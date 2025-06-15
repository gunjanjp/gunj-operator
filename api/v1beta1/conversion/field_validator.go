/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// FieldValidator provides comprehensive field mapping validation during conversions
type FieldValidator struct {
	logger logr.Logger
	// Track field usage for deprecation warnings
	fieldUsage map[string]int
	// Validation rules for different field types
	validators map[string]FieldValidationFunc
	// Compatibility matrix for version conversions
	compatibilityMatrix *CompatibilityMatrix
}

// FieldValidationFunc defines a validation function for a specific field
type FieldValidationFunc func(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList

// ValidationResult contains the result of field validation
type ValidationResult struct {
	Errors      field.ErrorList
	Warnings    []string
	Suggestions []string
	Metrics     ValidationMetrics
}

// ValidationMetrics tracks conversion metrics
type ValidationMetrics struct {
	FieldsValidated   int
	FieldsWithErrors  int
	FieldsWithWarnings int
	DeprecatedFields  int
	DataLossFields    int
	EnhancedFields    int
}

// NewFieldValidator creates a new field validator instance
func NewFieldValidator(logger logr.Logger) *FieldValidator {
	fv := &FieldValidator{
		logger:              logger,
		fieldUsage:          make(map[string]int),
		validators:          make(map[string]FieldValidationFunc),
		compatibilityMatrix: NewCompatibilityMatrix(),
	}
	
	// Register default validators
	fv.registerDefaultValidators()
	
	return fv
}

// registerDefaultValidators sets up the default field validators
func (fv *FieldValidator) registerDefaultValidators() {
	// Resource validation
	fv.validators["resources"] = fv.validateResources
	
	// Version validation
	fv.validators["version"] = fv.validateVersion
	
	// Storage validation
	fv.validators["storage"] = fv.validateStorage
	
	// Replica validation
	fv.validators["replicas"] = fv.validateReplicas
	
	// Label validation
	fv.validators["labels"] = fv.validateLabels
	
	// Selector validation
	fv.validators["selector"] = fv.validateSelector
}

// ValidateConversion performs comprehensive validation during conversion
func (fv *FieldValidator) ValidateConversion(srcVersion, dstVersion string, src, dst interface{}) *ValidationResult {
	result := &ValidationResult{
		Errors:   field.ErrorList{},
		Warnings: []string{},
		Metrics:  ValidationMetrics{},
	}
	
	// Get field mappings for the conversion
	mappings := fv.compatibilityMatrix.GetFieldMappings(srcVersion, dstVersion)
	
	// Validate each field mapping
	for _, mapping := range mappings {
		fv.validateFieldMapping(mapping, src, dst, result)
	}
	
	// Check for data loss
	fv.checkDataLoss(srcVersion, dstVersion, src, dst, result)
	
	// Generate suggestions
	fv.generateSuggestions(srcVersion, dstVersion, result)
	
	return result
}

// validateFieldMapping validates a single field mapping
func (fv *FieldValidator) validateFieldMapping(mapping FieldMapping, src, dst interface{}, result *ValidationResult) {
	result.Metrics.FieldsValidated++
	
	// Get field values using reflection
	srcValue := fv.getFieldValue(src, mapping.SourcePath)
	dstValue := fv.getFieldValue(dst, mapping.DestPath)
	
	// Check if field is deprecated
	if mapping.Deprecated {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"Field %s is deprecated and will be removed in %s. %s",
			mapping.SourcePath, mapping.DeprecationVersion, mapping.MigrationGuide))
		result.Metrics.DeprecatedFields++
		fv.fieldUsage[mapping.SourcePath]++
	}
	
	// Apply field-specific validation
	if validator, ok := fv.validators[mapping.ValidationType]; ok {
		fieldPath := field.NewPath(strings.Split(mapping.SourcePath, ".")...)
		errors := validator(fieldPath, srcValue, dstValue)
		result.Errors = append(result.Errors, errors...)
		if len(errors) > 0 {
			result.Metrics.FieldsWithErrors++
		}
	}
	
	// Check for semantic changes
	if mapping.SemanticChange != nil {
		warning := mapping.SemanticChange(srcValue, dstValue)
		if warning != "" {
			result.Warnings = append(result.Warnings, warning)
			result.Metrics.FieldsWithWarnings++
		}
	}
	
	// Track enhancements
	if mapping.Enhanced {
		result.Metrics.EnhancedFields++
		fv.logger.V(2).Info("Field enhanced during conversion",
			"field", mapping.SourcePath,
			"enhancement", mapping.EnhancementDescription)
	}
}

