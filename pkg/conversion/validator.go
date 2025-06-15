/*
Copyright 2025 The Gunj Operator Authors.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MigrationValidator validates resources before and after migration
type MigrationValidator struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
	rules  map[string][]ValidationRule
}

// ValidationRule represents a validation rule
type ValidationRule interface {
	Validate(ctx context.Context, obj *unstructured.Unstructured) error
	Name() string
	Description() string
}

// ValidationReport contains the results of validation
type ValidationReport struct {
	Valid       bool
	TotalChecks int
	PassedChecks int
	FailedChecks int
	Errors      []ValidationError
	Warnings    []ValidationWarning
	Summary     map[string]ValidationSummary
}

// ValidationError represents a validation error
type ValidationError struct {
	Resource    types.NamespacedName
	Rule        string
	Message     string
	Severity    ValidationSeverity
	Remediation string
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Resource types.NamespacedName
	Rule     string
	Message  string
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalResources   int
	ValidResources   int
	InvalidResources int
	Errors           int
	Warnings         int
}

// ValidationSeverity represents the severity of a validation error
type ValidationSeverity string

const (
	ValidationSeverityCritical ValidationSeverity = "Critical"
	ValidationSeverityHigh     ValidationSeverity = "High"
	ValidationSeverityMedium   ValidationSeverity = "Medium"
	ValidationSeverityLow      ValidationSeverity = "Low"
)

// NewMigrationValidator creates a new migration validator
func NewMigrationValidator(client client.Client, scheme *runtime.Scheme, log logr.Logger) *MigrationValidator {
	v := &MigrationValidator{
		client: client,
		scheme: scheme,
		log:    log.WithName("migration-validator"),
		rules:  make(map[string][]ValidationRule),
	}

	// Register default validation rules
	v.registerDefaultRules()

	return v
}

// registerDefaultRules registers default validation rules
func (v *MigrationValidator) registerDefaultRules() {
	// Register rules for ObservabilityPlatform
	v.RegisterRule("observability.io/ObservabilityPlatform", 
		&ResourceExistsRule{client: v.client},
		&VersionCompatibilityRule{},
		&ResourceQuotaRule{client: v.client},
		&FieldRequirementRule{},
		&SchemaValidationRule{scheme: v.scheme},
	)
}

// RegisterRule registers validation rules for a GVK
func (v *MigrationValidator) RegisterRule(gvk string, rules ...ValidationRule) {
	v.rules[gvk] = append(v.rules[gvk], rules...)
}

// ValidatePreMigration performs pre-migration validation
func (v *MigrationValidator) ValidatePreMigration(ctx context.Context, plan *MigrationPlan) (ValidationReport, error) {
	v.log.Info("Starting pre-migration validation", "resources", len(plan.Resources))

	report := ValidationReport{
		Valid:   true,
		Summary: make(map[string]ValidationSummary),
	}

	// Validate each resource
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 10) // Limit concurrent validations

	for _, resource := range plan.Resources {
		wg.Add(1)
		sem <- struct{}{}

		go func(res types.NamespacedName) {
			defer wg.Done()
			defer func() { <-sem }()

			err := v.validateResource(ctx, plan.SourceGVK, res, &report, &mu)
			if err != nil {
				v.log.Error(err, "Failed to validate resource", "resource", res)
			}
		}(resource)
	}

	wg.Wait()

	// Update report summary
	report.Valid = report.FailedChecks == 0
	v.log.Info("Pre-migration validation completed", 
		"valid", report.Valid,
		"passed", report.PassedChecks,
		"failed", report.FailedChecks)

	return report, nil
}

// ValidatePostMigration performs post-migration validation
func (v *MigrationValidator) ValidatePostMigration(ctx context.Context, plan *MigrationPlan) (ValidationReport, error) {
	v.log.Info("Starting post-migration validation", "resources", len(plan.Resources))

	report := ValidationReport{
		Valid:   true,
		Summary: make(map[string]ValidationSummary),
	}

	// Additional post-migration checks
	postMigrationRules := []ValidationRule{
		&DataIntegrityRule{client: v.client, scheme: v.scheme},
		&ResourceConsistencyRule{client: v.client},
		&StatusValidationRule{},
	}

	// Validate each resource with post-migration rules
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 10)

	for _, resource := range plan.Resources {
		wg.Add(1)
		sem <- struct{}{}

		go func(res types.NamespacedName) {
			defer wg.Done()
			defer func() { <-sem }()

			// Get the migrated resource
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(plan.TargetGVK)
			if err := v.client.Get(ctx, res, obj); err != nil {
				if !errors.IsNotFound(err) {
					mu.Lock()
					report.Errors = append(report.Errors, ValidationError{
						Resource: res,
						Rule:     "resource-exists",
						Message:  fmt.Sprintf("Failed to get migrated resource: %v", err),
						Severity: ValidationSeverityCritical,
					})
					report.FailedChecks++
					mu.Unlock()
				}
				return
			}

			// Apply post-migration rules
			for _, rule := range postMigrationRules {
				if err := rule.Validate(ctx, obj); err != nil {
					mu.Lock()
					report.Errors = append(report.Errors, ValidationError{
						Resource:    res,
						Rule:        rule.Name(),
						Message:     err.Error(),
						Severity:    ValidationSeverityHigh,
						Remediation: "Review resource configuration and re-run migration if necessary",
					})
					report.FailedChecks++
					mu.Unlock()
				} else {
					mu.Lock()
					report.PassedChecks++
					mu.Unlock()
				}
			}
		}(resource)
	}

	wg.Wait()

	report.Valid = report.FailedChecks == 0
	v.log.Info("Post-migration validation completed",
		"valid", report.Valid,
		"passed", report.PassedChecks,
		"failed", report.FailedChecks)

	return report, nil
}

// validateResource validates a single resource
func (v *MigrationValidator) validateResource(ctx context.Context, gvk schema.GroupVersionKind, resource types.NamespacedName, report *ValidationReport, mu *sync.Mutex) error {
	// Get the resource
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	if err := v.client.Get(ctx, resource, obj); err != nil {
		if errors.IsNotFound(err) {
			mu.Lock()
			report.Warnings = append(report.Warnings, ValidationWarning{
				Resource: resource,
				Rule:     "resource-exists",
				Message:  "Resource not found",
			})
			mu.Unlock()
			return nil
		}
		return fmt.Errorf("getting resource: %w", err)
	}

	// Get rules for this GVK
	gvkString := fmt.Sprintf("%s/%s", gvk.Group, gvk.Kind)
	rules, exists := v.rules[gvkString]
	if !exists {
		// Use generic rules if no specific rules
		rules = []ValidationRule{
			&ResourceExistsRule{client: v.client},
			&SchemaValidationRule{scheme: v.scheme},
		}
	}

	// Apply each rule
	for _, rule := range rules {
		mu.Lock()
		report.TotalChecks++
		mu.Unlock()

		if err := rule.Validate(ctx, obj); err != nil {
			mu.Lock()
			report.Errors = append(report.Errors, ValidationError{
				Resource:    resource,
				Rule:        rule.Name(),
				Message:     err.Error(),
				Severity:    ValidationSeverityMedium,
				Remediation: "Fix the issue before proceeding with migration",
			})
			report.FailedChecks++
			mu.Unlock()
		} else {
			mu.Lock()
			report.PassedChecks++
			mu.Unlock()
		}
	}

	// Update summary
	mu.Lock()
	summary := report.Summary[gvkString]
	summary.TotalResources++
	if len(report.Errors) == 0 {
		summary.ValidResources++
	} else {
		summary.InvalidResources++
	}
	report.Summary[gvkString] = summary
	mu.Unlock()

	return nil
}

// IsValid returns whether the validation report indicates success
func (r ValidationReport) IsValid() bool {
	return r.Valid && len(r.Errors) == 0
}

// Built-in Validation Rules

// ResourceExistsRule validates that the resource exists
type ResourceExistsRule struct {
	client client.Client
}

func (r *ResourceExistsRule) Name() string        { return "resource-exists" }
func (r *ResourceExistsRule) Description() string { return "Validates that the resource exists" }

func (r *ResourceExistsRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	key := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())
	if err := r.client.Get(ctx, key, existing); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("resource does not exist")
		}
		return fmt.Errorf("failed to check resource existence: %w", err)
	}
	
	return nil
}

// VersionCompatibilityRule validates version compatibility
type VersionCompatibilityRule struct{}

func (r *VersionCompatibilityRule) Name() string        { return "version-compatibility" }
func (r *VersionCompatibilityRule) Description() string { return "Validates API version compatibility" }

func (r *VersionCompatibilityRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	// Check if the resource has any deprecated annotations
	annotations := obj.GetAnnotations()
	if deprecated, exists := annotations["deprecated.kubernetes.io/api-version"]; exists {
		return fmt.Errorf("resource uses deprecated API version: %s", deprecated)
	}
	
	return nil
}

// ResourceQuotaRule validates resource quota compliance
type ResourceQuotaRule struct {
	client client.Client
}

func (r *ResourceQuotaRule) Name() string        { return "resource-quota" }
func (r *ResourceQuotaRule) Description() string { return "Validates resource quota compliance" }

func (r *ResourceQuotaRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	// For deployment-like resources, check resource requests/limits
	if obj.GetKind() == "ObservabilityPlatform" {
		spec, found, err := unstructured.NestedMap(obj.Object, "spec")
		if err != nil || !found {
			return fmt.Errorf("spec not found")
		}
		
		// Check if resources are specified
		components, found, err := unstructured.NestedMap(spec, "components")
		if err != nil || !found {
			return nil // No components specified
		}
		
		// Validate each component has resource specifications
		for component, config := range components {
			if configMap, ok := config.(map[string]interface{}); ok {
				if _, hasResources := configMap["resources"]; !hasResources {
					return fmt.Errorf("component %s missing resource specifications", component)
				}
			}
		}
	}
	
	return nil
}

// FieldRequirementRule validates required fields
type FieldRequirementRule struct{}

func (r *FieldRequirementRule) Name() string        { return "field-requirements" }
func (r *FieldRequirementRule) Description() string { return "Validates required fields are present" }

func (r *FieldRequirementRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	// Define required fields based on kind
	requiredFields := map[string][]string{
		"ObservabilityPlatform": {"spec.components"},
	}
	
	fields, exists := requiredFields[obj.GetKind()]
	if !exists {
		return nil // No required fields defined
	}
	
	// Check each required field
	for _, field := range fields {
		_, found, err := unstructured.NestedFieldNoCopy(obj.Object, field)
		if err != nil || !found {
			return fmt.Errorf("required field %s not found", field)
		}
	}
	
	return nil
}

// SchemaValidationRule validates against the OpenAPI schema
type SchemaValidationRule struct {
	scheme *runtime.Scheme
}

func (r *SchemaValidationRule) Name() string        { return "schema-validation" }
func (r *SchemaValidationRule) Description() string { return "Validates resource against schema" }

func (r *SchemaValidationRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	// Convert to typed object for schema validation
	typed, err := r.scheme.New(obj.GroupVersionKind())
	if err != nil {
		return fmt.Errorf("unknown resource type: %w", err)
	}
	
	// Convert unstructured to typed
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, typed); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	return nil
}

// DataIntegrityRule validates data integrity after migration
type DataIntegrityRule struct {
	client client.Client
	scheme *runtime.Scheme
}

func (r *DataIntegrityRule) Name() string        { return "data-integrity" }
func (r *DataIntegrityRule) Description() string { return "Validates data integrity after migration" }

func (r *DataIntegrityRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	// Check if all expected fields are present and valid
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("spec field missing after migration")
	}
	
	// Validate spec is not empty
	if len(spec) == 0 {
		return fmt.Errorf("spec is empty after migration")
	}
	
	// Check status exists (should be preserved during migration)
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status == nil {
		return fmt.Errorf("status was not preserved during migration")
	}
	
	return nil
}

// ResourceConsistencyRule validates resource consistency
type ResourceConsistencyRule struct {
	client client.Client
}

func (r *ResourceConsistencyRule) Name() string        { return "resource-consistency" }
func (r *ResourceConsistencyRule) Description() string { return "Validates resource consistency" }

func (r *ResourceConsistencyRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	// Check metadata consistency
	if obj.GetUID() == "" {
		return fmt.Errorf("UID is missing")
	}
	
	if obj.GetResourceVersion() == "" {
		return fmt.Errorf("ResourceVersion is missing")
	}
	
	// Check labels and annotations are preserved
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	
	// Verify migration label was added
	if _, exists := labels["migration.observability.io/migrated"]; !exists {
		return fmt.Errorf("migration label not added")
	}
	
	return nil
}

// StatusValidationRule validates resource status
type StatusValidationRule struct{}

func (r *StatusValidationRule) Name() string        { return "status-validation" }
func (r *StatusValidationRule) Description() string { return "Validates resource status" }

func (r *StatusValidationRule) Validate(ctx context.Context, obj *unstructured.Unstructured) error {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	
	if !found || status == nil {
		// Status might be optional for some resources
		return nil
	}
	
	// Check for common status fields
	if phase, found, _ := unstructured.NestedString(status, "phase"); found {
		if phase == "" {
			return fmt.Errorf("status phase is empty")
		}
	}
	
	return nil
}
