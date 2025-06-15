package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops/metrics"
)

// GitOpsHandler handles GitOps-related API endpoints
type GitOpsHandler struct {
	client   client.Client
	log      logr.Logger
	recorder *metrics.MetricsRecorder
}

// NewGitOpsHandler creates a new GitOpsHandler
func NewGitOpsHandler(client client.Client, log logr.Logger) *GitOpsHandler {
	return &GitOpsHandler{
		client:   client,
		log:      log.WithName("gitops-api"),
		recorder: metrics.NewMetricsRecorder(),
	}
}

// RegisterRoutes registers GitOps API routes
func (h *GitOpsHandler) RegisterRoutes(router *gin.RouterGroup) {
	gitops := router.Group("/gitops")
	{
		// GitOps deployment endpoints
		gitops.GET("/deployments", h.listGitOpsDeployments)
		gitops.GET("/deployments/:namespace/:name", h.getGitOpsDeployment)
		gitops.POST("/deployments", h.createGitOpsDeployment)
		gitops.PUT("/deployments/:namespace/:name", h.updateGitOpsDeployment)
		gitops.DELETE("/deployments/:namespace/:name", h.deleteGitOpsDeployment)
		gitops.POST("/deployments/:namespace/:name/sync", h.syncGitOpsDeployment)
		gitops.GET("/deployments/:namespace/:name/status", h.getGitOpsStatus)
		gitops.GET("/deployments/:namespace/:name/history", h.getGitOpsHistory)
		gitops.POST("/deployments/:namespace/:name/rollback", h.rollbackGitOpsDeployment)
		
		// Promotion endpoints
		gitops.GET("/promotions", h.listPromotions)
		gitops.GET("/promotions/:namespace/:name", h.getPromotion)
		gitops.POST("/promotions", h.createPromotion)
		gitops.POST("/promotions/:namespace/:name/approve", h.approvePromotion)
		gitops.POST("/promotions/:namespace/:name/reject", h.rejectPromotion)
		
		// Dashboard endpoints
		gitops.GET("/dashboard", h.getGitOpsDashboard)
		gitops.GET("/dashboard/summary", h.getGitOpsSummary)
		gitops.GET("/dashboard/health", h.getGitOpsHealth)
		gitops.GET("/dashboard/metrics", h.getGitOpsMetrics)
		gitops.GET("/dashboard/events", h.getGitOpsEvents)
		
		// Webhook endpoints
		gitops.POST("/webhook/:provider", h.handleWebhook)
		gitops.GET("/webhook/config/:namespace/:name", h.getWebhookConfig)
		
		// Drift detection endpoints
		gitops.GET("/drift/:namespace/:name", h.getDriftStatus)
		gitops.POST("/drift/:namespace/:name/correct", h.correctDrift)
	}
}

// GitOps deployment handlers

func (h *GitOpsHandler) listGitOpsDeployments(c *gin.Context) {
	namespace := c.Query("namespace")
	labelSelector := c.Query("labelSelector")
	
	list := &observabilityv1.GitOpsDeploymentList{}
	opts := []client.ListOption{}
	
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	
	if labelSelector != "" {
		selector, err := labels.Parse(labelSelector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid label selector",
				"details": err.Error(),
			})
			return
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}
	
	if err := h.client.List(c.Request.Context(), list, opts...); err != nil {
		h.log.Error(err, "Failed to list GitOps deployments")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list GitOps deployments",
		})
		return
	}
	
	c.JSON(http.StatusOK, list)
}

func (h *GitOpsHandler) getGitOpsDeployment(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	gitops := &observabilityv1.GitOpsDeployment{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	
	if err := h.client.Get(c.Request.Context(), key, gitops); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "GitOps deployment not found",
			})
			return
		}
		h.log.Error(err, "Failed to get GitOps deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get GitOps deployment",
		})
		return
	}
	
	c.JSON(http.StatusOK, gitops)
}

func (h *GitOpsHandler) createGitOpsDeployment(c *gin.Context) {
	var gitops observabilityv1.GitOpsDeployment
	if err := c.ShouldBindJSON(&gitops); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}
	
	if err := h.client.Create(c.Request.Context(), &gitops); err != nil {
		if errors.IsAlreadyExists(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "GitOps deployment already exists",
			})
			return
		}
		h.log.Error(err, "Failed to create GitOps deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create GitOps deployment",
		})
		return
	}
	
	c.JSON(http.StatusCreated, gitops)
}

