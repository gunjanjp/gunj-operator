/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"fmt"
	"sync"
)

// CompatibilityMatrix defines field mappings and compatibility rules between API versions
type CompatibilityMatrix struct {
	mu       sync.RWMutex
	mappings map[string]map[string][]FieldMapping // [srcVersion][dstVersion][]mappings
}

// FieldMapping represents a mapping between fields in different API versions
type FieldMapping struct {
	SourcePath            string
	DestPath              string
	ValidationType        string
	Required              bool
	Deprecated            bool
	DeprecationVersion    string
	MigrationGuide        string
	Enhanced              bool
	EnhancementDescription string
	DataTransform         DataTransformFunc
	SemanticChange        SemanticChangeFunc
	DefaultValue          interface{}
}

// DataTransformFunc transforms data during field mapping
type DataTransformFunc func(srcValue interface{}) (dstValue interface{}, err error)

// SemanticChangeFunc checks for semantic changes and returns a warning message
type SemanticChangeFunc func(srcValue, dstValue interface{}) string

// NewCompatibilityMatrix creates a new compatibility matrix
func NewCompatibilityMatrix() *CompatibilityMatrix {
	cm := &CompatibilityMatrix{
		mappings: make(map[string]map[string][]FieldMapping),
	}
	
	// Initialize default mappings
	cm.initializeV1Alpha1ToV1Beta1Mappings()
	cm.initializeV1Beta1ToV1Alpha1Mappings()
	
	return cm
}

// initializeV1Alpha1ToV1Beta1Mappings sets up mappings from v1alpha1 to v1beta1
func (cm *CompatibilityMatrix) initializeV1Alpha1ToV1Beta1Mappings() {
	mappings := []FieldMapping{
		// Core fields
		{
			SourcePath:     "Spec.Paused",
			DestPath:       "Spec.Paused",
			ValidationType: "boolean",
			Required:       false,
		},
		
		// Prometheus mappings
		{
			SourcePath:     "Spec.Components.Prometheus.Enabled",
			DestPath:       "Spec.Components.Prometheus.Enabled",
			ValidationType: "boolean",
			Required:       false,
		},
		{
			SourcePath:     "Spec.Components.Prometheus.Version",
			DestPath:       "Spec.Components.Prometheus.Version",
			ValidationType: "version",
			Required:       true,
		},
		{
			SourcePath:     "Spec.Components.Prometheus.Replicas",
			DestPath:       "Spec.Components.Prometheus.Replicas",
			ValidationType: "replicas",
			Required:       false,
			DefaultValue:   int32(1),
		},
		{
			SourcePath:     "Spec.Components.Prometheus.Resources",
			DestPath:       "Spec.Components.Prometheus.Resources",
			ValidationType: "resources",
			DataTransform:  transformResourcesV1Alpha1ToV1Beta1,
		},
		{
			SourcePath:     "Spec.Components.Prometheus.Storage.Size",
			DestPath:       "Spec.Components.Prometheus.Storage.Size",
			ValidationType: "storage",
			DataTransform:  transformStorageSizeV1Alpha1ToV1Beta1,
		},
		{
			SourcePath:         "Spec.Components.Prometheus.CustomConfig",
			DestPath:          "Spec.Components.Prometheus.ExternalLabels",
			Deprecated:        true,
			DeprecationVersion: "v1.0.0",
			MigrationGuide:    "Use Spec.Components.Prometheus.ExternalLabels instead of CustomConfig['externalLabels']",
			DataTransform:     transformCustomConfigToExternalLabels,
		},
		
		// Grafana mappings
		{
			SourcePath:     "Spec.Components.Grafana.Enabled",
			DestPath:       "Spec.Components.Grafana.Enabled",
			ValidationType: "boolean",
			Required:       false,
		},
		{
			SourcePath:     "Spec.Components.Grafana.Version",
			DestPath:       "Spec.Components.Grafana.Version",
			ValidationType: "version",
			Required:       true,
		},
		{
			SourcePath:             "Spec.Components.Grafana.Ingress",
			DestPath:              "Spec.Components.Grafana.Ingress",
			ValidationType:        "ingress",
			Enhanced:              true,
			EnhancementDescription: "Ingress now supports additional annotations and middleware configuration",
		},
		
		// New fields in v1beta1
		{
			SourcePath:             "",
			DestPath:              "Spec.Security",
			Enhanced:              true,
			EnhancementDescription: "Security configuration added with TLS, pod security policy, and network policy support",
			DefaultValue: map[string]interface{}{
				"tls": map[string]interface{}{
					"enabled": true,
				},
				"podSecurityPolicy": true,
				"networkPolicy":     true,
			},
		},
		{
			SourcePath:             "",
			DestPath:              "Spec.CostOptimization",
			Enhanced:              true,
			EnhancementDescription: "Cost optimization features including resource recommendations and auto-scaling",
			DefaultValue: map[string]interface{}{
				"enabled": false,
			},
		},
		
		// Global configuration
		{
			SourcePath:     "Spec.Global.ExternalLabels",
			DestPath:       "Spec.Global.ExternalLabels",
			ValidationType: "labels",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil && dst != nil {
					srcLabels := src.(map[string]string)
					dstLabels := dst.(map[string]string)
					if len(srcLabels) != len(dstLabels) {
						return "External labels have been modified during conversion"
					}
				}
				return ""
			},
		},
		
		// High Availability
		{
			SourcePath:     "Spec.HighAvailability.MinReplicas",
			DestPath:       "Spec.HighAvailability.MinReplicas",
			ValidationType: "replicas",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil {
					srcReplicas := src.(int32)
					if srcReplicas < 3 {
						return fmt.Sprintf("MinReplicas %d is below recommended value of 3 for high availability", srcReplicas)
					}
				}
				return ""
			},
		},
		
		// Status fields
		{
			SourcePath:     "Status.Phase",
			DestPath:       "Status.Phase",
			ValidationType: "phase",
			Required:       true,
		},
		{
			SourcePath:     "Status.ComponentStatus",
			DestPath:       "Status.ComponentStatus",
			DataTransform:  transformComponentStatusV1Alpha1ToV1Beta1,
		},
	}
	
	cm.setMappings("v1alpha1", "v1beta1", mappings)
}

