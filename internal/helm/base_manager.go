/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm

import (
	"context"
	"fmt"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BaseHelmManager provides common Helm functionality for component managers
type BaseHelmManager struct {
	Client         client.Client
	Scheme         *runtime.Scheme
	RestConfig     *rest.Config
	HelmClient     HelmClient
	ValueBuilder   ValueBuilder
	Repository     Repository
	VersionManager VersionManager
}

// NewBaseHelmManager creates a new base Helm manager
func NewBaseHelmManager(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	restConfig *rest.Config,
) (*BaseHelmManager, error) {
	// Create Helm client
	helmClient, err := NewHelmClient(restConfig, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}
	
	// Create repository manager
	repository, err := NewRepositoryManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create repository manager: %w", err)
	}
	
	// Create version manager
	versionManager := NewVersionManager(repository)
	
	// Create value builder
	valueBuilder := NewValueBuilder()
	
	return &BaseHelmManager{
		Client:         k8sClient,
		Scheme:         scheme,
		RestConfig:     restConfig,
		HelmClient:     helmClient,
		ValueBuilder:   valueBuilder,
		Repository:     repository,
		VersionManager: versionManager,
	}, nil
}

// InstallOrUpgradeComponent installs or upgrades a component using Helm
func (bhm *BaseHelmManager) InstallOrUpgradeComponent(
	ctx context.Context,
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
	chartRef string,
	version string,
	values map[string]interface{},
) error {
	logger := log.FromContext(ctx)
	releaseName := bhm.getReleaseName(platform, componentType)
	namespace := platform.Namespace
	
	// Check if release exists
	existingRelease, err := bhm.HelmClient.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return fmt.Errorf("failed to check existing release: %w", err)
	}
	
	if existingRelease == nil {
		// Install new release
		logger.Info("Installing component", "component", componentType, "release", releaseName)
		
		installOpts := &InstallOptions{
			ReleaseName:     releaseName,
			Namespace:       namespace,
			ChartRef:        chartRef,
			Version:         version,
			Values:          values,
			CreateNamespace: false, // Namespace should already exist
			Wait:            true,
			WaitForJobs:     true,
			Timeout:         300, // 5 minutes
			Atomic:          true,
			Description:     fmt.Sprintf("Installing %s for platform %s", componentType, platform.Name),
		}
		
		if _, err := bhm.HelmClient.InstallChart(ctx, installOpts); err != nil {
			return fmt.Errorf("failed to install %s: %w", componentType, err)
		}
		
		logger.Info("Component installed successfully", "component", componentType)
	} else {
		// Check if upgrade is needed
		needsUpgrade := false
		
		// Check version
		if existingRelease.Chart != nil && existingRelease.Chart.Version != version {
			needsUpgrade = true
			logger.Info("Version change detected", 
				"component", componentType,
				"current", existingRelease.Chart.Version,
				"target", version)
		}
		
		// Check values (simplified check - in production, do deep comparison)
		if !needsUpgrade {
			// TODO: Implement deep value comparison
			needsUpgrade = true // For now, always upgrade to ensure latest config
		}
		
		if needsUpgrade {
			// Check version compatibility
			if existingRelease.Chart != nil {
				compatible, err := bhm.VersionManager.IsVersionCompatible(
					existingRelease.Chart.Version,
					version,
				)
				if err != nil {
					return fmt.Errorf("failed to check version compatibility: %w", err)
				}
				
				if !compatible {
					// Get upgrade path
					upgradePath, err := bhm.VersionManager.GetUpgradePath(
						existingRelease.Chart.Version,
						version,
					)
					if err != nil {
						return fmt.Errorf("failed to get upgrade path: %w", err)
					}
					
					logger.Info("Multi-step upgrade required", 
						"component", componentType,
						"path", upgradePath)
					
					// Perform multi-step upgrade
					for _, stepVersion := range upgradePath {
						if err := bhm.upgradeComponent(ctx, releaseName, namespace, chartRef, stepVersion, values); err != nil {
							return fmt.Errorf("failed to upgrade to version %s: %w", stepVersion, err)
						}
					}
				} else {
					// Direct upgrade
					if err := bhm.upgradeComponent(ctx, releaseName, namespace, chartRef, version, values); err != nil {
						return fmt.Errorf("failed to upgrade component: %w", err)
					}
				}
			}
		} else {
			logger.Info("Component is up to date", "component", componentType)
		}
	}
	
	return nil
}

// upgradeComponent performs a single upgrade step
func (bhm *BaseHelmManager) upgradeComponent(
	ctx context.Context,
	releaseName string,
	namespace string,
	chartRef string,
	version string,
	values map[string]interface{},
) error {
	logger := log.FromContext(ctx)
	logger.Info("Upgrading component", "release", releaseName, "version", version)
	
	upgradeOpts := &UpgradeOptions{
		ReleaseName:   releaseName,
		Namespace:     namespace,
		ChartRef:      chartRef,
		Version:       version,
		Values:        values,
		ReuseValues:   false, // Use new values
		Wait:          true,
		WaitForJobs:   true,
		Timeout:       300, // 5 minutes
		Atomic:        true,
		CleanupOnFail: true,
		MaxHistory:    10,
		Description:   fmt.Sprintf("Upgrading to version %s", version),
	}
	
	if _, err := bhm.HelmClient.UpgradeChart(ctx, upgradeOpts); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}
	
	logger.Info("Component upgraded successfully", "release", releaseName)
	return nil
}

