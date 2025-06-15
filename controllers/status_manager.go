/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// StatusManager handles all status updates for ObservabilityPlatform
type StatusManager struct {
	client           client.Client
	log              logr.Logger
	conditionUtils   *ConditionUtils
	eventRecorder    *EnhancedEventRecorder
	updateQueue      chan statusUpdate
	metricsCollector *MetricsCollector
}

// statusUpdate represents a queued status update
type statusUpdate struct {
	platform *observabilityv1beta1.ObservabilityPlatform
	updateFn func(*observabilityv1beta1.ObservabilityPlatformStatus)
	ctx      context.Context
}

// ComponentStatus represents the status of a single component
type ComponentStatus struct {
	Name         string
	Ready        bool
	Version      string
	Replicas     int32
	ReadyReplicas int32
	Message      string
	LastUpdated  time.Time
	Resources    ResourceStatus
}

// ResourceStatus represents resource usage
type ResourceStatus struct {
	CPUUsage    string
	MemoryUsage string
	StorageUsage string
}

// ProgressInfo tracks progress for long-running operations
type ProgressInfo struct {
	Operation   string
	StartTime   time.Time
	Current     int
	Total       int
	Message     string
	SubTasks    []SubTaskProgress
}

// SubTaskProgress represents progress of a sub-task
type SubTaskProgress struct {
	Name      string
	Status    string
	Completed bool
	Message   string
}

// MetricsCollector collects metrics for status reporting
type MetricsCollector struct {
	reconciliationDuration time.Duration
	componentStats         map[string]ComponentMetrics
	errorCount             int
	lastReconcileTime      time.Time
}

// ComponentMetrics holds metrics for a component
type ComponentMetrics struct {
	DeploymentTime time.Duration
	RestartCount   int
	LastError      string
	HealthScore    float64
}

// NewStatusManager creates a new status manager
func NewStatusManager(client client.Client, log logr.Logger, eventRecorder *EnhancedEventRecorder) *StatusManager {
	sm := &StatusManager{
		client:         client,
		log:            log,
		conditionUtils: NewConditionUtils(),
		eventRecorder:  eventRecorder,
		updateQueue:    make(chan statusUpdate, 100),
		metricsCollector: &MetricsCollector{
			componentStats: make(map[string]ComponentMetrics),
		},
	}

	// Start the update processor
	go sm.processUpdates()

	return sm
}

// UpdatePlatformStatus updates the platform status with retries
func (sm *StatusManager) UpdatePlatformStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, updateFn func(*observabilityv1beta1.ObservabilityPlatformStatus)) error {
	// Queue the update
	select {
	case sm.updateQueue <- statusUpdate{
		platform: platform,
		updateFn: updateFn,
		ctx:      ctx,
	}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetCondition sets a condition on the platform
func (sm *StatusManager) SetCondition(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, condType string, status metav1.ConditionStatus, reason, message string) error {
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		condition := metav1.Condition{
			Type:               condType,
			Status:             status,
			ObservedGeneration: platform.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		}
		status.Conditions = sm.conditionUtils.SetCondition(status.Conditions, condition)
	})
}

// SetComponentStatus updates the status for a specific component
func (sm *StatusManager) SetComponentStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, component string, ready bool, message string) error {
	conditionType := ""
	eventReason := EventReasonComponentReady
	
	switch component {
	case "prometheus":
		conditionType = ConditionPrometheusReady
	case "grafana":
		conditionType = ConditionGrafanaReady
	case "loki":
		conditionType = ConditionLokiReady
	case "tempo":
		conditionType = ConditionTempoReady
	default:
		return fmt.Errorf("unknown component: %s", component)
	}

	status := metav1.ConditionFalse
	reason := ReasonComponentFailed
	if ready {
		status = metav1.ConditionTrue
		reason = ReasonComponentReady
	} else {
		eventReason = EventReasonComponentFailed
	}

	// Record event
	sm.eventRecorder.RecordComponentEvent(platform, component, eventReason, message)

	// Update condition
	return sm.SetCondition(ctx, platform, conditionType, status, reason, message)
}

