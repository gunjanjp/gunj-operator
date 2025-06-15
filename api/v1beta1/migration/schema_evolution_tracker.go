/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

// SchemaEvolutionTracker tracks schema evolution and migration patterns
type SchemaEvolutionTracker struct {
	logger logr.Logger
	
	mu       sync.RWMutex
	versions map[string]*VersionInfo
	migrations map[string]*MigrationPath
	analytics *MigrationAnalytics
}

// VersionInfo contains information about a schema version
type VersionInfo struct {
	Version        string
	ReleaseDate    time.Time
	DeprecatedDate *time.Time
	RemovalDate    *time.Time
	Features       []string
	Breaking       []string
	Deprecated     []string
	FieldChanges   map[string]FieldChange
}

// FieldChange represents a change to a field
type FieldChange struct {
	Type        ChangeType
	OldPath     string
	NewPath     string
	OldType     string
	NewType     string
	Description string
	Migration   string // Migration function name
}

// ChangeType represents the type of field change
type ChangeType string

const (
	ChangeTypeAdded       ChangeType = "added"
	ChangeTypeRemoved     ChangeType = "removed"
	ChangeTypeModified    ChangeType = "modified"
	ChangeTypeMoved       ChangeType = "moved"
	ChangeTypeRenamed     ChangeType = "renamed"
	ChangeTypeDeprecated  ChangeType = "deprecated"
)

// MigrationPath represents a migration path between versions
type MigrationPath struct {
	FromVersion   string
	ToVersion     string
	Direct        bool
	Steps         []string // Intermediate versions for multi-step migration
	Complexity    MigrationComplexity
	DataLossRisk  bool
	RequiresManual bool
	Transformations []TransformationStep
}

// MigrationComplexity represents the complexity of a migration
type MigrationComplexity string

const (
	ComplexitySimple   MigrationComplexity = "simple"
	ComplexityModerate MigrationComplexity = "moderate"
	ComplexityComplex  MigrationComplexity = "complex"
)

// TransformationStep represents a single transformation in a migration
type TransformationStep struct {
	Field        string
	Type         TransformationType
	Function     string
	Description  string
}

// TransformationType represents the type of transformation
type TransformationType string

const (
	TransformationTypeValue    TransformationType = "value"
	TransformationTypeStructure TransformationType = "structure"
	TransformationTypeValidation TransformationType = "validation"
)

// MigrationAnalytics tracks migration patterns and statistics
type MigrationAnalytics struct {
	mu               sync.RWMutex
	TotalMigrations  int64
	SuccessfulMigrations int64
	FailedMigrations int64
	MigrationsByVersion map[string]int64
	AverageDuration  time.Duration
	CommonPaths      map[string]int64
	ErrorPatterns    map[string]int64
}

// NewSchemaEvolutionTracker creates a new schema evolution tracker
func NewSchemaEvolutionTracker(logger logr.Logger) *SchemaEvolutionTracker {
	tracker := &SchemaEvolutionTracker{
		logger:     logger.WithName("schema-evolution"),
		versions:   make(map[string]*VersionInfo),
		migrations: make(map[string]*MigrationPath),
		analytics: &MigrationAnalytics{
			MigrationsByVersion: make(map[string]int64),
			CommonPaths:        make(map[string]int64),
			ErrorPatterns:      make(map[string]int64),
		},
	}
	
	// Initialize known versions
	tracker.initializeVersions()
	
	return tracker
}

