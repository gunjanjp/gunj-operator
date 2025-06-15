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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemaEvolutionTracker tracks schema changes and migration history
type SchemaEvolutionTracker struct {
	log     logr.Logger
	storage SchemaStorage
	mu      sync.RWMutex
	cache   map[string]*SchemaEvolution
}

// SchemaEvolution represents the evolution of a schema
type SchemaEvolution struct {
	GroupKind       schema.GroupKind
	Versions        []VersionInfo
	Migrations      []MigrationRecord
	CurrentVersion  string
	DeprecatedPaths []DeprecatedPath
}

// VersionInfo contains information about a specific version
type VersionInfo struct {
	Version         string
	ReleaseDate     time.Time
	Fields          map[string]FieldInfo
	Deprecated      bool
	DeprecationDate *time.Time
	RemovalDate     *time.Time
}

// FieldInfo contains information about a field
type FieldInfo struct {
	Path        string
	Type        string
	Description string
	Added       time.Time
	Deprecated  *time.Time
	Removed     *time.Time
	MigratedTo  string
}

// MigrationRecord represents a migration between versions
type MigrationRecord struct {
	ID             string
	FromVersion    string
	ToVersion      string
	StartTime      time.Time
	EndTime        *time.Time
	Status         MigrationStatus
	TotalResources int
	Successful     int
	Failed         int
	Details        map[string]interface{}
}

// MigrationStatus represents the status of a migration
type MigrationStatus string

const (
	MigrationStatusPending    MigrationStatus = "Pending"
	MigrationStatusInProgress MigrationStatus = "InProgress"
	MigrationStatusCompleted  MigrationStatus = "Completed"
	MigrationStatusFailed     MigrationStatus = "Failed"
	MigrationStatusRollback   MigrationStatus = "Rollback"
)

// DeprecatedPath represents a deprecated field path
type DeprecatedPath struct {
	Path            string
	DeprecatedSince string
	RemovedIn       string
	MigrationPath   string
	Description     string
}

// MigrationHistoryEntry represents an entry in migration history
type MigrationHistoryEntry struct {
	Timestamp   time.Time
	Plan        *MigrationPlan
	Result      *MigrationResult
	Annotations map[string]string
}

// SchemaStorage interface for persisting schema evolution data
type SchemaStorage interface {
	SaveEvolution(ctx context.Context, evolution *SchemaEvolution) error
	LoadEvolution(ctx context.Context, gk schema.GroupKind) (*SchemaEvolution, error)
	SaveMigrationHistory(ctx context.Context, entry *MigrationHistoryEntry) error
	LoadMigrationHistory(ctx context.Context, limit int) ([]*MigrationHistoryEntry, error)
}

// NewSchemaEvolutionTracker creates a new schema evolution tracker
func NewSchemaEvolutionTracker(log logr.Logger) *SchemaEvolutionTracker {
	return &SchemaEvolutionTracker{
		log:     log.WithName("schema-evolution-tracker"),
		storage: NewInMemorySchemaStorage(), // Default to in-memory storage
		cache:   make(map[string]*SchemaEvolution),
	}
}

// WithStorage sets a custom storage backend
func (t *SchemaEvolutionTracker) WithStorage(storage SchemaStorage) *SchemaEvolutionTracker {
	t.storage = storage
	return t
}

// TrackSchemaChange records a schema change
func (t *SchemaEvolutionTracker) TrackSchemaChange(gk schema.GroupKind, fromVersion, toVersion string, changes []FieldChange) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := gk.String()
	evolution, exists := t.cache[key]
	if !exists {
		evolution = &SchemaEvolution{
			GroupKind:       gk,
			Versions:        []VersionInfo{},
			Migrations:      []MigrationRecord{},
			DeprecatedPaths: []DeprecatedPath{},
		}
		t.cache[key] = evolution
	}

	// Update version info
	t.updateVersionInfo(evolution, fromVersion, toVersion, changes)

	// Save to storage
	if err := t.storage.SaveEvolution(context.Background(), evolution); err != nil {
		t.log.Error(err, "Failed to save schema evolution")
		return err
	}

	return nil
}