// initializeV1Beta1ToV1Alpha1Mappings sets up mappings from v1beta1 to v1alpha1
func (cm *CompatibilityMatrix) initializeV1Beta1ToV1Alpha1Mappings() {
	mappings := []FieldMapping{
		// Core fields
		{
			SourcePath:     "Spec.Paused",
			DestPath:       "Spec.Paused",
			ValidationType: "boolean",
			Required:       false,
		},
		
		// Fields that will be lost
		{
			SourcePath:     "Spec.Security",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil {
					return "Security configuration will be lost when converting to v1alpha1"
				}
				return ""
			},
		},
		{
			SourcePath:     "Spec.CostOptimization",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil {
					return "Cost optimization configuration will be lost when converting to v1alpha1"
				}
				return ""
			},
		},
		{
			SourcePath:     "Spec.Components.Prometheus.ExternalLabels",
			DestPath:       "Spec.Components.Prometheus.CustomConfig",
			DataTransform:  transformExternalLabelsToCustomConfig,
			SemanticChange: func(src, dst interface{}) string {
				return "ExternalLabels will be moved to CustomConfig map"
			},
		},
		
		// Component fields
		{
			SourcePath:     "Spec.Components.Prometheus.AdditionalScrapeConfigs",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil {
					return "Additional scrape configs will be lost when converting to v1alpha1"
				}
				return ""
			},
		},
		{
			SourcePath:     "Spec.Components.Grafana.Plugins",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil {
					return "Grafana plugins configuration will be lost when converting to v1alpha1"
				}
				return ""
			},
		},
		{
			SourcePath:     "Spec.Components.Grafana.SMTP",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil {
					return "Grafana SMTP configuration will be lost when converting to v1alpha1"
				}
				return ""
			},
		},
		{
			SourcePath:     "Spec.Components.Loki.CompactorEnabled",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil && src.(bool) {
					return "Loki compactor configuration will be lost when converting to v1alpha1"
				}
				return ""
			},
		},
		{
			SourcePath:     "Spec.Components.Tempo.SearchEnabled",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				if src != nil && src.(bool) {
					return "Tempo search feature will be disabled when converting to v1alpha1"
				}
				return ""
			},
		},
		
		// Status enhancements lost
		{
			SourcePath:     "Status.Version",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				return "Platform version tracking will be lost when converting to v1alpha1"
			},
		},
		{
			SourcePath:     "Status.Health",
			DestPath:       "",
			SemanticChange: func(src, dst interface{}) string {
				return "Health status information will be lost when converting to v1alpha1"
			},
		},
	}
	
	cm.setMappings("v1beta1", "v1alpha1", mappings)
}

