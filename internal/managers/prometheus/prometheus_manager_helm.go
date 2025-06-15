/*
Copyright 2025.

Licensed under the MIT License.
*/

package prometheus

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
	// Component name
	componentName = "prometheus"
	
	// Chart information
	chartRepository = "prometheus-community"
	chartName       = "prometheus"
	
	// Default values
	defaultPort = 9090
)

// PrometheusManagerHelm manages Prometheus deployments using Helm
type PrometheusManagerHelm struct {
	*helm.BaseHelmManager
}

// NewPrometheusManagerHelm creates a new Prometheus manager with Helm integration
func NewPrometheusManagerHelm(client client.Client, scheme *runtime.Scheme, restConfig *rest.Config) (managers.PrometheusManager, error) {
	baseManager, err := helm.NewBaseHelmManager(client, scheme, restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base Helm manager: %w", err)
	}
	
	return &PrometheusManagerHelm{
		BaseHelmManager: baseManager,
	}, nil
}

// Reconcile reconciles the Prometheus component
func (m *PrometheusManagerHelm) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Prometheus component with provided configuration
func (m *PrometheusManagerHelm) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	logger := log.FromContext(ctx).WithValues("component", componentName)
	
	// Check if Prometheus is enabled
	if platform.Spec.Components.Prometheus == nil || !platform.Spec.Components.Prometheus.Enabled {
		logger.V(1).Info("Prometheus is disabled, checking if it needs to be uninstalled")
		
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
	
	prometheusSpec := platform.Spec.Components.Prometheus
	logger.Info("Reconciling Prometheus", "version", prometheusSpec.Version)
	
	// Ensure repositories are configured
	if err := m.BaseHelmManager.EnsureRepositories(ctx); err != nil {
		return fmt.Errorf("failed to ensure repositories: %w", err)
	}
	
	// Build Helm values
	values, err := m.buildHelmValues(platform, prometheusSpec, config)
	if err != nil {
		return fmt.Errorf("failed to build Helm values: %w", err)
	}
	
	// Validate values
	if err := m.ValueBuilder.ValidateValues(componentName, values); err != nil {
		return fmt.Errorf("invalid Helm values: %w", err)
	}
	
	// Get chart reference
	chartRef := fmt.Sprintf("%s/%s", chartRepository, chartName)
	
	// Determine version to use
	version := prometheusSpec.Version
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
		componentName,
		chartRef,
		version,
		values,
	); err != nil {
		return fmt.Errorf("failed to install/upgrade Prometheus: %w", err)
	}
	
	logger.Info("Prometheus reconciliation completed successfully")
	return nil
}

// Delete removes the Prometheus component resources
func (m *PrometheusManagerHelm) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentName)
	logger.Info("Deleting Prometheus resources")
	
	if err := m.BaseHelmManager.UninstallComponent(ctx, platform, componentName); err != nil {
		return fmt.Errorf("failed to uninstall Prometheus: %w", err)
	}
	
	return nil
}

// GetStatus returns the current status of the Prometheus component
func (m *PrometheusManagerHelm) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	// Check if Prometheus is enabled
	if platform.Spec.Components.Prometheus == nil || !platform.Spec.Components.Prometheus.Enabled {
		return &observabilityv1beta1.ComponentStatus{
			Name:    componentName,
			Status:  "Disabled",
			Message: "Prometheus is disabled",
		}, nil
	}
	
	return m.BaseHelmManager.GetComponentStatus(ctx, platform, componentName)
}

// Validate validates the Prometheus configuration
func (m *PrometheusManagerHelm) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Prometheus == nil {
		return nil
	}
	
	// Build values to validate
	values, err := m.buildHelmValues(platform, platform.Spec.Components.Prometheus, nil)
	if err != nil {
		return err
	}
	
	// Validate using value builder
	return m.ValueBuilder.ValidateValues(componentName, values)
}

// GetServiceURL returns the service URL for Prometheus
func (m *PrometheusManagerHelm) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	releaseName := fmt.Sprintf("%s-%s", platform.Name, componentName)
	return fmt.Sprintf("http://%s-server.%s.svc.cluster.local:%d", 
		releaseName,
		platform.Namespace, 
		defaultPort)
}

