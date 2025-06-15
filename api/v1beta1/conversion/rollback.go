/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RollbackManager handles conversion rollbacks
type RollbackManager struct {
	logger    logr.Logger
	client    client.Client
	store     RollbackStore
	validator *FieldValidator
}

// RollbackStore interface for storing rollback data
type RollbackStore interface {
	// SaveSnapshot saves a snapshot before conversion
	SaveSnapshot(ctx context.Context, snapshot *ConversionSnapshot) error
	
	// GetSnapshot retrieves a snapshot
	GetSnapshot(ctx context.Context, key types.NamespacedName, version string) (*ConversionSnapshot, error)
	
	// ListSnapshots lists all snapshots for a resource
	ListSnapshots(ctx context.Context, key types.NamespacedName) ([]*ConversionSnapshot, error)
	
	// DeleteSnapshot removes a snapshot
	DeleteSnapshot(ctx context.Context, key types.NamespacedName, version string) error
	
	// Cleanup removes old snapshots
	Cleanup(ctx context.Context, retention time.Duration) error
}

// ConversionSnapshot represents a point-in-time backup
type ConversionSnapshot struct {
	// Metadata
	ResourceKey      types.NamespacedName `json:"resource_key"`
	SourceVersion    string               `json:"source_version"`
	TargetVersion    string               `json:"target_version"`
	Timestamp        time.Time            `json:"timestamp"`
	SnapshotID       string               `json:"snapshot_id"`
	
	// Original object data
	OriginalObject   json.RawMessage      `json:"original_object"`
	OriginalChecksum string               `json:"original_checksum"`
	
	// Conversion context
	ConversionReason string               `json:"conversion_reason"`
	ConversionMethod string               `json:"conversion_method"` // webhook, manual, migration
	Annotations      map[string]string    `json:"annotations"`
	
	// Validation state
	ValidationResult *ValidationResult    `json:"validation_result,omitempty"`
}

// RollbackRequest represents a request to rollback a conversion
type RollbackRequest struct {
	ResourceKey   types.NamespacedName
	SnapshotID    string
	Reason        string
	Force         bool // Force rollback even with warnings
	ValidateOnly  bool // Only validate, don't perform rollback
}

// RollbackResult contains the result of a rollback operation
type RollbackResult struct {
	Success        bool              `json:"success"`
	RolledBackTo   string            `json:"rolled_back_to_version"`
	ValidationResult *ValidationResult `json:"validation_result"`
	Warnings       []string          `json:"warnings"`
	Error          string            `json:"error,omitempty"`
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(logger logr.Logger, client client.Client, store RollbackStore) *RollbackManager {
	return &RollbackManager{
		logger:    logger.WithName("rollback-manager"),
		client:    client,
		store:     store,
		validator: NewFieldValidator(logger),
	}
}

// CreateSnapshot creates a snapshot before conversion
func (rm *RollbackManager) CreateSnapshot(ctx context.Context, obj runtime.Object, targetVersion, reason string) (*ConversionSnapshot, error) {
	// Get object metadata
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return nil, fmt.Errorf("getting object key: %w", err)
	}
	
	// Determine source version
	sourceVersion, err := rm.getObjectVersion(obj)
	if err != nil {
		return nil, fmt.Errorf("getting source version: %w", err)
	}
	
	// Serialize object
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("marshaling object: %w", err)
	}
	
	// Create snapshot
	snapshot := &ConversionSnapshot{
		ResourceKey:      key,
		SourceVersion:    sourceVersion,
		TargetVersion:    targetVersion,
		Timestamp:        time.Now(),
		SnapshotID:       rm.generateSnapshotID(key, sourceVersion),
		OriginalObject:   data,
		OriginalChecksum: rm.calculateChecksum(data),
		ConversionReason: reason,
		ConversionMethod: "webhook", // Default, can be overridden
		Annotations:      make(map[string]string),
	}
	
	// Validate the snapshot
	if err := rm.validateSnapshot(snapshot); err != nil {
		return nil, fmt.Errorf("validating snapshot: %w", err)
	}
	
	// Save snapshot
	if err := rm.store.SaveSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("saving snapshot: %w", err)
	}
	
	rm.logger.Info("Created conversion snapshot",
		"resource", key,
		"snapshotID", snapshot.SnapshotID,
		"sourceVersion", sourceVersion,
		"targetVersion", targetVersion)
	
	return snapshot, nil
}

