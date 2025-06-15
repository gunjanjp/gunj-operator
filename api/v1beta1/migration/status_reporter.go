/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MigrationStatusReporter provides detailed migration status reporting
type MigrationStatusReporter struct {
	logger         logr.Logger
	
	// Reporting channels
	statusChannel  chan MigrationStatusUpdate
	eventChannel   chan MigrationEvent
	
	// Storage
	mu             sync.RWMutex
	reports        map[string]*MigrationReport
	events         []MigrationEvent
	
	// Metrics
	metrics        *ReporterMetrics
	
	// Configuration
	config         ReporterConfig
}

// ReporterConfig defines configuration for the status reporter
type ReporterConfig struct {
	// MaxReports maximum number of reports to keep
	MaxReports int
	
	// MaxEvents maximum number of events to keep
	MaxEvents int
	
	// ReportRetention how long to keep reports
	ReportRetention time.Duration
	
	// EnablePrometheus enables Prometheus metrics
	EnablePrometheus bool
	
	// EnableWebhook enables webhook notifications
	EnableWebhook bool
	
	// WebhookURL URL for webhook notifications
	WebhookURL string
}

// MigrationStatusUpdate represents a status update
type MigrationStatusUpdate struct {
	TaskID    string
	Status    MigrationStatus
	Progress  MigrationProgress
	Message   string
	Timestamp time.Time
}

// MigrationEvent represents a migration event
type MigrationEvent struct {
	ID        string
	TaskID    string
	Type      EventType
	Level     EventLevel
	Message   string
	Details   map[string]interface{}
	Timestamp time.Time
}

// EventType represents the type of migration event
type EventType string

const (
	EventTypeStarted       EventType = "migration_started"
	EventTypeCompleted     EventType = "migration_completed"
	EventTypeFailed        EventType = "migration_failed"
	EventTypeProgress      EventType = "migration_progress"
	EventTypeWarning       EventType = "migration_warning"
	EventTypeRollback      EventType = "migration_rollback"
)

// EventLevel represents the severity level of an event
type EventLevel string

const (
	EventLevelInfo    EventLevel = "info"
	EventLevelWarning EventLevel = "warning"
	EventLevelError   EventLevel = "error"
)

// MigrationReport represents a complete migration report
type MigrationReport struct {
	TaskID           string
	StartTime        time.Time
	EndTime          *time.Time
	Duration         time.Duration
	Status           MigrationStatus
	TotalResources   int
	SuccessCount     int
	FailureCount     int
	SkippedCount     int
	RollbackCount    int
	Events           []MigrationEvent
	ResourceDetails  []ResourceMigrationDetail
	PerformanceStats PerformanceStats
	Recommendations  []string
}

// ResourceMigrationDetail represents details for a single resource migration
type ResourceMigrationDetail struct {
	Name          string
	Namespace     string
	FromVersion   string
	ToVersion     string
	Status        string
	Duration      time.Duration
	Error         string
	Optimizations []string
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	AverageDuration   time.Duration
	FastestMigration  time.Duration
	SlowestMigration  time.Duration
	TotalDataSize     int64
	CacheHitRate      float64
	ParallelizationFactor int
}

// ReporterMetrics tracks reporter metrics
type ReporterMetrics struct {
	migrationsTotal    *prometheus.CounterVec
	migrationDuration  *prometheus.HistogramVec
	migrationProgress  *prometheus.GaugeVec
	eventsTotal        *prometheus.CounterVec
}

// NewMigrationStatusReporter creates a new migration status reporter
func NewMigrationStatusReporter(logger logr.Logger) *MigrationStatusReporter {
	config := ReporterConfig{
		MaxReports:       100,
		MaxEvents:        1000,
		ReportRetention:  7 * 24 * time.Hour,
		EnablePrometheus: true,
		EnableWebhook:    false,
	}
	
	reporter := &MigrationStatusReporter{
		logger:        logger.WithName("status-reporter"),
		statusChannel: make(chan MigrationStatusUpdate, 100),
		eventChannel:  make(chan MigrationEvent, 100),
		reports:       make(map[string]*MigrationReport),
		events:        make([]MigrationEvent, 0),
		config:        config,
	}
	
	// Initialize metrics if Prometheus is enabled
	if config.EnablePrometheus {
		reporter.initializeMetrics()
	}
	
	// Start background workers
	go reporter.processStatusUpdates()
	go reporter.processEvents()
	go reporter.cleanupOldReports()
	
	return reporter
}

