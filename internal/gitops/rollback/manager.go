package rollback

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// Manager handles rollback operations
type Manager struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewManager creates a new rollback manager
func NewManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *Manager {
	return &Manager{
		Client: client,
		Scheme: scheme,
		Log:    log.WithName("rollback-manager"),
	}
}

// RollbackPoint represents a point in time that can be rolled back to
type RollbackPoint struct {
	// Revision is the Git revision
	Revision string `json:"revision"`

	// Timestamp is when this revision was deployed
	Timestamp time.Time `json:"timestamp"`

	// Configuration is the platform configuration at this point
	Configuration *observabilityv1.ObservabilityPlatformSpec `json:"configuration"`

	// Status is the platform status at this point
	Status *observabilityv1.ObservabilityPlatformStatus `json:"status"`

	// Reason is why this rollback point was created
	Reason string `json:"reason"`

	// Success indicates if this was a successful deployment
	Success bool `json:"success"`

	// Metrics contains health metrics at this point
	Metrics *HealthMetrics `json:"metrics,omitempty"`
}

// HealthMetrics represents health metrics for a deployment
type HealthMetrics struct {
	// ComponentHealth maps component names to health status
	ComponentHealth map[string]string `json:"componentHealth"`

	// ErrorRate is the error rate percentage
	ErrorRate float64 `json:"errorRate"`

	// ResponseTime is the average response time in ms
	ResponseTime float64 `json:"responseTime"`

	// Availability is the availability percentage
	Availability float64 `json:"availability"`

	// CustomMetrics contains custom health metrics
	CustomMetrics map[string]float64 `json:"customMetrics,omitempty"`
}

// RollbackHistory represents the rollback history for a platform
type RollbackHistory struct {
	// Points are the rollback points
	Points []RollbackPoint `json:"points"`

	// CurrentRevision is the current deployed revision
	CurrentRevision string `json:"currentRevision"`

	// LastRollback contains information about the last rollback
	LastRollback *RollbackInfo `json:"lastRollback,omitempty"`
}

// RollbackInfo contains information about a rollback operation
type RollbackInfo struct {
	// FromRevision is the revision we rolled back from
	FromRevision string `json:"fromRevision"`

	// ToRevision is the revision we rolled back to
	ToRevision string `json:"toRevision"`

	// Reason is why the rollback was performed
	Reason string `json:"reason"`

	// Timestamp is when the rollback occurred
	Timestamp time.Time `json:"timestamp"`

	// Success indicates if the rollback was successful
	Success bool `json:"success"`

	// Message provides additional information
	Message string `json:"message,omitempty"`
}

// CreateRollbackPoint creates a new rollback point
func (m *Manager) CreateRollbackPoint(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, reason string) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.V(1).Info("Creating rollback point", "reason", reason)

	// Get current health metrics
	metrics, err := m.collectHealthMetrics(ctx, platform)
	if err != nil {
		log.Error(err, "Failed to collect health metrics")
		// Continue without metrics
	}

	// Create rollback point
	point := RollbackPoint{
		Revision:      platform.Status.GitOps.Revision,
		Timestamp:     time.Now(),
		Configuration: platform.Spec.DeepCopy(),
		Status:        platform.Status.DeepCopy(),
		Reason:        reason,
		Success:       platform.Status.Phase == observabilityv1.PlatformPhaseReady,
		Metrics:       metrics,
	}

	// Store rollback point in ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rollback-%d", platform.Name, time.Now().Unix()),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"gunj-operator.io/platform":       platform.Name,
				"gunj-operator.io/rollback-point": "true",
				"gunj-operator.io/revision":       point.Revision,
			},
			Annotations: map[string]string{
				"gunj-operator.io/timestamp": point.Timestamp.Format(time.RFC3339),
				"gunj-operator.io/reason":    reason,
				"gunj-operator.io/success":   fmt.Sprintf("%t", point.Success),
			},
		},
		Data: map[string]string{
			"configuration": m.serializeConfiguration(point.Configuration),
			"status":        m.serializeStatus(point.Status),
			"metrics":       m.serializeMetrics(metrics),
		},
	}

	if err := m.Client.Create(ctx, cm); err != nil {
		return fmt.Errorf("creating rollback point: %w", err)
	}

	// Clean up old rollback points
	if platform.Spec.GitOps.Rollback != nil && platform.Spec.GitOps.Rollback.MaxHistory > 0 {
		if err := m.cleanupOldRollbackPoints(ctx, platform, int(platform.Spec.GitOps.Rollback.MaxHistory)); err != nil {
			log.Error(err, "Failed to cleanup old rollback points")
		}
	}

	log.Info("Created rollback point", "revision", point.Revision)
	return nil
}

