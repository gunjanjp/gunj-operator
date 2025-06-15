/*
Copyright 2025 The Gunj Operator Authors.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MigrationUtils provides utility functions for migration operations
type MigrationUtils struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

// NewMigrationUtils creates a new migration utilities instance
func NewMigrationUtils(client client.Client, scheme *runtime.Scheme, log logr.Logger) *MigrationUtils {
	return &MigrationUtils{
		client: client,
		scheme: scheme,
		log:    log.WithName("migration-utils"),
	}
}

// CompareVersions compares two API versions to determine migration direction
func (u *MigrationUtils) CompareVersions(v1, v2 string) int {
	// Simple version comparison logic
	// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
	
	// Extract version numbers
	v1Parts := extractVersionParts(v1)
	v2Parts := extractVersionParts(v2)
	
	// Compare major versions
	if v1Parts.major != v2Parts.major {
		if v1Parts.major < v2Parts.major {
			return -1
		}
		return 1
	}
	
	// Compare stability (alpha < beta < stable)
	if v1Parts.stability != v2Parts.stability {
		return compareStability(v1Parts.stability, v2Parts.stability)
	}
	
	// Compare minor versions
	if v1Parts.minor < v2Parts.minor {
		return -1
	} else if v1Parts.minor > v2Parts.minor {
		return 1
	}
	
	return 0
}

// versionParts represents parsed version components
type versionParts struct {
	major     int
	minor     int
	stability string // alpha, beta, or stable
}

// extractVersionParts extracts version components
func extractVersionParts(version string) versionParts {
	parts := versionParts{
		major:     1,
		stability: "stable",
	}
	
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	
	// Check for alpha/beta
	if strings.Contains(version, "alpha") {
		parts.stability = "alpha"
		idx := strings.Index(version, "alpha")
		version = version[:idx]
		if len(version) > idx+5 {
			fmt.Sscanf(version[idx+5:], "%d", &parts.minor)
		}
	} else if strings.Contains(version, "beta") {
		parts.stability = "beta"
		idx := strings.Index(version, "beta")
		version = version[:idx]
		if len(version) > idx+4 {
			fmt.Sscanf(version[idx+4:], "%d", &parts.minor)
		}
	}
	
	// Extract major version
	fmt.Sscanf(version, "%d", &parts.major)
	
	return parts
}

// compareStability compares version stability levels
func compareStability(s1, s2 string) int {
	order := map[string]int{
		"alpha":  1,
		"beta":   2,
		"stable": 3,
	}
	
	v1 := order[s1]
	v2 := order[s2]
	
	if v1 < v2 {
		return -1
	} else if v1 > v2 {
		return 1
	}
	return 0
}

// FindMigrationPath finds the migration path between versions
func (u *MigrationUtils) FindMigrationPath(gk schema.GroupKind, fromVersion, toVersion string) ([]string, error) {
	// For now, return direct path
	// In a real implementation, this would find intermediate versions if needed
	return []string{fromVersion, toVersion}, nil
}

// DiffResources compares two resources and returns differences
func (u *MigrationUtils) DiffResources(old, new *unstructured.Unstructured) (*ResourceDiff, error) {
	diff := &ResourceDiff{
		Resource: types.NamespacedName{
			Namespace: old.GetNamespace(),
			Name:      old.GetName(),
		},
		OldGVK: old.GroupVersionKind(),
		NewGVK: new.GroupVersionKind(),
		Fields: []FieldDiff{},
	}
	
	// Compare specs
	oldSpec, _, _ := unstructured.NestedMap(old.Object, "spec")
	newSpec, _, _ := unstructured.NestedMap(new.Object, "spec")
	
	u.diffMaps("spec", oldSpec, newSpec, &diff.Fields)
	
	return diff, nil
}

// diffMaps recursively compares two maps
func (u *MigrationUtils) diffMaps(path string, old, new map[string]interface{}, diffs *[]FieldDiff) {
	// Check for removed fields
	for key, oldValue := range old {
		fieldPath := fmt.Sprintf("%s.%s", path, key)
		newValue, exists := new[key]
		
		if !exists {
			*diffs = append(*diffs, FieldDiff{
				Path:     fieldPath,
				Type:     DiffTypeRemoved,
				OldValue: fmt.Sprintf("%v", oldValue),
			})
			continue
		}
		
		// Compare values
		if !reflect.DeepEqual(oldValue, newValue) {
			// Check if both are maps and recurse
			if oldMap, ok := oldValue.(map[string]interface{}); ok {
				if newMap, ok := newValue.(map[string]interface{}); ok {
					u.diffMaps(fieldPath, oldMap, newMap, diffs)
					continue
				}
			}
			
			*diffs = append(*diffs, FieldDiff{
				Path:     fieldPath,
				Type:     DiffTypeModified,
				OldValue: fmt.Sprintf("%v", oldValue),
				NewValue: fmt.Sprintf("%v", newValue),
			})
		}
	}
	
	// Check for added fields
	for key, newValue := range new {
		if _, exists := old[key]; !exists {
			fieldPath := fmt.Sprintf("%s.%s", path, key)
			*diffs = append(*diffs, FieldDiff{
				Path:     fieldPath,
				Type:     DiffTypeAdded,
				NewValue: fmt.Sprintf("%v", newValue),
			})
		}
	}
}

// ResourceDiff represents differences between resources
type ResourceDiff struct {
	Resource types.NamespacedName
	OldGVK   schema.GroupVersionKind
	NewGVK   schema.GroupVersionKind
	Fields   []FieldDiff
}

// FieldDiff represents a field difference
type FieldDiff struct {
	Path     string
	Type     DiffType
	OldValue string
	NewValue string
}

// DiffType represents the type of difference
type DiffType string

const (
	DiffTypeAdded    DiffType = "Added"
	DiffTypeRemoved  DiffType = "Removed"
	DiffTypeModified DiffType = "Modified"
)

// BackupResource creates a backup of a resource
func (u *MigrationUtils) BackupResource(ctx context.Context, resource *unstructured.Unstructured) (*ResourceBackup, error) {
	backup := &ResourceBackup{
		Resource:  resource.DeepCopy(),
		Timestamp: metav1.Now(),
		Annotations: map[string]string{
			"backup.migration.observability.io/original-version": resource.GetAPIVersion(),
			"backup.migration.observability.io/original-uid":     string(resource.GetUID()),
		},
	}
	
	// Store backup as a ConfigMap
	backupCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("backup-%s-%s", resource.GetName(), time.Now().Format("20060102-150405")),
			Namespace: resource.GetNamespace(),
			Labels: map[string]string{
				"migration.observability.io/backup": "true",
				"migration.observability.io/resource-name": resource.GetName(),
				"migration.observability.io/resource-kind": resource.GetKind(),
			},
		},
		Data: map[string]string{
			"resource.yaml": u.serializeResource(resource),
			"metadata.json": u.serializeBackupMetadata(backup),
		},
	}
	
	if err := u.client.Create(ctx, backupCM); err != nil {
		return nil, fmt.Errorf("creating backup: %w", err)
	}
	
	return backup, nil
}

// RestoreResource restores a resource from backup
func (u *MigrationUtils) RestoreResource(ctx context.Context, backup *ResourceBackup) error {
	resource := backup.Resource
	
	// Update resource version for update operation
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(resource.GroupVersionKind())
	
	key := types.NamespacedName{
		Namespace: resource.GetNamespace(),
		Name:      resource.GetName(),
	}
	
	if err := u.client.Get(ctx, key, current); err != nil {
		if errors.IsNotFound(err) {
			// Resource doesn't exist, create it
			return u.client.Create(ctx, resource)
		}
		return fmt.Errorf("getting current resource: %w", err)
	}
	
	// Preserve resource version
	resource.SetResourceVersion(current.GetResourceVersion())
	
	// Update the resource
	return u.client.Update(ctx, resource)
}

// ResourceBackup represents a resource backup
type ResourceBackup struct {
	Resource    *unstructured.Unstructured
	Timestamp   metav1.Time
	Annotations map[string]string
}

// serializeResource serializes a resource to YAML
func (u *MigrationUtils) serializeResource(resource *unstructured.Unstructured) string {
	data, err := yaml.Marshal(resource)
	if err != nil {
		u.log.Error(err, "Failed to serialize resource")
		return ""
	}
	return string(data)
}

// serializeBackupMetadata serializes backup metadata
func (u *MigrationUtils) serializeBackupMetadata(backup *ResourceBackup) string {
	metadata := map[string]interface{}{
		"timestamp":   backup.Timestamp,
		"annotations": backup.Annotations,
	}
	
	data, err := json.Marshal(metadata)
	if err != nil {
		u.log.Error(err, "Failed to serialize backup metadata")
		return "{}"
	}
	return string(data)
}

// AnalyzeMigrationRisk analyzes the risk level of a migration
func (u *MigrationUtils) AnalyzeMigrationRisk(plan *MigrationPlan) (*RiskAnalysis, error) {
	analysis := &RiskAnalysis{
		OverallRisk: RiskLevelLow,
		Factors:     []RiskFactor{},
	}
	
	// Factor 1: Number of resources
	if len(plan.Resources) > 100 {
		analysis.Factors = append(analysis.Factors, RiskFactor{
			Type:        "resource_count",
			Description: fmt.Sprintf("Large number of resources: %d", len(plan.Resources)),
			Risk:        RiskLevelMedium,
			Mitigation:  "Consider migrating in smaller batches",
		})
	}
	
	// Factor 2: Version gap
	versionGap := u.calculateVersionGap(plan.SourceGVK.Version, plan.TargetGVK.Version)
	if versionGap > 1 {
		analysis.Factors = append(analysis.Factors, RiskFactor{
			Type:        "version_gap",
			Description: fmt.Sprintf("Multiple version gap: %d versions", versionGap),
			Risk:        RiskLevelHigh,
			Mitigation:  "Consider intermediate version migrations",
		})
	}
	
	// Factor 3: Production resources
	for _, resource := range plan.Resources {
		if resource.Namespace == "production" || resource.Namespace == "prod" {
			analysis.Factors = append(analysis.Factors, RiskFactor{
				Type:        "production_resources",
				Description: "Migration includes production resources",
				Risk:        RiskLevelHigh,
				Mitigation:  "Ensure proper backup and rollback procedures",
			})
			break
		}
	}
	
	// Calculate overall risk
	analysis.OverallRisk = u.calculateOverallRisk(analysis.Factors)
	
	return analysis, nil
}

// calculateVersionGap calculates the gap between versions
func (u *MigrationUtils) calculateVersionGap(v1, v2 string) int {
	v1Parts := extractVersionParts(v1)
	v2Parts := extractVersionParts(v2)
	
	gap := 0
	
	// Major version difference
	if v1Parts.major != v2Parts.major {
		gap += abs(v1Parts.major - v2Parts.major)
	}
	
	// Stability difference
	if v1Parts.stability != v2Parts.stability {
		stab1 := compareStability(v1Parts.stability, "stable")
		stab2 := compareStability(v2Parts.stability, "stable")
		gap += abs(stab1 - stab2)
	}
	
	return gap
}

// calculateOverallRisk calculates overall risk level
func (u *MigrationUtils) calculateOverallRisk(factors []RiskFactor) RiskLevel {
	if len(factors) == 0 {
		return RiskLevelLow
	}
	
	highCount := 0
	mediumCount := 0
	
	for _, factor := range factors {
		switch factor.Risk {
		case RiskLevelHigh:
			highCount++
		case RiskLevelMedium:
			mediumCount++
		}
	}
	
	if highCount > 0 {
		return RiskLevelHigh
	}
	if mediumCount > 1 {
		return RiskLevelMedium
	}
	
	return RiskLevelLow
}

// RiskAnalysis represents migration risk analysis
type RiskAnalysis struct {
	OverallRisk RiskLevel
	Factors     []RiskFactor
}

// RiskFactor represents a risk factor
type RiskFactor struct {
	Type        string
	Description string
	Risk        RiskLevel
	Mitigation  string
}

// RiskLevel represents the risk level
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "Low"
	RiskLevelMedium RiskLevel = "Medium"
	RiskLevelHigh   RiskLevel = "High"
)

// abs returns absolute value
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// GenerateMigrationScript generates a migration script
func (u *MigrationUtils) GenerateMigrationScript(plan *MigrationPlan) (string, error) {
	var script strings.Builder
	
	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Auto-generated migration script\n")
	script.WriteString(fmt.Sprintf("# Generated: %s\n", time.Now().Format(time.RFC3339)))
	script.WriteString(fmt.Sprintf("# Source: %s\n", plan.SourceGVK))
	script.WriteString(fmt.Sprintf("# Target: %s\n\n", plan.TargetGVK))
	
	script.WriteString("set -euo pipefail\n\n")
	
	// Add functions
	script.WriteString("# Helper functions\n")
	script.WriteString("log() { echo \"[$(date +'%Y-%m-%d %H:%M:%S')] $*\"; }\n")
	script.WriteString("error() { log \"ERROR: $*\" >&2; exit 1; }\n\n")
	
	// Add migration commands
	script.WriteString("# Migration commands\n")
	script.WriteString("log \"Starting migration...\"\n\n")
	
	if plan.DryRun {
		script.WriteString("# DRY RUN MODE\n")
		script.WriteString("DRY_RUN='--dry-run=client'\n\n")
	}
	
	// Generate kubectl commands for each resource
	for i, resource := range plan.Resources {
		script.WriteString(fmt.Sprintf("# Resource %d/%d: %s/%s\n", 
			i+1, len(plan.Resources), resource.Namespace, resource.Name))
		script.WriteString(fmt.Sprintf("log \"Migrating %s/%s...\"\n", 
			resource.Namespace, resource.Name))
		
		// Get resource
		script.WriteString(fmt.Sprintf("kubectl get %s %s -n %s -o yaml > /tmp/resource-%d.yaml\n",
			plan.SourceGVK.Kind, resource.Name, resource.Namespace, i))
		
		// Convert (simplified - would need actual conversion logic)
		script.WriteString(fmt.Sprintf("# Convert from %s to %s\n", 
			plan.SourceGVK.Version, plan.TargetGVK.Version))
		script.WriteString(fmt.Sprintf("sed -i 's/apiVersion: %s\\/%s/apiVersion: %s\\/%s/' /tmp/resource-%d.yaml\n",
			plan.SourceGVK.Group, plan.SourceGVK.Version,
			plan.TargetGVK.Group, plan.TargetGVK.Version, i))
		
		// Apply
		script.WriteString(fmt.Sprintf("kubectl apply -f /tmp/resource-%d.yaml ${DRY_RUN:-}\n\n", i))
	}
	
	script.WriteString("log \"Migration completed successfully!\"\n")
	
	return script.String(), nil
}

// Additional utility functions for common types
var (
	corev1 = schema.GroupVersion{Group: "", Version: "v1"}
	appsv1 = schema.GroupVersion{Group: "apps", Version: "v1"}
	metav1 = schema.GroupVersion{Group: "meta", Version: "v1"}
	errors = apierrors{}
)

// Mock error types for compilation
type apierrors struct{}

func (apierrors) IsNotFound(err error) bool { return false }
