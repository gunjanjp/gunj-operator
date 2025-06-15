/*
Copyright 2025.

Licensed under the MIT License.
*/

package tempo

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
	componentNameHelm = "tempo"
	
	// Chart information
	chartRepository = "grafana"
	chartName       = "tempo"
	
	// Default values
	defaultHTTPPortHelm = 3200
	defaultGRPCPortHelm = 9095
)

// TempoManagerHelm manages Tempo deployments using Helm
type TempoManagerHelm struct {
	*helm.BaseHelmManager
}

// NewTempoManagerHelm creates a new Tempo manager with Helm integration
func NewTempoManagerHelm(client client.Client, scheme *runtime.Scheme, restConfig *rest.Config) (managers.TempoManager, error) {
	baseManager, err := helm.NewBaseHelmManager(client, scheme, restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base Helm manager: %w", err)
	}
	
	return &TempoManagerHelm{
		BaseHelmManager: baseManager,
	}, nil
}

// Reconcile reconciles the Tempo component
func (m *TempoManagerHelm) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Tempo component with provided configuration
func (m *TempoManagerHelm) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	
	// Check if Tempo is enabled
	if platform.Spec.Components.Tempo == nil || !platform.Spec.Components.Tempo.Enabled {
		logger.V(1).Info("Tempo is disabled, checking if it needs to be uninstalled")
		
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
	
	tempoSpec := platform.Spec.Components.Tempo
	logger.Info("Reconciling Tempo", "version", tempoSpec.Version)
	
	// Ensure repositories are configured
	if err := m.BaseHelmManager.EnsureRepositories(ctx); err != nil {
		return fmt.Errorf("failed to ensure repositories: %w", err)
	}
	
	// Build Helm values
	values, err := m.buildHelmValues(platform, tempoSpec, config)
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
	version := tempoSpec.Version
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
		return fmt.Errorf("failed to install/upgrade Tempo: %w", err)
	}
	
	logger.Info("Tempo reconciliation completed successfully")
	return nil
}

// Delete removes the Tempo component resources
func (m *TempoManagerHelm) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Deleting Tempo resources")
	
	if err := m.BaseHelmManager.UninstallComponent(ctx, platform, componentNameHelm); err != nil {
		return fmt.Errorf("failed to uninstall Tempo: %w", err)
	}
	
	return nil
}

// GetStatus returns the current status of the Tempo component
func (m *TempoManagerHelm) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	// Check if Tempo is enabled
	if platform.Spec.Components.Tempo == nil || !platform.Spec.Components.Tempo.Enabled {
		return &observabilityv1beta1.ComponentStatus{
			Name:    componentNameHelm,
			Status:  "Disabled",
			Message: "Tempo is disabled",
		}, nil
	}
	
	return m.BaseHelmManager.GetComponentStatus(ctx, platform, componentNameHelm)
}

// Validate validates the Tempo configuration
func (m *TempoManagerHelm) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Tempo == nil {
		return nil
	}
	
	// Build values to validate
	values, err := m.buildHelmValues(platform, platform.Spec.Components.Tempo, nil)
	if err != nil {
		return err
	}
	
	// Validate using value builder
	return m.ValueBuilder.ValidateValues(componentNameHelm, values)
}

// GetServiceURL returns the service URL for Tempo
func (m *TempoManagerHelm) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	releaseName := fmt.Sprintf("%s-%s", platform.Name, componentNameHelm)
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", 
		releaseName,
		platform.Namespace, 
		defaultHTTPPortHelm)
}

