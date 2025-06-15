package deprecation

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Checker provides functionality to check objects for deprecated fields and values
type Checker struct {
	registry *Registry
}

// NewChecker creates a new deprecation checker
func NewChecker() *Checker {
	return &Checker{
		registry: GetRegistry(),
	}
}

// Warning represents a deprecation warning found during checking
type Warning struct {
	// Field is the path to the deprecated field
	Field string
	// Value is the deprecated value (if applicable)
	Value string
	// Message is the warning message
	Message string
	// Severity indicates the warning severity
	Severity DeprecationSeverity
	// MigrationGuide provides migration instructions
	MigrationGuide string
}

// CheckResult contains all deprecation warnings found
type CheckResult struct {
	// Warnings contains all deprecation warnings found
	Warnings []Warning
	// HasCritical indicates if any critical warnings were found
	HasCritical bool
	// HasWarnings indicates if any warnings were found
	HasWarnings bool
}

// Check examines an object for deprecated fields and values
func (c *Checker) Check(obj *unstructured.Unstructured, gvk schema.GroupVersionKind) (*CheckResult, error) {
	result := &CheckResult{
		Warnings: []Warning{},
	}

	// Get all deprecations for this GVK
	deprecations := c.registry.GetDeprecations(gvk)
	if len(deprecations) == 0 {
		return result, nil
	}

	// Check each deprecation
	for _, dep := range deprecations {
		warning := c.checkDeprecation(obj, dep)
		if warning != nil {
			result.Warnings = append(result.Warnings, *warning)
			result.HasWarnings = true
			if warning.Severity == SeverityCritical {
				result.HasCritical = true
			}
		}
	}

	return result, nil
}

// checkDeprecation checks a single deprecation against an object
func (c *Checker) checkDeprecation(obj *unstructured.Unstructured, dep *DeprecationInfo) *Warning {
	switch dep.Type {
	case FieldDeprecation:
		return c.checkFieldDeprecation(obj, dep)
	case ValueDeprecation:
		return c.checkValueDeprecation(obj, dep)
	case APIVersionDeprecation:
		return c.checkAPIVersionDeprecation(obj, dep)
	case FeatureDeprecation:
		return c.checkFeatureDeprecation(obj, dep)
	default:
		return nil
	}
}

// checkFieldDeprecation checks if a deprecated field exists in the object
func (c *Checker) checkFieldDeprecation(obj *unstructured.Unstructured, dep *DeprecationInfo) *Warning {
	value, found, err := unstructured.NestedFieldNoCopy(obj.Object, strings.Split(dep.Path, ".")...)
	if err != nil || !found {
		return nil
	}

	// Field exists, so it's deprecated
	return &Warning{
		Field:          dep.Path,
		Message:        dep.Message,
		Severity:       dep.GetSeverity(),
		MigrationGuide: dep.MigrationGuide,
	}
}

// checkValueDeprecation checks if a field has a deprecated value
func (c *Checker) checkValueDeprecation(obj *unstructured.Unstructured, dep *DeprecationInfo) *Warning {
	value, found, err := unstructured.NestedFieldNoCopy(obj.Object, strings.Split(dep.Path, ".")...)
	if err != nil || !found {
		return nil
	}

	// Check if the value matches the deprecated value
	valueStr := fmt.Sprintf("%v", value)
	if valueStr == dep.Value {
		return &Warning{
			Field:          dep.Path,
			Value:          dep.Value,
			Message:        dep.Message,
			Severity:       dep.GetSeverity(),
			MigrationGuide: dep.MigrationGuide,
		}
	}

	// For version checks, we might need to do semantic version comparison
	if strings.Contains(dep.Path, "version") && isVersionDeprecated(valueStr, dep.Value) {
		return &Warning{
			Field:          dep.Path,
			Value:          valueStr,
			Message:        dep.Message,
			Severity:       dep.GetSeverity(),
			MigrationGuide: dep.MigrationGuide,
		}
	}

	return nil
}

// checkAPIVersionDeprecation checks if the API version is deprecated
func (c *Checker) checkAPIVersionDeprecation(obj *unstructured.Unstructured, dep *DeprecationInfo) *Warning {
	apiVersion := obj.GetAPIVersion()
	if apiVersion == dep.Value {
		return &Warning{
			Field:          "apiVersion",
			Value:          apiVersion,
			Message:        dep.Message,
			Severity:       dep.GetSeverity(),
			MigrationGuide: dep.MigrationGuide,
		}
	}
	return nil
}

// checkFeatureDeprecation checks for deprecated features
func (c *Checker) checkFeatureDeprecation(obj *unstructured.Unstructured, dep *DeprecationInfo) *Warning {
	// Feature deprecations might involve checking multiple fields or complex conditions
	value, found, err := unstructured.NestedFieldNoCopy(obj.Object, strings.Split(dep.Path, ".")...)
	if err != nil || !found {
		return nil
	}

	// Check if the feature is being used
	if isFeatureInUse(value) {
		return &Warning{
			Field:          dep.Path,
			Message:        dep.Message,
			Severity:       dep.GetSeverity(),
			MigrationGuide: dep.MigrationGuide,
		}
	}

	return nil
}

