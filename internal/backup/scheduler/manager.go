package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
	"github.com/gunjanjp/gunj-operator/internal/backup/controller"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Manager manages backup schedules
type Manager struct {
	scheduler      *gocron.Scheduler
	backupCtrl     backup.BackupController
	log            logr.Logger
	metrics        *schedulerMetrics
	
	// Schedule tracking
	schedulesMu sync.RWMutex
	schedules   map[string]*scheduleState
}

// scheduleState tracks a backup schedule
type scheduleState struct {
	name           string
	schedule       *backup.BackupSchedule
	job            *gocron.Job
	lastBackupTime *time.Time
	backupHistory  []backupHistoryEntry
}

// backupHistoryEntry represents a backup in history
type backupHistoryEntry struct {
	name      string
	timestamp time.Time
	success   bool
	error     string
}

// schedulerMetrics contains Prometheus metrics
type schedulerMetrics struct {
	scheduledBackupsTotal  *prometheus.CounterVec
	scheduledBackupsFailed *prometheus.CounterVec
	activeSchedules        prometheus.Gauge
	nextBackupTime         *prometheus.GaugeVec
}

// ManagerConfig contains scheduler configuration
type ManagerConfig struct {
	// Location is the timezone location
	Location *time.Location
	
	// MaxHistorySize is the maximum history entries to keep
	MaxHistorySize int
	
	// MetricsEnabled enables metrics collection
	MetricsEnabled bool
}

// NewManager creates a new schedule manager
func NewManager(
	backupCtrl backup.BackupController,
	log logr.Logger,
	config ManagerConfig,
) (*Manager, error) {
	// Create scheduler with location
	location := config.Location
	if location == nil {
		location = time.UTC
	}
	scheduler := gocron.NewScheduler(location)
	
	m := &Manager{
		scheduler:  scheduler,
		backupCtrl: backupCtrl,
		log:        log.WithName("backup-scheduler"),
		schedules:  make(map[string]*scheduleState),
	}
	
	// Initialize metrics
	if config.MetricsEnabled {
		m.metrics = m.initMetrics()
	}
	
	// Start scheduler
	scheduler.StartAsync()
	
	return m, nil
}

// initMetrics initializes Prometheus metrics
func (m *Manager) initMetrics() *schedulerMetrics {
	metrics := &schedulerMetrics{
		scheduledBackupsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_scheduled_backups_total",
				Help: "Total number of scheduled backups",
			},
			[]string{"schedule", "status"},
		),
		scheduledBackupsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_scheduled_backups_failed_total",
				Help: "Total number of failed scheduled backups",
			},
			[]string{"schedule", "reason"},
		),
		activeSchedules: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gunj_backup_schedules_active",
				Help: "Number of active backup schedules",
			},
		),
		nextBackupTime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_backup_schedule_next_run_timestamp",
				Help: "Timestamp of next scheduled backup",
			},
			[]string{"schedule"},
		),
	}
	
	// Register metrics
	prometheus.MustRegister(
		metrics.scheduledBackupsTotal,
		metrics.scheduledBackupsFailed,
		metrics.activeSchedules,
		metrics.nextBackupTime,
	)
	
	return metrics
}

// CreateSchedule creates a backup schedule
func (m *Manager) CreateSchedule(ctx context.Context, name string, schedule *backup.BackupSchedule) error {
	m.log.V(1).Info("Creating backup schedule", "name", name, "schedule", schedule.Schedule)
	
	// Check if schedule already exists
	m.schedulesMu.RLock()
	if _, exists := m.schedules[name]; exists {
		m.schedulesMu.RUnlock()
		return fmt.Errorf("schedule %s already exists", name)
	}
	m.schedulesMu.RUnlock()
	
	// Create scheduled job
	job, err := m.scheduler.Cron(schedule.Schedule).Do(m.runScheduledBackup, name, schedule)
	if err != nil {
		return fmt.Errorf("creating scheduled job: %w", err)
	}
	
	// Store schedule state
	state := &scheduleState{
		name:          name,
		schedule:      schedule,
		job:           job,
		backupHistory: make([]backupHistoryEntry, 0),
	}
	
	m.schedulesMu.Lock()
	m.schedules[name] = state
	m.schedulesMu.Unlock()
	
	// Update metrics
	if m.metrics != nil {
		m.metrics.activeSchedules.Inc()
		m.updateNextBackupMetric(name, job)
	}
	
	m.log.Info("Backup schedule created", "name", name, "nextRun", job.NextRun())
	
	return nil
}