// GetHistory gets the rollback history for a platform
func (m *Manager) GetHistory(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) ([]RollbackPoint, error) {
	// List all rollback point ConfigMaps
	cmList := &corev1.ConfigMapList{}
	if err := m.Client.List(ctx, cmList, client.InNamespace(platform.Namespace),
		client.MatchingLabels{
			"gunj-operator.io/platform":       platform.Name,
			"gunj-operator.io/rollback-point": "true",
		}); err != nil {
		return nil, fmt.Errorf("listing rollback points: %w", err)
	}

	// Convert to RollbackPoint objects
	var points []RollbackPoint
	for _, cm := range cmList.Items {
		point := RollbackPoint{
			Revision: cm.Labels["gunj-operator.io/revision"],
			Reason:   cm.Annotations["gunj-operator.io/reason"],
		}

		if timestamp, err := time.Parse(time.RFC3339, cm.Annotations["gunj-operator.io/timestamp"]); err == nil {
			point.Timestamp = timestamp
		}

		if success := cm.Annotations["gunj-operator.io/success"]; success == "true" {
			point.Success = true
		}

		// Deserialize configuration and status
		// In a real implementation, we would properly deserialize these
		
		points = append(points, point)
	}

	// Sort by timestamp (newest first)
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.After(points[j].Timestamp)
	})

	return points, nil
}

// RollbackToRevision rolls back to a specific revision
func (m *Manager) RollbackToRevision(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string, reason string) error {
	log := m.Log.WithValues("platform", platform.Name, "revision", revision)
	log.Info("Rolling back to revision", "reason", reason)

	// Find the rollback point for this revision
	history, err := m.GetHistory(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting rollback history: %w", err)
	}

	var targetPoint *RollbackPoint
	for _, point := range history {
		if point.Revision == revision {
			targetPoint = &point
			break
		}
	}

	if targetPoint == nil {
		return fmt.Errorf("rollback point not found for revision %s", revision)
	}

	// Create rollback record
	rollbackInfo := &RollbackInfo{
		FromRevision: platform.Status.GitOps.Revision,
		ToRevision:   revision,
		Reason:       reason,
		Timestamp:    time.Now(),
	}

	// Store current state as a rollback point before rolling back
	if err := m.CreateRollbackPoint(ctx, platform, fmt.Sprintf("Before rollback to %s", revision)); err != nil {
		log.Error(err, "Failed to create pre-rollback point")
	}

	// Perform rollback based on provider
	switch platform.Spec.GitOps.Provider {
	case observabilityv1.GitOpsProviderArgoCD:
		err = m.rollbackArgoCD(ctx, platform, targetPoint)
	case observabilityv1.GitOpsProviderFlux:
		err = m.rollbackFlux(ctx, platform, targetPoint)
	default:
		err = fmt.Errorf("unsupported GitOps provider: %s", platform.Spec.GitOps.Provider)
	}

	if err != nil {
		rollbackInfo.Success = false
		rollbackInfo.Message = err.Error()
		m.storeRollbackInfo(ctx, platform, rollbackInfo)
		return fmt.Errorf("performing rollback: %w", err)
	}

	rollbackInfo.Success = true
	rollbackInfo.Message = "Rollback completed successfully"
	m.storeRollbackInfo(ctx, platform, rollbackInfo)

	log.Info("Rollback completed successfully")
	return nil
}

