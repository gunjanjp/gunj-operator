/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm

import (
	"fmt"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v3"
)

// valueBuilder implements the ValueBuilder interface
type valueBuilder struct {
	defaultValues map[string]map[string]interface{}
}

// NewValueBuilder creates a new value builder
func NewValueBuilder() ValueBuilder {
	vb := &valueBuilder{
		defaultValues: make(map[string]map[string]interface{}),
	}
	
	// Initialize default values for each component
	vb.initializeDefaults()
	
	return vb
}

// initializeDefaults sets up default values for each component type
func (vb *valueBuilder) initializeDefaults() {
	// Prometheus defaults
	vb.defaultValues["prometheus"] = map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "prom/prometheus",
			"pullPolicy": "IfNotPresent",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"port": 9090,
		},
		"persistence": map[string]interface{}{
			"enabled": true,
			"size":    "8Gi",
		},
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "250m",
				"memory": "512Mi",
			},
			"limits": map[string]interface{}{
				"cpu":    "1000m",
				"memory": "2Gi",
			},
		},
		"retention": "15d",
		"scrapeInterval": "30s",
		"evaluationInterval": "30s",
	}
	
	// Grafana defaults
	vb.defaultValues["grafana"] = map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "grafana/grafana",
			"pullPolicy": "IfNotPresent",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"port": 3000,
		},
		"persistence": map[string]interface{}{
			"enabled": true,
			"size":    "2Gi",
		},
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			"limits": map[string]interface{}{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		},
		"adminPassword": "admin",
		"ingress": map[string]interface{}{
			"enabled": false,
		},
		"datasources": map[string]interface{}{
			"datasources.yaml": map[string]interface{}{
				"apiVersion": 1,
				"datasources": []interface{}{},
			},
		},
	}
	
	// Loki defaults
	vb.defaultValues["loki"] = map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "grafana/loki",
			"pullPolicy": "IfNotPresent",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"port": 3100,
		},
		"persistence": map[string]interface{}{
			"enabled": true,
			"size":    "10Gi",
		},
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "100m",
				"memory": "256Mi",
			},
			"limits": map[string]interface{}{
				"cpu":    "500m",
				"memory": "1Gi",
			},
		},
		"config": map[string]interface{}{
			"auth_enabled": false,
			"ingester": map[string]interface{}{
				"chunk_idle_period": "3m",
				"chunk_block_size":  262144,
			},
			"limits_config": map[string]interface{}{
				"retention_period": "168h",
			},
		},
	}
	
	// Tempo defaults
	vb.defaultValues["tempo"] = map[string]interface{}{
		"replicaCount": 1,
		"image": map[string]interface{}{
			"repository": "grafana/tempo",
			"pullPolicy": "IfNotPresent",
		},
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"httpPort": 3200,
			"grpcPort": 9095,
		},
		"persistence": map[string]interface{}{
			"enabled": true,
			"size":    "10Gi",
		},
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "100m",
				"memory": "256Mi",
			},
			"limits": map[string]interface{}{
				"cpu":    "500m",
				"memory": "1Gi",
			},
		},
		"config": map[string]interface{}{
			"search_enabled": true,
			"metrics_generator": map[string]interface{}{
				"registry": map[string]interface{}{
					"external_labels": map[string]interface{}{
						"source": "tempo",
					},
				},
			},
		},
	}
}

