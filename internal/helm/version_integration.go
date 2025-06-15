// Package helm provides version management integration
package helm

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/version"
)

// VersionAwareManager extends base manager with version management capabilities
type VersionAwareManager struct {
	BaseManager
	versionManager *version.Manager
	log            logr.Logger
}

// NewVersionAwareManager creates a new version-aware manager
func NewVersionAwareManager(
	helmClient Interface,
	versionManager *version.Manager,
	componentName string,
	chartName string,
	repository string,
) *VersionAwareManager {
	return &VersionAwareManager{
		BaseManager: BaseManager{
			helmClient:    helmClient,
			componentName: componentName,
			chartName:     chartName,
			repository:    repository,
		},
		versionManager: versionManager,
		log:            log.Log.WithName(fmt.Sprintf("%s-version-manager", componentName)),
	}
}

// GetEffectiveVersion determines the effective version to use
func (m *VersionAwareManager) GetEffectiveVersion(ctx context.Context, spec interface{}) (string, error) {
	// Extract requested version from spec
	requestedVersion := m.extractVersion(spec)
	
	// Register chart if not already registered
	if err := m.versionManager.RegisterChart(ctx, m.chartName, m.repository); err != nil {
		m.log.Error(err, "Failed to register chart", "chart", m.chartName)
	}

	// Apply version pinning and overrides
	effectiveVersion, decision := m.versionManager.GetPinning().ApplyToVersion(m.componentName, requestedVersion)
	
	if len(decision.Applied) > 0 {
		m.log.Info("Version decision applied",
			"component", m.componentName,
			"requested", requestedVersion,
			"effective", effectiveVersion,
			"applied", decision.Applied,
		)
	}

	// Validate the effective version
	if err := m.validateVersion(ctx, effectiveVersion); err != nil {
		return "", fmt.Errorf("version validation failed: %w", err)
	}

	return effectiveVersion, nil
}

// ValidateUpgrade validates a proposed upgrade
func (m *VersionAwareManager) ValidateUpgrade(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetVersion string) error {
	platformRef := version.PlatformReference{
		Name:      platform.Name,
		Namespace: platform.Namespace,
		UID:       string(platform.UID),
	}

	validation, err := m.versionManager.ValidateUpgrade(ctx, platformRef, m.componentName, targetVersion)
	if err != nil {
		return fmt.Errorf("upgrade validation failed: %w", err)
	}

	if !validation.Valid {
		var failedChecks []string
		for _, check := range validation.Checks {
			if !check.Passed {
				failedChecks = append(failedChecks, fmt.Sprintf("%s: %s", check.Name, check.Message))
			}
		}
		return fmt.Errorf("upgrade validation failed: %v", failedChecks)
	}

	// Log warnings if any
	for _, warning := range validation.Warnings {
		m.log.Info("Upgrade warning", "component", m.componentName, "warning", warning)
	}

	return nil
}

// RecordVersionChange records a version change in history
func (m *VersionAwareManager) RecordVersionChange(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, action string, fromVersion, toVersion string) {
	if m.versionManager.GetHistory() == nil {
		return
	}

	entry := version.VersionHistoryEntry{
		Platform: version.PlatformReference{
			Name:      platform.Name,
			Namespace: platform.Namespace,
			UID:       string(platform.UID),
		},
		Component:   m.componentName,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Action:      version.VersionAction(action),
		Status:      version.VersionStatusInProgress,
	}

	if err := m.versionManager.GetHistory().RecordChange(ctx, entry); err != nil {
		m.log.Error(err, "Failed to record version change")
	}
}

// CheckForUpdates checks if updates are available
func (m *VersionAwareManager) CheckForUpdates(ctx context.Context) (*version.ChartVersionInfo, error) {
	return m.versionManager.GetRegistry().GetChartInfo(m.chartName, m.repository)
}

// GetRecommendedVersion gets the recommended version
func (m *VersionAwareManager) GetRecommendedVersion(ctx context.Context, constraints map[string]string) (string, error) {
	return m.versionManager.GetRecommendedVersion(ctx, m.componentName, constraints)
}

// Private helper methods

func (m *VersionAwareManager) extractVersion(spec interface{}) string {
	switch s := spec.(type) {
	case *observabilityv1.PrometheusSpec:
		return s.Version
	case *observabilityv1.GrafanaSpec:
		return s.Version
	case *observabilityv1.LokiSpec:
		return s.Version
	case *observabilityv1.TempoSpec:
		return s.Version
	default:
		return ""
	}
}

func (m *VersionAwareManager) validateVersion(ctx context.Context, version string) error {
	// Get available versions
	chartInfo, err := m.versionManager.GetRegistry().GetChartInfo(m.chartName, m.repository)
	if err != nil {
		return fmt.Errorf("failed to get chart info: %w", err)
	}

	// Check if version exists
	versionExists := false
	for _, v := range chartInfo.AvailableVersions {
		if v == version {
			versionExists = true
			break
		}
	}

	if !versionExists {
		return fmt.Errorf("version %s not found in available versions", version)
	}

	return nil
}

// VersionManager getter for access to version manager
func (m *VersionAwareManager) GetVersionManager() *version.Manager {
	return m.versionManager
}