// UpdateSchedule updates a backup schedule
func (m *Manager) UpdateSchedule(ctx context.Context, name string, schedule *backup.BackupSchedule) error {
	m.log.V(1).Info("Updating backup schedule", "name", name, "schedule", schedule.Schedule)
	
	// Get existing schedule
	m.schedulesMu.RLock()
	state, exists := m.schedules[name]
	m.schedulesMu.RUnlock()
	
	if !exists {
		return fmt.Errorf("schedule %s not found", name)
	}
	
	// Remove old job
	m.scheduler.RemoveByReference(state.job)
	
	// Create new job
	job, err := m.scheduler.Cron(schedule.Schedule).Do(m.runScheduledBackup, name, schedule)
	if err != nil {
		return fmt.Errorf("creating scheduled job: %w", err)
	}
	
	// Update state
	m.schedulesMu.Lock()
	state.schedule = schedule
	state.job = job
	m.schedulesMu.Unlock()
	
	// Update metrics
	if m.metrics != nil {
		m.updateNextBackupMetric(name, job)
	}
	
	m.log.Info("Backup schedule updated", "name", name, "nextRun", job.NextRun())
	
	return nil
}

// DeleteSchedule deletes a backup schedule
func (m *Manager) DeleteSchedule(ctx context.Context, name string) error {
	m.log.V(1).Info("Deleting backup schedule", "name", name)
	
	// Get existing schedule
	m.schedulesMu.Lock()
	state, exists := m.schedules[name]
	if !exists {
		m.schedulesMu.Unlock()
		return fmt.Errorf("schedule %s not found", name)
	}
	
	// Remove job
	m.scheduler.RemoveByReference(state.job)
	
	// Remove from tracking
	delete(m.schedules, name)
	m.schedulesMu.Unlock()
	
	// Update metrics
	if m.metrics != nil {
		m.metrics.activeSchedules.Dec()
		m.metrics.nextBackupTime.DeleteLabelValues(name)
	}
	
	m.log.Info("Backup schedule deleted", "name", name)
	
	return nil
}

// GetSchedule gets a backup schedule
func (m *Manager) GetSchedule(ctx context.Context, name string) (*backup.BackupSchedule, error) {
	m.schedulesMu.RLock()
	defer m.schedulesMu.RUnlock()
	
	state, exists := m.schedules[name]
	if !exists {
		return nil, fmt.Errorf("schedule %s not found", name)
	}
	
	return state.schedule, nil
}

// ListSchedules lists all backup schedules
func (m *Manager) ListSchedules(ctx context.Context) ([]backup.BackupSchedule, error) {
	m.schedulesMu.RLock()
	defer m.schedulesMu.RUnlock()
	
	var schedules []backup.BackupSchedule
	for _, state := range m.schedules {
		schedules = append(schedules, *state.schedule)
	}
	
	return schedules, nil
}

// TriggerSchedule manually triggers a scheduled backup
func (m *Manager) TriggerSchedule(ctx context.Context, name string) error {
	m.log.V(1).Info("Manually triggering backup schedule", "name", name)
	
	// Get schedule
	m.schedulesMu.RLock()
	state, exists := m.schedules[name]
	if !exists {
		m.schedulesMu.RUnlock()
		return fmt.Errorf("schedule %s not found", name)
	}
	schedule := state.schedule
	m.schedulesMu.RUnlock()
	
	// Run backup
	go m.runScheduledBackup(name, schedule)
	
	return nil
}

// runScheduledBackup runs a scheduled backup
func (m *Manager) runScheduledBackup(name string, schedule *backup.BackupSchedule) {
	ctx := context.Background()
	
	m.log.Info("Running scheduled backup", "schedule", name)
	
	// Check if suspended
	if schedule.Suspend {
		m.log.V(1).Info("Schedule is suspended, skipping backup", "schedule", name)
		return
	}
	
	// Update timestamp in backup spec
	backupSpec := schedule.Template
	if backupSpec.StorageLocation.Config == nil {
		backupSpec.StorageLocation.Config = make(map[string]string)
	}
	backupSpec.StorageLocation.Config["timestamp"] = time.Now().Format(time.RFC3339)
	backupSpec.StorageLocation.Config["schedule"] = name
	
	// Create backup
	status, err := m.backupCtrl.CreateBackup(ctx, &backupSpec)
	
	// Record history
	m.recordBackupHistory(name, status != nil && status.Phase == backup.BackupPhaseCompleted, err)
	
	// Update metrics
	if m.metrics != nil {
		if err != nil {
			m.metrics.scheduledBackupsTotal.WithLabelValues(name, "failed").Inc()
			m.metrics.scheduledBackupsFailed.WithLabelValues(name, "backup_failed").Inc()
		} else {
			m.metrics.scheduledBackupsTotal.WithLabelValues(name, "success").Inc()
		}
	}
	
	// Apply retention policy
	if schedule.RetentionPolicy != nil {
		m.applyRetentionPolicy(ctx, name, schedule.RetentionPolicy)
	}
	
	// Cleanup old history
	m.cleanupHistory(name, schedule)
	
	// Update last backup time
	m.schedulesMu.Lock()
	if state, exists := m.schedules[name]; exists {
		now := time.Now()
		state.lastBackupTime = &now
		
		// Update next backup metric
		if m.metrics != nil && state.job != nil {
			m.updateNextBackupMetric(name, state.job)
		}
	}
	m.schedulesMu.Unlock()
	
	if err != nil {
		m.log.Error(err, "Scheduled backup failed", "schedule", name)
	} else {
		m.log.Info("Scheduled backup completed", "schedule", name)
	}
}