// updateVersionInfo updates version information based on changes
func (t *SchemaEvolutionTracker) updateVersionInfo(evolution *SchemaEvolution, fromVersion, toVersion string, changes []FieldChange) {
	// Find or create version info
	var toVersionInfo *VersionInfo
	for i := range evolution.Versions {
		if evolution.Versions[i].Version == toVersion {
			toVersionInfo = &evolution.Versions[i]
			break
		}
	}

	if toVersionInfo == nil {
		toVersionInfo = &VersionInfo{
			Version:     toVersion,
			ReleaseDate: time.Now(),
			Fields:      make(map[string]FieldInfo),
		}
		evolution.Versions = append(evolution.Versions, *toVersionInfo)
	}

	// Apply changes
	for _, change := range changes {
		switch change.Type {
		case FieldChangeTypeAdded:
			toVersionInfo.Fields[change.Path] = FieldInfo{
				Path:        change.Path,
				Type:        change.FieldType,
				Description: change.Description,
				Added:       time.Now(),
			}
		case FieldChangeTypeRemoved:
			if info, exists := toVersionInfo.Fields[change.Path]; exists {
				now := time.Now()
				info.Removed = &now
				toVersionInfo.Fields[change.Path] = info
			}
		case FieldChangeTypeModified:
			if info, exists := toVersionInfo.Fields[change.Path]; exists {
				info.Type = change.FieldType
				info.Description = change.Description
				toVersionInfo.Fields[change.Path] = info
			}
		case FieldChangeTypeDeprecated:
			if info, exists := toVersionInfo.Fields[change.Path]; exists {
				now := time.Now()
				info.Deprecated = &now
				info.MigratedTo = change.MigratedTo
				toVersionInfo.Fields[change.Path] = info
			}
			evolution.DeprecatedPaths = append(evolution.DeprecatedPaths, DeprecatedPath{
				Path:            change.Path,
				DeprecatedSince: toVersion,
				MigrationPath:   change.MigratedTo,
				Description:     change.Description,
			})
		}
	}

	evolution.CurrentVersion = toVersion
}

// RecordMigrationPlan records a migration plan
func (t *SchemaEvolutionTracker) RecordMigrationPlan(plan *MigrationPlan) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	gk := schema.GroupKind{
		Group: plan.SourceGVK.Group,
		Kind:  plan.SourceGVK.Kind,
	}

	key := gk.String()
	evolution, exists := t.cache[key]
	if !exists {
		evolution = &SchemaEvolution{
			GroupKind:  gk,
			Migrations: []MigrationRecord{},
		}
		t.cache[key] = evolution
	}

	record := MigrationRecord{
		ID:             generateMigrationID(),
		FromVersion:    plan.SourceGVK.Version,
		ToVersion:      plan.TargetGVK.Version,
		StartTime:      time.Now(),
		Status:         MigrationStatusPending,
		TotalResources: len(plan.Resources),
		Details: map[string]interface{}{
			"batchSize":  plan.BatchSize,
			"dryRun":    plan.DryRun,
			"parallel":  plan.Parallel,
			"resources": plan.Resources,
		},
	}

	evolution.Migrations = append(evolution.Migrations, record)
	return t.storage.SaveEvolution(context.Background(), evolution)
}

// RecordMigrationResult records the result of a migration
func (t *SchemaEvolutionTracker) RecordMigrationResult(plan *MigrationPlan, result *MigrationResult) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry := &MigrationHistoryEntry{
		Timestamp: time.Now(),
		Plan:      plan,
		Result:    result,
		Annotations: map[string]string{
			"duration": result.Duration.String(),
			"status":   t.determineMigrationStatus(result),
		},
	}

	return t.storage.SaveMigrationHistory(context.Background(), entry)
}

// determineMigrationStatus determines the overall migration status
func (t *SchemaEvolutionTracker) determineMigrationStatus(result *MigrationResult) string {
	if result.FailedCount == 0 {
		return string(MigrationStatusCompleted)
	} else if result.SuccessfulCount == 0 {
		return string(MigrationStatusFailed)
	}
	return "PartiallyCompleted"
}

// GetMigrationHistory retrieves migration history
func (t *SchemaEvolutionTracker) GetMigrationHistory(limit int) ([]*MigrationHistoryEntry, error) {
	return t.storage.LoadMigrationHistory(context.Background(), limit)
}

// GetSchemaEvolution retrieves schema evolution for a GroupKind
func (t *SchemaEvolutionTracker) GetSchemaEvolution(gk schema.GroupKind) (*SchemaEvolution, error) {
	t.mu.RLock()
	if evolution, exists := t.cache[gk.String()]; exists {
		t.mu.RUnlock()
		return evolution, nil
	}
	t.mu.RUnlock()

	// Load from storage
	evolution, err := t.storage.LoadEvolution(context.Background(), gk)
	if err != nil {
		return nil, err
	}

	// Cache it
	t.mu.Lock()
	t.cache[gk.String()] = evolution
	t.mu.Unlock()

	return evolution, nil
}

// GetDeprecatedFields returns all deprecated fields for a GroupKind
func (t *SchemaEvolutionTracker) GetDeprecatedFields(gk schema.GroupKind) ([]DeprecatedPath, error) {
	evolution, err := t.GetSchemaEvolution(gk)
	if err != nil {
		return nil, err
	}
	return evolution.DeprecatedPaths, nil
}

// GenerateMigrationReport generates a comprehensive migration report
func (t *SchemaEvolutionTracker) GenerateMigrationReport(gk schema.GroupKind, fromVersion, toVersion string) (*MigrationReport, error) {
	evolution, err := t.GetSchemaEvolution(gk)
	if err != nil {
		return nil, err
	}

	report := &MigrationReport{
		GroupKind:   gk,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Timestamp:   time.Now(),
	}

	// Analyze field changes
	report.FieldChanges = t.analyzeFieldChanges(evolution, fromVersion, toVersion)
	
	// Identify deprecated fields
	report.DeprecatedFields = t.identifyDeprecatedFields(evolution, fromVersion, toVersion)
	
	// Add migration recommendations
	report.Recommendations = t.generateRecommendations(report.FieldChanges, report.DeprecatedFields)

	return report, nil
}