// initializeMetrics initializes Prometheus metrics
func (r *MigrationStatusReporter) initializeMetrics() {
	r.metrics = &ReporterMetrics{
		migrationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_migrations_total",
				Help: "Total number of migrations by status",
			},
			[]string{"status", "target_version"},
		),
		migrationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gunj_operator_migration_duration_seconds",
				Help:    "Duration of migration operations",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"target_version"},
		),
		migrationProgress: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_operator_migration_progress_ratio",
				Help: "Current migration progress (0-1)",
			},
			[]string{"task_id"},
		),
		eventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_migration_events_total",
				Help: "Total number of migration events",
			},
			[]string{"type", "level"},
		),
	}
}

// ReportMigrationStart reports the start of a migration
func (r *MigrationStatusReporter) ReportMigrationStart(task *MigrationTask) {
	event := MigrationEvent{
		ID:        fmt.Sprintf("event-%d", time.Now().UnixNano()),
		TaskID:    task.ID,
		Type:      EventTypeStarted,
		Level:     EventLevelInfo,
		Message:   fmt.Sprintf("Migration started for %d resources to version %s", len(task.Resources), task.TargetVersion),
		Details: map[string]interface{}{
			"targetVersion":  task.TargetVersion,
			"resourceCount":  len(task.Resources),
			"resources":      task.Resources,
		},
		Timestamp: time.Now(),
	}
	
	r.eventChannel <- event
	
	// Create initial report
	r.mu.Lock()
	r.reports[task.ID] = &MigrationReport{
		TaskID:         task.ID,
		StartTime:      task.StartTime,
		Status:         task.Status,
		TotalResources: len(task.Resources),
		Events:         []MigrationEvent{event},
	}
	r.mu.Unlock()
}

// ReportMigrationComplete reports the completion of a migration
func (r *MigrationStatusReporter) ReportMigrationComplete(task *MigrationTask) {
	status := "completed"
	level := EventLevelInfo
	if task.Status == MigrationStatusFailed {
		status = "failed"
		level = EventLevelError
	}
	
	event := MigrationEvent{
		ID:        fmt.Sprintf("event-%d", time.Now().UnixNano()),
		TaskID:    task.ID,
		Type:      EventTypeCompleted,
		Level:     level,
		Message:   fmt.Sprintf("Migration %s after %s", status, task.EndTime.Sub(task.StartTime)),
		Details: map[string]interface{}{
			"status":           task.Status,
			"duration":         task.EndTime.Sub(task.StartTime).String(),
			"migratedCount":    task.Progress.MigratedResources,
			"failedCount":      task.Progress.FailedResources,
			"skippedCount":     task.Progress.SkippedResources,
		},
		Timestamp: time.Now(),
	}
	
	r.eventChannel <- event
	
	// Update report
	r.mu.Lock()
	if report, exists := r.reports[task.ID]; exists {
		report.EndTime = task.EndTime
		report.Duration = task.EndTime.Sub(task.StartTime)
		report.Status = task.Status
		report.SuccessCount = task.Progress.MigratedResources
		report.FailureCount = task.Progress.FailedResources
		report.SkippedCount = task.Progress.SkippedResources
		report.Events = append(report.Events, event)
	}
	r.mu.Unlock()
	
	// Update metrics
	if r.metrics != nil {
		r.metrics.migrationsTotal.WithLabelValues(string(task.Status), task.TargetVersion).Inc()
		r.metrics.migrationDuration.WithLabelValues(task.TargetVersion).Observe(task.EndTime.Sub(task.StartTime).Seconds())
		r.metrics.migrationProgress.DeleteLabelValues(task.ID)
	}
	
	// Send webhook notification if enabled
	if r.config.EnableWebhook && r.config.WebhookURL != "" {
		go r.sendWebhookNotification(task)
	}
}

