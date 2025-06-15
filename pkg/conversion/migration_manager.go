/*
Copyright 2025 The Gunj Operator Authors.

Licensed under the MIT License.
*/

package conversion

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
	ctrl "sigs.k8s.io/controller-runtime"
)

// MigrationManager manages the migration of resources between API versions
type MigrationManager struct {
	client         client.Client
	scheme         *runtime.Scheme
	log            logr.Logger
	tracker        *SchemaEvolutionTracker
	statusReporter *MigrationStatusReporter
	validator      *MigrationValidator
	optimizer      *ConversionOptimizer
	mu             sync.RWMutex
}

// MigrationPlan represents a plan for migrating resources
type MigrationPlan struct {
	SourceGVK      schema.GroupVersionKind
	TargetGVK      schema.GroupVersionKind
	Resources      []types.NamespacedName
	BatchSize      int
	DryRun         bool
	PreValidation  bool
	PostValidation bool
	Parallel       bool
	MaxConcurrency int
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	TotalResources   int
	SuccessfulCount  int
	FailedCount      int
	SkippedCount     int
	Duration         time.Duration
	Errors           []MigrationError
	ValidationReport ValidationReport
}

// MigrationError represents an error during migration
type MigrationError struct {
	Resource types.NamespacedName
	Error    error
	Phase    string
	Retry    bool
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *MigrationManager {
	return &MigrationManager{
		client:         client,
		scheme:         scheme,
		log:            log.WithName("migration-manager"),
		tracker:        NewSchemaEvolutionTracker(log),
		statusReporter: NewMigrationStatusReporter(log),
		validator:      NewMigrationValidator(client, scheme, log),
		optimizer:      NewConversionOptimizer(log),
	}
}

// PlanMigration creates a migration plan for resources
func (m *MigrationManager) PlanMigration(ctx context.Context, sourceGVK, targetGVK schema.GroupVersionKind, opts ...MigrationOption) (*MigrationPlan, error) {
	m.log.Info("Planning migration", "source", sourceGVK, "target", targetGVK)

	// Apply options
	options := &migrationOptions{
		batchSize:      10,
		parallel:       true,
		maxConcurrency: 3,
		preValidation:  true,
		postValidation: true,
	}
	for _, opt := range opts {
		opt(options)
	}

	// List resources of source type
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(sourceGVK)
	if err := m.client.List(ctx, list); err != nil {
		return nil, fmt.Errorf("listing resources: %w", err)
	}

	// Build resource list
	resources := make([]types.NamespacedName, 0, len(list.Items))
	for _, item := range list.Items {
		resources = append(resources, types.NamespacedName{
			Namespace: item.GetNamespace(),
			Name:      item.GetName(),
		})
	}

	plan := &MigrationPlan{
		SourceGVK:      sourceGVK,
		TargetGVK:      targetGVK,
		Resources:      resources,
		BatchSize:      options.batchSize,
		DryRun:         options.dryRun,
		PreValidation:  options.preValidation,
		PostValidation: options.postValidation,
		Parallel:       options.parallel,
		MaxConcurrency: options.maxConcurrency,
	}

	// Track schema evolution
	if err := m.tracker.RecordMigrationPlan(plan); err != nil {
		m.log.Error(err, "Failed to record migration plan")
	}

	return plan, nil
}

// ExecuteMigration executes a migration plan
func (m *MigrationManager) ExecuteMigration(ctx context.Context, plan *MigrationPlan) (*MigrationResult, error) {
	m.log.Info("Executing migration", "resources", len(plan.Resources), "dryRun", plan.DryRun)

	start := time.Now()
	result := &MigrationResult{
		TotalResources: len(plan.Resources),
	}

	// Pre-migration validation
	if plan.PreValidation {
		validationReport, err := m.validator.ValidatePreMigration(ctx, plan)
		if err != nil {
			return nil, fmt.Errorf("pre-migration validation failed: %w", err)
		}
		result.ValidationReport = validationReport
		if !validationReport.IsValid() {
			return result, fmt.Errorf("pre-migration validation failed with %d errors", len(validationReport.Errors))
		}
	}

	// Start migration status reporting
	statusCtx, cancelStatus := context.WithCancel(ctx)
	defer cancelStatus()
	m.statusReporter.Start(statusCtx, plan, result)

	// Execute migration
	if plan.Parallel {
		err := m.executeBatchedParallelMigration(ctx, plan, result)
		if err != nil {
			return result, err
		}
	} else {
		err := m.executeSequentialMigration(ctx, plan, result)
		if err != nil {
			return result, err
		}
	}

	// Post-migration validation
	if plan.PostValidation {
		validationReport, err := m.validator.ValidatePostMigration(ctx, plan)
		if err != nil {
			m.log.Error(err, "Post-migration validation failed")
		}
		result.ValidationReport = validationReport
	}

	result.Duration = time.Since(start)
	m.log.Info("Migration completed", "duration", result.Duration, "success", result.SuccessfulCount, "failed", result.FailedCount)

	// Record migration result
	if err := m.tracker.RecordMigrationResult(plan, result); err != nil {
		m.log.Error(err, "Failed to record migration result")
	}

	return result, nil
}

// executeBatchedParallelMigration executes migration in parallel batches
func (m *MigrationManager) executeBatchedParallelMigration(ctx context.Context, plan *MigrationPlan, result *MigrationResult) error {
	sem := make(chan struct{}, plan.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Process in batches
	for i := 0; i < len(plan.Resources); i += plan.BatchSize {
		end := i + plan.BatchSize
		if end > len(plan.Resources) {
			end = len(plan.Resources)
		}
		batch := plan.Resources[i:end]

		// Optimize batch for performance
		optimizedBatch := m.optimizer.OptimizeBatch(batch)

		for _, resource := range optimizedBatch {
			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore

			go func(res types.NamespacedName) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				err := m.migrateResource(ctx, plan, res)
				mu.Lock()
				if err != nil {
					result.FailedCount++
					result.Errors = append(result.Errors, MigrationError{
						Resource: res,
						Error:    err,
						Phase:    "migration",
						Retry:    errors.IsConflict(err) || errors.IsServerTimeout(err),
					})
				} else {
					result.SuccessfulCount++
				}
				mu.Unlock()

				// Update status
				m.statusReporter.UpdateProgress(result)
			}(resource)
		}
	}

	wg.Wait()
	return nil
}

// executeSequentialMigration executes migration sequentially
func (m *MigrationManager) executeSequentialMigration(ctx context.Context, plan *MigrationPlan, result *MigrationResult) error {
	for _, resource := range plan.Resources {
		err := m.migrateResource(ctx, plan, resource)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, MigrationError{
				Resource: resource,
				Error:    err,
				Phase:    "migration",
				Retry:    errors.IsConflict(err) || errors.IsServerTimeout(err),
			})
		} else {
			result.SuccessfulCount++
		}
		
		// Update status
		m.statusReporter.UpdateProgress(result)
	}
	return nil
}

