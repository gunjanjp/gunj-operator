/*
Copyright 2025.

Licensed under the MIT License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	conversionpkg "github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
)

var conversionLog = logf.Log.WithName("observabilityplatform-conversion")

// ConvertTo converts this ObservabilityPlatform to the Hub version (v1beta1).
// This is implemented in observabilityplatform_conversion.go with enhanced validation
// +kubebuilder:object:generate=false
func (src *ObservabilityPlatform) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.ObservabilityPlatform)
	logger := conversionLog.WithValues("name", src.Name, "namespace", src.Namespace)
	
	// Use the enhanced conversion validation
	validator := conversionpkg.NewFieldValidator(logger)
	dryRun := conversionpkg.NewDryRunConverter(logger)
	
	// Perform dry-run validation first
	ctx := context.Background()
	result, err := dryRun.DryRunConversion(ctx, src, "v1beta1")
	if err != nil {
		return fmt.Errorf("dry-run validation failed: %w", err)
	}
	
	// Check validation results
	if !result.Success {
		// Build error message from validation errors
		errMsg := "conversion validation failed:\n"
		for _, e := range result.ValidationResult.Errors {
			errMsg += fmt.Sprintf("- %s: %s\n", e.Field, e.Detail)
		}
		return fmt.Errorf(errMsg)
	}
	
	// Log warnings
	for _, warning := range result.ValidationResult.Warnings {
		logger.Info("Conversion warning", "warning", warning)
	}
	
	// Perform the actual conversion (delegate to the existing implementation)
	// This is implemented in observabilityplatform_conversion.go
	return src.convertTo(dst)
}

// ConvertFrom converts from the Hub version (v1beta1) to this version.
// This is implemented in observabilityplatform_conversion.go with enhanced validation
// +kubebuilder:object:generate=false
func (dst *ObservabilityPlatform) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.ObservabilityPlatform)
	logger := conversionLog.WithValues("name", src.Name, "namespace", src.Namespace)
	
	// Use the enhanced conversion validation
	validator := conversionpkg.NewFieldValidator(logger)
	dryRun := conversionpkg.NewDryRunConverter(logger)
	
	// Perform dry-run validation first
	ctx := context.Background()
	result, err := dryRun.DryRunConversion(ctx, src, "v1alpha1")
	if err != nil {
		return fmt.Errorf("dry-run validation failed: %w", err)
	}
	
	// Check validation results
	if !result.Success {
		// Build error message from validation errors
		errMsg := "conversion validation failed:\n"
		for _, e := range result.ValidationResult.Errors {
			errMsg += fmt.Sprintf("- %s: %s\n", e.Field, e.Detail)
		}
		return fmt.Errorf(errMsg)
	}
	
	// Log warnings about data loss
	if result.EstimatedImpact.DataLoss {
		logger.Info("Data loss will occur during conversion",
			"lostFields", result.EstimatedImpact.LostFields)
		for _, warning := range result.ValidationResult.Warnings {
			logger.Info("Conversion warning", "warning", warning)
		}
	}
	
	// Perform the actual conversion (delegate to the existing implementation)
	// This is implemented in observabilityplatform_conversion.go
	return dst.convertFrom(src)
}

// ConversionReview handles conversion between versions with enhanced validation
type ConversionReview struct {
	// Map of deprecated fields to new fields
	FieldMappings map[string]string
	
	// Conversion warnings to be emitted
	Warnings []string
	
	// Validation result from enhanced validation
	ValidationResult *conversionpkg.ValidationResult
	
	// Metrics for tracking conversions
	Metrics *conversionpkg.ConversionMetrics
}

// NewConversionReview creates a new conversion review with validation
func NewConversionReview() *ConversionReview {
	return &ConversionReview{
		FieldMappings: v1alpha1ToV1beta1FieldMappings,
		Warnings:      []string{},
		Metrics:       conversionpkg.GetMetrics(),
	}
}

// v1alpha1ToV1beta1FieldMappings defines field mappings from v1alpha1 to v1beta1
var v1alpha1ToV1beta1FieldMappings = map[string]string{
	"spec.components.prometheus.customConfig": "spec.components.prometheus.externalLabels",
	"spec.alerting.alertManager":              "spec.alerting.alertmanager",
	"spec.backup.destination.pvc":             "spec.backup.destination.local",
}

// ValidateConversion validates that a conversion is possible with enhanced checks
func ValidateConversion(src, dst string) error {
	validConversions := map[string][]string{
		"v1alpha1": {"v1beta1"},
		"v1beta1":  {"v1alpha1", "v1"},
		"v1":       {"v1beta1"},
	}
	
	allowed, ok := validConversions[src]
	if !ok {
		return fmt.Errorf("unknown source version: %s", src)
	}
	
	for _, v := range allowed {
		if v == dst {
			return nil
		}
	}
	
	return fmt.Errorf("conversion from %s to %s not supported", src, dst)
}

// MigrateFieldValue migrates a field value between versions with validation
func MigrateFieldValue(fieldName string, value interface{}, fromVersion, toVersion string) (interface{}, error) {
	// Get compatibility matrix
	matrix := conversionpkg.NewCompatibilityMatrix()
	mappings := matrix.GetFieldMappings(fromVersion, toVersion)
	
	// Find mapping for this field
	for _, mapping := range mappings {
		if mapping.SourcePath == fieldName && mapping.DataTransform != nil {
			return mapping.DataTransform(value)
		}
	}
	
	// Special handling for certain fields
	switch fieldName {
	case "retention":
		// v1alpha1 uses days (int), v1beta1 uses duration string
		if fromVersion == "v1alpha1" && toVersion == "v1beta1" {
			if days, ok := value.(int); ok {
				return fmt.Sprintf("%dd", days), nil
			}
		}
		// v1beta1 to v1alpha1
		if fromVersion == "v1beta1" && toVersion == "v1alpha1" {
			if duration, ok := value.(string); ok {
				// Parse duration and convert to days
				if len(duration) > 1 && duration[len(duration)-1] == 'd' {
					days := duration[:len(duration)-1]
					return days, nil
				}
			}
		}
	}
	
	// Default: return value unchanged
	return value, nil
}

// GenerateConversionWarnings generates warnings for deprecated fields using validation framework
func GenerateConversionWarnings(obj *ObservabilityPlatform) []string {
	logger := conversionLog.WithValues("name", obj.Name, "namespace", obj.Namespace)
	validator := conversionpkg.NewFieldValidator(logger)
	
	// Validate conversion to v1beta1 to detect issues
	result := validator.ValidateConversion("v1alpha1", "v1beta1", obj, &v1beta1.ObservabilityPlatform{})
	
	warnings := make([]string, 0, len(result.Warnings))
	warnings = append(warnings, result.Warnings...)
	
	// Add custom warnings
	if obj.Spec.Components.Prometheus != nil && obj.Spec.Components.Prometheus.CustomConfig != nil {
		if _, hasOldConfig := obj.Spec.Components.Prometheus.CustomConfig["memory_request"]; hasOldConfig {
			warnings = append(warnings, "Field 'memory_request' in customConfig is deprecated. Use resources.requests.memory instead")
		}
	}
	
	return warnings
}

// DefaultMissingFields adds default values for fields that are required in newer versions
func DefaultMissingFields(obj *ObservabilityPlatform, targetVersion string) {
	switch targetVersion {
	case "v1beta1":
		// Add defaults required for v1beta1
		if obj.Spec.Components.Prometheus != nil {
			if obj.Spec.Components.Prometheus.Retention == "" {
				obj.Spec.Components.Prometheus.Retention = "30d"
			}
			if obj.Spec.Components.Prometheus.Replicas == 0 {
				obj.Spec.Components.Prometheus.Replicas = 1
			}
		}
		
		if obj.Spec.Global != nil && obj.Spec.Global.LogLevel == "" {
			obj.Spec.Global.LogLevel = "info"
		}
		
	case "v1":
		// Add defaults required for v1
		if obj.Spec.Global != nil && obj.Spec.Global.SecurityContext == nil {
			obj.Spec.Global.SecurityContext = &SecurityContext{
				RunAsNonRoot:           boolPtr(true),
				RunAsUser:              int64Ptr(65534),
				FSGroup:                int64Ptr(65534),
				ReadOnlyRootFilesystem: boolPtr(true),
			}
		}
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

// VersionAnnotations adds version-related annotations with tracking
func VersionAnnotations(version string) map[string]string {
	return map[string]string{
		"observability.io/api-version":           version,
		"observability.io/conversion-timestamp":  fmt.Sprintf("%d", time.Now().Unix()),
		"observability.io/original-version":      version,
		"observability.io/conversion-validated":  "true",
	}
}

// RecordConversionMetrics records conversion metrics
func RecordConversionMetrics(srcVersion, dstVersion string, success bool, duration time.Duration) {
	metrics := conversionpkg.GetMetrics()
	metrics.RecordConversion(srcVersion, dstVersion, "ObservabilityPlatform", duration, success)
}
