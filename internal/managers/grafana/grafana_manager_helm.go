/*
Copyright 2025.

Licensed under the MIT License.
*/

package grafana

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
	componentNameHelm = "grafana"
	
	// Chart information
	chartRepository = "grafana"
	chartName       = "grafana"
	
	// Default values
	defaultPortHelm = 3000
)

// GrafanaManagerHelm manages Grafana deployments using Helm
type GrafanaManagerHelm struct {
	*helm.BaseHelmManager
}

// NewGrafanaManagerHelm creates a new Grafana manager with Helm integration
func NewGrafanaManagerHelm(client client.Client, scheme *runtime.Scheme, restConfig *rest.Config) (managers.GrafanaManager, error) {
	baseManager, err := helm.NewBaseHelmManager(client, scheme, restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base Helm manager: %w", err)
	}
	
	return &GrafanaManagerHelm{
		BaseHelmManager: baseManager,
	}, nil
}

// Reconcile reconciles the Grafana component
func (m *GrafanaManagerHelm) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Grafana component with provided configuration
func (m *GrafanaManagerHelm) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	
	// Check if Grafana is enabled
	if platform.Spec.Components.Grafana == nil || !platform.Spec.Components.Grafana.Enabled {
		logger.V(1).Info("Grafana is disabled, checking if it needs to be uninstalled")
		
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
	
	grafanaSpec := platform.Spec.Components.Grafana
	logger.Info("Reconciling Grafana", "version", grafanaSpec.Version)
	
	// Ensure repositories are configured
	if err := m.BaseHelmManager.EnsureRepositories(ctx); err != nil {
		return fmt.Errorf("failed to ensure repositories: %w", err)
	}
	
	// Build Helm values
	values, err := m.buildHelmValues(platform, grafanaSpec, config)
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
	version := grafanaSpec.Version
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
		return fmt.Errorf("failed to install/upgrade Grafana: %w", err)
	}
	
	// After installation, configure data sources if needed
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		// Data sources will be configured via Helm values
		logger.V(1).Info("Data sources configured via Helm values")
	}
	
	logger.Info("Grafana reconciliation completed successfully")
	return nil
}

// Delete removes the Grafana component resources
func (m *GrafanaManagerHelm) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Deleting Grafana resources")
	
	if err := m.BaseHelmManager.UninstallComponent(ctx, platform, componentNameHelm); err != nil {
		return fmt.Errorf("failed to uninstall Grafana: %w", err)
	}
	
	return nil
}

// GetStatus returns the current status of the Grafana component
func (m *GrafanaManagerHelm) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	// Check if Grafana is enabled
	if platform.Spec.Components.Grafana == nil || !platform.Spec.Components.Grafana.Enabled {
		return &observabilityv1beta1.ComponentStatus{
			Name:    componentNameHelm,
			Status:  "Disabled",
			Message: "Grafana is disabled",
		}, nil
	}
	
	return m.BaseHelmManager.GetComponentStatus(ctx, platform, componentNameHelm)
}

// Validate validates the Grafana configuration
func (m *GrafanaManagerHelm) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Grafana == nil {
		return nil
	}
	
	// Build values to validate
	values, err := m.buildHelmValues(platform, platform.Spec.Components.Grafana, nil)
	if err != nil {
		return err
	}
	
	// Validate using value builder
	return m.ValueBuilder.ValidateValues(componentNameHelm, values)
}

// GetServiceURL returns the service URL for Grafana
func (m *GrafanaManagerHelm) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	releaseName := fmt.Sprintf("%s-%s", platform.Name, componentNameHelm)
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", 
		releaseName,
		platform.Namespace, 
		defaultPortHelm)
}

// ImportDashboards imports dashboards into Grafana
func (m *GrafanaManagerHelm) ImportDashboards(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Importing dashboards")
	
	// With Helm, dashboards are imported via ConfigMaps with specific labels
	// The sidecar container in Grafana will automatically import them
	// This is configured in the Helm values
	
	logger.Info("Dashboards will be imported via sidecar container")
	return nil
}

