package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"golang.org/x/time/rate"
)

// Logger returns a middleware that logs HTTP requests
func Logger(log logr.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info("HTTP request processed",
			"status", statusCode,
			"method", method,
			"path", path,
			"ip", clientIP,
			"latency", latency,
			"user-agent", c.Request.UserAgent(),
			"request-id", c.GetString("request-id"),
			"error", errorMessage,
		)
	}
}

// Authenticate validates JWT tokens and extracts user information
func Authenticate(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			return
		}

		// Validate token (implementation depends on auth provider)
		claims, err := validateToken(tokenString, config)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			return
		}

		// Set user information in context
		c.Set("user", claims.Subject)
		c.Set("groups", claims.Groups)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// Authorize checks if the authenticated user has permission for the requested resource
func Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user information from context
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "user not authenticated",
			})
			return
		}

		groups, _ := c.Get("groups")

		// Check permissions based on path and method
		if !checkPermission(user.(string), groups.([]string), c.Request.Method, c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// RateLimit implements rate limiting middleware
func RateLimit(limiter *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}