// validateResources validates resource requirements conversion
func (fv *FieldValidator) validateResources(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList {
	var errors field.ErrorList
	
	// Handle v1alpha1 to v1beta1 resource conversion
	if oldRes, ok := oldValue.(v1alpha1.ResourceRequirements); ok {
		if newRes, ok := newValue.(v1beta1.ResourceRequirements); ok {
			// Validate CPU
			if oldRes.Requests.CPU != "" {
				if _, err := resource.ParseQuantity(oldRes.Requests.CPU); err != nil {
					errors = append(errors, field.Invalid(fieldPath.Child("requests", "cpu"), 
						oldRes.Requests.CPU, "invalid CPU quantity"))
				}
			}
			
			// Validate Memory
			if oldRes.Requests.Memory != "" {
				if _, err := resource.ParseQuantity(oldRes.Requests.Memory); err != nil {
					errors = append(errors, field.Invalid(fieldPath.Child("requests", "memory"), 
						oldRes.Requests.Memory, "invalid memory quantity"))
				}
			}
			
			// Check for resource scaling
			if oldRes.Limits.CPU != "" && newRes.Limits.CPU != "" {
				oldCPU, _ := resource.ParseQuantity(oldRes.Limits.CPU)
				newCPU, _ := resource.ParseQuantity(newRes.Limits.CPU)
				if oldCPU.Cmp(newCPU) != 0 {
					fv.logger.V(1).Info("CPU limit changed during conversion",
						"old", oldCPU.String(), "new", newCPU.String())
				}
			}
		}
	}
	
	return errors
}

// validateVersion validates component version compatibility
func (fv *FieldValidator) validateVersion(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList {
	var errors field.ErrorList
	
	oldVersion, oldOk := oldValue.(string)
	newVersion, newOk := newValue.(string)
	
	if !oldOk || !newOk {
		return errors
	}
	
	// Check version format
	if !isValidVersion(oldVersion) {
		errors = append(errors, field.Invalid(fieldPath, oldVersion, 
			"version must follow semantic versioning (e.g., v2.48.0)"))
	}
	
	// Check for major version changes
	if oldVersion != "" && newVersion != "" && oldVersion != newVersion {
		oldMajor := extractMajorVersion(oldVersion)
		newMajor := extractMajorVersion(newVersion)
		
		if oldMajor != newMajor {
			errors = append(errors, field.Invalid(fieldPath, newVersion,
				fmt.Sprintf("major version change not allowed during conversion (from %s to %s)", 
					oldVersion, newVersion)))
		}
	}
	
	return errors
}

// validateStorage validates storage configuration conversion
func (fv *FieldValidator) validateStorage(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList {
	var errors field.ErrorList
	
	// Handle string to resource.Quantity conversion
	if oldSize, ok := oldValue.(string); ok {
		if oldSize != "" {
			quantity, err := resource.ParseQuantity(oldSize)
			if err != nil {
				errors = append(errors, field.Invalid(fieldPath.Child("size"), 
					oldSize, "invalid storage size"))
			} else {
				// Check for minimum storage requirements
				minSize := resource.MustParse("1Gi")
				if quantity.Cmp(minSize) < 0 {
					errors = append(errors, field.Invalid(fieldPath.Child("size"),
						oldSize, "storage size must be at least 1Gi"))
				}
			}
		}
	}
	
	return errors
}

// validateReplicas validates replica count changes
func (fv *FieldValidator) validateReplicas(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList {
	var errors field.ErrorList
	
	oldReplicas, oldOk := oldValue.(int32)
	newReplicas, newOk := newValue.(int32)
	
	if !oldOk || !newOk {
		return errors
	}
	
	// Check for valid replica counts
	if oldReplicas < 0 {
		errors = append(errors, field.Invalid(fieldPath, oldReplicas, 
			"replicas cannot be negative"))
	}
	
	// Warn about scaling down in production
	if oldReplicas > newReplicas && oldReplicas > 1 {
		fv.logger.Info("Scaling down replicas during conversion",
			"component", fieldPath.String(),
			"from", oldReplicas,
			"to", newReplicas)
	}
	
	return errors
}

// validateLabels validates label changes
func (fv *FieldValidator) validateLabels(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList {
	var errors field.ErrorList
	
	oldLabels, oldOk := oldValue.(map[string]string)
	newLabels, newOk := newValue.(map[string]string)
	
	if !oldOk || !newOk {
		return errors
	}
	
	// Check for removed labels
	for key := range oldLabels {
		if _, exists := newLabels[key]; !exists {
			errors = append(errors, field.Invalid(fieldPath.Key(key),
				key, "label cannot be removed during conversion"))
		}
	}
	
	// Validate label values
	for key, value := range newLabels {
		if len(key) > 63 || len(value) > 63 {
			errors = append(errors, field.Invalid(fieldPath.Key(key),
				value, "label key and value must be 63 characters or less"))
		}
	}
	
	return errors
}

// validateSelector validates selector changes
func (fv *FieldValidator) validateSelector(fieldPath *field.Path, oldValue, newValue interface{}) field.ErrorList {
	var errors field.ErrorList
	
	// Selectors should not change during conversion
	if !reflect.DeepEqual(oldValue, newValue) {
		errors = append(errors, field.Invalid(fieldPath, newValue,
			"selector cannot be changed during conversion"))
	}
	
	return errors
}

// checkDataLoss checks for potential data loss during conversion
func (fv *FieldValidator) checkDataLoss(srcVersion, dstVersion string, src, dst interface{}, result *ValidationResult) {
	// Get fields that will be lost
	lostFields := fv.compatibilityMatrix.GetLostFields(srcVersion, dstVersion)
	
	for _, fieldPath := range lostFields {
		value := fv.getFieldValue(src, fieldPath)
		if !fv.isZeroValue(value) {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"Field %s will be lost during conversion from %s to %s",
				fieldPath, srcVersion, dstVersion))
			result.Metrics.DataLossFields++
		}
	}
}

// generateSuggestions generates helpful suggestions for the conversion
func (fv *FieldValidator) generateSuggestions(srcVersion, dstVersion string, result *ValidationResult) {
	// Suggest using newer fields
	if srcVersion == "v1alpha1" && dstVersion == "v1beta1" {
		if result.Metrics.DeprecatedFields > 0 {
			result.Suggestions = append(result.Suggestions,
				"Consider updating your configuration to use the newer field names before the deprecated fields are removed.")
		}
		
		if result.Metrics.DataLossFields == 0 {
			result.Suggestions = append(result.Suggestions,
				"This conversion is safe with no data loss. Consider using v1beta1 directly for new resources.")
		}
	}
	
	// Suggest backup before conversion
	if result.Metrics.DataLossFields > 0 {
		result.Suggestions = append(result.Suggestions,
			"Back up your resources before conversion as some fields will be lost.")
	}
}

// getFieldValue retrieves a field value using reflection
func (fv *FieldValidator) getFieldValue(obj interface{}, fieldPath string) interface{} {
	// Simplified field access - in production, use more robust reflection
	// or a library like github.com/tidwall/gjson for JSON path access
	
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	parts := strings.Split(fieldPath, ".")
	for _, part := range parts {
		if v.Kind() != reflect.Struct {
			return nil
		}
		
		f := v.FieldByName(part)
		if !f.IsValid() {
			return nil
		}
		v = f
	}
	
	return v.Interface()
}

// isZeroValue checks if a value is the zero value for its type
func (fv *FieldValidator) isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String() == ""
	case reflect.Int, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Map, reflect.Slice:
		return rv.Len() == 0
	case reflect.Ptr:
		return rv.IsNil()
	default:
		return rv.IsZero()
	}
}

// Helper functions

func isValidVersion(version string) bool {
	// Simple version validation - should use semver library in production
	if version == "" {
		return true
	}
	
	// Must start with 'v' and contain major.minor.patch
	if !strings.HasPrefix(version, "v") {
		return false
	}
	
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	return len(parts) == 3
}

func extractMajorVersion(version string) string {
	if version == "" {
		return ""
	}
	
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	
	return ""
}
