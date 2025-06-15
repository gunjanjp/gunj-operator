/*
Copyright 2025.

Licensed under the MIT License.
*/

package loki

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/helm"
	"github.com/gunjanjp/gunj-operator/internal/managers"
)

const (
	// Component name for Helm
	componentNameHelm = "loki"
	
	// Chart information
	chartRepository = "grafana"
	chartName       = "loki"
	
	// Default values
	defaultPortHelm = 3100
)

// LokiManagerHelm manages Loki deployments using Helm
type LokiManagerHelm struct {
	*helm.BaseHelmManager
}

// NewLokiManagerHelm creates a new Loki manager with Helm integration
func NewLokiManagerHelm(client client.Client, scheme *runtime.Scheme, restConfig *rest.Config) (managers.LokiManager, error) {
	baseManager, err := helm.NewBaseHelmManager(client, scheme, restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base Helm manager: %w", err)
	}
	
	return &LokiManagerHelm{
		BaseHelmManager: baseManager,
	}, nil
}

// Reconcile reconciles the Loki component
func (m *LokiManagerHelm) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Loki component with provided configuration
func (m *LokiManagerHelm) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	
	// Check if Loki is enabled
	if platform.Spec.Components.Loki == nil || !platform.Spec.Components.Loki.Enabled {
		logger.V(1).Info("Loki is disabled, checking if it needs to be uninstalled")
		
		// Check if it's installed and uninstall if needed
		status, err := m.GetStatus(ctx, platform)
		if err != nil {
			return err
		}
		
		if status.Status != "NotInstalled" && status.Status != "Disabled" {
			return m.Delete(ctx, platform)
		}
		
		return nil
	}
	
	lokiSpec := platform.Spec.Components.Loki
	logger.Info("Reconciling Loki", "version", lokiSpec.Version)
	
	// Ensure repositories are configured
	if err := m.BaseHelmManager.EnsureRepositories(ctx); err != nil {
		return fmt.Errorf("failed to ensure repositories: %w", err)
	}
	
	// Build Helm values
	values, err := m.buildHelmValues(platform, lokiSpec, config)
	if err != nil {
		return fmt.Errorf("failed to build Helm values: %w", err)
	}
	
	// Validate values
	if err := m.ValueBuilder.ValidateValues(componentNameHelm, values); err != nil {
		return fmt.Errorf("invalid Helm values: %w", err)
	}
	
	// Get chart reference
	chartRef := fmt.Sprintf("%s/%s", chartRepository, chartName)
	
	// Determine version to use
	version := lokiSpec.Version
	if version == "" || version == "latest" {
		// Get latest stable version
		latestVersion, err := m.VersionManager.GetLatestVersion(ctx, chartRef)
		if err != nil {
			return fmt.Errorf("failed to get latest version: %w", err)
		}
		version = latestVersion
		logger.Info("Using latest version", "version", version)
	}
	
	// Install or upgrade
	if err := m.BaseHelmManager.InstallOrUpgradeComponent(
		ctx,
		platform,
		componentNameHelm,
		chartRef,
		version,
		values,
	); err != nil {
		return fmt.Errorf("failed to install/upgrade Loki: %w", err)
	}
	
	logger.Info("Loki reconciliation completed successfully")
	return nil
}

// Delete removes the Loki component resources
func (m *LokiManagerHelm) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Deleting Loki resources")
	
	if err := m.BaseHelmManager.UninstallComponent(ctx, platform, componentNameHelm); err != nil {
		return fmt.Errorf("failed to uninstall Loki: %w", err)
	}
	
	return nil
}

// GetStatus returns the current status of the Loki component
func (m *LokiManagerHelm) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	// Check if Loki is enabled
	if platform.Spec.Components.Loki == nil || !platform.Spec.Components.Loki.Enabled {
		return &observabilityv1beta1.ComponentStatus{
			Name:    componentNameHelm,
			Status:  "Disabled",
			Message: "Loki is disabled",
		}, nil
	}
	
	return m.BaseHelmManager.GetComponentStatus(ctx, platform, componentNameHelm)
}