// migrateResource migrates a single resource
func (m *MigrationManager) migrateResource(ctx context.Context, plan *MigrationPlan, resource types.NamespacedName) error {
	log := m.log.WithValues("resource", resource, "dryRun", plan.DryRun)

	// Get the resource
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(plan.SourceGVK)
	if err := m.client.Get(ctx, resource, obj); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Resource not found, skipping")
			return nil
		}
		return fmt.Errorf("getting resource: %w", err)
	}

	// Convert to target version
	targetObj := &unstructured.Unstructured{}
	targetObj.SetGroupVersionKind(plan.TargetGVK)
	
	// Use scheme for conversion
	if err := m.scheme.Convert(obj, targetObj, nil); err != nil {
		return fmt.Errorf("converting resource: %w", err)
	}

	// Apply migration hooks if any
	if err := m.applyMigrationHooks(ctx, obj, targetObj); err != nil {
		return fmt.Errorf("applying migration hooks: %w", err)
	}

	if plan.DryRun {
		log.Info("Dry run: would migrate resource")
		return nil
	}

	// Update the resource with retry
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get latest version
		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(plan.SourceGVK)
		if err := m.client.Get(ctx, resource, current); err != nil {
			return err
		}

		// Update resource version
		targetObj.SetResourceVersion(current.GetResourceVersion())
		
		// Update the resource
		if err := m.client.Update(ctx, targetObj); err != nil {
			return err
		}

		log.Info("Resource migrated successfully")
		return nil
	})
}

