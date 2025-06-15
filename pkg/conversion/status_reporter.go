/*
Copyright 2025 The Gunj Operator Authors.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// MigrationStatusReporter provides real-time migration status reporting
type MigrationStatusReporter struct {
	log            logr.Logger
	mu             sync.RWMutex
	currentStatus  *MigrationStatus
	statusHandlers []StatusHandler
	updateInterval time.Duration
}

// MigrationStatus represents the current status of a migration
type MigrationStatus struct {
	ID               string
	Plan             *MigrationPlan
	StartTime        time.Time
	LastUpdateTime   time.Time
	Phase            MigrationPhase
	Progress         MigrationProgress
	ResourceStatuses map[string]*ResourceMigrationStatus
	Metrics          MigrationMetrics
	Events           []MigrationEvent
}

// MigrationPhase represents the current phase of migration
type MigrationPhase string

const (
	MigrationPhaseInitializing MigrationPhase = "Initializing"
	MigrationPhaseValidating   MigrationPhase = "Validating"
	MigrationPhaseMigrating    MigrationPhase = "Migrating"
	MigrationPhaseVerifying    MigrationPhase = "Verifying"
	MigrationPhaseCompleted    MigrationPhase = "Completed"
	MigrationPhaseFailed       MigrationPhase = "Failed"
	MigrationPhaseRollingBack  MigrationPhase = "RollingBack"
)

// MigrationProgress tracks migration progress
type MigrationProgress struct {
	TotalResources     int
	ProcessedResources int
	SuccessfulResources int
	FailedResources    int
	SkippedResources   int
	PercentComplete    float64
	EstimatedTimeLeft  time.Duration
	CurrentBatch       int
	TotalBatches       int
}

// ResourceMigrationStatus tracks individual resource migration status
type ResourceMigrationStatus struct {
	Name        string
	Namespace   string
	Status      ResourceStatus
	StartTime   time.Time
	EndTime     *time.Time
	Duration    time.Duration
	Error       string
	RetryCount  int
	LastAttempt time.Time
}

// ResourceStatus represents the status of a resource migration
type ResourceStatus string

const (
	ResourceStatusPending    ResourceStatus = "Pending"
	ResourceStatusInProgress ResourceStatus = "InProgress"
	ResourceStatusCompleted  ResourceStatus = "Completed"
	ResourceStatusFailed     ResourceStatus = "Failed"
	ResourceStatusSkipped    ResourceStatus = "Skipped"
	ResourceStatusRetrying   ResourceStatus = "Retrying"
)

// MigrationMetrics contains migration performance metrics
type MigrationMetrics struct {
	AverageResourceTime   time.Duration
	TotalDuration         time.Duration
	ResourcesPerSecond    float64
	SuccessRate           float64
	RetryRate             float64
	EstimatedCompletion   time.Time
	MemoryUsageMB         float64
	CPUUsagePercent       float64
	NetworkBandwidthMBps  float64
}

// MigrationEvent represents an event during migration
type MigrationEvent struct {
	Timestamp time.Time
	Type      EventType
	Level     EventLevel
	Message   string
	Resource  string
	Details   map[string]interface{}
}

// EventType represents the type of migration event
type EventType string

const (
	EventTypeStarted      EventType = "Started"
	EventTypeProgress     EventType = "Progress"
	EventTypeResourceDone EventType = "ResourceDone"
	EventTypeWarning      EventType = "Warning"
	EventTypeError        EventType = "Error"
	EventTypeCompleted    EventType = "Completed"
	EventTypeRollback     EventType = "Rollback"
)

// EventLevel represents the severity level of an event
type EventLevel string

const (
	EventLevelInfo    EventLevel = "Info"
	EventLevelWarning EventLevel = "Warning"
	EventLevelError   EventLevel = "Error"
)

// StatusHandler interface for handling status updates
type StatusHandler interface {
	HandleStatusUpdate(status *MigrationStatus)
}

// NewMigrationStatusReporter creates a new migration status reporter
func NewMigrationStatusReporter(log logr.Logger) *MigrationStatusReporter {
	return &MigrationStatusReporter{
		log:            log.WithName("migration-status-reporter"),
		statusHandlers: []StatusHandler{},
		updateInterval: 5 * time.Second,
	}
}

// AddStatusHandler adds a status update handler
func (r *MigrationStatusReporter) AddStatusHandler(handler StatusHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusHandlers = append(r.statusHandlers, handler)
}

// Start begins status reporting for a migration
func (r *MigrationStatusReporter) Start(ctx context.Context, plan *MigrationPlan, result *MigrationResult) {
	r.mu.Lock()
	r.currentStatus = &MigrationStatus{
		ID:               generateMigrationID(),
		Plan:             plan,
		StartTime:        time.Now(),
		LastUpdateTime:   time.Now(),
		Phase:            MigrationPhaseInitializing,
		ResourceStatuses: make(map[string]*ResourceMigrationStatus),
		Events:           []MigrationEvent{},
	}
	
	// Initialize resource statuses
	for _, resource := range plan.Resources {
		key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
		r.currentStatus.ResourceStatuses[key] = &ResourceMigrationStatus{
			Name:      resource.Name,
			Namespace: resource.Namespace,
			Status:    ResourceStatusPending,
		}
	}
	
	// Initialize progress
	r.currentStatus.Progress = MigrationProgress{
		TotalResources: len(plan.Resources),
		TotalBatches:   (len(plan.Resources) + plan.BatchSize - 1) / plan.BatchSize,
	}
	r.mu.Unlock()

	// Add start event
	r.addEvent(EventTypeStarted, EventLevelInfo, "Migration started", "", nil)

	// Start periodic status updates
	go r.runStatusUpdater(ctx)
}

// runStatusUpdater runs periodic status updates
func (r *MigrationStatusReporter) runStatusUpdater(ctx context.Context) {
	ticker := time.NewTicker(r.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.updateMetrics()
			r.notifyHandlers()
		}
	}
}

// UpdateProgress updates the migration progress
func (r *MigrationStatusReporter) UpdateProgress(result *MigrationResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentStatus == nil {
		return
	}

	// Update progress
	r.currentStatus.Progress.ProcessedResources = result.SuccessfulCount + result.FailedCount + result.SkippedCount
	r.currentStatus.Progress.SuccessfulResources = result.SuccessfulCount
	r.currentStatus.Progress.FailedResources = result.FailedCount
	r.currentStatus.Progress.SkippedResources = result.SkippedCount
	
	if r.currentStatus.Progress.TotalResources > 0 {
		r.currentStatus.Progress.PercentComplete = float64(r.currentStatus.Progress.ProcessedResources) / float64(r.currentStatus.Progress.TotalResources) * 100
	}

	// Update phase
	if r.currentStatus.Progress.ProcessedResources == r.currentStatus.Progress.TotalResources {
		if result.FailedCount == 0 {
			r.currentStatus.Phase = MigrationPhaseCompleted
		} else {
			r.currentStatus.Phase = MigrationPhaseFailed
		}
	} else {
		r.currentStatus.Phase = MigrationPhaseMigrating
	}

	r.currentStatus.LastUpdateTime = time.Now()

	// Estimate time left
	r.estimateTimeLeft()
}

// UpdateResourceStatus updates the status of a specific resource
func (r *MigrationStatusReporter) UpdateResourceStatus(namespace, name string, status ResourceStatus, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentStatus == nil {
		return
	}

	key := fmt.Sprintf("%s/%s", namespace, name)
	resourceStatus, exists := r.currentStatus.ResourceStatuses[key]
	if !exists {
		resourceStatus = &ResourceMigrationStatus{
			Name:      name,
			Namespace: namespace,
		}
		r.currentStatus.ResourceStatuses[key] = resourceStatus
	}

	// Update status
	previousStatus := resourceStatus.Status
	resourceStatus.Status = status
	resourceStatus.LastAttempt = time.Now()

	// Handle status transitions
	switch status {
	case ResourceStatusInProgress:
		if resourceStatus.StartTime.IsZero() {
			resourceStatus.StartTime = time.Now()
		}
	case ResourceStatusCompleted, ResourceStatusFailed, ResourceStatusSkipped:
		now := time.Now()
		resourceStatus.EndTime = &now
		if !resourceStatus.StartTime.IsZero() {
			resourceStatus.Duration = now.Sub(resourceStatus.StartTime)
		}
		if err != nil {
			resourceStatus.Error = err.Error()
		}
	case ResourceStatusRetrying:
		resourceStatus.RetryCount++
	}

	// Add event
	eventType := EventTypeProgress
	if status == ResourceStatusCompleted || status == ResourceStatusFailed {
		eventType = EventTypeResourceDone
	}

	level := EventLevelInfo
	if status == ResourceStatusFailed {
		level = EventLevelError
	} else if status == ResourceStatusRetrying {
		level = EventLevelWarning
	}

	r.addEvent(eventType, level, fmt.Sprintf("Resource %s transitioned from %s to %s", key, previousStatus, status), key, map[string]interface{}{
		"previousStatus": previousStatus,
		"newStatus":      status,
		"error":          resourceStatus.Error,
		"retryCount":     resourceStatus.RetryCount,
	})
}

// SetPhase sets the current migration phase
func (r *MigrationStatusReporter) SetPhase(phase MigrationPhase) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentStatus == nil {
		return
	}

	previousPhase := r.currentStatus.Phase
	r.currentStatus.Phase = phase
	r.currentStatus.LastUpdateTime = time.Now()

	r.addEvent(EventTypeProgress, EventLevelInfo, fmt.Sprintf("Phase changed from %s to %s", previousPhase, phase), "", nil)
}

// AddWarning adds a warning event
func (r *MigrationStatusReporter) AddWarning(message string, resource string, details map[string]interface{}) {
	r.addEvent(EventTypeWarning, EventLevelWarning, message, resource, details)
}

// AddError adds an error event
func (r *MigrationStatusReporter) AddError(message string, resource string, err error) {
	details := map[string]interface{}{}
	if err != nil {
		details["error"] = err.Error()
	}
	r.addEvent(EventTypeError, EventLevelError, message, resource, details)
}

// addEvent adds a migration event
func (r *MigrationStatusReporter) addEvent(eventType EventType, level EventLevel, message string, resource string, details map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentStatus == nil {
		return
	}

	event := MigrationEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		Level:     level,
		Message:   message,
		Resource:  resource,
		Details:   details,
	}

	r.currentStatus.Events = append(r.currentStatus.Events, event)
	
	// Log the event
	switch level {
	case EventLevelError:
		r.log.Error(nil, message, "resource", resource, "details", details)
	case EventLevelWarning:
		r.log.Info(message, "level", "warning", "resource", resource, "details", details)
	default:
		r.log.V(1).Info(message, "resource", resource, "details", details)
	}
}

// updateMetrics updates migration metrics
func (r *MigrationStatusReporter) updateMetrics() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentStatus == nil {
		return
	}

	metrics := &r.currentStatus.Metrics

	// Calculate duration
	metrics.TotalDuration = time.Since(r.currentStatus.StartTime)

	// Calculate resources per second
	if metrics.TotalDuration.Seconds() > 0 {
		metrics.ResourcesPerSecond = float64(r.currentStatus.Progress.ProcessedResources) / metrics.TotalDuration.Seconds()
	}

	// Calculate success rate
	if r.currentStatus.Progress.ProcessedResources > 0 {
		metrics.SuccessRate = float64(r.currentStatus.Progress.SuccessfulResources) / float64(r.currentStatus.Progress.ProcessedResources) * 100
	}

	// Calculate retry rate
	totalRetries := 0
	for _, status := range r.currentStatus.ResourceStatuses {
		totalRetries += status.RetryCount
	}
	if r.currentStatus.Progress.ProcessedResources > 0 {
		metrics.RetryRate = float64(totalRetries) / float64(r.currentStatus.Progress.ProcessedResources)
	}

	// Calculate average resource time
	var totalResourceTime time.Duration
	completedResources := 0
	for _, status := range r.currentStatus.ResourceStatuses {
		if status.Duration > 0 {
			totalResourceTime += status.Duration
			completedResources++
		}
	}
	if completedResources > 0 {
		metrics.AverageResourceTime = totalResourceTime / time.Duration(completedResources)
	}

	// Estimate completion
	if r.currentStatus.Progress.ProcessedResources > 0 && metrics.ResourcesPerSecond > 0 {
		remainingResources := r.currentStatus.Progress.TotalResources - r.currentStatus.Progress.ProcessedResources
		estimatedSeconds := float64(remainingResources) / metrics.ResourcesPerSecond
		metrics.EstimatedCompletion = time.Now().Add(time.Duration(estimatedSeconds) * time.Second)
		r.currentStatus.Progress.EstimatedTimeLeft = time.Duration(estimatedSeconds) * time.Second
	}

	// TODO: Add actual system metrics collection
	// For now, using placeholder values
	metrics.MemoryUsageMB = 128.5
	metrics.CPUUsagePercent = 25.3
	metrics.NetworkBandwidthMBps = 10.2
}

// estimateTimeLeft estimates the time left for migration
func (r *MigrationStatusReporter) estimateTimeLeft() {
	if r.currentStatus.Progress.ProcessedResources == 0 {
		return
	}

	elapsed := time.Since(r.currentStatus.StartTime)
	resourcesPerSecond := float64(r.currentStatus.Progress.ProcessedResources) / elapsed.Seconds()
	
	if resourcesPerSecond > 0 {
		remainingResources := r.currentStatus.Progress.TotalResources - r.currentStatus.Progress.ProcessedResources
		estimatedSeconds := float64(remainingResources) / resourcesPerSecond
		r.currentStatus.Progress.EstimatedTimeLeft = time.Duration(estimatedSeconds) * time.Second
	}
}

// GetCurrentStatus returns the current migration status
func (r *MigrationStatusReporter) GetCurrentStatus() (*MigrationStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentStatus == nil {
		return nil, fmt.Errorf("no active migration")
	}

	// Create a copy to avoid race conditions
	statusCopy := *r.currentStatus
	return &statusCopy, nil
}

// notifyHandlers notifies all registered handlers of status update
func (r *MigrationStatusReporter) notifyHandlers() {
	r.mu.RLock()
	status := r.currentStatus
	handlers := r.statusHandlers
	r.mu.RUnlock()

	if status == nil {
		return
	}

	for _, handler := range handlers {
		// Run handlers in goroutines to avoid blocking
		go func(h StatusHandler) {
			defer runtime.HandleCrash()
			h.HandleStatusUpdate(status)
		}(handler)
	}
}

// GenerateReport generates a migration status report
func (r *MigrationStatusReporter) GenerateReport() (*MigrationStatusReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.currentStatus == nil {
		return nil, fmt.Errorf("no active migration")
	}

	report := &MigrationStatusReport{
		ID:        r.currentStatus.ID,
		StartTime: r.currentStatus.StartTime,
		EndTime:   time.Now(),
		Duration:  r.currentStatus.Metrics.TotalDuration,
		Plan:      r.currentStatus.Plan,
		Progress:  r.currentStatus.Progress,
		Metrics:   r.currentStatus.Metrics,
		Summary:   r.generateSummary(),
		Events:    r.currentStatus.Events,
	}

	// Group resources by status
	report.ResourcesByStatus = make(map[ResourceStatus][]*ResourceMigrationStatus)
	for _, status := range r.currentStatus.ResourceStatuses {
		report.ResourcesByStatus[status.Status] = append(report.ResourcesByStatus[status.Status], status)
	}

	return report, nil
}

// generateSummary generates a summary of the migration
func (r *MigrationStatusReporter) generateSummary() string {
	status := r.currentStatus
	
	summary := fmt.Sprintf(
		"Migration %s: %s\n"+
		"Progress: %d/%d resources (%.1f%%)\n"+
		"Success: %d, Failed: %d, Skipped: %d\n"+
		"Duration: %s, Rate: %.2f resources/sec\n"+
		"Success Rate: %.1f%%, Retry Rate: %.2f",
		status.ID,
		status.Phase,
		status.Progress.ProcessedResources,
		status.Progress.TotalResources,
		status.Progress.PercentComplete,
		status.Progress.SuccessfulResources,
		status.Progress.FailedResources,
		status.Progress.SkippedResources,
		status.Metrics.TotalDuration.Round(time.Second),
		status.Metrics.ResourcesPerSecond,
		status.Metrics.SuccessRate,
		status.Metrics.RetryRate,
	)

	return summary
}

// MigrationStatusReport represents a comprehensive status report
type MigrationStatusReport struct {
	ID                string
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	Plan              *MigrationPlan
	Progress          MigrationProgress
	Metrics           MigrationMetrics
	Summary           string
	ResourcesByStatus map[ResourceStatus][]*ResourceMigrationStatus
	Events            []MigrationEvent
}

// WebSocketStatusHandler sends status updates via WebSocket
type WebSocketStatusHandler struct {
	log logr.Logger
	// WebSocket connection would be injected here
}

// HandleStatusUpdate handles status updates for WebSocket
func (h *WebSocketStatusHandler) HandleStatusUpdate(status *MigrationStatus) {
	// Convert status to JSON
	data, err := json.Marshal(status)
	if err != nil {
		h.log.Error(err, "Failed to marshal status")
		return
	}

	// TODO: Send data over WebSocket connection
	h.log.V(2).Info("Status update", "data", string(data))
}

// MetricsStatusHandler updates Prometheus metrics
type MetricsStatusHandler struct {
	// Prometheus metrics would be injected here
}

// HandleStatusUpdate updates Prometheus metrics
func (h *MetricsStatusHandler) HandleStatusUpdate(status *MigrationStatus) {
	// TODO: Update Prometheus metrics
	// Example:
	// migrationProgress.WithLabelValues(status.ID).Set(status.Progress.PercentComplete)
	// migrationSuccessRate.WithLabelValues(status.ID).Set(status.Metrics.SuccessRate)
}
