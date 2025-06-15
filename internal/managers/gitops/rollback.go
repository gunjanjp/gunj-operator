package gitops

import (
	"context"
	"fmt"
	"strings"
	"time"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RollbackManager handles automated rollback scenarios
type RollbackManager struct {
	client        client.Client
	scheme        *runtime.Scheme
	gitOpsManager GitOpsManager
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(client client.Client, scheme *runtime.Scheme, gitOpsManager GitOpsManager) *RollbackManager {
	return &RollbackManager{
		client:        client,
		scheme:        scheme,
		gitOpsManager: gitOpsManager,
	}
}

// CheckAndRollback checks if rollback is needed and performs it
func (m *RollbackManager) CheckAndRollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx).WithName("rollback-manager")

	if !m.isRollbackEnabled(platform) {
		return nil
	}

	// Check if rollback conditions are met
	shouldRollback, reason := m.shouldRollback(ctx, platform)
	if !shouldRollback {
		return nil
	}

	log.Info("Rollback conditions met", "reason", reason)

	// Get last known good revision
	revision, err := m.getLastKnownGoodRevision(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting last known good revision: %w", err)
	}

	if revision == "" {
		log.Info("No previous good revision found, skipping rollback")
		return nil
	}

	// Perform rollback
	log.Info("Performing automatic rollback", "revision", revision)
	if err := m.gitOpsManager.Rollback(ctx, platform, revision); err != nil {
		return fmt.Errorf("performing rollback: %w", err)
	}

	// Update rollback status
	if err := m.updateRollbackStatus(ctx, platform, revision, reason); err != nil {
		return fmt.Errorf("updating rollback status: %w", err)
	}

	// Send notification
	if err := m.sendRollbackNotification(ctx, platform, revision, reason); err != nil {
		log.Error(err, "Failed to send rollback notification")
	}

	return nil
}

// isRollbackEnabled checks if automatic rollback is enabled
func (m *RollbackManager) isRollbackEnabled(platform *observabilityv1beta1.ObservabilityPlatform) bool {
	return platform.Spec.GitOps != nil &&
		platform.Spec.GitOps.RollbackConfig != nil &&
		platform.Spec.GitOps.RollbackConfig.AutoRollback
}

// shouldRollback determines if rollback conditions are met
func (m *RollbackManager) shouldRollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (bool, string) {
	log := ctrl.LoggerFrom(ctx)
	config := platform.Spec.GitOps.RollbackConfig

	// Check if we're within the detection window
	window, err := time.ParseDuration(config.Window)
	if err != nil {
		log.Error(err, "Invalid rollback window duration")
		window = 30 * time.Minute // default
	}

	// Get recent events
	events, err := m.getRecentEvents(ctx, platform, window)
	if err != nil {
		log.Error(err, "Failed to get recent events")
		return false, ""
	}

	// Count failures
	failureCount := 0
	for _, event := range events {
		if event.Type == corev1.EventTypeWarning && isFailureEvent(event) {
			failureCount++
		}
	}

	if failureCount >= config.FailureThreshold {
		return true, fmt.Sprintf("Failure threshold exceeded: %d failures in %s", failureCount, window)
	}

	// Check component health
	unhealthyComponents := m.checkComponentHealth(ctx, platform)
	if len(unhealthyComponents) > 0 {
		return true, fmt.Sprintf("Unhealthy components detected: %v", unhealthyComponents)
	}

	// Check sync status
	syncStatus, err := m.gitOpsManager.GetSyncStatus(ctx, platform)
	if err == nil && syncStatus.SyncState == SyncStateError {
		return true, fmt.Sprintf("GitOps sync error: %s", syncStatus.Message)
	}

	return false, ""
}

// getLastKnownGoodRevision retrieves the last successful revision
func (m *RollbackManager) getLastKnownGoodRevision(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (string, error) {
	// Check ConfigMap for revision history
	historyConfigMap := &corev1.ConfigMap{}
	historyKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-gitops-history", platform.Name),
		Namespace: platform.Namespace,
	}

	err := m.client.Get(ctx, historyKey, historyConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create history ConfigMap if it doesn't exist
			historyConfigMap = m.createHistoryConfigMap(platform)
			if err := m.client.Create(ctx, historyConfigMap); err != nil {
				return "", fmt.Errorf("creating history ConfigMap: %w", err)
			}
			return "", nil
		}
		return "", fmt.Errorf("getting history ConfigMap: %w", err)
	}

	// Find last good revision
	if historyConfigMap.Data != nil {
		if lastGood, ok := historyConfigMap.Data["lastKnownGood"]; ok {
			return lastGood, nil
		}
	}

	return "", nil
}

// updateRollbackStatus updates the platform status with rollback information
func (m *RollbackManager) updateRollbackStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision, reason string) error {
	platform.Status.LastRollback = &observabilityv1beta1.RollbackStatus{
		Revision:  revision,
		Timestamp: time.Now().Format(time.RFC3339),
		Reason:    reason,
		Automatic: true,
	}

	// Add condition
	condition := metav1.Condition{
		Type:               "RollbackPerformed",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "AutomaticRollback",
		Message:            fmt.Sprintf("Automatically rolled back to revision %s: %s", revision, reason),
	}
	
	meta.SetStatusCondition(&platform.Status.Conditions, condition)

	return m.client.Status().Update(ctx, platform)
}