// UninstallComponent uninstalls a component using Helm
func (bhm *BaseHelmManager) UninstallComponent(
	ctx context.Context,
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
) error {
	logger := log.FromContext(ctx)
	releaseName := bhm.getReleaseName(platform, componentType)
	namespace := platform.Namespace
	
	logger.Info("Uninstalling component", "component", componentType, "release", releaseName)
	
	if err := bhm.HelmClient.UninstallChart(ctx, releaseName, namespace); err != nil {
		return fmt.Errorf("failed to uninstall %s: %w", componentType, err)
	}
	
	logger.Info("Component uninstalled successfully", "component", componentType)
	return nil
}

// GetComponentStatus gets the status of a Helm release
func (bhm *BaseHelmManager) GetComponentStatus(
	ctx context.Context,
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
) (*observabilityv1beta1.ComponentStatus, error) {
	releaseName := bhm.getReleaseName(platform, componentType)
	namespace := platform.Namespace
	
	release, err := bhm.HelmClient.GetRelease(ctx, releaseName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get release status: %w", err)
	}
	
	status := &observabilityv1beta1.ComponentStatus{
		Name: componentType,
	}
	
	if release == nil {
		status.Status = "NotInstalled"
		status.Message = "Component is not installed"
		return status, nil
	}
	
	// Map Helm status to component status
	switch release.Status {
	case "deployed":
		status.Status = "Ready"
		status.Message = "Component is running"
		if release.Chart != nil {
			status.Version = release.Chart.Version
		}
	case "failed":
		status.Status = "Failed"
		status.Message = "Component deployment failed"
	case "pending-install", "pending-upgrade":
		status.Status = "Installing"
		status.Message = "Component is being installed/upgraded"
	case "uninstalling":
		status.Status = "Terminating"
		status.Message = "Component is being uninstalled"
	default:
		status.Status = "Unknown"
		status.Message = fmt.Sprintf("Helm status: %s", release.Status)
	}
	
	status.LastUpdated = &release.LastDeployed
	
	return status, nil
}

// RollbackComponent rolls back a component to a previous version
func (bhm *BaseHelmManager) RollbackComponent(
	ctx context.Context,
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
	revision int,
) error {
	logger := log.FromContext(ctx)
	releaseName := bhm.getReleaseName(platform, componentType)
	namespace := platform.Namespace
	
	logger.Info("Rolling back component", "component", componentType, "revision", revision)
	
	if err := bhm.HelmClient.RollbackRelease(ctx, releaseName, namespace, revision); err != nil {
		return fmt.Errorf("failed to rollback %s: %w", componentType, err)
	}
	
	logger.Info("Component rolled back successfully", "component", componentType)
	return nil
}

// TestComponent runs Helm tests for a component
func (bhm *BaseHelmManager) TestComponent(
	ctx context.Context,
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
) error {
	logger := log.FromContext(ctx)
	releaseName := bhm.getReleaseName(platform, componentType)
	namespace := platform.Namespace
	
	logger.Info("Running tests for component", "component", componentType)
	
	if err := bhm.HelmClient.TestRelease(ctx, releaseName, namespace); err != nil {
		return fmt.Errorf("tests failed for %s: %w", componentType, err)
	}
	
	logger.Info("Component tests passed", "component", componentType)
	return nil
}

// GetComponentValues gets the current values for a component
func (bhm *BaseHelmManager) GetComponentValues(
	ctx context.Context,
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
) (map[string]interface{}, error) {
	releaseName := bhm.getReleaseName(platform, componentType)
	namespace := platform.Namespace
	
	values, err := bhm.HelmClient.GetValues(ctx, releaseName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get values for %s: %w", componentType, err)
	}
	
	return values, nil
}

// getReleaseName generates a consistent release name for a component
func (bhm *BaseHelmManager) getReleaseName(
	platform *observabilityv1beta1.ObservabilityPlatform,
	componentType string,
) string {
	return fmt.Sprintf("%s-%s", platform.Name, componentType)
}

// GetChartReference returns the chart reference for a component
func (bhm *BaseHelmManager) GetChartReference(componentType string) string {
	chartRefs := map[string]string{
		"prometheus": "prometheus-community/prometheus",
		"grafana":    "grafana/grafana",
		"loki":       "grafana/loki",
		"tempo":      "grafana/tempo",
	}
	
	if ref, ok := chartRefs[componentType]; ok {
		return ref
	}
	
	return ""
}

// UpdateRepository updates the Helm repository index
func (bhm *BaseHelmManager) UpdateRepository(ctx context.Context, repoName string) error {
	logger := log.FromContext(ctx)
	logger.Info("Updating Helm repository", "repository", repoName)
	
	if err := bhm.Repository.UpdateRepository(ctx, repoName); err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}
	
	logger.Info("Repository updated successfully", "repository", repoName)
	return nil
}

// EnsureRepositories ensures required repositories are added
func (bhm *BaseHelmManager) EnsureRepositories(ctx context.Context) error {
	requiredRepos := map[string]string{
		"prometheus-community": "https://prometheus-community.github.io/helm-charts",
		"grafana":             "https://grafana.github.io/helm-charts",
	}
	
	for name, url := range requiredRepos {
		// Check if repo exists
		repos, err := bhm.Repository.ListRepositories(ctx)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}
		
		exists := false
		for _, repo := range repos {
			if repo.Name == name {
				exists = true
				break
			}
		}
		
		if !exists {
			if err := bhm.Repository.AddRepository(ctx, name, url, nil); err != nil {
				return fmt.Errorf("failed to add repository %s: %w", name, err)
			}
		}
		
		// Update repository index
		if err := bhm.Repository.UpdateRepository(ctx, name); err != nil {
			return fmt.Errorf("failed to update repository %s: %w", name, err)
		}
	}
	
	return nil
}