// GetFieldMappings returns field mappings for a specific conversion
func (cm *CompatibilityMatrix) GetFieldMappings(srcVersion, dstVersion string) []FieldMapping {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if srcMappings, ok := cm.mappings[srcVersion]; ok {
		if dstMappings, ok := srcMappings[dstVersion]; ok {
			return dstMappings
		}
	}
	
	return []FieldMapping{}
}

// GetLostFields returns fields that will be lost during conversion
func (cm *CompatibilityMatrix) GetLostFields(srcVersion, dstVersion string) []string {
	lostFields := []string{}
	
	mappings := cm.GetFieldMappings(srcVersion, dstVersion)
	for _, mapping := range mappings {
		if mapping.DestPath == "" && mapping.SourcePath != "" {
			lostFields = append(lostFields, mapping.SourcePath)
		}
	}
	
	// Additional lost fields for v1beta1 to v1alpha1
	if srcVersion == "v1beta1" && dstVersion == "v1alpha1" {
		additionalLostFields := []string{
			"Spec.Security",
			"Spec.CostOptimization",
			"Spec.ServiceMesh",
			"Spec.MultiCluster",
			"Spec.Components.Prometheus.AdditionalScrapeConfigs",
			"Spec.Components.Grafana.Plugins",
			"Spec.Components.Grafana.SMTP",
			"Spec.Components.Loki.CompactorEnabled",
			"Spec.Components.Tempo.SearchEnabled",
			"Spec.Global.Affinity",
			"Spec.Global.ImagePullSecrets",
			"Spec.HighAvailability.AntiAffinity",
			"Status.Version",
			"Status.Health",
		}
		lostFields = append(lostFields, additionalLostFields...)
	}
	
	return lostFields
}

// setMappings sets the mappings for a specific version conversion
func (cm *CompatibilityMatrix) setMappings(srcVersion, dstVersion string, mappings []FieldMapping) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cm.mappings[srcVersion] == nil {
		cm.mappings[srcVersion] = make(map[string][]FieldMapping)
	}
	
	cm.mappings[srcVersion][dstVersion] = mappings
}

// Transform functions

func transformResourcesV1Alpha1ToV1Beta1(src interface{}) (interface{}, error) {
	// Transform v1alpha1.ResourceRequirements to corev1.ResourceRequirements
	// This is handled in the main conversion logic
	return src, nil
}

func transformStorageSizeV1Alpha1ToV1Beta1(src interface{}) (interface{}, error) {
	// Transform string size to resource.Quantity
	// This is handled in the main conversion logic
	return src, nil
}

func transformCustomConfigToExternalLabels(src interface{}) (interface{}, error) {
	customConfig, ok := src.(map[string]string)
	if !ok || customConfig == nil {
		return map[string]string{}, nil
	}
	
	if labels, exists := customConfig["externalLabels"]; exists {
		// Parse labels string into map
		// In real implementation, parse JSON or YAML
		return map[string]string{"converted": labels}, nil
	}
	
	return map[string]string{}, nil
}

func transformExternalLabelsToCustomConfig(src interface{}) (interface{}, error) {
	labels, ok := src.(map[string]string)
	if !ok || labels == nil {
		return map[string]string{}, nil
	}
	
	// Convert labels map to custom config format
	customConfig := make(map[string]string)
	if len(labels) > 0 {
		// In real implementation, serialize to JSON or YAML
		customConfig["externalLabels"] = fmt.Sprintf("%v", labels)
	}
	
	return customConfig, nil
}

func transformComponentStatusV1Alpha1ToV1Beta1(src interface{}) (interface{}, error) {
	// Component status transformation is handled in the main conversion
	return src, nil
}