// analyzeFieldChanges analyzes changes between versions
func (t *SchemaEvolutionTracker) analyzeFieldChanges(evolution *SchemaEvolution, fromVersion, toVersion string) []FieldChange {
	changes := []FieldChange{}

	var fromFields, toFields map[string]FieldInfo
	for _, v := range evolution.Versions {
		if v.Version == fromVersion {
			fromFields = v.Fields
		}
		if v.Version == toVersion {
			toFields = v.Fields
		}
	}

	// Find added fields
	for path, field := range toFields {
		if _, exists := fromFields[path]; !exists {
			changes = append(changes, FieldChange{
				Type:        FieldChangeTypeAdded,
				Path:        path,
				FieldType:   field.Type,
				Description: field.Description,
			})
		}
	}

	// Find removed fields
	for path, field := range fromFields {
		if _, exists := toFields[path]; !exists {
			changes = append(changes, FieldChange{
				Type:        FieldChangeTypeRemoved,
				Path:        path,
				FieldType:   field.Type,
				Description: field.Description,
			})
		}
	}

	return changes
}

// identifyDeprecatedFields identifies deprecated fields between versions
func (t *SchemaEvolutionTracker) identifyDeprecatedFields(evolution *SchemaEvolution, fromVersion, toVersion string) []DeprecatedPath {
	deprecated := []DeprecatedPath{}
	
	for _, path := range evolution.DeprecatedPaths {
		// Check if the deprecation is relevant for this version transition
		deprecated = append(deprecated, path)
	}
	
	return deprecated
}

// generateRecommendations generates migration recommendations
func (t *SchemaEvolutionTracker) generateRecommendations(changes []FieldChange, deprecated []DeprecatedPath) []string {
	recommendations := []string{}

	if len(changes) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Review %d field changes before migration", len(changes)))
	}

	if len(deprecated) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Update %d deprecated field references", len(deprecated)))
	}

	recommendations = append(recommendations, "Create backup before migration")
	recommendations = append(recommendations, "Test migration in non-production environment first")

	return recommendations
}

// FieldChange represents a change in a field
type FieldChange struct {
	Type        FieldChangeType
	Path        string
	FieldType   string
	Description string
	MigratedTo  string
}

// FieldChangeType represents the type of field change
type FieldChangeType string

const (
	FieldChangeTypeAdded      FieldChangeType = "Added"
	FieldChangeTypeRemoved    FieldChangeType = "Removed"
	FieldChangeTypeModified   FieldChangeType = "Modified"
	FieldChangeTypeDeprecated FieldChangeType = "Deprecated"
)

// MigrationReport represents a comprehensive migration report
type MigrationReport struct {
	GroupKind        schema.GroupKind
	FromVersion      string
	ToVersion        string
	Timestamp        time.Time
	FieldChanges     []FieldChange
	DeprecatedFields []DeprecatedPath
	Recommendations  []string
}

// InMemorySchemaStorage provides in-memory storage for schema evolution
type InMemorySchemaStorage struct {
	mu         sync.RWMutex
	evolutions map[string]*SchemaEvolution
	history    []*MigrationHistoryEntry
}

// NewInMemorySchemaStorage creates a new in-memory storage
func NewInMemorySchemaStorage() *InMemorySchemaStorage {
	return &InMemorySchemaStorage{
		evolutions: make(map[string]*SchemaEvolution),
		history:    make([]*MigrationHistoryEntry, 0),
	}
}

// SaveEvolution saves schema evolution
func (s *InMemorySchemaStorage) SaveEvolution(ctx context.Context, evolution *SchemaEvolution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := evolution.GroupKind.String()
	s.evolutions[key] = evolution
	return nil
}

// LoadEvolution loads schema evolution
func (s *InMemorySchemaStorage) LoadEvolution(ctx context.Context, gk schema.GroupKind) (*SchemaEvolution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	key := gk.String()
	evolution, exists := s.evolutions[key]
	if !exists {
		return nil, fmt.Errorf("schema evolution not found for %s", key)
	}
	return evolution, nil
}

// SaveMigrationHistory saves migration history
func (s *InMemorySchemaStorage) SaveMigrationHistory(ctx context.Context, entry *MigrationHistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.history = append(s.history, entry)
	return nil
}

// LoadMigrationHistory loads migration history
func (s *InMemorySchemaStorage) LoadMigrationHistory(ctx context.Context, limit int) ([]*MigrationHistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}
	
	// Return most recent entries
	start := len(s.history) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]*MigrationHistoryEntry, limit)
	copy(result, s.history[start:])
	return result, nil
}

// generateMigrationID generates a unique migration ID
func generateMigrationID() string {
	return fmt.Sprintf("migration-%d", time.Now().UnixNano())
}
