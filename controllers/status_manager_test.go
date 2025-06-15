/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestStatusManager(t *testing.T) {
	// Setup
	s := scheme.Scheme
	err := observabilityv1beta1.AddToScheme(s)
	require.NoError(t, err)

	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-platform",
			Namespace:  "test-namespace",
			Generation: 1,
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Prometheus: &observabilityv1beta1.PrometheusSpec{
					Enabled: true,
				},
				Grafana: &observabilityv1beta1.GrafanaSpec{
					Enabled: true,
				},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(platform).
		WithStatusSubresource(&observabilityv1beta1.ObservabilityPlatform{}).
		Build()

	recorder := record.NewFakeRecorder(100)
	eventRecorder := NewEnhancedEventRecorder(recorder, "test")
	logger := log.FromContext(context.Background())

	sm := NewStatusManager(client, logger, eventRecorder)

	ctx := context.Background()

	t.Run("SetCondition", func(t *testing.T) {
		// Test setting a condition
		err := sm.SetCondition(ctx, platform, ConditionProgressing, metav1.ConditionTrue, ReasonReconciling, "Test reconciliation")
		require.NoError(t, err)

		// Verify condition was set
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		assert.Len(t, updated.Status.Conditions, 1)
		assert.Equal(t, ConditionProgressing, updated.Status.Conditions[0].Type)
		assert.Equal(t, metav1.ConditionTrue, updated.Status.Conditions[0].Status)
		assert.Equal(t, ReasonReconciling, updated.Status.Conditions[0].Reason)
		assert.Equal(t, "Test reconciliation", updated.Status.Conditions[0].Message)
	})

	t.Run("SetComponentStatus", func(t *testing.T) {
		// Test setting component status
		err := sm.SetComponentStatus(ctx, platform, "prometheus", true, "Prometheus is ready")
		require.NoError(t, err)

		err = sm.SetComponentStatus(ctx, platform, "grafana", false, "Grafana deployment failed")
		require.NoError(t, err)

		// Verify component conditions were set
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		// Check Prometheus condition
		promCond := sm.conditionUtils.GetCondition(updated.Status.Conditions, ConditionPrometheusReady)
		require.NotNil(t, promCond)
		assert.Equal(t, metav1.ConditionTrue, promCond.Status)
		assert.Equal(t, "Prometheus is ready", promCond.Message)

		// Check Grafana condition
		grafanaCond := sm.conditionUtils.GetCondition(updated.Status.Conditions, ConditionGrafanaReady)
		require.NotNil(t, grafanaCond)
		assert.Equal(t, metav1.ConditionFalse, grafanaCond.Status)
		assert.Equal(t, "Grafana deployment failed", grafanaCond.Message)
	})

	t.Run("UpdateProgress", func(t *testing.T) {
		// Test progress updates
		err := sm.UpdateProgress(ctx, platform, "test-operation", 5, 10, "Processing items")
		require.NoError(t, err)

		// Verify progress was updated
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		require.NotNil(t, updated.Status.Progress)
		assert.Equal(t, "test-operation", updated.Status.Progress.Operation)
		assert.Equal(t, int32(5), updated.Status.Progress.Current)
		assert.Equal(t, int32(10), updated.Status.Progress.Total)
		assert.Equal(t, int32(50), updated.Status.Progress.Percentage)
		assert.Equal(t, "Processing items", updated.Status.Progress.Message)
	})

	t.Run("CompleteOperation", func(t *testing.T) {
		// Test operation completion
		duration := 30 * time.Second
		err := sm.CompleteOperation(ctx, platform, "test-operation", true, "Operation completed successfully", duration)
		require.NoError(t, err)

		// Verify operation was recorded
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		// Progress should be cleared
		assert.Nil(t, updated.Status.Progress)

		// Operation should be in history
		require.Len(t, updated.Status.OperationHistory, 1)
		assert.Equal(t, "test-operation", updated.Status.OperationHistory[0].Operation)
		assert.True(t, updated.Status.OperationHistory[0].Success)
		assert.Equal(t, "Operation completed successfully", updated.Status.OperationHistory[0].Message)
		assert.Equal(t, "30s", updated.Status.OperationHistory[0].Duration)
	})

	t.Run("AggregateComponentStatuses", func(t *testing.T) {
		// Set up component conditions
		sm.SetComponentStatus(ctx, platform, "prometheus", true, "Ready")
		sm.SetComponentStatus(ctx, platform, "grafana", true, "Ready")
		sm.SetComponentStatus(ctx, platform, "loki", true, "Ready")
		sm.SetComponentStatus(ctx, platform, "tempo", false, "Failed to deploy")

		// Aggregate statuses
		err := sm.AggregateComponentStatuses(ctx, platform)
		require.NoError(t, err)

		// Verify aggregation
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		// Check Ready condition
		readyCond := sm.conditionUtils.GetCondition(updated.Status.Conditions, ConditionReady)
		require.NotNil(t, readyCond)
		assert.Equal(t, metav1.ConditionFalse, readyCond.Status)
		assert.Contains(t, readyCond.Message, "3 of 4 components ready")

		// Check component summary
		assert.Equal(t, "3/4", updated.Status.ReadyComponents)
		assert.Len(t, updated.Status.ComponentsStatus, 4)
		assert.True(t, updated.Status.ComponentsStatus["prometheus"].Ready)
		assert.False(t, updated.Status.ComponentsStatus["tempo"].Ready)
	})

	t.Run("CalculateAndSetPhase", func(t *testing.T) {
		// Test phase calculation
		err := sm.CalculateAndSetPhase(ctx, platform)
		require.NoError(t, err)

		// Verify phase
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		// Should be Installing since not all components are ready
		assert.Equal(t, PhaseInstalling, updated.Status.Phase)

		// Set all components to ready
		sm.SetComponentStatus(ctx, platform, "tempo", true, "Ready")
		sm.AggregateComponentStatuses(ctx, platform)
		sm.CalculateAndSetPhase(ctx, platform)

		// Check phase again
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)
		assert.Equal(t, PhaseReady, updated.Status.Phase)
	})

	t.Run("RecordError", func(t *testing.T) {
		// Test error recording
		testErr := assert.AnError
		err := sm.RecordError(ctx, platform, testErr, "Test context")
		require.NoError(t, err)

		// Verify error condition
		updated := &observabilityv1beta1.ObservabilityPlatform{}
		err = client.Get(ctx, types.NamespacedName{Name: platform.Name, Namespace: platform.Namespace}, updated)
		require.NoError(t, err)

		errorCond := sm.conditionUtils.GetCondition(updated.Status.Conditions, ConditionError)
		require.NotNil(t, errorCond)
		assert.Equal(t, metav1.ConditionTrue, errorCond.Status)
		assert.Contains(t, errorCond.Message, "Test context")
		assert.Contains(t, errorCond.Message, testErr.Error())
	})
}