// recordBackupHistory records backup in history
func (m *Manager) recordBackupHistory(scheduleName string, success bool, err error) {
	m.schedulesMu.Lock()
	defer m.schedulesMu.Unlock()
	
	state, exists := m.schedules[scheduleName]
	if !exists {
		return
	}
	
	entry := backupHistoryEntry{
		name:      fmt.Sprintf("%s-%d", scheduleName, time.Now().Unix()),
		timestamp: time.Now(),
		success:   success,
	}
	
	if err != nil {
		entry.error = err.Error()
	}
	
	state.backupHistory = append(state.backupHistory, entry)
	
	// Keep only recent history (e.g., last 100 entries)
	maxHistory := 100
	if len(state.backupHistory) > maxHistory {
		state.backupHistory = state.backupHistory[len(state.backupHistory)-maxHistory:]
	}
}

// applyRetentionPolicy applies retention policy to backups
func (m *Manager) applyRetentionPolicy(ctx context.Context, scheduleName string, policy *backup.RetentionPolicy) {
	m.log.V(1).Info("Applying retention policy", "schedule", scheduleName)
	
	// Get all backups for this schedule
	backups, err := m.backupCtrl.ListBackups(ctx)
	if err != nil {
		m.log.Error(err, "Failed to list backups for retention", "schedule", scheduleName)
		return
	}
	
	// Filter backups for this schedule
	var scheduleBackups []backup.BackupStatus
	for _, b := range backups {
		if b.BackupLocation != "" && b.Phase == backup.BackupPhaseCompleted {
			// Check if this backup belongs to the schedule
			// This is a simplified check - in production, you'd want better metadata
			scheduleBackups = append(scheduleBackups, b)
		}
	}
	
	// Sort by timestamp (newest first)
	// Apply retention rules
	toDelete := m.selectBackupsToDelete(scheduleBackups, policy)
	
	// Delete old backups
	for _, backup := range toDelete {
		// Extract backup name from location or other metadata
		// This is simplified - you'd need proper backup naming
		backupName := fmt.Sprintf("backup-%d", backup.StartTimestamp.Unix())
		
		if err := m.backupCtrl.DeleteBackup(ctx, backupName); err != nil {
			m.log.Error(err, "Failed to delete backup per retention policy", "backup", backupName)
		} else {
			m.log.V(1).Info("Deleted backup per retention policy", "backup", backupName)
		}
	}
}

// selectBackupsToDelete selects backups to delete based on retention policy
func (m *Manager) selectBackupsToDelete(backups []backup.BackupStatus, policy *backup.RetentionPolicy) []backup.BackupStatus {
	// This is a simplified implementation
	// In production, you'd implement sophisticated retention logic
	// considering daily, weekly, monthly, yearly retention rules
	
	var toDelete []backup.BackupStatus
	
	// Simple implementation: keep only the last N backups
	if policy.KeepLast > 0 && len(backups) > policy.KeepLast {
		// Assuming backups are sorted by timestamp (newest first)
		toDelete = backups[policy.KeepLast:]
	}
	
	return toDelete
}

// cleanupHistory cleans up old backup history
func (m *Manager) cleanupHistory(scheduleName string, schedule *backup.BackupSchedule) {
	m.schedulesMu.Lock()
	defer m.schedulesMu.Unlock()
	
	state, exists := m.schedules[scheduleName]
	if !exists {
		return
	}
	
	// Separate successful and failed backups
	var successful, failed []backupHistoryEntry
	for _, entry := range state.backupHistory {
		if entry.success {
			successful = append(successful, entry)
		} else {
			failed = append(failed, entry)
		}
	}
	
	// Apply history limits
	if schedule.SuccessfulBackupsHistoryLimit != nil {
		limit := int(*schedule.SuccessfulBackupsHistoryLimit)
		if len(successful) > limit {
			successful = successful[len(successful)-limit:]
		}
	}
	
	if schedule.FailedBackupsHistoryLimit != nil {
		limit := int(*schedule.FailedBackupsHistoryLimit)
		if len(failed) > limit {
			failed = failed[len(failed)-limit:]
		}
	}
	
	// Merge back
	state.backupHistory = append(successful, failed...)
}

// updateNextBackupMetric updates the next backup time metric
func (m *Manager) updateNextBackupMetric(scheduleName string, job *gocron.Job) {
	if m.metrics != nil && job != nil {
		nextRun := job.NextRun()
		m.metrics.nextBackupTime.WithLabelValues(scheduleName).Set(float64(nextRun.Unix()))
	}
}

// Stop stops the scheduler
func (m *Manager) Stop() {
	m.log.Info("Stopping backup scheduler")
	m.scheduler.Stop()
}