func (h *GitOpsHandler) updateGitOpsDeployment(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	var updates observabilityv1.GitOpsDeployment
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}
	
	gitops := &observabilityv1.GitOpsDeployment{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	
	if err := h.client.Get(c.Request.Context(), key, gitops); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "GitOps deployment not found",
			})
			return
		}
		h.log.Error(err, "Failed to get GitOps deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get GitOps deployment",
		})
		return
	}
	
	// Update spec
	gitops.Spec = updates.Spec
	
	if err := h.client.Update(c.Request.Context(), gitops); err != nil {
		h.log.Error(err, "Failed to update GitOps deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update GitOps deployment",
		})
		return
	}
	
	c.JSON(http.StatusOK, gitops)
}

func (h *GitOpsHandler) deleteGitOpsDeployment(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	gitops := &observabilityv1.GitOpsDeployment{}
	gitops.Namespace = namespace
	gitops.Name = name
	
	if err := h.client.Delete(c.Request.Context(), gitops); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "GitOps deployment not found",
			})
			return
		}
		h.log.Error(err, "Failed to delete GitOps deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete GitOps deployment",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "GitOps deployment deleted successfully",
	})
}

func (h *GitOpsHandler) syncGitOpsDeployment(c *gin.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")
	
	gitops := &observabilityv1.GitOpsDeployment{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	
	if err := h.client.Get(c.Request.Context(), key, gitops); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "GitOps deployment not found",
			})
			return
		}
		h.log.Error(err, "Failed to get GitOps deployment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get GitOps deployment",
		})
		return
	}
	
	// Trigger sync by updating annotation
	if gitops.Annotations == nil {
		gitops.Annotations = make(map[string]string)
	}
	gitops.Annotations["observability.io/sync-requested"] = time.Now().Format(time.RFC3339)
	
	if err := h.client.Update(c.Request.Context(), gitops); err != nil {
		h.log.Error(err, "Failed to trigger sync")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to trigger sync",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Sync triggered successfully",
		"syncTime": gitops.Annotations["observability.io/sync-requested"],
	})
}

// Dashboard handlers

func (h *GitOpsHandler) getGitOpsDashboard(c *gin.Context) {
	ctx := c.Request.Context()
	namespace := c.Query("namespace")
	
	dashboard := &GitOpsDashboard{
		Timestamp: time.Now(),
		Summary:   &GitOpsSummary{},
		Health:    &GitOpsHealth{},
	}
	
	// Get all GitOps deployments
	list := &observabilityv1.GitOpsDeploymentList{}
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	
	if err := h.client.List(ctx, list, opts...); err != nil {
		h.log.Error(err, "Failed to list GitOps deployments")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get dashboard data",
		})
		return
	}
	
	// Calculate summary
	dashboard.Summary.TotalDeployments = len(list.Items)
	syncProviderCount := make(map[string]int)
	phaseCount := make(map[string]int)
	
	for _, gitops := range list.Items {
		// Count by sync provider
		provider := string(gitops.Spec.SyncProvider)
		syncProviderCount[provider]++
		
		// Count by phase
		phase := string(gitops.Status.Phase)
		phaseCount[phase]++
		
		// Check health
		if gitops.Status.Phase == observabilityv1.GitOpsPhaseReady {
			dashboard.Health.HealthyDeployments++
		} else if gitops.Status.Phase == observabilityv1.GitOpsPhaseFailed {
			dashboard.Health.FailedDeployments++
		}
		
		// Check sync status
		if gitops.Status.LastSyncedTime != nil {
			syncAge := time.Since(gitops.Status.LastSyncedTime.Time)
			if syncAge > 24*time.Hour {
				dashboard.Health.OutOfSyncDeployments++
			}
		}
	}
	
	dashboard.Summary.ByProvider = syncProviderCount
	dashboard.Summary.ByPhase = phaseCount
	dashboard.Health.TotalDeployments = dashboard.Summary.TotalDeployments
	
	// Get recent events
	events, err := h.getRecentEvents(ctx, namespace, 10)
	if err != nil {
		h.log.Error(err, "Failed to get recent events")
	} else {
		dashboard.RecentEvents = events
	}
	
	// Get active promotions
	promotions := &observabilityv1.GitOpsPromotionList{}
	if err := h.client.List(ctx, promotions, opts...); err != nil {
		h.log.Error(err, "Failed to list promotions")
	} else {
		for _, promo := range promotions.Items {
			if promo.Status.Phase == observabilityv1.PromotionPhaseProgressing ||
				promo.Status.Phase == observabilityv1.PromotionPhasePendingApproval {
				dashboard.Summary.ActivePromotions++
			}
		}
	}
	
	c.JSON(http.StatusOK, dashboard)
}