// ReportProgress reports migration progress
func (r *MigrationStatusReporter) ReportProgress(taskID string, progress MigrationProgress) {
	update := MigrationStatusUpdate{
		TaskID:    taskID,
		Status:    MigrationStatusInProgress,
		Progress:  progress,
		Message:   fmt.Sprintf("Processed %d/%d resources", progress.MigratedResources+progress.FailedResources+progress.SkippedResources, progress.TotalResources),
		Timestamp: time.Now(),
	}
	
	r.statusChannel <- update
	
	// Update metrics
	if r.metrics != nil && progress.TotalResources > 0 {
		progressRatio := float64(progress.MigratedResources+progress.FailedResources+progress.SkippedResources) / float64(progress.TotalResources)
		r.metrics.migrationProgress.WithLabelValues(taskID).Set(progressRatio)
	}
}

// ReportBatchResults reports results from batch processing
func (r *MigrationStatusReporter) ReportBatchResults(taskID string, results []BatchConversionResult) {
	// Aggregate results
	successCount := 0
	failureCount := 0
	skippedCount := 0
	totalDuration := time.Duration(0)
	
	resourceDetails := make([]ResourceMigrationDetail, 0, len(results))
	
	for _, result := range results {
		detail := ResourceMigrationDetail{
			Name:      result.Resource.Name,
			Namespace: result.Resource.Namespace,
			Status:    string(result.Status),
			Duration:  result.Duration,
		}
		
		if result.Error != nil {
			detail.Error = result.Error.Error()
		}
		
		resourceDetails = append(resourceDetails, detail)
		totalDuration += result.Duration
		
		switch result.Status {
		case BatchResultStatusSuccess:
			successCount++
		case BatchResultStatusFailed:
			failureCount++
		case BatchResultStatusSkipped:
			skippedCount++
		}
	}
	
	// Create batch event
	event := MigrationEvent{
		ID:        fmt.Sprintf("event-%d", time.Now().UnixNano()),
		TaskID:    taskID,
		Type:      EventTypeProgress,
		Level:     EventLevelInfo,
		Message:   fmt.Sprintf("Batch completed: %d success, %d failed, %d skipped", successCount, failureCount, skippedCount),
		Details: map[string]interface{}{
			"successCount":   successCount,
			"failureCount":   failureCount,
			"skippedCount":   skippedCount,
			"averageDuration": totalDuration / time.Duration(len(results)),
		},
		Timestamp: time.Now(),
	}
	
	r.eventChannel <- event
	
	// Update report
	r.mu.Lock()
	if report, exists := r.reports[taskID]; exists {
		report.ResourceDetails = append(report.ResourceDetails, resourceDetails...)
	}
	r.mu.Unlock()
}

// processStatusUpdates processes status updates
func (r *MigrationStatusReporter) processStatusUpdates() {
	for update := range r.statusChannel {
		r.mu.Lock()
		if report, exists := r.reports[update.TaskID]; exists {
			// Update report with latest status
			if update.Progress.TotalResources > 0 {
				report.SuccessCount = update.Progress.MigratedResources
				report.FailureCount = update.Progress.FailedResources
				report.SkippedCount = update.Progress.SkippedResources
			}
		}
		r.mu.Unlock()
		
		r.logger.V(1).Info("Processed status update",
			"taskID", update.TaskID,
			"status", update.Status,
			"message", update.Message)
	}
}

// processEvents processes migration events
func (r *MigrationStatusReporter) processEvents() {
	for event := range r.eventChannel {
		r.mu.Lock()
		
		// Add to global events
		r.events = append(r.events, event)
		
		// Trim if exceeds max
		if len(r.events) > r.config.MaxEvents {
			r.events = r.events[len(r.events)-r.config.MaxEvents:]
		}
		
		// Add to report
		if report, exists := r.reports[event.TaskID]; exists {
			report.Events = append(report.Events, event)
		}
		
		r.mu.Unlock()
		
		// Update metrics
		if r.metrics != nil {
			r.metrics.eventsTotal.WithLabelValues(string(event.Type), string(event.Level)).Inc()
		}
		
		r.logger.Info("Migration event",
			"taskID", event.TaskID,
			"type", event.Type,
			"level", event.Level,
			"message", event.Message)
	}
}