// UpdateConfiguration updates Prometheus configuration without restart
func (m *PrometheusManagerHelm) UpdateConfiguration(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentName)
	logger.Info("Updating Prometheus configuration")
	
	// This will trigger a Helm upgrade with new values
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ScaleReplicas scales Prometheus replicas
func (m *PrometheusManagerHelm) ScaleReplicas(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, replicas int32) error {
	logger := log.FromContext(ctx).WithValues("component", componentName)
	logger.Info("Scaling Prometheus replicas", "replicas", replicas)
	
	// Update the replica count in the spec
	if platform.Spec.Components.Prometheus != nil {
		platform.Spec.Components.Prometheus.Replicas = &replicas
	}
	
	// Trigger reconciliation with updated replicas
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// buildHelmValues builds Helm values for Prometheus
func (m *PrometheusManagerHelm) buildHelmValues(
	platform *observabilityv1beta1.ObservabilityPlatform,
	prometheusSpec *observabilityv1beta1.PrometheusSpec,
	overrides map[string]interface{},
) (map[string]interface{}, error) {
	// Get base values from value builder
	values, err := m.ValueBuilder.BuildValues(componentName, prometheusSpec)
	if err != nil {
		return nil, err
	}
	
	// Add platform-specific configurations
	
	// Server configuration
	server := values["server"].(map[string]interface{})
	
	// Set replicas
	if prometheusSpec.Replicas != nil {
		server["replicaCount"] = *prometheusSpec.Replicas
	}
	
	// Set retention
	if prometheusSpec.Retention != "" {
		server["retention"] = prometheusSpec.Retention
	}
	
	// Global configuration
	global := map[string]interface{}{
		"scrape_interval":     "15s",
		"evaluation_interval": "15s",
	}
	
	// External labels
	externalLabels := make(map[string]interface{})
	
	// Add global external labels
	if platform.Spec.Global != nil {
		for k, v := range platform.Spec.Global.ExternalLabels {
			externalLabels[k] = v
		}
	}
	
	// Add Prometheus-specific external labels (override global)
	if prometheusSpec.ExternalLabels != nil {
		for k, v := range prometheusSpec.ExternalLabels {
			externalLabels[k] = v
		}
	}
	
	if len(externalLabels) > 0 {
		global["external_labels"] = externalLabels
	}
	
	server["global"] = global
	
	// Remote write configuration
	if len(prometheusSpec.RemoteWrite) > 0 {
		remoteWrite := make([]interface{}, 0, len(prometheusSpec.RemoteWrite))
		for _, rw := range prometheusSpec.RemoteWrite {
			rwConfig := map[string]interface{}{
				"url": rw.URL,
			}
			if rw.RemoteTimeout != "" {
				rwConfig["remote_timeout"] = rw.RemoteTimeout
			}
			if len(rw.Headers) > 0 {
				rwConfig["headers"] = rw.Headers
			}
			remoteWrite = append(remoteWrite, rwConfig)
		}
		server["remoteWrite"] = remoteWrite
	}
	
	// Storage configuration
	if prometheusSpec.Storage != nil {
		persistence := server["persistentVolume"].(map[string]interface{})
		if prometheusSpec.Storage.Size != "" {
			persistence["size"] = prometheusSpec.Storage.Size
		}
		if prometheusSpec.Storage.StorageClass != "" {
			persistence["storageClass"] = prometheusSpec.Storage.StorageClass
		}
	}
	
	// Resources
	if prometheusSpec.Resources != nil {
		server["resources"] = prometheusSpec.Resources
	}
	
	// Service monitor for self-monitoring
	server["serviceMonitor"] = map[string]interface{}{
		"enabled": true,
		"labels": map[string]interface{}{
			"observability.io/platform": platform.Name,
		},
	}
	
	// Pod Security Policy
	values["podSecurityPolicy"] = map[string]interface{}{
		"enabled": false, // Deprecated in newer K8s versions
	}
	
	// RBAC
	values["rbac"] = map[string]interface{}{
		"create": true,
	}
	
	// Service Account
	values["serviceAccounts"] = map[string]interface{}{
		"server": map[string]interface{}{
			"create": true,
			"name":   fmt.Sprintf("%s-prometheus", platform.Name),
		},
	}
	
	// Node selector, tolerations, and affinity from global settings
	if platform.Spec.Global != nil {
		if len(platform.Spec.Global.NodeSelector) > 0 {
			server["nodeSelector"] = platform.Spec.Global.NodeSelector
		}
		if len(platform.Spec.Global.Tolerations) > 0 {
			server["tolerations"] = platform.Spec.Global.Tolerations
		}
		if platform.Spec.Global.Affinity != nil {
			server["affinity"] = platform.Spec.Global.Affinity
		}
	}
	
	// Additional scrape configs
	if prometheusSpec.AdditionalScrapeConfigs != "" {
		server["extraScrapeConfigs"] = prometheusSpec.AdditionalScrapeConfigs
	}
	
	// Custom config override
	if prometheusSpec.Config != nil {
		// Deep merge custom config
		if serverConfig, ok := prometheusSpec.Config["server"].(map[string]interface{}); ok {
			for k, v := range serverConfig {
				server[k] = v
			}
		}
		
		// Apply other top-level custom configs
		for k, v := range prometheusSpec.Config {
			if k != "server" {
				values[k] = v
			}
		}
	}
	
	// Disable components we're managing separately
	values["alertmanager"] = map[string]interface{}{
		"enabled": false,
	}
	values["pushgateway"] = map[string]interface{}{
		"enabled": false,
	}
	values["nodeExporter"] = map[string]interface{}{
		"enabled": false,
	}
	
	// Apply any overrides
	if overrides != nil {
		values = m.ValueBuilder.MergeValues(values, overrides)
	}
	
	return values, nil
}
