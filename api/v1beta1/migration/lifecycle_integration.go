/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// LifecycleIntegrationManager manages integration with operator lifecycle
type LifecycleIntegrationManager struct {
	client         client.Client
	logger         logr.Logger
	eventRecorder  record.EventRecorder
	hooks          map[string][]LifecycleHook
	validators     map[string][]MigrationValidator
	rollbackMgr    *RollbackManager
}

// LifecycleHook defines a hook to be executed during migration
type LifecycleHook interface {
	Name() string
	Phase() HookPhase
	Execute(ctx context.Context, obj runtime.Object) error
}

// HookPhase defines when a hook should be executed
type HookPhase string

const (
	HookPhasePre  HookPhase = "pre"
	HookPhasePost HookPhase = "post"
)

// MigrationValidator validates migration readiness and results
type MigrationValidator interface {
	Name() string
	ValidatePreMigration(ctx context.Context, resource types.NamespacedName, targetVersion string) error
	ValidatePostMigration(ctx context.Context, resource types.NamespacedName, targetVersion string) error
}

// RollbackManager handles migration rollbacks
type RollbackManager struct {
	client    client.Client
	logger    logr.Logger
	snapshots map[string]*ResourceSnapshot
}

// ResourceSnapshot represents a snapshot of a resource before migration
type ResourceSnapshot struct {
	Resource      runtime.Object
	Version       string
	Timestamp     time.Time
	Checksum      string
	RelatedResources []types.NamespacedName
}

// NewLifecycleIntegrationManager creates a new lifecycle integration manager
func NewLifecycleIntegrationManager(client client.Client, logger logr.Logger) *LifecycleIntegrationManager {
	return &LifecycleIntegrationManager{
		client:      client,
		logger:      logger.WithName("lifecycle-integration"),
		hooks:       make(map[string][]LifecycleHook),
		validators:  make(map[string][]MigrationValidator),
		rollbackMgr: NewRollbackManager(client, logger),
	}
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(client client.Client, logger logr.Logger) *RollbackManager {
	return &RollbackManager{
		client:    client,
		logger:    logger.WithName("rollback-manager"),
		snapshots: make(map[string]*ResourceSnapshot),
	}
}

// PreMigrationCheck performs pre-migration validation
func (l *LifecycleIntegrationManager) PreMigrationCheck(ctx context.Context, resource types.NamespacedName, targetVersion string) error {
	l.logger.Info("Performing pre-migration checks",
		"resource", resource,
		"targetVersion", targetVersion)
	
	// Create snapshot for rollback
	if err := l.rollbackMgr.CreateSnapshot(ctx, resource); err != nil {
		return fmt.Errorf("failed to create rollback snapshot: %w", err)
	}
	
	// Run validators
	validators := l.validators[targetVersion]
	for _, validator := range validators {
		if err := validator.ValidatePreMigration(ctx, resource, targetVersion); err != nil {
			return fmt.Errorf("pre-migration validation failed (%s): %w", validator.Name(), err)
		}
	}
	
	// Check resource health
	if err := l.checkResourceHealth(ctx, resource); err != nil {
		return fmt.Errorf("resource health check failed: %w", err)
	}
	
	// Check dependencies
	if err := l.checkDependencies(ctx, resource); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}
	
	// Record event
	l.recordEvent(resource, corev1.EventTypeNormal, "PreMigrationCheckPassed", 
		fmt.Sprintf("Pre-migration checks passed for version %s", targetVersion))
	
	return nil
}

// ApplyMigrationHooks applies lifecycle hooks during migration
func (l *LifecycleIntegrationManager) ApplyMigrationHooks(ctx context.Context, obj runtime.Object) error {
	// Get object metadata
	accessor, err := metav1.NewAccessor().ObjectMetadata(obj)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}
	
	version := accessor.GetAPIVersion()
	
	// Apply pre-migration hooks
	preHooks := l.getHooksByPhase(version, HookPhasePre)
	for _, hook := range preHooks {
		l.logger.V(1).Info("Applying pre-migration hook", "hook", hook.Name())
		if err := hook.Execute(ctx, obj); err != nil {
			return fmt.Errorf("pre-migration hook failed (%s): %w", hook.Name(), err)
		}
	}
	
	// The actual migration happens here (handled by caller)
	
	// Apply post-migration hooks
	postHooks := l.getHooksByPhase(version, HookPhasePost)
	for _, hook := range postHooks {
		l.logger.V(1).Info("Applying post-migration hook", "hook", hook.Name())
		if err := hook.Execute(ctx, obj); err != nil {
			return fmt.Errorf("post-migration hook failed (%s): %w", hook.Name(), err)
		}
	}
	
	return nil
}