func TestConditionUtils(t *testing.T) {
	cu := NewConditionUtils()

	t.Run("SetCondition", func(t *testing.T) {
		conditions := []metav1.Condition{}

		// Add new condition
		newCond := metav1.Condition{
			Type:               "TestCondition",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "TestReason",
			Message:            "Test message",
		}

		conditions = cu.SetCondition(conditions, newCond)
		assert.Len(t, conditions, 1)
		assert.Equal(t, "TestCondition", conditions[0].Type)

		// Update existing condition
		updatedCond := metav1.Condition{
			Type:               "TestCondition",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "UpdatedReason",
			Message:            "Updated message",
		}

		conditions = cu.SetCondition(conditions, updatedCond)
		assert.Len(t, conditions, 1)
		assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
		assert.Equal(t, "UpdatedReason", conditions[0].Reason)
	})

	t.Run("GetCondition", func(t *testing.T) {
		conditions := []metav1.Condition{
			{Type: "Condition1", Status: metav1.ConditionTrue},
			{Type: "Condition2", Status: metav1.ConditionFalse},
		}

		cond := cu.GetCondition(conditions, "Condition1")
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)

		cond = cu.GetCondition(conditions, "NonExistent")
		assert.Nil(t, cond)
	})

	t.Run("IsConditionTrue", func(t *testing.T) {
		conditions := []metav1.Condition{
			{Type: "TrueCondition", Status: metav1.ConditionTrue},
			{Type: "FalseCondition", Status: metav1.ConditionFalse},
		}

		assert.True(t, cu.IsConditionTrue(conditions, "TrueCondition"))
		assert.False(t, cu.IsConditionTrue(conditions, "FalseCondition"))
		assert.False(t, cu.IsConditionTrue(conditions, "NonExistent"))
	})

	t.Run("AggregateComponentConditions", func(t *testing.T) {
		// Test with all components ready
		conditions := []metav1.Condition{
			{Type: ConditionPrometheusReady, Status: metav1.ConditionTrue},
			{Type: ConditionGrafanaReady, Status: metav1.ConditionTrue},
			{Type: ConditionLokiReady, Status: metav1.ConditionTrue},
			{Type: ConditionTempoReady, Status: metav1.ConditionTrue},
		}

		aggregated := cu.AggregateComponentConditions(conditions)
		assert.Equal(t, ConditionReady, aggregated.Type)
		assert.Equal(t, metav1.ConditionTrue, aggregated.Status)
		assert.Equal(t, ReasonReady, aggregated.Reason)
		assert.Contains(t, aggregated.Message, "All 4 components are ready")

		// Test with some components not ready
		conditions[1].Status = metav1.ConditionFalse
		conditions[3].Status = metav1.ConditionFalse

		aggregated = cu.AggregateComponentConditions(conditions)
		assert.Equal(t, metav1.ConditionFalse, aggregated.Status)
		assert.Equal(t, ReasonInProgress, aggregated.Reason)
		assert.Contains(t, aggregated.Message, "2 of 4 components ready")
	})

	t.Run("CalculatePhase", func(t *testing.T) {
		// Test Failed phase
		conditions := []metav1.Condition{
			{Type: ConditionError, Status: metav1.ConditionTrue},
		}
		assert.Equal(t, PhaseFailed, cu.CalculatePhase(conditions))

		// Test Ready phase
		conditions = []metav1.Condition{
			{Type: ConditionReady, Status: metav1.ConditionTrue},
			{Type: ConditionError, Status: metav1.ConditionFalse},
		}
		assert.Equal(t, PhaseReady, cu.CalculatePhase(conditions))

		// Test Degraded phase
		conditions = []metav1.Condition{
			{Type: ConditionReady, Status: metav1.ConditionTrue},
			{Type: ConditionDegraded, Status: metav1.ConditionTrue},
		}
		assert.Equal(t, PhaseDegraded, cu.CalculatePhase(conditions))

		// Test Installing phase
		conditions = []metav1.Condition{
			{Type: ConditionProgressing, Status: metav1.ConditionTrue, Reason: ReasonReconciling},
		}
		assert.Equal(t, PhaseInstalling, cu.CalculatePhase(conditions))

		// Test Upgrading phase
		conditions = []metav1.Condition{
			{Type: ConditionProgressing, Status: metav1.ConditionTrue, Reason: ReasonComponentUpgrading},
		}
		assert.Equal(t, PhaseUpgrading, cu.CalculatePhase(conditions))
	})
}