// UpdateProgress updates progress for long-running operations
func (sm *StatusManager) UpdateProgress(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, operation string, current, total int, message string) error {
	// Record progress event
	sm.eventRecorder.RecordProgressEvent(platform, operation, current, total, message)

	// Update status
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		if status.Progress == nil {
			status.Progress = &observabilityv1beta1.ProgressStatus{}
		}
		
		status.Progress.Operation = operation
		status.Progress.Current = int32(current)
		status.Progress.Total = int32(total)
		status.Progress.Message = message
		status.Progress.LastUpdated = metav1.Now()
		
		// Calculate percentage
		if total > 0 {
			status.Progress.Percentage = int32((current * 100) / total)
		}
	})
}

// CompleteOperation marks an operation as complete
func (sm *StatusManager) CompleteOperation(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, operation string, success bool, message string, duration time.Duration) error {
	// Record completion event
	if success {
		sm.eventRecorder.RecordOperationEvent(platform, operation, EventReasonPlatformReady, fmt.Sprintf("Operation '%s' completed successfully: %s", operation, message), duration)
	} else {
		sm.eventRecorder.RecordOperationEvent(platform, operation, EventReasonPlatformFailed, fmt.Sprintf("Operation '%s' failed: %s", operation, message), duration)
	}

	// Clear progress
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		status.Progress = nil
		
		// Update operation history
		if status.OperationHistory == nil {
			status.OperationHistory = []observabilityv1beta1.OperationResult{}
		}
		
		result := observabilityv1beta1.OperationResult{
			Operation:  operation,
			Success:    success,
			Message:    message,
			Duration:   duration.String(),
			FinishedAt: metav1.Now(),
		}
		
		// Keep last 10 operations
		status.OperationHistory = append([]observabilityv1beta1.OperationResult{result}, status.OperationHistory...)
		if len(status.OperationHistory) > 10 {
			status.OperationHistory = status.OperationHistory[:10]
		}
	})
}

// CalculateAndSetPhase calculates and sets the phase based on conditions
func (sm *StatusManager) CalculateAndSetPhase(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		oldPhase := status.Phase
		status.Phase = sm.conditionUtils.CalculatePhase(status.Conditions)
		
		// Record phase transition
		if oldPhase != status.Phase {
			message := fmt.Sprintf("Phase changed from %s to %s", oldPhase, status.Phase)
			reason := EventReasonPlatformUpdated
			
			switch status.Phase {
			case PhaseReady:
				reason = EventReasonPlatformReady
			case PhaseFailed:
				reason = EventReasonPlatformFailed
			case PhaseDegraded:
				reason = EventReasonPlatformDegraded
			}
			
			sm.eventRecorder.RecordPlatformEvent(platform, reason, message)
		}
	})
}

// AggregateComponentStatuses aggregates component statuses and updates overall status
func (sm *StatusManager) AggregateComponentStatuses(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		// Create aggregated ready condition
		readyCondition := sm.conditionUtils.AggregateComponentConditions(status.Conditions)
		status.Conditions = sm.conditionUtils.SetCondition(status.Conditions, readyCondition)
		
		// Update component summary
		if status.ComponentsStatus == nil {
			status.ComponentsStatus = make(map[string]observabilityv1beta1.ComponentSummary)
		}
		
		// Check each component condition
		components := []struct {
			name      string
			condition string
			enabled   func() bool
		}{
			{"prometheus", ConditionPrometheusReady, func() bool { return platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled }},
			{"grafana", ConditionGrafanaReady, func() bool { return platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled }},
			{"loki", ConditionLokiReady, func() bool { return platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled }},
			{"tempo", ConditionTempoReady, func() bool { return platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled }},
		}
		
		readyCount := 0
		totalCount := 0
		
		for _, comp := range components {
			if !comp.enabled() {
				continue
			}
			
			totalCount++
			cond := sm.conditionUtils.GetCondition(status.Conditions, comp.condition)
			
			summary := observabilityv1beta1.ComponentSummary{
				Enabled: true,
			}
			
			if cond != nil {
				summary.Ready = cond.Status == metav1.ConditionTrue
				summary.Message = cond.Message
				summary.LastTransitionTime = cond.LastTransitionTime
				
				if summary.Ready {
					readyCount++
				}
			}
			
			status.ComponentsStatus[comp.name] = summary
		}
		
		// Update ready components count
		status.ReadyComponents = fmt.Sprintf("%d/%d", readyCount, totalCount)
	})
}

