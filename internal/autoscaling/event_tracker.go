package autoscaling

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EventTracker tracks scaling events and provides reporting
type EventTracker struct {
	client   client.Client
	recorder record.EventRecorder
	storage  EventStorage
	mu       sync.RWMutex
}

// EventStorage defines the interface for event storage
type EventStorage interface {
	Store(ctx context.Context, event ScalingEvent) error
	List(ctx context.Context, component v1beta1.ComponentType, duration time.Duration) ([]ScalingEvent, error)
	Delete(ctx context.Context, olderThan time.Time) error
}

// NewEventTracker creates a new event tracker
func NewEventTracker(client client.Client, recorder record.EventRecorder, storage EventStorage) *EventTracker {
	return &EventTracker{
		client:   client,
		recorder: recorder,
		storage:  storage,
	}
}

// RecordScalingEvent records a scaling event
func (t *EventTracker) RecordScalingEvent(ctx context.Context, platform *v1beta1.ObservabilityPlatform, event ScalingEvent) error {
	log := log.FromContext(ctx)
	
	// Generate event ID if not set
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s-%s-%d", event.Component, event.Type, time.Now().UnixNano())
	}
	
	// Store in persistent storage
	if err := t.storage.Store(ctx, event); err != nil {
		log.Error(err, "Failed to store scaling event", "eventID", event.ID)
	}
	
	// Emit Kubernetes event
	t.emitKubernetesEvent(platform, event)
	
	// Update metrics
	t.updateMetrics(event)
	
	// Create audit log
	t.createAuditLog(ctx, platform, event)
	
	log.Info("Recorded scaling event",
		"eventID", event.ID,
		"component", event.Component,
		"type", event.Type,
		"fromReplicas", event.FromReplicas,
		"toReplicas", event.ToReplicas,
	)
	
	return nil
}

// GetScalingHistory retrieves scaling history for a component
func (t *EventTracker) GetScalingHistory(ctx context.Context, component v1beta1.ComponentType, duration time.Duration) ([]ScalingEvent, error) {
	return t.storage.List(ctx, component, duration)
}

// GenerateReport generates a scaling report
func (t *EventTracker) GenerateReport(ctx context.Context, platform *v1beta1.ObservabilityPlatform, duration time.Duration) (*ScalingReport, error) {
	log := log.FromContext(ctx)
	
	report := &ScalingReport{
		PlatformName: platform.Name,
		Namespace:    platform.Namespace,
		StartTime:    time.Now().Add(-duration),
		EndTime:      time.Now(),
		Components:   make(map[v1beta1.ComponentType]*ComponentReport),
	}
	
	// Get events for each component
	components := []v1beta1.ComponentType{
		v1beta1.ComponentPrometheus,
		v1beta1.ComponentGrafana,
		v1beta1.ComponentLoki,
		v1beta1.ComponentTempo,
	}
	
	for _, component := range components {
		events, err := t.GetScalingHistory(ctx, component, duration)
		if err != nil {
			log.Error(err, "Failed to get scaling history", "component", component)
			continue
		}
		
		if len(events) > 0 {
			report.Components[component] = t.analyzeComponentEvents(events)
		}
	}
	
	// Calculate totals
	report.TotalEvents = 0
	report.TotalCostSavings = 0
	
	for _, compReport := range report.Components {
		report.TotalEvents += compReport.TotalEvents
		report.TotalCostSavings += compReport.TotalCostSavings
	}
	
	return report, nil
}

// analyzeComponentEvents analyzes events for a single component
func (t *EventTracker) analyzeComponentEvents(events []ScalingEvent) *ComponentReport {
	report := &ComponentReport{
		TotalEvents:      len(events),
		ScalingsByType:   make(map[ScalingType]int),
		SuccessRate:      0,
		AverageScaleTime: 0,
		TotalCostSavings: 0,
	}
	
	successCount := 0
	totalDuration := time.Duration(0)
	
	for _, event := range events {
		// Count by type
		report.ScalingsByType[event.Type]++
		
		// Success rate
		if event.Success {
			successCount++
		}
		
		// Average scale time
		if event.Duration > 0 {
			totalDuration += event.Duration
		}
		
		// Cost savings
		if event.CostImpact != nil && event.CostImpact.MonthlySavings > 0 {
			report.TotalCostSavings += event.CostImpact.MonthlySavings
		}
		
		// Track min/max replicas
		if report.MinReplicas == 0 || event.ToReplicas < report.MinReplicas {
			report.MinReplicas = event.ToReplicas
		}
		if event.ToReplicas > report.MaxReplicas {
			report.MaxReplicas = event.ToReplicas
		}
	}
	
	// Calculate averages
	if len(events) > 0 {
		report.SuccessRate = float64(successCount) / float64(len(events)) * 100
		if totalDuration > 0 {
			report.AverageScaleTime = totalDuration / time.Duration(len(events))
		}
	}
	
	// Find most common scaling reason
	reasonCounts := make(map[string]int)
	for _, event := range events {
		reasonCounts[event.Reason]++
	}
	
	maxCount := 0
	for reason, count := range reasonCounts {
		if count > maxCount {
			maxCount = count
			report.MostCommonReason = reason
		}
	}
	
	return report
}

