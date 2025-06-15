/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// MigrationManager handles automated version migrations for ObservabilityPlatform resources
type MigrationManager struct {
	client            client.Client
	scheme            *runtime.Scheme
	logger            logr.Logger
	tracker           *SchemaEvolutionTracker
	optimizer         *ConversionOptimizer
	lifecycleManager  *LifecycleIntegrationManager
	batchProcessor    *BatchConversionProcessor
	statusReporter    *MigrationStatusReporter
	
	// Configuration
	config MigrationConfig
	
	// Runtime state
	mu              sync.RWMutex
	activeMigrations map[string]*MigrationTask
}

// MigrationConfig defines configuration for the migration manager
type MigrationConfig struct {
	// MaxConcurrentMigrations limits concurrent migrations
	MaxConcurrentMigrations int
	
	// BatchSize for batch conversions
	BatchSize int
	
	// RetryAttempts for failed migrations
	RetryAttempts int
	
	// RetryInterval between attempts
	RetryInterval time.Duration
	
	// EnableOptimizations enables conversion optimizations
	EnableOptimizations bool
	
	// DryRun mode for testing migrations
	DryRun bool
	
	// ProgressReportInterval for status updates
	ProgressReportInterval time.Duration
}

// MigrationTask represents an active migration
type MigrationTask struct {
	ID            string
	SourceVersion string
	TargetVersion string
	Resources     []types.NamespacedName
	Status        MigrationStatus
	StartTime     time.Time
	EndTime       *time.Time
	Error         error
	Progress      MigrationProgress
}

// MigrationStatus represents the status of a migration
type MigrationStatus string

const (
	MigrationStatusPending    MigrationStatus = "Pending"
	MigrationStatusInProgress MigrationStatus = "InProgress"
	MigrationStatusCompleted  MigrationStatus = "Completed"
	MigrationStatusFailed     MigrationStatus = "Failed"
	MigrationStatusRolledBack MigrationStatus = "RolledBack"
)