// PostMigrationValidation performs post-migration validation
func (l *LifecycleIntegrationManager) PostMigrationValidation(ctx context.Context, resource types.NamespacedName, targetVersion string) error {
	l.logger.Info("Performing post-migration validation",
		"resource", resource,
		"targetVersion", targetVersion)
	
	// Run validators
	validators := l.validators[targetVersion]
	for _, validator := range validators {
		if err := validator.ValidatePostMigration(ctx, resource, targetVersion); err != nil {
			return fmt.Errorf("post-migration validation failed (%s): %w", validator.Name(), err)
		}
	}
	
	// Verify resource integrity
	if err := l.verifyResourceIntegrity(ctx, resource, targetVersion); err != nil {
		return fmt.Errorf("resource integrity verification failed: %w", err)
	}
	
	// Verify functionality
	if err := l.verifyFunctionality(ctx, resource); err != nil {
		return fmt.Errorf("functionality verification failed: %w", err)
	}
	
	// Clean up snapshot if validation passed
	l.rollbackMgr.DeleteSnapshot(resource)
	
	// Record event
	l.recordEvent(resource, corev1.EventTypeNormal, "MigrationCompleted",
		fmt.Sprintf("Successfully migrated to version %s", targetVersion))
	
	return nil
}

// RollbackMigration rolls back a failed migration
func (l *LifecycleIntegrationManager) RollbackMigration(ctx context.Context, resource types.NamespacedName) error {
	l.logger.Info("Rolling back migration", "resource", resource)
	
	// Get snapshot
	snapshot, err := l.rollbackMgr.GetSnapshot(resource)
	if err != nil {
		return fmt.Errorf("failed to get rollback snapshot: %w", err)
	}
	
	// Restore resource
	if err := l.rollbackMgr.RestoreSnapshot(ctx, resource); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}
	
	// Verify rollback
	if err := l.verifyRollback(ctx, resource, snapshot); err != nil {
		return fmt.Errorf("rollback verification failed: %w", err)
	}
	
	// Record event
	l.recordEvent(resource, corev1.EventTypeWarning, "MigrationRolledBack",
		"Migration was rolled back due to failure")
	
	return nil
}

// RegisterHook registers a lifecycle hook
func (l *LifecycleIntegrationManager) RegisterHook(version string, hook LifecycleHook) {
	l.hooks[version] = append(l.hooks[version], hook)
	l.logger.Info("Registered lifecycle hook",
		"version", version,
		"hook", hook.Name(),
		"phase", hook.Phase())
}

// RegisterValidator registers a migration validator
func (l *LifecycleIntegrationManager) RegisterValidator(version string, validator MigrationValidator) {
	l.validators[version] = append(l.validators[version], validator)
	l.logger.Info("Registered migration validator",
		"version", version,
		"validator", validator.Name())
}

// checkResourceHealth checks if the resource is healthy before migration
func (l *LifecycleIntegrationManager) checkResourceHealth(ctx context.Context, resource types.NamespacedName) error {
	// Get the platform
	platform := &observabilityv1beta1.ObservabilityPlatform{}
	if err := l.client.Get(ctx, resource, platform); err != nil {
		return fmt.Errorf("failed to get platform: %w", err)
	}
	
	// Check status
	if platform.Status.Phase == "Failed" || platform.Status.Phase == "Unknown" {
		return fmt.Errorf("platform is in unhealthy state: %s", platform.Status.Phase)
	}
	
	// Check conditions
	for _, condition := range platform.Status.Conditions {
		if condition.Type == "Ready" && condition.Status != metav1.ConditionTrue {
			return fmt.Errorf("platform is not ready: %s", condition.Message)
		}
	}
	
	return nil
}

// checkDependencies checks if dependencies allow migration
func (l *LifecycleIntegrationManager) checkDependencies(ctx context.Context, resource types.NamespacedName) error {
	// Check if any dependent resources exist
	// This is a simplified check - in real implementation, this would check
	// for actual dependencies like ConfigMaps, Secrets, etc.
	
	platform := &observabilityv1beta1.ObservabilityPlatform{}
	if err := l.client.Get(ctx, resource, platform); err != nil {
		return fmt.Errorf("failed to get platform: %w", err)
	}
	
	// Check if components are being updated
	if platform.Status.Phase == "Upgrading" {
		return fmt.Errorf("cannot migrate while platform is upgrading")
	}
	
	return nil
}

// verifyResourceIntegrity verifies resource integrity after migration
func (l *LifecycleIntegrationManager) verifyResourceIntegrity(ctx context.Context, resource types.NamespacedName, targetVersion string) error {
	// Get the migrated resource
	platform := &observabilityv1beta1.ObservabilityPlatform{}
	if err := l.client.Get(ctx, resource, platform); err != nil {
		return fmt.Errorf("failed to get migrated platform: %w", err)
	}
	
	// Verify API version
	if platform.APIVersion != fmt.Sprintf("observability.io/%s", targetVersion) {
		return fmt.Errorf("API version mismatch: expected %s, got %s", targetVersion, platform.APIVersion)
	}
	
	// Verify required fields are present
	if platform.Spec.Components == nil {
		return fmt.Errorf("components field is missing after migration")
	}
	
	return nil
}