func (h *GitOpsHandler) getGitOpsSummary(c *gin.Context) {
	ctx := c.Request.Context()
	namespace := c.Query("namespace")
	
	summary := &GitOpsSummary{}
	
	// Get deployments
	list := &observabilityv1.GitOpsDeploymentList{}
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	
	if err := h.client.List(ctx, list, opts...); err != nil {
		h.log.Error(err, "Failed to list GitOps deployments")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get summary",
		})
		return
	}
	
	summary.TotalDeployments = len(list.Items)
	summary.ByProvider = make(map[string]int)
	summary.ByPhase = make(map[string]int)
	
	for _, gitops := range list.Items {
		provider := string(gitops.Spec.SyncProvider)
		summary.ByProvider[provider]++
		
		phase := string(gitops.Status.Phase)
		summary.ByPhase[phase]++
	}
	
	// Get promotions
	promotions := &observabilityv1.GitOpsPromotionList{}
	if err := h.client.List(ctx, promotions, opts...); err != nil {
		h.log.Error(err, "Failed to list promotions")
	} else {
		for _, promo := range promotions.Items {
			if promo.Status.Phase == observabilityv1.PromotionPhaseProgressing ||
				promo.Status.Phase == observabilityv1.PromotionPhasePendingApproval {
				summary.ActivePromotions++
			}
		}
		summary.TotalPromotions = len(promotions.Items)
	}
	
	// Get drift status
	for _, gitops := range list.Items {
		if gitops.Status.DriftStatus != nil && gitops.Status.DriftStatus.DriftDetected {
			summary.DriftedDeployments++
		}
	}
	
	c.JSON(http.StatusOK, summary)
}

func (h *GitOpsHandler) getGitOpsHealth(c *gin.Context) {
	ctx := c.Request.Context()
	namespace := c.Query("namespace")
	
	health := &GitOpsHealth{
		Timestamp: time.Now(),
	}
	
	list := &observabilityv1.GitOpsDeploymentList{}
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	
	if err := h.client.List(ctx, list, opts...); err != nil {
		h.log.Error(err, "Failed to list GitOps deployments")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get health status",
		})
		return
	}
	
	health.TotalDeployments = len(list.Items)
	health.Details = make([]DeploymentHealth, 0, len(list.Items))
	
	for _, gitops := range list.Items {
		detail := DeploymentHealth{
			Name:      gitops.Name,
			Namespace: gitops.Namespace,
			Phase:     string(gitops.Status.Phase),
			Healthy:   gitops.Status.Phase == observabilityv1.GitOpsPhaseReady,
		}
		
		// Check sync status
		if gitops.Status.LastSyncedTime != nil {
			detail.LastSync = gitops.Status.LastSyncedTime.Time
			syncAge := time.Since(detail.LastSync)
			detail.SyncStatus = "Synced"
			if syncAge > 24*time.Hour {
				detail.SyncStatus = "OutOfSync"
				health.OutOfSyncDeployments++
			}
		} else {
			detail.SyncStatus = "Unknown"
		}
		
		// Check health
		if detail.Healthy {
			health.HealthyDeployments++
		} else if gitops.Status.Phase == observabilityv1.GitOpsPhaseFailed {
			health.FailedDeployments++
			detail.Error = gitops.Status.Message
		}
		
		// Check drift
		if gitops.Status.DriftStatus != nil && gitops.Status.DriftStatus.DriftDetected {
			detail.DriftDetected = true
			detail.DriftCount = gitops.Status.DriftStatus.DriftedResources
		}
		
		health.Details = append(health.Details, detail)
	}
	
	// Calculate health score (0-100)
	if health.TotalDeployments > 0 {
		health.HealthScore = float64(health.HealthyDeployments) / float64(health.TotalDeployments) * 100
	}
	
	c.JSON(http.StatusOK, health)
}