// Rollback performs a rollback to a previous version
func (rm *RollbackManager) Rollback(ctx context.Context, req RollbackRequest) (*RollbackResult, error) {
	result := &RollbackResult{
		Success:  false,
		Warnings: []string{},
	}
	
	// Retrieve snapshot
	snapshot, err := rm.store.GetSnapshot(ctx, req.ResourceKey, req.SnapshotID)
	if err != nil {
		result.Error = fmt.Sprintf("retrieving snapshot: %v", err)
		return result, err
	}
	
	// Verify snapshot integrity
	if !rm.verifySnapshotIntegrity(snapshot) {
		result.Error = "snapshot integrity check failed"
		return result, fmt.Errorf("snapshot corrupted or tampered")
	}
	
	// Unmarshal original object
	originalObj, err := rm.unmarshalSnapshot(snapshot)
	if err != nil {
		result.Error = fmt.Sprintf("unmarshaling snapshot: %v", err)
		return result, err
	}
	
	// Get current object
	currentObj, err := rm.getCurrentObject(ctx, req.ResourceKey, snapshot.TargetVersion)
	if err != nil {
		result.Error = fmt.Sprintf("getting current object: %v", err)
		return result, err
	}
	
	// Validate rollback
	validationResult := rm.validator.ValidateConversion(
		snapshot.TargetVersion,
		snapshot.SourceVersion,
		currentObj,
		originalObj,
	)
	result.ValidationResult = validationResult
	
	// Check for critical issues
	if len(validationResult.Errors) > 0 && !req.Force {
		result.Error = "validation failed, use force flag to override"
		result.Warnings = append(result.Warnings, 
			"Rollback validation failed with errors. Use force flag to proceed anyway.")
		return result, nil
	}
	
	// Add warnings
	result.Warnings = append(result.Warnings, validationResult.Warnings...)
	
	// If validate only, return here
	if req.ValidateOnly {
		result.Success = true
		result.RolledBackTo = snapshot.SourceVersion
		return result, nil
	}
	
	// Perform the rollback
	if err := rm.performRollback(ctx, req.ResourceKey, originalObj); err != nil {
		result.Error = fmt.Sprintf("performing rollback: %v", err)
		return result, err
	}
	
	// Record rollback metrics
	metrics := GetMetrics()
	metrics.RecordConversion(
		snapshot.TargetVersion,
		snapshot.SourceVersion,
		"rollback",
		time.Since(snapshot.Timestamp),
		true,
	)
	
	result.Success = true
	result.RolledBackTo = snapshot.SourceVersion
	
	rm.logger.Info("Successfully rolled back conversion",
		"resource", req.ResourceKey,
		"from", snapshot.TargetVersion,
		"to", snapshot.SourceVersion,
		"reason", req.Reason)
	
	return result, nil
}

// ListAvailableRollbacks lists available rollback points
func (rm *RollbackManager) ListAvailableRollbacks(ctx context.Context, key types.NamespacedName) ([]*RollbackOption, error) {
	snapshots, err := rm.store.ListSnapshots(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("listing snapshots: %w", err)
	}
	
	options := make([]*RollbackOption, 0, len(snapshots))
	for _, snapshot := range snapshots {
		option := &RollbackOption{
			SnapshotID:    snapshot.SnapshotID,
			Version:       snapshot.SourceVersion,
			Timestamp:     snapshot.Timestamp,
			Reason:        snapshot.ConversionReason,
			CanRollback:   rm.canRollback(snapshot),
			RollbackRisks: rm.assessRollbackRisks(snapshot),
		}
		options = append(options, option)
	}
	
	return options, nil
}

// RollbackOption represents an available rollback point
type RollbackOption struct {
	SnapshotID    string    `json:"snapshot_id"`
	Version       string    `json:"version"`
	Timestamp     time.Time `json:"timestamp"`
	Reason        string    `json:"reason"`
	CanRollback   bool      `json:"can_rollback"`
	RollbackRisks []string  `json:"rollback_risks"`
}