// RecordSuccessfulDeployment records a successful deployment for future rollback
func (m *RollbackManager) RecordSuccessfulDeployment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("revision", revision)
	log.Info("Recording successful deployment")

	// Get or create history ConfigMap
	historyConfigMap := &corev1.ConfigMap{}
	historyKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-gitops-history", platform.Name),
		Namespace: platform.Namespace,
	}

	err := m.client.Get(ctx, historyKey, historyConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			historyConfigMap = m.createHistoryConfigMap(platform)
		} else {
			return fmt.Errorf("getting history ConfigMap: %w", err)
		}
	}

	// Update revision history
	if historyConfigMap.Data == nil {
		historyConfigMap.Data = make(map[string]string)
	}

	// Record as last known good
	historyConfigMap.Data["lastKnownGood"] = revision
	historyConfigMap.Data["lastKnownGoodTime"] = time.Now().Format(time.RFC3339)

	// Maintain revision history
	m.updateRevisionHistory(historyConfigMap, revision, platform.Spec.GitOps.RollbackConfig.MaxHistory)

	// Create or update ConfigMap
	if historyConfigMap.ResourceVersion == "" {
		if err := m.client.Create(ctx, historyConfigMap); err != nil {
			return fmt.Errorf("creating history ConfigMap: %w", err)
		}
	} else {
		if err := m.client.Update(ctx, historyConfigMap); err != nil {
			return fmt.Errorf("updating history ConfigMap: %w", err)
		}
	}

	return nil
}

// Helper methods

func (m *RollbackManager) getRecentEvents(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, window time.Duration) ([]corev1.Event, error) {
	eventList := &corev1.EventList{}
	
	// List events for the platform
	if err := m.client.List(ctx, eventList, client.InNamespace(platform.Namespace)); err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}

	// Filter recent events
	cutoff := time.Now().Add(-window)
	var recentEvents []corev1.Event
	
	for _, event := range eventList.Items {
		if event.InvolvedObject.Name == platform.Name || 
		   (event.InvolvedObject.Name != "" && event.InvolvedObject.FieldPath == fmt.Sprintf("spec.platforms[%s]", platform.Name)) {
			if event.LastTimestamp.After(cutoff) {
				recentEvents = append(recentEvents, event)
			}
		}
	}

	return recentEvents, nil
}

func (m *RollbackManager) checkComponentHealth(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) []string {
	var unhealthy []string

	// Check each component's health
	if platform.Status.ComponentStatuses != nil {
		for component, status := range platform.Status.ComponentStatuses {
			if status.Health != "Healthy" && status.Health != "" {
				unhealthy = append(unhealthy, component)
			}
		}
	}

	return unhealthy
}

func (m *RollbackManager) createHistoryConfigMap(platform *observabilityv1beta1.ObservabilityPlatform) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-gitops-history", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":   platform.Name,
			},
		},
		Data: map[string]string{
			"created": time.Now().Format(time.RFC3339),
		},
	}
}

func (m *RollbackManager) updateRevisionHistory(configMap *corev1.ConfigMap, revision string, maxHistory int) {
	// Parse existing history
	var history []string
	if historyData, ok := configMap.Data["history"]; ok {
		history = parseHistory(historyData)
	}

	// Add new revision
	entry := fmt.Sprintf("%s|%s", time.Now().Format(time.RFC3339), revision)
	history = append(history, entry)

	// Trim to max history
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	// Update ConfigMap
	configMap.Data["history"] = formatHistory(history)
}

func (m *RollbackManager) sendRollbackNotification(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision, reason string) error {
	// Create an event for the rollback
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rollback-%d", platform.Name, time.Now().Unix()),
			Namespace: platform.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "ObservabilityPlatform",
			Name:       platform.Name,
			Namespace:  platform.Namespace,
			UID:        platform.UID,
			APIVersion: observabilityv1beta1.GroupVersion.String(),
		},
		Type:    corev1.EventTypeWarning,
		Reason:  "AutomaticRollback",
		Message: fmt.Sprintf("Automatically rolled back to revision %s: %s", revision, reason),
		Source: corev1.EventSource{
			Component: "gunj-operator-rollback-manager",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	return m.client.Create(ctx, event)
}

// Helper functions

func isFailureEvent(event corev1.Event) bool {
	failureReasons := []string{
		"Failed",
		"BackOff",
		"CrashLoopBackOff",
		"Error",
		"Unhealthy",
		"FailedSync",
		"ReconcileError",
	}

	for _, reason := range failureReasons {
		if event.Reason == reason {
			return true
		}
	}

	return false
}

func parseHistory(data string) []string {
	// Simple newline-separated format
	if data == "" {
		return []string{}
	}
	return strings.Split(data, "\n")
}

func formatHistory(history []string) string {
	return strings.Join(history, "\n")
}
