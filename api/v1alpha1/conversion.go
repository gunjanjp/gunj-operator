/*
Copyright 2025.

Licensed under the MIT License.
*/

package v1alpha1

import (
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/conversion"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var conversionLog = logf.Log.WithName("observabilityplatform-conversion")

// ConvertTo converts this ObservabilityPlatform to the Hub version (v1).
// +kubebuilder:object:generate=false
func (src *ObservabilityPlatform) ConvertTo(dstRaw conversion.Hub) error {
	conversionLog.Info("Converting from v1alpha1 to Hub version", "name", src.Name)
	
	// For now, v1alpha1 is the hub version
	// This will change when we introduce v1beta1 and v1
	return fmt.Errorf("v1alpha1 is currently the storage version")
}

// ConvertFrom converts from the Hub version (v1) to this version.
// +kubebuilder:object:generate=false
func (dst *ObservabilityPlatform) ConvertFrom(srcRaw conversion.Hub) error {
	conversionLog.Info("Converting from Hub version to v1alpha1", "name", dst.Name)
	
	// For now, v1alpha1 is the hub version
	// This will change when we introduce v1beta1 and v1
	return fmt.Errorf("v1alpha1 is currently the storage version")
}

// ConversionReview handles conversion between versions
type ConversionReview struct {
	// Map of deprecated fields to new fields
	FieldMappings map[string]string
	
	// Conversion warnings to be emitted
	Warnings []string
}

// v1alpha1ToV1beta1FieldMappings defines field mappings from v1alpha1 to v1beta1
var v1alpha1ToV1beta1FieldMappings = map[string]string{
	"prometheusEnabled": "components.prometheus.enabled",
	"prometheusVersion": "components.prometheus.version",
	"grafanaEnabled":    "components.grafana.enabled",
	"grafanaVersion":    "components.grafana.version",
	"lokiEnabled":       "components.loki.enabled",
	"lokiVersion":       "components.loki.version",
	"tempoEnabled":      "components.tempo.enabled",
	"tempoVersion":      "components.tempo.version",
}

// ValidateConversion validates that a conversion is possible
func ValidateConversion(src, dst string) error {
	validConversions := map[string][]string{
		"v1alpha1": {"v1beta1", "v1"},
		"v1beta1":  {"v1alpha1", "v1"},
		"v1":       {"v1alpha1", "v1beta1"},
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

// MigrateFieldValue migrates a field value between versions
func MigrateFieldValue(fieldName string, value interface{}, fromVersion, toVersion string) (interface{}, error) {
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
				// Simple implementation for demonstration
				if len(duration) > 1 && duration[len(duration)-1] == 'd' {
					days := duration[:len(duration)-1]
					return days, nil
				}
			}
		}
	
	case "resources":
		// v1alpha1 uses separate fields, v1beta1 uses ResourceRequirements
		if fromVersion == "v1alpha1" && toVersion == "v1beta1" {
			// Convert old format to new format
			// This would need the full object context
			return value, nil
		}
	}
	
	// Default: return value unchanged
	return value, nil
}

// GenerateConversionWarnings generates warnings for deprecated fields
func GenerateConversionWarnings(obj *ObservabilityPlatform) []string {
	var warnings []string
	
	// Check for deprecated patterns
	if obj.Spec.Components.Prometheus != nil && obj.Spec.Components.Prometheus.CustomConfig != nil {
		if _, hasOldConfig := obj.Spec.Components.Prometheus.CustomConfig["memory_request"]; hasOldConfig {
			warnings = append(warnings, "Field 'memory_request' in customConfig is deprecated. Use resources.requests.memory instead")
		}
	}
	
	// Check for soon-to-be deprecated features
	if obj.Spec.Backup != nil && obj.Spec.Backup.Destination.S3 == nil && obj.Spec.Backup.Destination.PVC == nil {
		warnings = append(warnings, "Backup destination must be specified. This will be required in v1beta1")
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
		
		if obj.Spec.Global.LogLevel == "" {
			obj.Spec.Global.LogLevel = "info"
		}
		
	case "v1":
		// Add defaults required for v1
		// v1 might require additional fields
		if obj.Spec.Global.SecurityContext == nil {
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

// VersionAnnotations adds version-related annotations
func VersionAnnotations(version string) map[string]string {
	return map[string]string{
		"observability.io/api-version":           version,
		"observability.io/conversion-timestamp":  fmt.Sprintf("%d", time.Now().Unix()),
		"observability.io/original-version":      version,
	}
}