// MigrationProgress tracks migration progress
type MigrationProgress struct {
	TotalResources     int
	MigratedResources  int
	FailedResources    int
	SkippedResources   int
	CurrentResource    string
	EstimatedTimeLeft  time.Duration
	AverageProcessTime time.Duration
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(client client.Client, scheme *runtime.Scheme, logger logr.Logger, config MigrationConfig) *MigrationManager {
	return &MigrationManager{
		client:           client,
		scheme:           scheme,
		logger:           logger.WithName("migration-manager"),
		tracker:          NewSchemaEvolutionTracker(logger),
		optimizer:        NewConversionOptimizer(logger),
		lifecycleManager: NewLifecycleIntegrationManager(client, logger),
		batchProcessor:   NewBatchConversionProcessor(client, scheme, logger, config.BatchSize),
		statusReporter:   NewMigrationStatusReporter(logger),
		config:           config,
		activeMigrations: make(map[string]*MigrationTask),
	}
}

// MigrateResource migrates a single resource to the target version
func (m *MigrationManager) MigrateResource(ctx context.Context, resource types.NamespacedName, targetVersion string) error {
	m.logger.Info("Starting resource migration",
		"resource", resource,
		"targetVersion", targetVersion)
	
	// Create migration task
	task := &MigrationTask{
		ID:            fmt.Sprintf("migrate-%s-%s-%d", resource.Namespace, resource.Name, time.Now().Unix()),
		TargetVersion: targetVersion,
		Resources:     []types.NamespacedName{resource},
		Status:        MigrationStatusPending,
		StartTime:     time.Now(),
		Progress: MigrationProgress{
			TotalResources: 1,
		},
	}
	
	// Register task
	m.mu.Lock()
	m.activeMigrations[task.ID] = task
	m.mu.Unlock()
	
	// Execute migration
	err := m.executeMigration(ctx, task)
	
	// Update task status
	m.mu.Lock()
	if err != nil {
		task.Status = MigrationStatusFailed
		task.Error = err
	} else {
		task.Status = MigrationStatusCompleted
	}
	endTime := time.Now()
	task.EndTime = &endTime
	m.mu.Unlock()
	
	// Report status
	m.statusReporter.ReportMigrationComplete(task)
	
	return err
}

// MigrateBatch migrates multiple resources in batch
func (m *MigrationManager) MigrateBatch(ctx context.Context, resources []types.NamespacedName, targetVersion string) (*MigrationTask, error) {
	m.logger.Info("Starting batch migration",
		"resourceCount", len(resources),
		"targetVersion", targetVersion)
	
	// Create migration task
	task := &MigrationTask{
		ID:            fmt.Sprintf("batch-migrate-%d-%d", len(resources), time.Now().Unix()),
		TargetVersion: targetVersion,
		Resources:     resources,
		Status:        MigrationStatusInProgress,
		StartTime:     time.Now(),
		Progress: MigrationProgress{
			TotalResources: len(resources),
		},
	}
	
	// Register task
	m.mu.Lock()
	m.activeMigrations[task.ID] = task
	m.mu.Unlock()
	
	// Execute batch migration asynchronously
	go func() {
		err := m.executeBatchMigration(ctx, task)
		
		// Update task status
		m.mu.Lock()
		if err != nil {
			task.Status = MigrationStatusFailed
			task.Error = err
		} else {
			task.Status = MigrationStatusCompleted
		}
		endTime := time.Now()
		task.EndTime = &endTime
		m.mu.Unlock()
		
		// Report status
		m.statusReporter.ReportMigrationComplete(task)
	}()
	
	return task, nil
}

// executeMigration performs the actual migration
func (m *MigrationManager) executeMigration(ctx context.Context, task *MigrationTask) error {
	resource := task.Resources[0]
	
	// Pre-migration checks
	if err := m.lifecycleManager.PreMigrationCheck(ctx, resource, task.TargetVersion); err != nil {
		return fmt.Errorf("pre-migration check failed: %w", err)
	}
	
	// Get current resource
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1alpha1",
		Kind:    "ObservabilityPlatform",
	})
	
	if err := m.client.Get(ctx, resource, u); err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}
	
	// Track schema evolution
	m.tracker.RecordMigration(u.GetAPIVersion(), task.TargetVersion, resource)
	
	// Optimize conversion if enabled
	if m.config.EnableOptimizations {
		optimized, err := m.optimizer.OptimizeConversion(u, task.TargetVersion)
		if err == nil && optimized != nil {
			u = optimized
		}
	}
	
	// Perform migration with retries
	var migrationErr error
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Convert to target version
		converted, err := m.convertToVersion(u, task.TargetVersion)
		if err != nil {
			migrationErr = fmt.Errorf("conversion failed: %w", err)
			return err
		}
		
		// Apply lifecycle hooks
		if err := m.lifecycleManager.ApplyMigrationHooks(ctx, converted); err != nil {
			migrationErr = fmt.Errorf("lifecycle hooks failed: %w", err)
			return err
		}
		
		// Update resource
		if !m.config.DryRun {
			if err := m.client.Update(ctx, converted); err != nil {
				if !errors.IsConflict(err) {
					migrationErr = err
				}
				return err
			}
		}
		
		return nil
	})
	
	if err != nil {
		// Attempt rollback on failure
		if rollbackErr := m.lifecycleManager.RollbackMigration(ctx, resource); rollbackErr != nil {
			m.logger.Error(rollbackErr, "Failed to rollback migration")
		}
		return migrationErr
	}
	
	// Post-migration validation
	if err := m.lifecycleManager.PostMigrationValidation(ctx, resource, task.TargetVersion); err != nil {
		return fmt.Errorf("post-migration validation failed: %w", err)
	}
	
	// Update progress
	m.updateProgress(task, 1, 0, 0)
	
	return nil
}

// executeBatchMigration performs batch migration
func (m *MigrationManager) executeBatchMigration(ctx context.Context, task *MigrationTask) error {
	// Use batch processor for efficient batch conversion
	results, err := m.batchProcessor.ProcessBatch(ctx, task.Resources, task.TargetVersion)
	if err != nil {
		return fmt.Errorf("batch processing failed: %w", err)
	}
	
	// Update progress based on results
	var migrated, failed, skipped int
	for _, result := range results {
		switch result.Status {
		case BatchResultStatusSuccess:
			migrated++
		case BatchResultStatusFailed:
			failed++
		case BatchResultStatusSkipped:
			skipped++
		}
	}
	
	m.updateProgress(task, migrated, failed, skipped)
	
	// Report batch results
	m.statusReporter.ReportBatchResults(task.ID, results)
	
	if failed > 0 {
		return fmt.Errorf("batch migration completed with %d failures", failed)
	}
	
	return nil
}

