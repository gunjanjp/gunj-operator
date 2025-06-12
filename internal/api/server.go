// Package api provides the REST and GraphQL API server for Gunj Operator
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/time/rate"

	"github.com/gunjanjp/gunj-operator/internal/api/handlers"
	"github.com/gunjanjp/gunj-operator/internal/api/middleware"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Server represents the API server
type Server struct {
	router     *gin.Engine
	client     client.Client
	log        logr.Logger
	config     *Config
	httpServer *http.Server
}

// Config holds API server configuration
type Config struct {
	Port              int
	TLSEnabled        bool
	TLSCertPath       string
	TLSKeyPath        string
	CORSAllowOrigins  []string
	RateLimitRPS      int
	EnableGraphQL     bool
	EnableWebSocket   bool
	EnableSSE         bool
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// NewServer creates a new API server instance
func NewServer(client client.Client, log logr.Logger, config *Config) *Server {
	// Set Gin mode based on environment
	if config.EnableDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	return &Server{
		router: router,
		client: client,
		log:    log.WithName("api-server"),
		config: config,
	}
}

// setupMiddleware configures all middleware for the API server
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request ID middleware
	s.router.Use(requestid.New())

	// OpenTelemetry instrumentation
	s.router.Use(otelgin.Middleware("gunj-operator-api"))

	// Structured logging middleware
	s.router.Use(middleware.Logger(s.log))

	// CORS configuration
	corsConfig := cors.Config{
		AllowOrigins:     s.config.CORSAllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	s.router.Use(cors.New(corsConfig))

	// Rate limiting
	limiter := rate.NewLimiter(rate.Limit(s.config.RateLimitRPS), s.config.RateLimitRPS*2)
	s.router.Use(middleware.RateLimit(limiter))

	// Authentication middleware (applied to specific routes)
	// Authorization middleware (applied after authentication)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health and readiness endpoints
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/ready", s.handleReady)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Apply authentication middleware to API routes
		v1.Use(middleware.Authenticate(s.config))
		v1.Use(middleware.Authorize())

		// Platform management
		platforms := v1.Group("/platforms")
		{
			platforms.GET("", handlers.ListPlatforms(s.client))
			platforms.POST("", handlers.CreatePlatform(s.client))
			platforms.GET("/:name", handlers.GetPlatform(s.client))
			platforms.PUT("/:name", handlers.UpdatePlatform(s.client))
			platforms.DELETE("/:name", handlers.DeletePlatform(s.client))
			platforms.PATCH("/:name", handlers.PatchPlatform(s.client))

			// Platform operations
			platforms.POST("/:name/operations/backup", handlers.BackupPlatform(s.client))
			platforms.POST("/:name/operations/restore", handlers.RestorePlatform(s.client))
			platforms.POST("/:name/operations/upgrade", handlers.UpgradePlatform(s.client))

			// Platform metrics and health
			platforms.GET("/:name/metrics", handlers.GetPlatformMetrics(s.client))
			platforms.GET("/:name/health", handlers.GetPlatformHealth(s.client))

			// Component management
			platforms.GET("/:name/components", handlers.ListComponents(s.client))
			platforms.PUT("/:name/components/:component", handlers.UpdateComponent(s.client))
		}

		// Alerting rules
		alerts := v1.Group("/alerts")
		{
			alerts.GET("", handlers.ListAlertingRules(s.client))
			alerts.POST("", handlers.CreateAlertingRule(s.client))
			alerts.GET("/:name", handlers.GetAlertingRule(s.client))
			alerts.PUT("/:name", handlers.UpdateAlertingRule(s.client))
			alerts.DELETE("/:name", handlers.DeleteAlertingRule(s.client))
		}

		// Dashboards
		dashboards := v1.Group("/dashboards")
		{
			dashboards.GET("", handlers.ListDashboards(s.client))
			dashboards.POST("", handlers.CreateDashboard(s.client))
			dashboards.GET("/:name", handlers.GetDashboard(s.client))
			dashboards.PUT("/:name", handlers.UpdateDashboard(s.client))
			dashboards.DELETE("/:name", handlers.DeleteDashboard(s.client))
		}
	}

	// GraphQL endpoint (if enabled)
	if s.config.EnableGraphQL {
		s.setupGraphQL()
	}

	// WebSocket endpoint (if enabled)
	if s.config.EnableWebSocket {
		s.setupWebSocket()
	}

	// Server-Sent Events (if enabled)
	if s.config.EnableSSE {
		s.setupSSE()
	}
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	s.setupMiddleware()
	s.setupRoutes()

	// Configure HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// Start server
	if s.config.TLSEnabled {
		s.log.Info("Starting HTTPS server", "port", s.config.Port)
		return s.httpServer.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
	}

	s.log.Info("Starting HTTP server", "port", s.config.Port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down API server")

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": "v2.0.0",
		"time":    time.Now().UTC(),
	})
}

// handleReady handles readiness check requests
func (s *Server) handleReady(c *gin.Context) {
	// Check if we can connect to Kubernetes API
	if err := s.client.List(c.Request.Context(), &client.ListOptions{}); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "cannot connect to Kubernetes API",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC(),
	})
}