// Validate validates the Loki configuration
func (m *LokiManagerHelm) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Loki == nil {
		return nil
	}
	
	// Build values to validate
	values, err := m.buildHelmValues(platform, platform.Spec.Components.Loki, nil)
	if err != nil {
		return err
	}
	
	// Validate using value builder
	return m.ValueBuilder.ValidateValues(componentNameHelm, values)
}

// GetServiceURL returns the service URL for Loki
func (m *LokiManagerHelm) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	releaseName := fmt.Sprintf("%s-%s", platform.Name, componentNameHelm)
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", 
		releaseName,
		platform.Namespace, 
		defaultPortHelm)
}

// ConfigureStorage configures storage backend for Loki
func (m *LokiManagerHelm) ConfigureStorage(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Configuring storage")
	
	// Storage is configured via Helm values
	// Trigger a reconciliation to update storage configuration
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// UpdateRetention updates log retention policies
func (m *LokiManagerHelm) UpdateRetention(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Updating retention policies")
	
	// Retention is configured via Helm values
	// Trigger a reconciliation to update retention
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// buildHelmValues builds Helm values for Loki
func (m *LokiManagerHelm) buildHelmValues(
	platform *observabilityv1beta1.ObservabilityPlatform,
	lokiSpec *observabilityv1beta1.LokiSpec,
	overrides map[string]interface{},
) (map[string]interface{}, error) {
	// Get base values from value builder
	values, err := m.ValueBuilder.BuildValues(componentNameHelm, lokiSpec)
	if err != nil {
		return nil, err
	}
	
	// Loki deployment mode
	// Use simple scalable mode for production
	values["deploymentMode"] = "SingleBinary"
	
	// Configure single binary settings
	singleBinary := map[string]interface{}{
		"replicas": 1,
	}
	
	if lokiSpec.Replicas != nil {
		singleBinary["replicas"] = *lokiSpec.Replicas
	}
	
	if lokiSpec.Resources != nil {
		singleBinary["resources"] = lokiSpec.Resources
	}
	
	values["singleBinary"] = singleBinary
	
	// Configure storage
	storage := map[string]interface{}{
		"type": "filesystem",
	}
	
	// Check if S3 is configured
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		storage["type"] = "s3"
		storage["s3"] = map[string]interface{}{
			"s3":              fmt.Sprintf("s3://%s", lokiSpec.S3.BucketName),
			"bucketNames":     lokiSpec.S3.BucketName,
			"region":          lokiSpec.S3.Region,
			"accessKeyId":     lokiSpec.S3.AccessKeyID,
			"secretAccessKey": lokiSpec.S3.SecretAccessKey,
			"insecure":        false,
			"s3ForcePathStyle": true,
		}
	} else if lokiSpec.Storage != nil {
		// Use filesystem storage with PVC
		storage["filesystem"] = map[string]interface{}{
			"size": lokiSpec.Storage.Size,
		}
		
		if lokiSpec.Storage.StorageClass != "" {
			storage["filesystem"].(map[string]interface{})["storageClass"] = lokiSpec.Storage.StorageClass
		}
	}
	
	values["storage"] = storage
	
	// Configure Loki config
	lokiConfig := map[string]interface{}{
		"auth_enabled": false,
		"server": map[string]interface{}{
			"http_listen_port": 3100,
			"grpc_listen_port": 9095,
			"log_level": "info",
		},
		"common": map[string]interface{}{
			"path_prefix": "/loki",
			"storage": map[string]interface{}{
				"filesystem": map[string]interface{}{
					"chunks_directory": "/loki/chunks",
					"rules_directory":  "/loki/rules",
				},
			},
			"replication_factor": 1,
		},
		"query_range": map[string]interface{}{
			"results_cache": map[string]interface{}{
				"cache": map[string]interface{}{
					"embedded_cache": map[string]interface{}{
						"enabled": true,
						"max_size_mb": 100,
					},
				},
			},
		},
		"schema_config": map[string]interface{}{
			"configs": []interface{}{
				map[string]interface{}{
					"from": "2020-05-15",
					"store": "tsdb",
					"object_store": "filesystem",
					"schema": "v13",
					"index": map[string]interface{}{
						"prefix": "index_",
						"period": "24h",
					},
				},
			},
		},
		"ruler": map[string]interface{}{
			"alertmanager_url": fmt.Sprintf("http://%s-alertmanager.%s.svc.cluster.local:9093", 
				platform.Name, platform.Namespace),
		},
		"analytics": map[string]interface{}{
			"reporting_enabled": false,
		},
	}
	
	// Configure retention
	if lokiSpec.RetentionDays > 0 {
		lokiConfig["limits_config"] = map[string]interface{}{
			"retention_period": fmt.Sprintf("%dh", lokiSpec.RetentionDays*24),
			"retention_stream": []interface{}{
				map[string]interface{}{
					"selector": `{namespace="kube-system"}`,
					"priority": 1,
					"period":   "24h",
				},
			},
		}
		
		lokiConfig["compactor"] = map[string]interface{}{
			"retention_enabled": true,
			"retention_delete_delay": "2h",
			"retention_delete_worker_count": 150,
		}
	}
	
	// Configure S3 in Loki config if enabled
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		lokiConfig["common"].(map[string]interface{})["storage"] = map[string]interface{}{
			"s3": map[string]interface{}{
				"s3":              fmt.Sprintf("s3://%s", lokiSpec.S3.BucketName),
				"bucketnames":     lokiSpec.S3.BucketName,
				"region":          lokiSpec.S3.Region,
				"access_key_id":   lokiSpec.S3.AccessKeyID,
				"secret_access_key": lokiSpec.S3.SecretAccessKey,
			},
		}
		
		// Update schema config for S3
		schemaConfigs := lokiConfig["schema_config"].(map[string]interface{})["configs"].([]interface{})
		if len(schemaConfigs) > 0 {
			schemaConfigs[0].(map[string]interface{})["object_store"] = "s3"
		}
	}
	
	// Merge with custom config if provided
	if lokiSpec.Config != nil {
		// Deep merge custom config
		for k, v := range lokiSpec.Config {
			lokiConfig[k] = v
		}
	}
	
	values["loki"] = lokiConfig
	
	// Configure monitoring
	values["monitoring"] = map[string]interface{}{
		"dashboards": map[string]interface{}{
			"enabled": true,
			"namespace": platform.Namespace,
			"labels": map[string]interface{}{
				"grafana_dashboard": "1",
			},
		},
		"rules": map[string]interface{}{
			"enabled": true,
			"namespace": platform.Namespace,
		},
		"serviceMonitor": map[string]interface{}{
			"enabled": true,
			"labels": map[string]interface{}{
				"observability.io/platform": platform.Name,
			},
		},
	}
	
	// Gateway configuration
	values["gateway"] = map[string]interface{}{
		"enabled": false, // We'll use direct access
	}
	
	// RBAC
	values["rbac"] = map[string]interface{}{
		"pspEnabled": false,
		"sccEnabled": false,
	}
	
	// Service Account
	values["serviceAccount"] = map[string]interface{}{
		"create": true,
		"name":   fmt.Sprintf("%s-loki", platform.Name),
	}
	
	// Test
	values["test"] = map[string]interface{}{
		"enabled": false,
	}
	
	// Node selector, tolerations, and affinity from global settings
	if platform.Spec.Global != nil {
		globalSettings := map[string]interface{}{}
		
		if len(platform.Spec.Global.NodeSelector) > 0 {
			globalSettings["nodeSelector"] = platform.Spec.Global.NodeSelector
		}
		if len(platform.Spec.Global.Tolerations) > 0 {
			globalSettings["tolerations"] = platform.Spec.Global.Tolerations
		}
		if platform.Spec.Global.Affinity != nil {
			globalSettings["affinity"] = platform.Spec.Global.Affinity
		}
		
		// Apply to single binary
		if sb, ok := values["singleBinary"].(map[string]interface{}); ok {
			for k, v := range globalSettings {
				sb[k] = v
			}
		}
	}
	
	// Apply any overrides
	if overrides != nil {
		values = m.ValueBuilder.MergeValues(values, overrides)
	}
	
	return values, nil
}