// emitKubernetesEvent emits a Kubernetes event for the scaling action
func (t *EventTracker) emitKubernetesEvent(platform *v1beta1.ObservabilityPlatform, event ScalingEvent) {
	eventType := corev1.EventTypeNormal
	reason := "Scaling"
	
	if !event.Success {
		eventType = corev1.EventTypeWarning
		reason = "ScalingFailed"
	}
	
	message := fmt.Sprintf("%s scaling for %s: %d -> %d replicas. Reason: %s",
		event.Type, event.Component, event.FromReplicas, event.ToReplicas, event.Reason)
	
	if event.Error != "" {
		message += fmt.Sprintf(". Error: %s", event.Error)
	}
	
	t.recorder.Event(platform, eventType, reason, message)
}

// updateMetrics updates Prometheus metrics for the scaling event
func (t *EventTracker) updateMetrics(event ScalingEvent) {
	// These would be actual Prometheus metrics in production
	scalingEventTotal.WithLabelValues(
		string(event.Component),
		string(event.Type),
		fmt.Sprintf("%v", event.Success),
	).Inc()
	
	if event.Duration > 0 {
		scalingDuration.WithLabelValues(
			string(event.Component),
			string(event.Type),
		).Observe(event.Duration.Seconds())
	}
	
	if event.CostImpact != nil && event.CostImpact.MonthlySavings != 0 {
		costSavings.WithLabelValues(
			string(event.Component),
		).Add(event.CostImpact.MonthlySavings)
	}
}

// createAuditLog creates an audit log entry for the scaling event
func (t *EventTracker) createAuditLog(ctx context.Context, platform *v1beta1.ObservabilityPlatform, event ScalingEvent) {
	log := log.FromContext(ctx)
	
	auditLog := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("scaling-audit-%s-%d", event.Component, time.Now().Unix()),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "gunj-operator",
				"app.kubernetes.io/component":  "audit-log",
				"observability.io/platform":    platform.Name,
				"observability.io/component":   string(event.Component),
				"observability.io/event-type":  string(event.Type),
			},
		},
		Data: map[string]string{
			"event.json": t.serializeEvent(event),
			"timestamp":  event.Timestamp.Format(time.RFC3339),
			"component":  string(event.Component),
			"type":       string(event.Type),
		},
	}
	
	if err := t.client.Create(ctx, auditLog); err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create audit log")
		}
	}
}

// serializeEvent serializes an event to JSON
func (t *EventTracker) serializeEvent(event ScalingEvent) string {
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to serialize: %v"}`, err)
	}
	return string(data)
}

// CleanupOldEvents removes events older than the retention period
func (t *EventTracker) CleanupOldEvents(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return t.storage.Delete(ctx, cutoff)
}

// InMemoryEventStorage is a simple in-memory implementation of EventStorage
type InMemoryEventStorage struct {
	mu     sync.RWMutex
	events []ScalingEvent
}

// NewInMemoryEventStorage creates a new in-memory event storage
func NewInMemoryEventStorage() *InMemoryEventStorage {
	return &InMemoryEventStorage{
		events: make([]ScalingEvent, 0),
	}
}

// Store stores an event in memory
func (s *InMemoryEventStorage) Store(ctx context.Context, event ScalingEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.events = append(s.events, event)
	
	// Keep only last 10000 events to prevent unbounded growth
	if len(s.events) > 10000 {
		s.events = s.events[len(s.events)-10000:]
	}
	
	return nil
}