// AutoRollback performs automatic rollback on conversion failure
func (rm *RollbackManager) AutoRollback(ctx context.Context, key types.NamespacedName, conversionError error) error {
	rm.logger.Info("Attempting automatic rollback due to conversion failure",
		"resource", key,
		"error", conversionError)
	
	// Get the most recent snapshot
	snapshots, err := rm.store.ListSnapshots(ctx, key)
	if err != nil {
		return fmt.Errorf("listing snapshots for auto-rollback: %w", err)
	}
	
	if len(snapshots) == 0 {
		return fmt.Errorf("no snapshots available for rollback")
	}
	
	// Find the most recent valid snapshot
	var latestSnapshot *ConversionSnapshot
	for _, snapshot := range snapshots {
		if rm.canRollback(snapshot) {
			if latestSnapshot == nil || snapshot.Timestamp.After(latestSnapshot.Timestamp) {
				latestSnapshot = snapshot
			}
		}
	}
	
	if latestSnapshot == nil {
		return fmt.Errorf("no valid snapshot found for auto-rollback")
	}
	
	// Perform rollback
	req := RollbackRequest{
		ResourceKey: key,
		SnapshotID:  latestSnapshot.SnapshotID,
		Reason:      fmt.Sprintf("Auto-rollback due to conversion error: %v", conversionError),
		Force:       false, // Don't force in auto-rollback
	}
	
	result, err := rm.Rollback(ctx, req)
	if err != nil {
		return fmt.Errorf("auto-rollback failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("auto-rollback unsuccessful: %s", result.Error)
	}
	
	return nil
}

// CleanupOldSnapshots removes old snapshots based on retention policy
func (rm *RollbackManager) CleanupOldSnapshots(ctx context.Context, retention time.Duration) error {
	return rm.store.Cleanup(ctx, retention)
}

// Helper methods

// getObjectVersion determines the API version of an object
func (rm *RollbackManager) getObjectVersion(obj runtime.Object) (string, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Version == "" {
		return "", fmt.Errorf("object has no version")
	}
	return gvk.Version, nil
}

// generateSnapshotID generates a unique snapshot ID
func (rm *RollbackManager) generateSnapshotID(key types.NamespacedName, version string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s-%s", key.Namespace, key.Name, version, timestamp)
}

// calculateChecksum calculates checksum for data integrity
func (rm *RollbackManager) calculateChecksum(data []byte) string {
	// In production, use crypto/sha256
	return fmt.Sprintf("%x", len(data)) // Simplified for example
}

// validateSnapshot validates a snapshot before saving
func (rm *RollbackManager) validateSnapshot(snapshot *ConversionSnapshot) error {
	if snapshot.ResourceKey.Name == "" || snapshot.ResourceKey.Namespace == "" {
		return fmt.Errorf("invalid resource key")
	}
	
	if snapshot.SourceVersion == "" || snapshot.TargetVersion == "" {
		return fmt.Errorf("missing version information")
	}
	
	if len(snapshot.OriginalObject) == 0 {
		return fmt.Errorf("empty object data")
	}
	
	return nil
}

// verifySnapshotIntegrity verifies snapshot hasn't been corrupted
func (rm *RollbackManager) verifySnapshotIntegrity(snapshot *ConversionSnapshot) bool {
	checksum := rm.calculateChecksum(snapshot.OriginalObject)
	return checksum == snapshot.OriginalChecksum
}

// unmarshalSnapshot unmarshals the snapshot data
func (rm *RollbackManager) unmarshalSnapshot(snapshot *ConversionSnapshot) (runtime.Object, error) {
	// In production, use scheme.Scheme to decode
	// This is simplified for the example
	var obj runtime.Object
	
	// Determine object type based on version
	switch snapshot.SourceVersion {
	case "v1alpha1":
		obj = &v1alpha1.ObservabilityPlatform{}
	case "v1beta1":
		obj = &v1beta1.ObservabilityPlatform{}
	default:
		return nil, fmt.Errorf("unknown version: %s", snapshot.SourceVersion)
	}
	
	if err := json.Unmarshal(snapshot.OriginalObject, obj); err != nil {
		return nil, err
	}
	
	return obj, nil
}

// getCurrentObject retrieves the current object from the cluster
func (rm *RollbackManager) getCurrentObject(ctx context.Context, key types.NamespacedName, version string) (runtime.Object, error) {
	// In production, use the client to get the object
	// This is simplified for the example
	var obj runtime.Object
	
	switch version {
	case "v1alpha1":
		obj = &v1alpha1.ObservabilityPlatform{}
	case "v1beta1":
		obj = &v1beta1.ObservabilityPlatform{}
	default:
		return nil, fmt.Errorf("unknown version: %s", version)
	}
	
	if err := rm.client.Get(ctx, key, obj.(client.Object)); err != nil {
		return nil, err
	}
	
	return obj, nil
}

// performRollback applies the rollback
func (rm *RollbackManager) performRollback(ctx context.Context, key types.NamespacedName, originalObj runtime.Object) error {
	// In production, this would use the client to update the object
	// ensuring proper resource version handling
	
	return rm.client.Update(ctx, originalObj.(client.Object))
}

// canRollback checks if a snapshot can be rolled back to
func (rm *RollbackManager) canRollback(snapshot *ConversionSnapshot) bool {
	// Check age - don't rollback to very old snapshots
	if time.Since(snapshot.Timestamp) > 7*24*time.Hour {
		return false
	}
	
	// Check if the version is still supported
	supportedVersions := []string{"v1alpha1", "v1beta1"}
	for _, v := range supportedVersions {
		if v == snapshot.SourceVersion {
			return true
		}
	}
	
	return false
}

// assessRollbackRisks evaluates risks of rolling back
func (rm *RollbackManager) assessRollbackRisks(snapshot *ConversionSnapshot) []string {
	risks := []string{}
	
	// Check age
	age := time.Since(snapshot.Timestamp)
	if age > 24*time.Hour {
		risks = append(risks, fmt.Sprintf("Snapshot is %v old, state may have diverged significantly", age))
	}
	
	// Check version downgrade
	if snapshot.SourceVersion < snapshot.TargetVersion {
		risks = append(risks, "Rolling back to an older API version may lose features")
	}
	
	// Check for known breaking changes
	if snapshot.TargetVersion == "v1beta1" && snapshot.SourceVersion == "v1alpha1" {
		risks = append(risks, "Security and cost optimization configurations will be lost")
	}
	
	return risks
}

// InMemoryRollbackStore provides an in-memory implementation of RollbackStore
type InMemoryRollbackStore struct {
	snapshots map[string]*ConversionSnapshot
	mu        sync.RWMutex
}

// NewInMemoryRollbackStore creates a new in-memory store
func NewInMemoryRollbackStore() *InMemoryRollbackStore {
	return &InMemoryRollbackStore{
		snapshots: make(map[string]*ConversionSnapshot),
	}
}

// SaveSnapshot saves a snapshot
func (s *InMemoryRollbackStore) SaveSnapshot(ctx context.Context, snapshot *ConversionSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := s.makeKey(snapshot.ResourceKey, snapshot.SnapshotID)
	s.snapshots[key] = snapshot
	return nil
}

// GetSnapshot retrieves a snapshot
func (s *InMemoryRollbackStore) GetSnapshot(ctx context.Context, key types.NamespacedName, snapshotID string) (*ConversionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	mapKey := s.makeKey(key, snapshotID)
	snapshot, ok := s.snapshots[mapKey]
	if !ok {
		return nil, fmt.Errorf("snapshot not found")
	}
	
	return snapshot, nil
}

// ListSnapshots lists all snapshots for a resource
func (s *InMemoryRollbackStore) ListSnapshots(ctx context.Context, key types.NamespacedName) ([]*ConversionSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var snapshots []*ConversionSnapshot
	prefix := fmt.Sprintf("%s/%s/", key.Namespace, key.Name)
	
	for k, v := range s.snapshots {
		if hasPrefix(k, prefix) {
			snapshots = append(snapshots, v)
		}
	}
	
	return snapshots, nil
}

// DeleteSnapshot removes a snapshot
func (s *InMemoryRollbackStore) DeleteSnapshot(ctx context.Context, key types.NamespacedName, snapshotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	mapKey := s.makeKey(key, snapshotID)
	delete(s.snapshots, mapKey)
	return nil
}

// Cleanup removes old snapshots
func (s *InMemoryRollbackStore) Cleanup(ctx context.Context, retention time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-retention)
	
	for key, snapshot := range s.snapshots {
		if snapshot.Timestamp.Before(cutoff) {
			delete(s.snapshots, key)
		}
	}
	
	return nil
}

// makeKey creates a map key
func (s *InMemoryRollbackStore) makeKey(key types.NamespacedName, snapshotID string) string {
	return fmt.Sprintf("%s/%s/%s", key.Namespace, key.Name, snapshotID)
}

// hasPrefix checks if a string has a prefix (Go 1.19 compatibility)
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}