// BuildValues builds Helm values for a component
func (vb *valueBuilder) BuildValues(componentType string, spec interface{}) (map[string]interface{}, error) {
	// Get default values for the component type
	defaults, ok := vb.defaultValues[componentType]
	if !ok {
		return nil, fmt.Errorf("unknown component type: %s", componentType)
	}
	
	// Make a copy of defaults
	values := make(map[string]interface{})
	if err := mergo.Merge(&values, defaults); err != nil {
		return nil, fmt.Errorf("failed to copy defaults: %w", err)
	}
	
	// Build component-specific values
	switch componentType {
	case "prometheus":
		if prometheusSpec, ok := spec.(*observabilityv1beta1.PrometheusSpec); ok {
			vb.buildPrometheusValues(values, prometheusSpec)
		}
	case "grafana":
		if grafanaSpec, ok := spec.(*observabilityv1beta1.GrafanaSpec); ok {
			vb.buildGrafanaValues(values, grafanaSpec)
		}
	case "loki":
		if lokiSpec, ok := spec.(*observabilityv1beta1.LokiSpec); ok {
			vb.buildLokiValues(values, lokiSpec)
		}
	case "tempo":
		if tempoSpec, ok := spec.(*observabilityv1beta1.TempoSpec); ok {
			vb.buildTempoValues(values, tempoSpec)
		}
	default:
		return nil, fmt.Errorf("unsupported component type: %s", componentType)
	}
	
	return values, nil
}