// convertToVersion converts a resource to the target version
func (m *MigrationManager) convertToVersion(u *unstructured.Unstructured, targetVersion string) (runtime.Object, error) {
	switch targetVersion {
	case "v1beta1":
		// Convert from v1alpha1 to v1beta1
		v1alpha1Obj := &observabilityv1alpha1.ObservabilityPlatform{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, v1alpha1Obj); err != nil {
			return nil, fmt.Errorf("failed to convert to v1alpha1: %w", err)
		}
		
		v1beta1Obj := &observabilityv1beta1.ObservabilityPlatform{}
		if err := v1alpha1Obj.ConvertTo(v1beta1Obj); err != nil {
			return nil, fmt.Errorf("failed to convert to v1beta1: %w", err)
		}
		
		return v1beta1Obj, nil
		
	case "v1alpha1":
		// Convert from v1beta1 to v1alpha1 (backward compatibility)
		v1beta1Obj := &observabilityv1beta1.ObservabilityPlatform{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, v1beta1Obj); err != nil {
			return nil, fmt.Errorf("failed to convert to v1beta1: %w", err)
		}
		
		v1alpha1Obj := &observabilityv1alpha1.ObservabilityPlatform{}
		if err := v1beta1Obj.ConvertTo(v1alpha1Obj); err != nil {
			return nil, fmt.Errorf("failed to convert to v1alpha1: %w", err)
		}
		
		return v1alpha1Obj, nil
		
	default:
		return nil, fmt.Errorf("unsupported target version: %s", targetVersion)
	}
}

// updateProgress updates migration progress
func (m *MigrationManager) updateProgress(task *MigrationTask, migrated, failed, skipped int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task.Progress.MigratedResources += migrated
	task.Progress.FailedResources += failed
	task.Progress.SkippedResources += skipped
	
	// Calculate estimated time left
	elapsed := time.Since(task.StartTime)
	processedCount := task.Progress.MigratedResources + task.Progress.FailedResources + task.Progress.SkippedResources
	if processedCount > 0 {
		task.Progress.AverageProcessTime = elapsed / time.Duration(processedCount)
		remaining := task.Progress.TotalResources - processedCount
		task.Progress.EstimatedTimeLeft = task.Progress.AverageProcessTime * time.Duration(remaining)
	}
}

// GetMigrationStatus returns the status of a migration task
func (m *MigrationManager) GetMigrationStatus(taskID string) (*MigrationTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	task, exists := m.activeMigrations[taskID]
	if !exists {
		return nil, fmt.Errorf("migration task not found: %s", taskID)
	}
	
	// Return a copy to avoid race conditions
	taskCopy := *task
	return &taskCopy, nil
}

// ListActiveMigrations returns all active migrations
func (m *MigrationManager) ListActiveMigrations() []*MigrationTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	tasks := make([]*MigrationTask, 0, len(m.activeMigrations))
	for _, task := range m.activeMigrations {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	
	return tasks
}

// CancelMigration cancels an active migration
func (m *MigrationManager) CancelMigration(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	task, exists := m.activeMigrations[taskID]
	if !exists {
		return fmt.Errorf("migration task not found: %s", taskID)
	}
	
	if task.Status != MigrationStatusInProgress {
		return fmt.Errorf("cannot cancel migration in status: %s", task.Status)
	}
	
	// Mark as cancelled and trigger rollback
	task.Status = MigrationStatusRolledBack
	
	// Rollback migrated resources
	for i := 0; i < task.Progress.MigratedResources; i++ {
		if i < len(task.Resources) {
			if err := m.lifecycleManager.RollbackMigration(context.Background(), task.Resources[i]); err != nil {
				m.logger.Error(err, "Failed to rollback resource during cancellation",
					"resource", task.Resources[i])
			}
		}
	}
	
	return nil
}