// verifyFunctionality verifies that the resource is functional after migration
func (l *LifecycleIntegrationManager) verifyFunctionality(ctx context.Context, resource types.NamespacedName) error {
	// Wait for resource to become ready
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for resource to become ready")
		case <-ticker.C:
			platform := &observabilityv1beta1.ObservabilityPlatform{}
			if err := l.client.Get(ctx, resource, platform); err != nil {
				if errors.IsNotFound(err) {
					return fmt.Errorf("resource not found after migration")
				}
				continue
			}
			
			if platform.Status.Phase == "Ready" {
				return nil
			}
			
			if platform.Status.Phase == "Failed" {
				return fmt.Errorf("resource failed after migration: %s", platform.Status.Message)
			}
		}
	}
}

// verifyRollback verifies that rollback was successful
func (l *LifecycleIntegrationManager) verifyRollback(ctx context.Context, resource types.NamespacedName, snapshot *ResourceSnapshot) error {
	// Get the rolled back resource
	platform := &observabilityv1beta1.ObservabilityPlatform{}
	if err := l.client.Get(ctx, resource, platform); err != nil {
		return fmt.Errorf("failed to get rolled back platform: %w", err)
	}
	
	// Verify version matches snapshot
	if platform.APIVersion != snapshot.Version {
		return fmt.Errorf("rollback version mismatch: expected %s, got %s", 
			snapshot.Version, platform.APIVersion)
	}
	
	return nil
}

// getHooksByPhase returns hooks for a specific phase
func (l *LifecycleIntegrationManager) getHooksByPhase(version string, phase HookPhase) []LifecycleHook {
	var phaseHooks []LifecycleHook
	
	for _, hook := range l.hooks[version] {
		if hook.Phase() == phase {
			phaseHooks = append(phaseHooks, hook)
		}
	}
	
	return phaseHooks
}

// recordEvent records a Kubernetes event
func (l *LifecycleIntegrationManager) recordEvent(resource types.NamespacedName, eventType, reason, message string) {
	if l.eventRecorder != nil {
		// In real implementation, this would record an event on the actual resource
		l.logger.Info("Recording event",
			"resource", resource,
			"type", eventType,
			"reason", reason,
			"message", message)
	}
}

// CreateSnapshot creates a snapshot of a resource
func (r *RollbackManager) CreateSnapshot(ctx context.Context, resource types.NamespacedName) error {
	platform := &observabilityv1beta1.ObservabilityPlatform{}
	if err := r.client.Get(ctx, resource, platform); err != nil {
		return fmt.Errorf("failed to get resource for snapshot: %w", err)
	}
	
	snapshot := &ResourceSnapshot{
		Resource:  platform.DeepCopy(),
		Version:   platform.APIVersion,
		Timestamp: time.Now(),
		Checksum:  r.calculateChecksum(platform),
	}
	
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	r.snapshots[key] = snapshot
	
	r.logger.Info("Created resource snapshot",
		"resource", resource,
		"version", snapshot.Version)
	
	return nil
}

// GetSnapshot retrieves a resource snapshot
func (r *RollbackManager) GetSnapshot(resource types.NamespacedName) (*ResourceSnapshot, error) {
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	snapshot, exists := r.snapshots[key]
	if !exists {
		return nil, fmt.Errorf("snapshot not found for resource: %s", key)
	}
	
	return snapshot, nil
}

// RestoreSnapshot restores a resource from snapshot
func (r *RollbackManager) RestoreSnapshot(ctx context.Context, resource types.NamespacedName) error {
	snapshot, err := r.GetSnapshot(resource)
	if err != nil {
		return err
	}
	
	// Update the resource
	if err := r.client.Update(ctx, snapshot.Resource); err != nil {
		return fmt.Errorf("failed to restore resource: %w", err)
	}
	
	r.logger.Info("Restored resource from snapshot",
		"resource", resource,
		"snapshotTime", snapshot.Timestamp)
	
	return nil
}

// DeleteSnapshot deletes a resource snapshot
func (r *RollbackManager) DeleteSnapshot(resource types.NamespacedName) {
	key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
	delete(r.snapshots, key)
	
	r.logger.V(1).Info("Deleted resource snapshot", "resource", resource)
}

// calculateChecksum calculates a checksum for a resource
func (r *RollbackManager) calculateChecksum(obj runtime.Object) string {
	// In real implementation, this would calculate a proper checksum
	return fmt.Sprintf("%x", time.Now().Unix())
}