// ConfigureReceivers configures trace receivers
func (m *TempoManagerHelm) ConfigureReceivers(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Configuring receivers")
	
	// Receivers are configured via Helm values
	// Trigger a reconciliation to update receiver configuration
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// UpdateSampling updates sampling configuration
func (m *TempoManagerHelm) UpdateSampling(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Updating sampling configuration")
	
	// Sampling is configured via Helm values
	// Trigger a reconciliation to update sampling
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// buildHelmValues builds Helm values for Tempo
func (m *TempoManagerHelm) buildHelmValues(
	platform *observabilityv1beta1.ObservabilityPlatform,
	tempoSpec *observabilityv1beta1.TempoSpec,
	overrides map[string]interface{},
) (map[string]interface{}, error) {
	// Get base values from value builder
	values, err := m.ValueBuilder.BuildValues(componentNameHelm, tempoSpec)
	if err != nil {
		return nil, err
	}
	
	// Configure Tempo
	tempo := map[string]interface{}{
		"repository": "grafana/tempo",
		"tag":        tempoSpec.Version,
		"pullPolicy": "IfNotPresent",
		"resources":  tempoSpec.Resources,
	}
	
	// Update replicas
	if tempoSpec.Replicas != nil {
		tempo["replicas"] = *tempoSpec.Replicas
	} else {
		tempo["replicas"] = 1
	}
	
	// Configure storage
	storage := map[string]interface{}{
		"trace": map[string]interface{}{
			"backend": "local",
			"local": map[string]interface{}{
				"path": "/var/tempo/traces",
			},
			"wal": map[string]interface{}{
				"path": "/var/tempo/wal",
			},
		},
	}
	
	// Configure persistence
	persistence := map[string]interface{}{
		"enabled": true,
		"size":    "10Gi",
	}
	
	if tempoSpec.Storage != nil {
		if tempoSpec.Storage.Size != "" {
			persistence["size"] = tempoSpec.Storage.Size
		}
		if tempoSpec.Storage.StorageClass != "" {
			persistence["storageClass"] = tempoSpec.Storage.StorageClass
		}
	}
	
	// Configure receivers
	receivers := map[string]interface{}{
		"jaeger": map[string]interface{}{
			"protocols": map[string]interface{}{
				"thrift_http": map[string]interface{}{
					"endpoint": "0.0.0.0:14268",
				},
				"grpc": map[string]interface{}{
					"endpoint": "0.0.0.0:14250",
				},
				"thrift_binary": map[string]interface{}{
					"endpoint": "0.0.0.0:6832",
				},
				"thrift_compact": map[string]interface{}{
					"endpoint": "0.0.0.0:6831",
				},
			},
		},
		"zipkin": map[string]interface{}{
			"endpoint": "0.0.0.0:9411",
		},
		"otlp": map[string]interface{}{
			"protocols": map[string]interface{}{
				"grpc": map[string]interface{}{
					"endpoint": "0.0.0.0:4317",
				},
				"http": map[string]interface{}{
					"endpoint": "0.0.0.0:4318",
				},
			},
		},
		"opencensus": map[string]interface{}{
			"endpoint": "0.0.0.0:55678",
		},
	}
	
	// Override with custom receivers if provided
	if tempoSpec.Receivers != nil {
		for k, v := range tempoSpec.Receivers {
			receivers[k] = v
		}
	}
	
	// Configure server
	server := map[string]interface{}{
		"http_listen_port": defaultHTTPPortHelm,
		"grpc_listen_port": defaultGRPCPortHelm,
		"log_level":        "info",
	}
	
	// Configure distributor
	distributor := map[string]interface{}{
		"receivers": receivers,
	}
	
	// Configure ingester
	ingester := map[string]interface{}{
		"max_block_duration": "5m",
	}
	
	// Configure compactor
	compactor := map[string]interface{}{
		"compaction": map[string]interface{}{
			"block_retention": fmt.Sprintf("%dd", tempoSpec.RetentionDays),
		},
	}
	
	// Configure querier
	querier := map[string]interface{}{
		"frontend_worker": map[string]interface{}{
			"frontend_address": fmt.Sprintf("%s-tempo-query-frontend:%d", platform.Name, defaultGRPCPortHelm),
		},
	}
	
	// Configure metrics generator
	metricsGenerator := map[string]interface{}{
		"registry": map[string]interface{}{
			"external_labels": map[string]interface{}{
				"source": "tempo",
				"cluster": platform.Name,
			},
		},
		"storage": map[string]interface{}{
			"path": "/var/tempo/generator/wal",
			"remote_write": []interface{}{
				map[string]interface{}{
					"url": fmt.Sprintf("http://%s-prometheus-server.%s.svc.cluster.local:9090/api/v1/write",
						platform.Name, platform.Namespace),
				},
			},
		},
	}
	
	// Build config
	config := map[string]interface{}{
		"auth_enabled":      false,
		"server":            server,
		"distributor":       distributor,
		"ingester":          ingester,
		"compactor":         compactor,
		"querier":           querier,
		"storage":           storage,
		"metrics_generator": metricsGenerator,
		"overrides": map[string]interface{}{
			"max_traces_per_user": 10000,
			"max_search_duration": "48h",
		},
	}
	
	// Merge with custom config if provided
	if tempoSpec.Config != nil {
		// Deep merge custom config
		for k, v := range tempoSpec.Config {
			config[k] = v
		}
	}
	
	// Set values
	values["tempo"] = tempo
	values["tempoConfig"] = config
	values["persistence"] = persistence
	
	// Service configuration
	values["service"] = map[string]interface{}{
		"type": "ClusterIP",
	}
	
	// Service monitor
	values["serviceMonitor"] = map[string]interface{}{
		"enabled": true,
		"labels": map[string]interface{}{
			"observability.io/platform": platform.Name,
		},
	}
	
	// Prometheus rule
	values["prometheusRule"] = map[string]interface{}{
		"enabled": true,
		"labels": map[string]interface{}{
			"observability.io/platform": platform.Name,
		},
	}
	
	// Gateway configuration (disabled, we expose services directly)
	values["gateway"] = map[string]interface{}{
		"enabled": false,
	}
	
	// Enterprise features (disabled)
	values["enterprise"] = map[string]interface{}{
		"enabled": false,
	}
	
	// Multi-tenancy (disabled for simplicity)
	values["multitenancy"] = map[string]interface{}{
		"enabled": false,
	}
	
	// Minio (disabled, using local storage or external S3)
	values["minio"] = map[string]interface{}{
		"enabled": false,
	}
	
	// Service Account
	values["serviceAccount"] = map[string]interface{}{
		"create": true,
		"name":   fmt.Sprintf("%s-tempo", platform.Name),
	}
	
	// Security context
	values["securityContext"] = map[string]interface{}{
		"runAsNonRoot": true,
		"runAsUser":    10001,
		"runAsGroup":   10001,
		"fsGroup":      10001,
	}
	
	// Node selector, tolerations, and affinity from global settings
	if platform.Spec.Global != nil {
		if len(platform.Spec.Global.NodeSelector) > 0 {
			values["nodeSelector"] = platform.Spec.Global.NodeSelector
		}
		if len(platform.Spec.Global.Tolerations) > 0 {
			values["tolerations"] = platform.Spec.Global.Tolerations
		}
		if platform.Spec.Global.Affinity != nil {
			values["affinity"] = platform.Spec.Global.Affinity
		}
	}
	
	// Apply any overrides
	if overrides != nil {
		values = m.ValueBuilder.MergeValues(values, overrides)
	}
	
	return values, nil
}