// isVersionDeprecated checks if a version is considered deprecated
func isVersionDeprecated(current, deprecated string) bool {
	// Simple string comparison for now
	// In a real implementation, you'd use semantic version comparison
	// For example, if deprecated is "< v2.40.0", check if current is less than v2.40.0
	
	// Remove 'v' prefix if present
	current = strings.TrimPrefix(current, "v")
	deprecated = strings.TrimPrefix(deprecated, "v")
	
	// For demonstration, assume deprecated means "less than this version"
	// In production, use a proper semver library
	return current < deprecated
}

// isFeatureInUse checks if a deprecated feature is being used
func isFeatureInUse(value interface{}) bool {
	// Check if the value indicates the feature is enabled
	switch v := value.(type) {
	case bool:
		return v
	case map[string]interface{}:
		// If it's a map and has content, the feature is probably in use
		return len(v) > 0
	default:
		// If the field exists with a non-nil value, consider it in use
		return value != nil
	}
}

// CheckStruct checks a typed struct for deprecations using reflection
func (c *Checker) CheckStruct(obj interface{}, gvk schema.GroupVersionKind) (*CheckResult, error) {
	result := &CheckResult{
		Warnings: []Warning{},
	}

	deprecations := c.registry.GetDeprecations(gvk)
	if len(deprecations) == 0 {
		return result, nil
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, dep := range deprecations {
		warning := c.checkStructDeprecation(v, dep, "")
		if warning != nil {
			result.Warnings = append(result.Warnings, *warning)
			result.HasWarnings = true
			if warning.Severity == SeverityCritical {
				result.HasCritical = true
			}
		}
	}

	return result, nil
}

// checkStructDeprecation checks for deprecations in a struct using reflection
func (c *Checker) checkStructDeprecation(v reflect.Value, dep *DeprecationInfo, currentPath string) *Warning {
	if !v.IsValid() {
		return nil
	}

	// Build the current path
	if currentPath != "" && !strings.HasPrefix(dep.Path, currentPath) {
		return nil
	}

	switch v.Kind() {
	case reflect.Struct:
		// Check each field of the struct
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" { // Skip unexported fields
				continue
			}

			fieldPath := currentPath
			if fieldPath != "" {
				fieldPath += "."
			}
			fieldPath += strings.ToLower(field.Name)

			if warning := c.checkStructDeprecation(v.Field(i), dep, fieldPath); warning != nil {
				return warning
			}
		}

	case reflect.Map:
		// Check map entries
		for _, key := range v.MapKeys() {
			keyPath := currentPath
			if keyPath != "" {
				keyPath += "."
			}
			keyPath += key.String()

			if warning := c.checkStructDeprecation(v.MapIndex(key), dep, keyPath); warning != nil {
				return warning
			}
		}

	case reflect.Slice, reflect.Array:
		// For slices/arrays, we might want to check each element
		// This is simplified - in practice you'd handle this more carefully
		for i := 0; i < v.Len(); i++ {
			if warning := c.checkStructDeprecation(v.Index(i), dep, currentPath); warning != nil {
				return warning
			}
		}

	default:
		// Check if this is the field we're looking for
		if currentPath == dep.Path {
			// Check based on deprecation type
			switch dep.Type {
			case FieldDeprecation:
				// Field exists, so it's deprecated
				return &Warning{
					Field:          dep.Path,
					Message:        dep.Message,
					Severity:       dep.GetSeverity(),
					MigrationGuide: dep.MigrationGuide,
				}
			case ValueDeprecation:
				// Check if value matches
				valueStr := fmt.Sprintf("%v", v.Interface())
				if valueStr == dep.Value {
					return &Warning{
						Field:          dep.Path,
						Value:          dep.Value,
						Message:        dep.Message,
						Severity:       dep.GetSeverity(),
						MigrationGuide: dep.MigrationGuide,
					}
				}
			}
		}
	}

	return nil
}

// FormatWarnings formats all warnings in a result for display
func (c *Checker) FormatWarnings(result *CheckResult) string {
	if len(result.Warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Deprecation warnings found:\n\n")

	// Group by severity
	var critical, warning, info []Warning
	for _, w := range result.Warnings {
		switch w.Severity {
		case SeverityCritical:
			critical = append(critical, w)
		case SeverityWarning:
			warning = append(warning, w)
		default:
			info = append(info, w)
		}
	}

	// Format critical warnings first
	if len(critical) > 0 {
		sb.WriteString("🚨 CRITICAL DEPRECATIONS:\n")
		for _, w := range critical {
			sb.WriteString(c.formatSingleWarning(&w))
			sb.WriteString("\n")
		}
	}

	// Then warnings
	if len(warning) > 0 {
		sb.WriteString("⚠️  DEPRECATIONS:\n")
		for _, w := range warning {
			sb.WriteString(c.formatSingleWarning(&w))
			sb.WriteString("\n")
		}
	}

	// Finally info
	if len(info) > 0 {
		sb.WriteString("ℹ️  DEPRECATION NOTICES:\n")
		for _, w := range info {
			sb.WriteString(c.formatSingleWarning(&w))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatSingleWarning formats a single warning
func (c *Checker) formatSingleWarning(w *Warning) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  - %s\n", w.Message))
	if w.Field != "" {
		sb.WriteString(fmt.Sprintf("    Field: %s\n", w.Field))
	}
	if w.Value != "" {
		sb.WriteString(fmt.Sprintf("    Value: %s\n", w.Value))
	}
	return sb.String()
}