// initializeVersions initializes known version information
func (s *SchemaEvolutionTracker) initializeVersions() {
	// v1alpha1 version info
	s.versions["v1alpha1"] = &VersionInfo{
		Version:     "v1alpha1",
		ReleaseDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Features: []string{
			"Basic ObservabilityPlatform CRD",
			"Prometheus integration",
			"Grafana integration",
			"Basic alerting",
		},
		FieldChanges: map[string]FieldChange{},
	}
	
	// v1beta1 version info
	deprecatedDate := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	s.versions["v1beta1"] = &VersionInfo{
		Version:        "v1beta1",
		ReleaseDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		DeprecatedDate: &deprecatedDate,
		Features: []string{
			"Enhanced ObservabilityPlatform CRD",
			"Multi-cluster support",
			"Advanced alerting rules",
			"GitOps integration",
			"Service mesh integration",
			"Cost optimization features",
		},
		Breaking: []string{
			"Changed spec.components structure",
			"Renamed spec.monitoring to spec.observability",
		},
		Deprecated: []string{
			"spec.legacyConfig field",
			"spec.deprecatedOptions field",
		},
		FieldChanges: map[string]FieldChange{
			"spec.monitoring": {
				Type:        ChangeTypeRenamed,
				OldPath:     "spec.monitoring",
				NewPath:     "spec.observability",
				Description: "Renamed for clarity",
				Migration:   "renameMonitoringToObservability",
			},
			"spec.components.prometheus.config": {
				Type:        ChangeTypeModified,
				OldPath:     "spec.components.prometheus.config",
				NewPath:     "spec.components.prometheus.configuration",
				OldType:     "string",
				NewType:     "PrometheusConfiguration",
				Description: "Changed from string to structured config",
				Migration:   "convertPrometheusConfig",
			},
			"spec.multiCluster": {
				Type:        ChangeTypeAdded,
				NewPath:     "spec.multiCluster",
				NewType:     "MultiClusterConfig",
				Description: "Added multi-cluster support",
			},
			"spec.serviceMesh": {
				Type:        ChangeTypeAdded,
				NewPath:     "spec.serviceMesh",
				NewType:     "ServiceMeshSpec",
				Description: "Added service mesh integration",
			},
		},
	}
	
	// Define migration paths
	s.migrations["v1alpha1->v1beta1"] = &MigrationPath{
		FromVersion:  "v1alpha1",
		ToVersion:    "v1beta1",
		Direct:       true,
		Complexity:   ComplexityModerate,
		DataLossRisk: false,
		Transformations: []TransformationStep{
			{
				Field:       "spec.monitoring",
				Type:        TransformationTypeStructure,
				Function:    "renameMonitoringToObservability",
				Description: "Rename monitoring to observability",
			},
			{
				Field:       "spec.components.prometheus.config",
				Type:        TransformationTypeStructure,
				Function:    "convertPrometheusConfig",
				Description: "Convert string config to structured format",
			},
		},
	}
	
	s.migrations["v1beta1->v1alpha1"] = &MigrationPath{
		FromVersion:    "v1beta1",
		ToVersion:      "v1alpha1",
		Direct:         true,
		Complexity:     ComplexityComplex,
		DataLossRisk:   true,
		RequiresManual: true,
		Transformations: []TransformationStep{
			{
				Field:       "spec.observability",
				Type:        TransformationTypeStructure,
				Function:    "renameObservabilityToMonitoring",
				Description: "Rename observability to monitoring",
			},
			{
				Field:       "spec.multiCluster",
				Type:        TransformationTypeStructure,
				Function:    "removeMultiCluster",
				Description: "Remove multi-cluster config (data loss)",
			},
			{
				Field:       "spec.serviceMesh",
				Type:        TransformationTypeStructure,
				Function:    "removeServiceMesh",
				Description: "Remove service mesh config (data loss)",
			},
		},
	}
}

// RecordMigration records a migration event
func (s *SchemaEvolutionTracker) RecordMigration(fromVersion, toVersion string, resource types.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Update analytics
	s.analytics.TotalMigrations++
	s.analytics.MigrationsByVersion[fmt.Sprintf("%s->%s", fromVersion, toVersion)]++
	
	pathKey := fmt.Sprintf("%s->%s", fromVersion, toVersion)
	s.analytics.CommonPaths[pathKey]++
	
	s.logger.Info("Recorded migration",
		"from", fromVersion,
		"to", toVersion,
		"resource", resource)
}