// CheckHealthAndRollback checks health and performs automatic rollback if needed
func (m *Manager) CheckHealthAndRollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name)
	log.V(1).Info("Checking health for automatic rollback")

	// Check if automatic rollback is enabled
	if platform.Spec.GitOps.Rollback == nil || !platform.Spec.GitOps.Rollback.Enabled {
		return nil
	}

	// Get health metrics
	metrics, err := m.collectHealthMetrics(ctx, platform)
	if err != nil {
		return fmt.Errorf("collecting health metrics: %w", err)
	}

	// Check if rollback is needed
	needsRollback, reason := m.evaluateRollbackCriteria(platform, metrics)
	if !needsRollback {
		log.V(1).Info("Health check passed, no rollback needed")
		return nil
	}

	log.Info("Health check failed, initiating automatic rollback", "reason", reason)

	// Get last successful deployment
	history, err := m.GetHistory(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting rollback history: %w", err)
	}

	var lastGoodRevision string
	for _, point := range history {
		if point.Success && point.Revision != platform.Status.GitOps.Revision {
			lastGoodRevision = point.Revision
			break
		}
	}

	if lastGoodRevision == "" {
		return fmt.Errorf("no successful revision found to rollback to")
	}

	// Perform rollback
	return m.RollbackToRevision(ctx, platform, lastGoodRevision, reason)
}

// RollbackPromotion rolls back a failed promotion
func (m *Manager) RollbackPromotion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, environment string, reason string) error {
	log := m.Log.WithValues("platform", platform.Name, "environment", environment)
	log.Info("Rolling back promotion", "reason", reason)

	// Find the promotion platform
	promotionPlatform := &observabilityv1.ObservabilityPlatform{}
	err := m.Client.Get(ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-%s", platform.Name, environment),
		Namespace: platform.Namespace,
	}, promotionPlatform)
	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Promotion platform not found, nothing to rollback")
			return nil
		}
		return fmt.Errorf("getting promotion platform: %w", err)
	}

	// Delete the promotion platform
	if err := m.Client.Delete(ctx, promotionPlatform); err != nil {
		return fmt.Errorf("deleting promotion platform: %w", err)
	}

	// Create rollback record
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-promotion-rollback-%d", platform.Name, time.Now().Unix()),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"gunj-operator.io/platform":           platform.Name,
				"gunj-operator.io/promotion-rollback": "true",
				"gunj-operator.io/environment":        environment,
			},
		},
		Data: map[string]string{
			"reason":    reason,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	if err := m.Client.Create(ctx, cm); err != nil {
		log.Error(err, "Failed to create rollback record")
	}

	log.Info("Promotion rollback completed")
	return nil
}

// collectHealthMetrics collects health metrics for the platform
func (m *Manager) collectHealthMetrics(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*HealthMetrics, error) {
	metrics := &HealthMetrics{
		ComponentHealth: make(map[string]string),
		CustomMetrics:   make(map[string]float64),
	}

	// Check component health
	if platform.Status.ComponentStatuses != nil {
		for component, status := range platform.Status.ComponentStatuses {
			metrics.ComponentHealth[component] = status.Phase
		}
	}

	// In a real implementation, we would:
	// 1. Query Prometheus for error rates, response times, etc.
	// 2. Check component endpoints
	// 3. Validate service availability

	// For now, use placeholder values
	metrics.ErrorRate = 0.1
	metrics.ResponseTime = 100
	metrics.Availability = 99.9

	return metrics, nil
}