func TestEnhancedEventRecorder(t *testing.T) {
	recorder := record.NewFakeRecorder(100)
	er := NewEnhancedEventRecorder(recorder, "test-component")

	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}

	t.Run("RecordPlatformEvent", func(t *testing.T) {
		er.RecordPlatformEvent(platform, EventReasonPlatformCreated, "Platform created successfully")

		// Check event was recorded
		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, "Platform created successfully")
			assert.Contains(t, event, string(EventReasonPlatformCreated))
		case <-time.After(time.Second):
			t.Fatal("No event recorded")
		}

		// Check event history
		history := er.GetEventHistory()
		assert.Len(t, history, 1)
		assert.Equal(t, EventReasonPlatformCreated, history[0].Reason)
	})

	t.Run("RecordComponentEvent", func(t *testing.T) {
		er.RecordComponentEvent(platform, "prometheus", EventReasonComponentReady, "Prometheus is ready")

		// Check event was recorded
		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, "[prometheus] Prometheus is ready")
		case <-time.After(time.Second):
			t.Fatal("No event recorded")
		}
	})

	t.Run("RecordOperationEvent", func(t *testing.T) {
		duration := 45 * time.Second
		er.RecordOperationEvent(platform, "deployment", EventReasonComponentReady, "Deployment completed", duration)

		// Check event was recorded
		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, "Deployment completed (took 45s)")
		case <-time.After(time.Second):
			t.Fatal("No event recorded")
		}
	})

	t.Run("RecordProgressEvent", func(t *testing.T) {
		er.RecordProgressEvent(platform, "reconciliation", 3, 5, "Processing components")

		// Check event was recorded
		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, "[reconciliation] Processing components (3/5 - 60%)")
		case <-time.After(time.Second):
			t.Fatal("No event recorded")
		}
	})

	t.Run("RecordErrorEvent", func(t *testing.T) {
		testErr := assert.AnError
		er.RecordErrorEvent(platform, EventReasonPlatformFailed, testErr, "Failed to create resource")

		// Check event was recorded
		select {
		case event := <-recorder.Events:
			assert.Contains(t, event, "Failed to create resource")
			assert.Contains(t, event, testErr.Error())
			assert.Contains(t, event, string(corev1.EventTypeWarning))
		case <-time.After(time.Second):
			t.Fatal("No event recorded")
		}
	})

	t.Run("EventHistory", func(t *testing.T) {
		// Record multiple events
		er.RecordPlatformEvent(platform, EventReasonPlatformCreated, "Event 1")
		er.RecordPlatformEvent(platform, EventReasonPlatformUpdated, "Event 2")
		er.RecordErrorEvent(platform, EventReasonPlatformFailed, assert.AnError, "Error event")

		// Check history
		history := er.GetEventHistory()
		assert.GreaterOrEqual(t, len(history), 3)

		// Check filtering by reason
		createdEvents := er.GetEventsByReason(EventReasonPlatformCreated)
		assert.GreaterOrEqual(t, len(createdEvents), 1)

		// Check recent errors
		errors := er.GetRecentErrors(10)
		assert.GreaterOrEqual(t, len(errors), 1)

		// Check event summary
		summary := er.GenerateEventSummary()
		assert.NotNil(t, summary["totalEvents"])
		assert.NotNil(t, summary["eventsByType"])
		assert.NotNil(t, summary["eventsByReason"])
	})
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{2*time.Hour + 15*time.Minute, "2h15m"},
		{25*time.Hour + 30*time.Minute, "1d1h"},
		{48 * time.Hour, "2d0h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