// List lists events for a component within a duration
func (s *InMemoryEventStorage) List(ctx context.Context, component v1beta1.ComponentType, duration time.Duration) ([]ScalingEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	cutoff := time.Now().Add(-duration)
	result := []ScalingEvent{}
	
	for _, event := range s.events {
		if event.Component == component && event.Timestamp.After(cutoff) {
			result = append(result, event)
		}
	}
	
	return result, nil
}

// Delete deletes events older than the specified time
func (s *InMemoryEventStorage) Delete(ctx context.Context, olderThan time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	filtered := []ScalingEvent{}
	for _, event := range s.events {
		if event.Timestamp.After(olderThan) {
			filtered = append(filtered, event)
		}
	}
	
	s.events = filtered
	return nil
}

// ScalingReport represents a comprehensive scaling report
type ScalingReport struct {
	PlatformName     string
	Namespace        string
	StartTime        time.Time
	EndTime          time.Time
	TotalEvents      int
	TotalCostSavings float64
	Components       map[v1beta1.ComponentType]*ComponentReport
}

// ComponentReport represents scaling statistics for a component
type ComponentReport struct {
	TotalEvents      int
	ScalingsByType   map[ScalingType]int
	SuccessRate      float64
	AverageScaleTime time.Duration
	MinReplicas      int32
	MaxReplicas      int32
	TotalCostSavings float64
	MostCommonReason string
}

// ConfigMapEventStorage stores events in ConfigMaps
type ConfigMapEventStorage struct {
	client    client.Client
	namespace string
}

// NewConfigMapEventStorage creates a new ConfigMap-based event storage
func NewConfigMapEventStorage(client client.Client, namespace string) *ConfigMapEventStorage {
	return &ConfigMapEventStorage{
		client:    client,
		namespace: namespace,
	}
}

// Store stores an event in a ConfigMap
func (s *ConfigMapEventStorage) Store(ctx context.Context, event ScalingEvent) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("scaling-event-%s", event.ID),
			Namespace: s.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "gunj-operator",
				"app.kubernetes.io/component": "scaling-event",
				"observability.io/component":  string(event.Component),
				"observability.io/type":       string(event.Type),
			},
		},
		Data: map[string]string{
			"event.json": s.serializeEvent(event),
			"timestamp":  event.Timestamp.Format(time.RFC3339),
		},
	}
	
	return s.client.Create(ctx, cm)
}

// List lists events from ConfigMaps
func (s *ConfigMapEventStorage) List(ctx context.Context, component v1beta1.ComponentType, duration time.Duration) ([]ScalingEvent, error) {
	cutoff := time.Now().Add(-duration)
	
	cmList := &corev1.ConfigMapList{}
	if err := s.client.List(ctx, cmList, &client.ListOptions{
		Namespace: s.namespace,
		LabelSelector: client.MatchingLabels{
			"app.kubernetes.io/component": "scaling-event",
			"observability.io/component":  string(component),
		},
	}); err != nil {
		return nil, err
	}
	
	events := []ScalingEvent{}
	for _, cm := range cmList.Items {
		if eventJSON, ok := cm.Data["event.json"]; ok {
			var event ScalingEvent
			if err := json.Unmarshal([]byte(eventJSON), &event); err == nil {
				if event.Timestamp.After(cutoff) {
					events = append(events, event)
				}
			}
		}
	}
	
	return events, nil
}

// Delete deletes old events from ConfigMaps
func (s *ConfigMapEventStorage) Delete(ctx context.Context, olderThan time.Time) error {
	cmList := &corev1.ConfigMapList{}
	if err := s.client.List(ctx, cmList, &client.ListOptions{
		Namespace: s.namespace,
		LabelSelector: client.MatchingLabels{
			"app.kubernetes.io/component": "scaling-event",
		},
	}); err != nil {
		return err
	}
	
	for _, cm := range cmList.Items {
		if timestampStr, ok := cm.Data["timestamp"]; ok {
			timestamp, err := time.Parse(time.RFC3339, timestampStr)
			if err == nil && timestamp.Before(olderThan) {
				if err := s.client.Delete(ctx, &cm); err != nil && !errors.IsNotFound(err) {
					return err
				}
			}
		}
	}
	
	return nil
}

// serializeEvent serializes an event to JSON
func (s *ConfigMapEventStorage) serializeEvent(event ScalingEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		return "{}"
	}
	return string(data)
}