// RecordError records an error in the status
func (sm *StatusManager) RecordError(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, err error, context string) error {
	// Record error event
	sm.eventRecorder.RecordErrorEvent(platform, EventReasonPlatformFailed, err, context)
	
	// Update error condition
	return sm.SetCondition(ctx, platform, ConditionError, metav1.ConditionTrue, ReasonFailed, fmt.Sprintf("%s: %v", context, err))
}

// ClearError clears the error condition
func (sm *StatusManager) ClearError(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		status.Conditions = sm.conditionUtils.RemoveCondition(status.Conditions, ConditionError)
	})
}

// SetDegraded marks the platform as degraded
func (sm *StatusManager) SetDegraded(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, reason, message string) error {
	sm.eventRecorder.RecordPlatformEvent(platform, EventReasonPlatformDegraded, message)
	return sm.SetCondition(ctx, platform, ConditionDegraded, metav1.ConditionTrue, reason, message)
}

// UpdateMetrics updates metrics in the status
func (sm *StatusManager) UpdateMetrics(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return sm.UpdatePlatformStatus(ctx, platform, func(status *observabilityv1beta1.ObservabilityPlatformStatus) {
		if status.Metrics == nil {
			status.Metrics = &observabilityv1beta1.PlatformMetrics{}
		}
		
		status.Metrics.ReconciliationDuration = sm.metricsCollector.reconciliationDuration.String()
		status.Metrics.LastReconcileTime = metav1.NewTime(sm.metricsCollector.lastReconcileTime)
		status.Metrics.ErrorCount = int32(sm.metricsCollector.errorCount)
		
		// Calculate health score
		healthScore := 100.0
		for _, metrics := range sm.metricsCollector.componentStats {
			healthScore = (healthScore + metrics.HealthScore) / 2
		}
		status.Metrics.HealthScore = int32(healthScore)
	})
}

// processUpdates processes queued status updates
func (sm *StatusManager) processUpdates() {
	for update := range sm.updateQueue {
		sm.applyStatusUpdate(update)
	}
}

// applyStatusUpdate applies a single status update with retries
func (sm *StatusManager) applyStatusUpdate(update statusUpdate) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		// Get latest version
		latest := &observabilityv1beta1.ObservabilityPlatform{}
		key := types.NamespacedName{
			Name:      update.platform.Name,
			Namespace: update.platform.Namespace,
		}
		
		if err := sm.client.Get(update.ctx, key, latest); err != nil {
			if !apierrors.IsNotFound(err) {
				sm.log.Error(err, "Failed to get latest platform", "attempt", i+1)
			}
			return
		}
		
		// Apply the update function
		update.updateFn(&latest.Status)
		
		// Update observed generation
		latest.Status.ObservedGeneration = latest.Generation
		latest.Status.LastUpdated = metav1.Now()
		
		// Try to update
		if err := sm.client.Status().Update(update.ctx, latest); err != nil {
			if apierrors.IsConflict(err) && i < maxRetries-1 {
				// Retry on conflict
				time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
				continue
			}
			sm.log.Error(err, "Failed to update platform status", "attempt", i+1)
			return
		}
		
		// Success
		return
	}
}

// GetEventSummary returns a summary of recent events
func (sm *StatusManager) GetEventSummary() map[string]interface{} {
	return sm.eventRecorder.GenerateEventSummary()
}

// Close closes the status manager
func (sm *StatusManager) Close() {
	close(sm.updateQueue)
}