// buildPrometheusValues builds Prometheus-specific values
func (vb *valueBuilder) buildPrometheusValues(values map[string]interface{}, spec *observabilityv1beta1.PrometheusSpec) {
	if spec == nil {
		return
	}
	
	// Set version
	if spec.Version != "" {
		values["image"].(map[string]interface{})["tag"] = spec.Version
	}
	
	// Set replicas
	if spec.Replicas != nil {
		values["replicaCount"] = *spec.Replicas
	}
	
	// Set resources
	if spec.Resources != nil {
		resources := values["resources"].(map[string]interface{})
		if spec.Resources.Requests != nil {
			requests := resources["requests"].(map[string]interface{})
			if cpu := spec.Resources.Requests.Cpu(); cpu != nil {
				requests["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Requests.Memory(); memory != nil {
				requests["memory"] = memory.String()
			}
		}
		if spec.Resources.Limits != nil {
			limits := resources["limits"].(map[string]interface{})
			if cpu := spec.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
	}
	
	// Set storage
	if spec.Storage != nil {
		persistence := values["persistence"].(map[string]interface{})
		if spec.Storage.Size != "" {
			persistence["size"] = spec.Storage.Size
		}
		if spec.Storage.StorageClass != "" {
			persistence["storageClass"] = spec.Storage.StorageClass
		}
	}
	
	// Set retention
	if spec.Retention != "" {
		values["retention"] = spec.Retention
	}
	
	// Set custom config
	if spec.Config != nil {
		values["config"] = spec.Config
	}
	
	// Set external labels
	if spec.ExternalLabels != nil {
		values["externalLabels"] = spec.ExternalLabels
	}
	
	// Set remote write
	if spec.RemoteWrite != nil {
		values["remoteWrite"] = spec.RemoteWrite
	}
}

// buildGrafanaValues builds Grafana-specific values
func (vb *valueBuilder) buildGrafanaValues(values map[string]interface{}, spec *observabilityv1beta1.GrafanaSpec) {
	if spec == nil {
		return
	}
	
	// Set version
	if spec.Version != "" {
		values["image"].(map[string]interface{})["tag"] = spec.Version
	}
	
	// Set replicas
	if spec.Replicas != nil {
		values["replicaCount"] = *spec.Replicas
	}
	
	// Set resources
	if spec.Resources != nil {
		resources := values["resources"].(map[string]interface{})
		if spec.Resources.Requests != nil {
			requests := resources["requests"].(map[string]interface{})
			if cpu := spec.Resources.Requests.Cpu(); cpu != nil {
				requests["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Requests.Memory(); memory != nil {
				requests["memory"] = memory.String()
			}
		}
		if spec.Resources.Limits != nil {
			limits := resources["limits"].(map[string]interface{})
			if cpu := spec.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
	}
	
	// Set storage
	if spec.Storage != nil {
		persistence := values["persistence"].(map[string]interface{})
		if spec.Storage.Size != "" {
			persistence["size"] = spec.Storage.Size
		}
		if spec.Storage.StorageClass != "" {
			persistence["storageClass"] = spec.Storage.StorageClass
		}
	}
	
	// Set admin password
	if spec.AdminPassword != "" {
		values["adminPassword"] = spec.AdminPassword
	}
	
	// Set ingress
	if spec.Ingress != nil {
		ingress := values["ingress"].(map[string]interface{})
		ingress["enabled"] = spec.Ingress.Enabled
		if spec.Ingress.Host != "" {
			ingress["hosts"] = []interface{}{
				map[string]interface{}{
					"host": spec.Ingress.Host,
					"paths": []interface{}{
						map[string]interface{}{
							"path": "/",
							"pathType": "Prefix",
						},
					},
				},
			}
		}
		if spec.Ingress.TLSSecret != "" {
			ingress["tls"] = []interface{}{
				map[string]interface{}{
					"secretName": spec.Ingress.TLSSecret,
					"hosts":      []interface{}{spec.Ingress.Host},
				},
			}
		}
	}
	
	// Set datasources
	if spec.DataSources != nil {
		values["datasources"] = spec.DataSources
	}
	
	// Set dashboards
	if spec.Dashboards != nil {
		values["dashboards"] = spec.Dashboards
	}
}

// buildLokiValues builds Loki-specific values
func (vb *valueBuilder) buildLokiValues(values map[string]interface{}, spec *observabilityv1beta1.LokiSpec) {
	if spec == nil {
		return
	}
	
	// Set version
	if spec.Version != "" {
		values["image"].(map[string]interface{})["tag"] = spec.Version
	}
	
	// Set replicas
	if spec.Replicas != nil {
		values["replicaCount"] = *spec.Replicas
	}
	
	// Set resources
	if spec.Resources != nil {
		resources := values["resources"].(map[string]interface{})
		if spec.Resources.Requests != nil {
			requests := resources["requests"].(map[string]interface{})
			if cpu := spec.Resources.Requests.Cpu(); cpu != nil {
				requests["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Requests.Memory(); memory != nil {
				requests["memory"] = memory.String()
			}
		}
		if spec.Resources.Limits != nil {
			limits := resources["limits"].(map[string]interface{})
			if cpu := spec.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
	}
	
	// Set storage
	if spec.Storage != nil {
		persistence := values["persistence"].(map[string]interface{})
		if spec.Storage.Size != "" {
			persistence["size"] = spec.Storage.Size
		}
		if spec.Storage.StorageClass != "" {
			persistence["storageClass"] = spec.Storage.StorageClass
		}
	}
	
	// Set retention days
	if spec.RetentionDays > 0 {
		config := values["config"].(map[string]interface{})
		limitsConfig := config["limits_config"].(map[string]interface{})
		limitsConfig["retention_period"] = fmt.Sprintf("%dh", spec.RetentionDays*24)
	}
	
	// Set S3 configuration
	if spec.S3 != nil && spec.S3.Enabled {
		config := values["config"].(map[string]interface{})
		config["storage_config"] = map[string]interface{}{
			"aws": map[string]interface{}{
				"s3": spec.S3.BucketName,
				"region": spec.S3.Region,
				"access_key_id": spec.S3.AccessKeyID,
				"secret_access_key": spec.S3.SecretAccessKey,
			},
		}
	}
}

// buildTempoValues builds Tempo-specific values
func (vb *valueBuilder) buildTempoValues(values map[string]interface{}, spec *observabilityv1beta1.TempoSpec) {
	if spec == nil {
		return
	}
	
	// Set version
	if spec.Version != "" {
		values["image"].(map[string]interface{})["tag"] = spec.Version
	}
	
	// Set replicas
	if spec.Replicas != nil {
		values["replicaCount"] = *spec.Replicas
	}
	
	// Set resources
	if spec.Resources != nil {
		resources := values["resources"].(map[string]interface{})
		if spec.Resources.Requests != nil {
			requests := resources["requests"].(map[string]interface{})
			if cpu := spec.Resources.Requests.Cpu(); cpu != nil {
				requests["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Requests.Memory(); memory != nil {
				requests["memory"] = memory.String()
			}
		}
		if spec.Resources.Limits != nil {
			limits := resources["limits"].(map[string]interface{})
			if cpu := spec.Resources.Limits.Cpu(); cpu != nil {
				limits["cpu"] = cpu.String()
			}
			if memory := spec.Resources.Limits.Memory(); memory != nil {
				limits["memory"] = memory.String()
			}
		}
	}
	
	// Set storage
	if spec.Storage != nil {
		persistence := values["persistence"].(map[string]interface{})
		if spec.Storage.Size != "" {
			persistence["size"] = spec.Storage.Size
		}
		if spec.Storage.StorageClass != "" {
			persistence["storageClass"] = spec.Storage.StorageClass
		}
	}
	
	// Set retention days
	if spec.RetentionDays > 0 {
		config := values["config"].(map[string]interface{})
		config["retention"] = fmt.Sprintf("%dd", spec.RetentionDays)
	}
	
	// Set receivers
	if spec.Receivers != nil {
		values["receivers"] = spec.Receivers
	}
}

// MergeValues merges multiple value sets
func (vb *valueBuilder) MergeValues(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Deep copy base
	if err := mergo.Merge(&result, base, mergo.WithOverride); err != nil {
		// If merge fails, do simple copy
		for k, v := range base {
			result[k] = v
		}
	}
	
	// Override with values from override map
	if err := mergo.Merge(&result, override, mergo.WithOverride); err != nil {
		// If merge fails, do simple override
		for k, v := range override {
			result[k] = v
		}
	}
	
	return result
}

// ValidateValues validates Helm values against a schema
func (vb *valueBuilder) ValidateValues(componentType string, values map[string]interface{}) error {
	// Basic validation - ensure required fields are present
	switch componentType {
	case "prometheus":
		return vb.validatePrometheusValues(values)
	case "grafana":
		return vb.validateGrafanaValues(values)
	case "loki":
		return vb.validateLokiValues(values)
	case "tempo":
		return vb.validateTempoValues(values)
	default:
		return fmt.Errorf("unknown component type: %s", componentType)
	}
}

// validatePrometheusValues validates Prometheus values
func (vb *valueBuilder) validatePrometheusValues(values map[string]interface{}) error {
	// Check required fields
	if _, ok := values["image"]; !ok {
		return fmt.Errorf("image configuration is required")
	}
	
	if _, ok := values["resources"]; !ok {
		return fmt.Errorf("resources configuration is required")
	}
	
	return nil
}

// validateGrafanaValues validates Grafana values
func (vb *valueBuilder) validateGrafanaValues(values map[string]interface{}) error {
	// Check required fields
	if _, ok := values["image"]; !ok {
		return fmt.Errorf("image configuration is required")
	}
	
	if _, ok := values["adminPassword"]; !ok {
		return fmt.Errorf("adminPassword is required")
	}
	
	return nil
}

// validateLokiValues validates Loki values
func (vb *valueBuilder) validateLokiValues(values map[string]interface{}) error {
	// Check required fields
	if _, ok := values["image"]; !ok {
		return fmt.Errorf("image configuration is required")
	}
	
	if _, ok := values["config"]; !ok {
		return fmt.Errorf("config is required")
	}
	
	return nil
}

// validateTempoValues validates Tempo values
func (vb *valueBuilder) validateTempoValues(values map[string]interface{}) error {
	// Check required fields
	if _, ok := values["image"]; !ok {
		return fmt.Errorf("image configuration is required")
	}
	
	if _, ok := values["config"]; !ok {
		return fmt.Errorf("config is required")
	}
	
	return nil
}

// GetDefaultValues returns default values for a component
func (vb *valueBuilder) GetDefaultValues(componentType string) map[string]interface{} {
	defaults, ok := vb.defaultValues[componentType]
	if !ok {
		return make(map[string]interface{})
	}
	
	// Return a copy
	result := make(map[string]interface{})
	if err := mergo.Merge(&result, defaults); err != nil {
		// If merge fails, return empty map
		return make(map[string]interface{})
	}
	
	return result
}