// cleanupOldReports periodically cleans up old reports
func (r *MigrationStatusReporter) cleanupOldReports() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		
		cutoff := time.Now().Add(-r.config.ReportRetention)
		deleted := 0
		
		for taskID, report := range r.reports {
			if report.EndTime != nil && report.EndTime.Before(cutoff) {
				delete(r.reports, taskID)
				deleted++
			}
		}
		
		r.mu.Unlock()
		
		if deleted > 0 {
			r.logger.Info("Cleaned up old migration reports", "count", deleted)
		}
	}
}

// GetReport returns a specific migration report
func (r *MigrationStatusReporter) GetReport(taskID string) (*MigrationReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	report, exists := r.reports[taskID]
	if !exists {
		return nil, fmt.Errorf("report not found for task: %s", taskID)
	}
	
	// Return a copy
	reportCopy := *report
	return &reportCopy, nil
}

// GetAllReports returns all migration reports
func (r *MigrationStatusReporter) GetAllReports() []*MigrationReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	reports := make([]*MigrationReport, 0, len(r.reports))
	for _, report := range r.reports {
		reportCopy := *report
		reports = append(reports, &reportCopy)
	}
	
	return reports
}

// GetRecentEvents returns recent migration events
func (r *MigrationStatusReporter) GetRecentEvents(limit int) []MigrationEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if limit > len(r.events) {
		limit = len(r.events)
	}
	
	// Return most recent events
	start := len(r.events) - limit
	if start < 0 {
		start = 0
	}
	
	events := make([]MigrationEvent, limit)
	copy(events, r.events[start:])
	
	return events
}

// GenerateHTMLReport generates an HTML report for a migration
func (r *MigrationStatusReporter) GenerateHTMLReport(taskID string) (string, error) {
	report, err := r.GetReport(taskID)
	if err != nil {
		return "", err
	}
	
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Migration Report - {{.TaskID}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 10px; }
        .status-{{.Status}} { color: {{if eq .Status "Completed"}}green{{else}}red{{end}}; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .event-warning { color: orange; }
        .event-error { color: red; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Migration Report</h1>
        <p>Task ID: {{.TaskID}}</p>
        <p>Status: <span class="status-{{.Status}}">{{.Status}}</span></p>
        <p>Duration: {{.Duration}}</p>
    </div>
    
    <h2>Summary</h2>
    <table>
        <tr>
            <th>Total Resources</th>
            <th>Success</th>
            <th>Failed</th>
            <th>Skipped</th>
        </tr>
        <tr>
            <td>{{.TotalResources}}</td>
            <td>{{.SuccessCount}}</td>
            <td>{{.FailureCount}}</td>
            <td>{{.SkippedCount}}</td>
        </tr>
    </table>
    
    <h2>Events</h2>
    <table>
        <tr>
            <th>Time</th>
            <th>Type</th>
            <th>Level</th>
            <th>Message</th>
        </tr>
        {{range .Events}}
        <tr>
            <td>{{.Timestamp.Format "2006-01-02 15:04:05"}}</td>
            <td>{{.Type}}</td>
            <td class="event-{{.Level}}">{{.Level}}</td>
            <td>{{.Message}}</td>
        </tr>
        {{end}}
    </table>
    
    {{if .Recommendations}}
    <h2>Recommendations</h2>
    <ul>
        {{range .Recommendations}}
        <li>{{.}}</li>
        {{end}}
    </ul>
    {{end}}
</body>
</html>
`
	
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	var buf strings.Builder
	if err := t.Execute(&buf, report); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return buf.String(), nil
}

// ExportReportJSON exports a report as JSON
func (r *MigrationStatusReporter) ExportReportJSON(taskID string) ([]byte, error) {
	report, err := r.GetReport(taskID)
	if err != nil {
		return nil, err
	}
	
	return json.MarshalIndent(report, "", "  ")
}

// sendWebhookNotification sends a webhook notification
func (r *MigrationStatusReporter) sendWebhookNotification(task *MigrationTask) {
	// This is a placeholder - in real implementation, this would
	// send an HTTP POST request to the configured webhook URL
	r.logger.V(1).Info("Would send webhook notification",
		"url", r.config.WebhookURL,
		"taskID", task.ID,
		"status", task.Status)
}
