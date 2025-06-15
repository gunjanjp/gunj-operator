// Package deprecation provides a system for tracking and warning about deprecated API fields
// and configurations in the Gunj Operator.
package deprecation

import (
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DeprecationType indicates the type of deprecation
type DeprecationType string

const (
	// FieldDeprecation indicates a deprecated field
	FieldDeprecation DeprecationType = "field"
	// ValueDeprecation indicates a deprecated value for a field
	ValueDeprecation DeprecationType = "value"
	// FeatureDeprecation indicates a deprecated feature
	FeatureDeprecation DeprecationType = "feature"
	// APIVersionDeprecation indicates a deprecated API version
	APIVersionDeprecation DeprecationType = "apiVersion"
)

// DeprecationPolicy defines when a deprecation becomes a removal
type DeprecationPolicy struct {
	// DeprecatedInVersion is the version where the deprecation was introduced
	DeprecatedInVersion string
	// RemovedInVersion is the version where the deprecated item will be removed
	RemovedInVersion string
	// DeprecatedSince is the timestamp when deprecation was introduced
	DeprecatedSince time.Time
	// RemovalDate is the planned removal date (optional)
	RemovalDate *time.Time
}

// DeprecationInfo contains information about a deprecated field or feature
type DeprecationInfo struct {
	// Type indicates what kind of deprecation this is
	Type DeprecationType
	// Path is the field path (e.g., "spec.components.prometheus.version")
	Path string
	// Value is the deprecated value (for ValueDeprecation type)
	Value string
	// Message is the deprecation message shown to users
	Message string
	// MigrationGuide provides guidance on how to migrate
	MigrationGuide string
	// AlternativePath is the new field path to use instead
	AlternativePath string
	// Policy defines the deprecation timeline
	Policy DeprecationPolicy
	// Severity indicates how critical this deprecation is
	Severity DeprecationSeverity
	// AffectedVersions lists the API versions affected by this deprecation
	AffectedVersions []string
}

// DeprecationSeverity indicates how critical a deprecation is
type DeprecationSeverity string

const (
	// SeverityInfo is for informational deprecations
	SeverityInfo DeprecationSeverity = "info"
	// SeverityWarning is for deprecations that should be addressed soon
	SeverityWarning DeprecationSeverity = "warning"
	// SeverityCritical is for deprecations that will be removed soon
	SeverityCritical DeprecationSeverity = "critical"
)

// Registry manages all deprecation information
type Registry struct {
	mu           sync.RWMutex
	deprecations map[string]*DeprecationInfo
	byGVK        map[schema.GroupVersionKind][]*DeprecationInfo
}

var (
	// globalRegistry is the singleton registry instance
	globalRegistry *Registry
	once           sync.Once
)

// GetRegistry returns the global deprecation registry
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			deprecations: make(map[string]*DeprecationInfo),
			byGVK:        make(map[schema.GroupVersionKind][]*DeprecationInfo),
		}
		// Initialize with known deprecations
		globalRegistry.initializeDeprecations()
	})
	return globalRegistry
}

// Register adds a new deprecation to the registry
func (r *Registry) Register(info *DeprecationInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.generateKey(info)
	r.deprecations[key] = info

	// Index by affected versions
	for _, version := range info.AffectedVersions {
		gvk := schema.GroupVersionKind{
			Group:   "observability.io",
			Version: version,
			Kind:    "ObservabilityPlatform",
		}
		r.byGVK[gvk] = append(r.byGVK[gvk], info)
	}
}

// GetDeprecations returns all deprecations for a given GVK
func (r *Registry) GetDeprecations(gvk schema.GroupVersionKind) []*DeprecationInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byGVK[gvk]
}

// GetDeprecationByPath returns deprecation info for a specific field path
func (r *Registry) GetDeprecationByPath(path string, gvk schema.GroupVersionKind) *DeprecationInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deprecations := r.byGVK[gvk]
	for _, dep := range deprecations {
		if dep.Path == path {
			return dep
		}
	}
	return nil
}

// CheckFieldValue checks if a specific field value is deprecated
func (r *Registry) CheckFieldValue(path, value string, gvk schema.GroupVersionKind) *DeprecationInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deprecations := r.byGVK[gvk]
	for _, dep := range deprecations {
		if dep.Type == ValueDeprecation && dep.Path == path && dep.Value == value {
			return dep
		}
	}
	return nil
}

// generateKey creates a unique key for a deprecation
func (r *Registry) generateKey(info *DeprecationInfo) string {
	parts := []string{string(info.Type), info.Path}
	if info.Value != "" {
		parts = append(parts, info.Value)
	}
	return strings.Join(parts, ":")
}