// ConfigureDataSources configures data sources in Grafana
func (m *GrafanaManagerHelm) ConfigureDataSources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	logger := log.FromContext(ctx).WithValues("component", componentNameHelm)
	logger.Info("Configuring data sources")
	
	// Data sources are configured via Helm values
	// Trigger a reconciliation to update them
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// buildHelmValues builds Helm values for Grafana
func (m *GrafanaManagerHelm) buildHelmValues(
	platform *observabilityv1beta1.ObservabilityPlatform,
	grafanaSpec *observabilityv1beta1.GrafanaSpec,
	overrides map[string]interface{},
) (map[string]interface{}, error) {
	// Get base values from value builder
	values, err := m.ValueBuilder.BuildValues(componentNameHelm, grafanaSpec)
	if err != nil {
		return nil, err
	}
	
	// Set replicas
	if grafanaSpec.Replicas != nil {
		values["replicas"] = *grafanaSpec.Replicas
	}
	
	// Set admin password
	if grafanaSpec.AdminPassword != "" {
		values["adminPassword"] = grafanaSpec.AdminPassword
	} else {
		// Generate a random password if not specified
		values["adminPassword"] = "admin" // In production, generate a secure password
	}
	
	// Configure ingress
	if grafanaSpec.Ingress != nil && grafanaSpec.Ingress.Enabled {
		ingress := map[string]interface{}{
			"enabled": true,
			"hosts":   []string{grafanaSpec.Ingress.Host},
		}
		
		if grafanaSpec.Ingress.TLSSecret != "" {
			ingress["tls"] = []map[string]interface{}{
				{
					"secretName": grafanaSpec.Ingress.TLSSecret,
					"hosts":      []string{grafanaSpec.Ingress.Host},
				},
			}
		}
		
		values["ingress"] = ingress
	}
	
	// Configure persistence
	if grafanaSpec.Storage != nil {
		persistence := map[string]interface{}{
			"enabled": true,
			"size":    grafanaSpec.Storage.Size,
		}
		
		if grafanaSpec.Storage.StorageClass != "" {
			persistence["storageClassName"] = grafanaSpec.Storage.StorageClass
		}
		
		values["persistence"] = persistence
	}
	
	// Configure resources
	if grafanaSpec.Resources != nil {
		values["resources"] = grafanaSpec.Resources
	}
	
	// Configure data sources
	datasources := map[string]interface{}{
		"datasources.yaml": map[string]interface{}{
			"apiVersion":  1,
			"datasources": m.buildDataSources(platform),
		},
	}
	values["datasources"] = datasources
	
	// Configure dashboards
	if grafanaSpec.Dashboards != nil {
		values["dashboards"] = grafanaSpec.Dashboards
	}
	
	// Configure sidecar for dynamic dashboard loading
	sidecar := map[string]interface{}{
		"dashboards": map[string]interface{}{
			"enabled": true,
			"label":   "grafana_dashboard",
			"folder":  "/tmp/dashboards",
			"provider": map[string]interface{}{
				"name":         "sidecarProvider",
				"orgid":        1,
				"folder":       "",
				"type":         "file",
				"disableDelete": false,
				"allowUiUpdates": false,
			},
		},
		"datasources": map[string]interface{}{
			"enabled": true,
			"label":   "grafana_datasource",
		},
	}
	values["sidecar"] = sidecar
	
	// RBAC
	values["rbac"] = map[string]interface{}{
		"create": true,
	}
	
	// Service Account
	values["serviceAccount"] = map[string]interface{}{
		"create": true,
		"name":   fmt.Sprintf("%s-grafana", platform.Name),
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
	
	// Custom grafana.ini configuration
	if grafanaSpec.Config != nil {
		values["grafana.ini"] = grafanaSpec.Config
	}
	
	// Apply any overrides
	if overrides != nil {
		values = m.ValueBuilder.MergeValues(values, overrides)
	}
	
	return values, nil
}

// buildDataSources builds the data sources configuration
func (m *GrafanaManagerHelm) buildDataSources(platform *observabilityv1beta1.ObservabilityPlatform) []interface{} {
	datasources := []interface{}{}
	
	// Add Prometheus data source if enabled
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		prometheusURL := fmt.Sprintf("http://%s-prometheus-server.%s.svc.cluster.local:9090",
			platform.Name,
			platform.Namespace)
		
		datasources = append(datasources, map[string]interface{}{
			"name":      "Prometheus",
			"type":      "prometheus",
			"url":       prometheusURL,
			"access":    "proxy",
			"isDefault": true,
			"jsonData": map[string]interface{}{
				"timeInterval": "30s",
			},
		})
	}
	
	// Add Loki data source if enabled
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		lokiURL := fmt.Sprintf("http://%s-loki.%s.svc.cluster.local:3100",
			platform.Name,
			platform.Namespace)
		
		datasources = append(datasources, map[string]interface{}{
			"name":   "Loki",
			"type":   "loki",
			"url":    lokiURL,
			"access": "proxy",
			"jsonData": map[string]interface{}{
				"maxLines": 1000,
			},
		})
	}
	
	// Add Tempo data source if enabled
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		tempoURL := fmt.Sprintf("http://%s-tempo.%s.svc.cluster.local:3200",
			platform.Name,
			platform.Namespace)
		
		tempoDS := map[string]interface{}{
			"name":   "Tempo",
			"type":   "tempo",
			"url":    tempoURL,
			"access": "proxy",
			"jsonData": map[string]interface{}{
				"nodeGraph": map[string]interface{}{
					"enabled": true,
				},
				"search": map[string]interface{}{
					"hide": false,
				},
			},
		}
		
		// Configure trace to logs if Loki is enabled
		if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
			tempoDS["jsonData"].(map[string]interface{})["tracesToLogs"] = map[string]interface{}{
				"datasourceUid": "loki",
				"tags":          []string{"job", "instance", "pod", "namespace"},
				"mappedTags": []map[string]string{
					{"key": "service.name", "value": "service"},
				},
				"mapTagNamesEnabled": true,
				"spanStartTimeShift": "1h",
				"spanEndTimeShift":   "1h",
				"filterByTraceID":    true,
				"filterBySpanID":     true,
			}
		}
		
		// Configure service map if Prometheus is enabled
		if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
			tempoDS["jsonData"].(map[string]interface{})["serviceMap"] = map[string]interface{}{
				"datasourceUid": "prometheus",
			}
		}
		
		datasources = append(datasources, tempoDS)
	}
	
	return datasources
}