// RecordMigrationSuccess records a successful migration
func (s *SchemaEvolutionTracker) RecordMigrationSuccess(fromVersion, toVersion string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.analytics.SuccessfulMigrations++
	
	// Update average duration
	if s.analytics.AverageDuration == 0 {
		s.analytics.AverageDuration = duration
	} else {
		// Calculate running average
		total := s.analytics.SuccessfulMigrations + s.analytics.FailedMigrations
		s.analytics.AverageDuration = (s.analytics.AverageDuration*time.Duration(total-1) + duration) / time.Duration(total)
	}
}

// RecordMigrationFailure records a failed migration
func (s *SchemaEvolutionTracker) RecordMigrationFailure(fromVersion, toVersion string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.analytics.FailedMigrations++
	
	// Track error patterns
	errorPattern := fmt.Sprintf("%s->%s: %T", fromVersion, toVersion, err)
	s.analytics.ErrorPatterns[errorPattern]++
	
	s.logger.Error(err, "Migration failed",
		"from", fromVersion,
		"to", toVersion)
}

// GetMigrationPath returns the migration path between two versions
func (s *SchemaEvolutionTracker) GetMigrationPath(fromVersion, toVersion string) (*MigrationPath, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	directPath := fmt.Sprintf("%s->%s", fromVersion, toVersion)
	if path, exists := s.migrations[directPath]; exists {
		return path, nil
	}
	
	// TODO: Implement multi-step path finding for indirect migrations
	
	return nil, fmt.Errorf("no migration path found from %s to %s", fromVersion, toVersion)
}

// GetVersionInfo returns information about a specific version
func (s *SchemaEvolutionTracker) GetVersionInfo(version string) (*VersionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	info, exists := s.versions[version]
	if !exists {
		return nil, fmt.Errorf("version not found: %s", version)
	}
	
	return info, nil
}

// GetFieldChanges returns field changes between two versions
func (s *SchemaEvolutionTracker) GetFieldChanges(fromVersion, toVersion string) (map[string]FieldChange, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	toInfo, exists := s.versions[toVersion]
	if !exists {
		return nil, fmt.Errorf("version not found: %s", toVersion)
	}
	
	// Return field changes for the target version
	return toInfo.FieldChanges, nil
}

// GetMigrationAnalytics returns migration analytics
func (s *SchemaEvolutionTracker) GetMigrationAnalytics() MigrationAnalytics {
	s.analytics.mu.RLock()
	defer s.analytics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return MigrationAnalytics{
		TotalMigrations:      s.analytics.TotalMigrations,
		SuccessfulMigrations: s.analytics.SuccessfulMigrations,
		FailedMigrations:     s.analytics.FailedMigrations,
		MigrationsByVersion:  copyMap(s.analytics.MigrationsByVersion),
		AverageDuration:      s.analytics.AverageDuration,
		CommonPaths:          copyMap(s.analytics.CommonPaths),
		ErrorPatterns:        copyMap(s.analytics.ErrorPatterns),
	}
}

// ExportAnalytics exports analytics data as JSON
func (s *SchemaEvolutionTracker) ExportAnalytics() ([]byte, error) {
	analytics := s.GetMigrationAnalytics()
	return json.MarshalIndent(analytics, "", "  ")
}

// IsVersionSupported checks if a version is supported
func (s *SchemaEvolutionTracker) IsVersionSupported(version string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	info, exists := s.versions[version]
	if !exists {
		return false
	}
	
	// Check if version is not yet removed
	return info.RemovalDate == nil || time.Now().Before(*info.RemovalDate)
}

// GetSupportedVersions returns all supported versions
func (s *SchemaEvolutionTracker) GetSupportedVersions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	versions := make([]string, 0, len(s.versions))
	for version := range s.versions {
		if s.IsVersionSupported(version) {
			versions = append(versions, version)
		}
	}
	
	return versions
}

// GetDeprecatedFields returns deprecated fields for a version
func (s *SchemaEvolutionTracker) GetDeprecatedFields(version string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	info, exists := s.versions[version]
	if !exists {
		return nil, fmt.Errorf("version not found: %s", version)
	}
	
	return info.Deprecated, nil
}

// copyMap creates a copy of a map[string]int64
func copyMap(m map[string]int64) map[string]int64 {
	copy := make(map[string]int64, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