// applyMigrationHooks applies any custom migration logic
func (m *MigrationManager) applyMigrationHooks(ctx context.Context, source, target *unstructured.Unstructured) error {
	// Apply any custom migration logic based on GVK
	gvk := source.GroupVersionKind()
	
	switch gvk.Group {
	case "observability.io":
		switch gvk.Kind {
		case "ObservabilityPlatform":
			return m.migrateObservabilityPlatform(source, target)
		}
	}

	return nil
}

// migrateObservabilityPlatform applies specific migration logic for ObservabilityPlatform
func (m *MigrationManager) migrateObservabilityPlatform(source, target *unstructured.Unstructured) error {
	// Example: Handle field renames or restructuring
	spec, found, err := unstructured.NestedMap(source.Object, "spec")
	if err != nil || !found {
		return err
	}

	// Example migration: rename field
	if oldValue, found := spec["deprecatedField"]; found {
		spec["newField"] = oldValue
		delete(spec, "deprecatedField")
	}

	// Set the migrated spec
	return unstructured.SetNestedMap(target.Object, spec, "spec")
}

// Rollback performs a rollback of a failed migration
func (m *MigrationManager) Rollback(ctx context.Context, plan *MigrationPlan, result *MigrationResult) error {
	m.log.Info("Starting rollback", "failed", len(result.Errors))

	// Create reverse migration plan
	reversePlan := &MigrationPlan{
		SourceGVK:      plan.TargetGVK,
		TargetGVK:      plan.SourceGVK,
		Resources:      make([]types.NamespacedName, 0, result.SuccessfulCount),
		BatchSize:      plan.BatchSize,
		DryRun:         false,
		PreValidation:  false,
		PostValidation: true,
		Parallel:       plan.Parallel,
		MaxConcurrency: plan.MaxConcurrency,
	}

	// Only rollback successfully migrated resources
	for _, resource := range plan.Resources {
		migrated := true
		for _, err := range result.Errors {
			if err.Resource == resource {
				migrated = false
				break
			}
		}
		if migrated {
			reversePlan.Resources = append(reversePlan.Resources, resource)
		}
	}

	// Execute rollback
	rollbackResult, err := m.ExecuteMigration(ctx, reversePlan)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if rollbackResult.FailedCount > 0 {
		return fmt.Errorf("rollback completed with %d failures", rollbackResult.FailedCount)
	}

	m.log.Info("Rollback completed successfully")
	return nil
}

// GetMigrationStatus returns the current status of migrations
func (m *MigrationManager) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	return m.statusReporter.GetCurrentStatus()
}

// GetMigrationHistory returns the history of migrations
func (m *MigrationManager) GetMigrationHistory(ctx context.Context, limit int) ([]*MigrationHistoryEntry, error) {
	return m.tracker.GetMigrationHistory(limit)
}

// MigrationOption is a function that configures migration options
type MigrationOption func(*migrationOptions)

type migrationOptions struct {
	batchSize      int
	dryRun         bool
	parallel       bool
	maxConcurrency int
	preValidation  bool
	postValidation bool
}

// WithBatchSize sets the batch size for migration
func WithBatchSize(size int) MigrationOption {
	return func(o *migrationOptions) {
		o.batchSize = size
	}
}

// WithDryRun enables dry run mode
func WithDryRun(dryRun bool) MigrationOption {
	return func(o *migrationOptions) {
		o.dryRun = dryRun
	}
}

// WithParallel enables parallel migration
func WithParallel(parallel bool) MigrationOption {
	return func(o *migrationOptions) {
		o.parallel = parallel
	}
}

// WithMaxConcurrency sets the maximum concurrency for parallel migration
func WithMaxConcurrency(max int) MigrationOption {
	return func(o *migrationOptions) {
		o.maxConcurrency = max
	}
}

// WithValidation configures validation options
func WithValidation(pre, post bool) MigrationOption {
	return func(o *migrationOptions) {
		o.preValidation = pre
		o.postValidation = post
	}
}