// evaluateRollbackCriteria evaluates if rollback is needed based on health metrics
func (m *Manager) evaluateRollbackCriteria(platform *observabilityv1.ObservabilityPlatform, metrics *HealthMetrics) (bool, string) {
	// Check component health
	for component, health := range metrics.ComponentHealth {
		if health == string(observabilityv1.ComponentPhaseFailed) {
			return true, fmt.Sprintf("Component %s is in failed state", component)
		}
	}

	// Check error rate threshold (example: 5%)
	if metrics.ErrorRate > 5.0 {
		return true, fmt.Sprintf("Error rate too high: %.2f%%", metrics.ErrorRate)
	}

	// Check availability threshold (example: 95%)
	if metrics.Availability < 95.0 {
		return true, fmt.Sprintf("Availability too low: %.2f%%", metrics.Availability)
	}

	// Check response time threshold (example: 1000ms)
	if metrics.ResponseTime > 1000 {
		return true, fmt.Sprintf("Response time too high: %.2fms", metrics.ResponseTime)
	}

	return false, ""
}

// rollbackArgoCD performs rollback for ArgoCD
func (m *Manager) rollbackArgoCD(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, point *RollbackPoint) error {
	// In a real implementation, this would:
	// 1. Update the ArgoCD Application to use the target revision
	// 2. Trigger a sync
	// 3. Wait for sync to complete
	m.Log.V(1).Info("Performing ArgoCD rollback", "revision", point.Revision)
	return nil
}

// rollbackFlux performs rollback for Flux
func (m *Manager) rollbackFlux(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, point *RollbackPoint) error {
	// In a real implementation, this would:
	// 1. Update the GitRepository to use the target revision
	// 2. Trigger a reconciliation
	// 3. Wait for reconciliation to complete
	m.Log.V(1).Info("Performing Flux rollback", "revision", point.Revision)
	return nil
}

// cleanupOldRollbackPoints removes old rollback points beyond the max history
func (m *Manager) cleanupOldRollbackPoints(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, maxHistory int) error {
	history, err := m.GetHistory(ctx, platform)
	if err != nil {
		return err
	}

	if len(history) <= maxHistory {
		return nil
	}

	// Delete oldest points
	for i := maxHistory; i < len(history); i++ {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-rollback-%d", platform.Name, history[i].Timestamp.Unix()),
				Namespace: platform.Namespace,
			},
		}
		if err := m.Client.Delete(ctx, cm); err != nil && !errors.IsNotFound(err) {
			m.Log.Error(err, "Failed to delete old rollback point", "revision", history[i].Revision)
		}
	}

	return nil
}

// storeRollbackInfo stores rollback information
func (m *Manager) storeRollbackInfo(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, info *RollbackInfo) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rollback-info", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"gunj-operator.io/platform":      platform.Name,
				"gunj-operator.io/rollback-info": "true",
			},
		},
		Data: map[string]string{
			"fromRevision": info.FromRevision,
			"toRevision":   info.ToRevision,
			"reason":       info.Reason,
			"timestamp":    info.Timestamp.Format(time.RFC3339),
			"success":      fmt.Sprintf("%t", info.Success),
			"message":      info.Message,
		},
	}

	// Create or update
	existing := &corev1.ConfigMap{}
	err := m.Client.Get(ctx, client.ObjectKeyFromObject(cm), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			m.Client.Create(ctx, cm)
		}
	} else {
		existing.Data = cm.Data
		m.Client.Update(ctx, existing)
	}
}

// serializeConfiguration serializes platform configuration
func (m *Manager) serializeConfiguration(spec *observabilityv1.ObservabilityPlatformSpec) string {
	// In a real implementation, we would marshal to JSON/YAML
	return "serialized-configuration"
}

// serializeStatus serializes platform status
func (m *Manager) serializeStatus(status *observabilityv1.ObservabilityPlatformStatus) string {
	// In a real implementation, we would marshal to JSON/YAML
	return "serialized-status"
}

// serializeMetrics serializes health metrics
func (m *Manager) serializeMetrics(metrics *HealthMetrics) string {
	// In a real implementation, we would marshal to JSON
	return "serialized-metrics"
}