func (h *GitOpsHandler) getGitOpsMetrics(c *gin.Context) {
	namespace := c.Query("namespace")
	timeRange := c.Query("range") // e.g., "1h", "24h", "7d"
	
	// This would typically query Prometheus or another metrics backend
	// For now, we'll return a mock response
	metrics := &GitOpsMetrics{
		Timestamp: time.Now(),
		Range:     timeRange,
		Namespace: namespace,
		Metrics: map[string]interface{}{
			"sync_total": map[string]interface{}{
				"success": 145,
				"failed":  12,
			},
			"sync_duration_seconds": map[string]interface{}{
				"p50": 2.5,
				"p90": 5.2,
				"p99": 12.8,
			},
			"drift_detected_total": 23,
			"rollback_total":       3,
			"promotion_total": map[string]interface{}{
				"completed": 18,
				"failed":    2,
				"pending":   1,
			},
			"webhook_events_total": map[string]interface{}{
				"github":    89,
				"gitlab":    45,
				"bitbucket": 12,
			},
		},
	}
	
	c.JSON(http.StatusOK, metrics)
}

func (h *GitOpsHandler) getGitOpsEvents(c *gin.Context) {
	namespace := c.Query("namespace")
	limitStr := c.Query("limit")
	
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	events, err := h.getRecentEvents(c.Request.Context(), namespace, limit)
	if err != nil {
		h.log.Error(err, "Failed to get events")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get events",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// Helper functions

func (h *GitOpsHandler) getRecentEvents(ctx context.Context, namespace string, limit int) ([]GitOpsEvent, error) {
	// This is a simplified implementation
	// In production, you would query actual Kubernetes events
	events := []GitOpsEvent{
		{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Type:      "Sync",
			Level:     "Info",
			Message:   "Successfully synced GitOps deployment 'production'",
			Source:    "GitOpsController",
			Object:    "gitopsdeployment/production",
		},
		{
			Timestamp: time.Now().Add(-15 * time.Minute),
			Type:      "Drift",
			Level:     "Warning",
			Message:   "Drift detected in 3 resources",
			Source:    "DriftDetector",
			Object:    "gitopsdeployment/staging",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Hour),
			Type:      "Promotion",
			Level:     "Info",
			Message:   "Promotion from dev to staging completed",
			Source:    "PromotionController",
			Object:    "gitopspromotion/dev-to-staging",
		},
	}
	
	if len(events) > limit {
		events = events[:limit]
	}
	
	return events, nil
}

// Response types

type GitOpsDashboard struct {
	Timestamp    time.Time       `json:"timestamp"`
	Summary      *GitOpsSummary  `json:"summary"`
	Health       *GitOpsHealth   `json:"health"`
	RecentEvents []GitOpsEvent   `json:"recentEvents,omitempty"`
}

type GitOpsSummary struct {
	TotalDeployments   int            `json:"totalDeployments"`
	ByProvider         map[string]int `json:"byProvider"`
	ByPhase            map[string]int `json:"byPhase"`
	ActivePromotions   int            `json:"activePromotions"`
	TotalPromotions    int            `json:"totalPromotions"`
	DriftedDeployments int            `json:"driftedDeployments"`
}

type GitOpsHealth struct {
	Timestamp            time.Time           `json:"timestamp"`
	TotalDeployments     int                 `json:"totalDeployments"`
	HealthyDeployments   int                 `json:"healthyDeployments"`
	FailedDeployments    int                 `json:"failedDeployments"`
	OutOfSyncDeployments int                 `json:"outOfSyncDeployments"`
	HealthScore          float64             `json:"healthScore"`
	Details              []DeploymentHealth  `json:"details,omitempty"`
}

type DeploymentHealth struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	Phase         string    `json:"phase"`
	Healthy       bool      `json:"healthy"`
	SyncStatus    string    `json:"syncStatus"`
	LastSync      time.Time `json:"lastSync,omitempty"`
	DriftDetected bool      `json:"driftDetected"`
	DriftCount    int       `json:"driftCount,omitempty"`
	Error         string    `json:"error,omitempty"`
}

type GitOpsMetrics struct {
	Timestamp time.Time              `json:"timestamp"`
	Range     string                 `json:"range"`
	Namespace string                 `json:"namespace,omitempty"`
	Metrics   map[string]interface{} `json:"metrics"`
}

type GitOpsEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	Object    string    `json:"object,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// Additional endpoint handlers would go here...
func (h *GitOpsHandler) getGitOpsStatus(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) getGitOpsHistory(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) rollbackGitOpsDeployment(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) listPromotions(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) getPromotion(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) createPromotion(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) approvePromotion(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) rejectPromotion(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) handleWebhook(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) getWebhookConfig(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) getDriftStatus(c *gin.Context) {
	// Implementation
}

func (h *GitOpsHandler) correctDrift(c *gin.Context) {
	// Implementation
}