// initializeDeprecations sets up known deprecations
func (r *Registry) initializeDeprecations() {
	// Example deprecations for v1alpha1 -> v1beta1 migration
	
	// Deprecated field: spec.monitoring (replaced by spec.components)
	r.Register(&DeprecationInfo{
		Type:    FieldDeprecation,
		Path:    "spec.monitoring",
		Message: "The 'spec.monitoring' field is deprecated and will be removed in v1",
		MigrationGuide: `Migrate from 'spec.monitoring' to 'spec.components':
  Before:
    spec:
      monitoring:
        prometheus:
          enabled: true
  
  After:
    spec:
      components:
        prometheus:
          enabled: true`,
		AlternativePath: "spec.components",
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v1",
			DeprecatedSince:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		Severity:         SeverityWarning,
		AffectedVersions: []string{"v1alpha1"},
	})

	// Deprecated value: spec.components.prometheus.version < v2.40.0
	r.Register(&DeprecationInfo{
		Type:    ValueDeprecation,
		Path:    "spec.components.prometheus.version",
		Value:   "v2.30.0",
		Message: "Prometheus versions below v2.40.0 are deprecated due to security vulnerabilities",
		MigrationGuide: `Update to a supported Prometheus version:
  Minimum supported version: v2.40.0
  Recommended version: v2.48.0 or later
  
  Update your configuration:
    spec:
      components:
        prometheus:
          version: v2.48.0`,
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v1",
			DeprecatedSince:     time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		Severity:         SeverityCritical,
		AffectedVersions: []string{"v1alpha1", "v1beta1"},
	})

	// Deprecated feature: manual TLS configuration
	r.Register(&DeprecationInfo{
		Type:    FeatureDeprecation,
		Path:    "spec.tls.manual",
		Message: "Manual TLS configuration is deprecated. Use cert-manager integration instead",
		MigrationGuide: `Migrate to cert-manager based TLS:
  1. Install cert-manager in your cluster
  2. Remove spec.tls.manual configuration
  3. Use spec.tls.certManager instead:
  
    spec:
      tls:
        certManager:
          enabled: true
          issuerRef:
            name: letsencrypt-prod
            kind: ClusterIssuer`,
		AlternativePath: "spec.tls.certManager",
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v1",
			DeprecatedSince:     time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		},
		Severity:         SeverityWarning,
		AffectedVersions: []string{"v1alpha1", "v1beta1"},
	})

	// API version deprecation
	r.Register(&DeprecationInfo{
		Type:    APIVersionDeprecation,
		Path:    "apiVersion",
		Value:   "observability.io/v1alpha1",
		Message: "API version v1alpha1 is deprecated and will be removed in v2.0.0",
		MigrationGuide: `Update your resources to use v1beta1:
  1. Change apiVersion from observability.io/v1alpha1 to observability.io/v1beta1
  2. Update any deprecated fields as indicated in other warnings
  3. Test your configurations thoroughly before the v1alpha1 removal`,
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v2.0.0",
			DeprecatedSince:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			RemovalDate:         &[]time.Time{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}[0],
		},
		Severity:         SeverityWarning,
		AffectedVersions: []string{"v1alpha1"},
	})

	// Storage configuration deprecation
	r.Register(&DeprecationInfo{
		Type:    FieldDeprecation,
		Path:    "spec.storage.class",
		Message: "The 'spec.storage.class' field is deprecated. Use 'spec.storage.storageClassName' instead",
		MigrationGuide: `Update your storage configuration:
  Before:
    spec:
      storage:
        class: fast-ssd
  
  After:
    spec:
      storage:
        storageClassName: fast-ssd`,
		AlternativePath: "spec.storage.storageClassName",
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v1",
			DeprecatedSince:     time.Date(2025, 5, 15, 0, 0, 0, 0, time.UTC),
		},
		Severity:         SeverityInfo,
		AffectedVersions: []string{"v1alpha1", "v1beta1"},
	})
}

// FormatWarning formats a deprecation warning for display
func (r *Registry) FormatWarning(info *DeprecationInfo) string {
	var sb strings.Builder
	
	// Header based on severity
	switch info.Severity {
	case SeverityCritical:
		sb.WriteString("⚠️  CRITICAL DEPRECATION WARNING ⚠️\n")
	case SeverityWarning:
		sb.WriteString("⚠️  DEPRECATION WARNING ⚠️\n")
	default:
		sb.WriteString("ℹ️  DEPRECATION NOTICE ℹ️\n")
	}
	
	// Main message
	sb.WriteString(fmt.Sprintf("  %s\n", info.Message))
	
	// Affected field/value
	if info.Type == ValueDeprecation {
		sb.WriteString(fmt.Sprintf("  Field: %s\n  Deprecated Value: %s\n", info.Path, info.Value))
	} else {
		sb.WriteString(fmt.Sprintf("  Field: %s\n", info.Path))
	}
	
	// Timeline information
	sb.WriteString(fmt.Sprintf("  Deprecated Since: %s (version %s)\n", 
		info.Policy.DeprecatedSince.Format("2006-01-02"),
		info.Policy.DeprecatedInVersion))
	sb.WriteString(fmt.Sprintf("  Will be removed in: %s\n", info.Policy.RemovedInVersion))
	
	if info.Policy.RemovalDate != nil {
		sb.WriteString(fmt.Sprintf("  Planned removal date: %s\n", 
			info.Policy.RemovalDate.Format("2006-01-02")))
	}
	
	// Migration guide
	if info.MigrationGuide != "" {
		sb.WriteString("\n  Migration Guide:\n")
		lines := strings.Split(info.MigrationGuide, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}
	
	// Alternative
	if info.AlternativePath != "" {
		sb.WriteString(fmt.Sprintf("\n  Use instead: %s\n", info.AlternativePath))
	}
	
	return sb.String()
}

// GetSeverity calculates the current severity based on timeline
func (info *DeprecationInfo) GetSeverity() DeprecationSeverity {
	now := time.Now()
	
	// If removal date is set and we're within 3 months, it's critical
	if info.Policy.RemovalDate != nil {
		daysUntilRemoval := info.Policy.RemovalDate.Sub(now).Hours() / 24
		if daysUntilRemoval <= 90 {
			return SeverityCritical
		} else if daysUntilRemoval <= 180 {
			return SeverityWarning
		}
	}
	
	// Check how long it's been deprecated
	daysSinceDeprecation := now.Sub(info.Policy.DeprecatedSince).Hours() / 24
	if daysSinceDeprecation > 180 {
		return SeverityWarning
	}
	
	return info.Severity
}
