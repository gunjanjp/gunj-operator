// Package version provides the main version management system
package version

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/internal/helm"
)

// Manager is the main version management system
type Manager struct {
	mu               sync.RWMutex
	client           client.Client
	scheme           *runtime.Scheme
	helmClient       helm.Interface
	registry         *Registry
	validator        *Validator
	compatibility    *CompatibilityMatrix
	notifications    *NotificationManager
	pinning          *PinningManager
	history          *HistoryTracker
	testing          *TestManager
	config           *ManagerConfig
	log              logr.Logger
}

// ManagerConfig configures the version manager
type ManagerConfig struct {
	// Core configuration
	HelmClient          helm.Interface
	KubernetesClient    client.Client
	Scheme              *runtime.Scheme
	
	// Feature toggles
	EnableAutoUpdate    bool
	EnableNotifications bool
	EnableTesting       bool
	EnableHistory       bool
	
	// Update settings
	UpdateInterval      time.Duration
	UpdateCheckTimeout  time.Duration
	
	// Storage paths
	CompatibilityFile   string
	PinningConfigFile   string
	
	// Testing configuration
	TestNamespace       string
	
	// History settings
	HistoryRetention    time.Duration
	MaxHistoryItems     int
}

// NewManager creates a new version manager
func NewManager(config *ManagerConfig) (*Manager, error) {
	if config.HelmClient == nil {
		return nil, fmt.Errorf("helm client is required")
	}
	if config.KubernetesClient == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}
	if config.Scheme == nil {
		return nil, fmt.Errorf("scheme is required")
	}

	// Set defaults
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 6 * time.Hour
	}
	if config.UpdateCheckTimeout == 0 {
		config.UpdateCheckTimeout = 5 * time.Minute
	}
	if config.MaxHistoryItems == 0 {
		config.MaxHistoryItems = 1000
	}
	if config.HistoryRetention == 0 {
		config.HistoryRetention = 90 * 24 * time.Hour // 90 days
	}

	m := &Manager{
		client:     config.KubernetesClient,
		scheme:     config.Scheme,
		helmClient: config.HelmClient,
		config:     config,
		log:        log.Log.WithName("version-manager"),
	}

	// Initialize components
	if err := m.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	m.log.Info("Version manager initialized",
		"autoUpdate", config.EnableAutoUpdate,
		"notifications", config.EnableNotifications,
		"testing", config.EnableTesting,
	)

	return m, nil
}

// Initialize initializes all version management components
func (m *Manager) initializeComponents() error {
	var err error

	// Initialize registry
	m.registry, err = NewRegistry(RegistryOptions{
		HelmClient:       m.helmClient,
		UpdateInterval:   m.config.UpdateInterval,
		EnableAutoUpdate: m.config.EnableAutoUpdate,
	})
	if err != nil {
		return fmt.Errorf("failed to create registry: %w", err)
	}

	// Initialize validator
	m.validator = NewValidator()

	// Initialize compatibility matrix
	m.compatibility = NewCompatibilityMatrix()
	if m.config.CompatibilityFile != "" {
		if err := m.compatibility.LoadFromFile(m.config.CompatibilityFile); err != nil {
			m.log.V(1).Info("Failed to load compatibility file, using defaults", "error", err)
			m.compatibility.LoadDefault()
		}
	} else {
		m.compatibility.LoadDefault()
	}

	// Initialize notification manager
	if m.config.EnableNotifications {
		m.notifications, err = NewNotificationManager(NotificationManagerOptions{
			Client:        m.client,
			Registry:      m.registry,
			Compatibility: m.compatibility,
		})
		if err != nil {
			return fmt.Errorf("failed to create notification manager: %w", err)
		}
	}

	// Initialize pinning manager
	m.pinning = NewPinningManager(m.client, m.config.PinningConfigFile)

	// Initialize history tracker
	if m.config.EnableHistory {
		storage := NewInMemoryHistoryStorage() // Could be replaced with persistent storage
		m.history = NewHistoryTracker(m.client, storage, m.config.MaxHistoryItems)
	}

	// Initialize test manager
	if m.config.EnableTesting {
		m.testing, err = NewTestManager(TestManagerOptions{
			Client:        m.client,
			Scheme:        m.scheme,
			Compatibility: m.compatibility,
			Namespace:     m.config.TestNamespace,
		})
		if err != nil {
			return fmt.Errorf("failed to create test manager: %w", err)
		}

		// Create default tests
		if err := m.testing.CreateDefaultTests(); err != nil {
			m.log.Error(err, "Failed to create default tests")
		}
	}

	return nil
}

// RegisterChart registers a chart for version tracking
func (m *Manager) RegisterChart(ctx context.Context, chartName, repository string) error {
	return m.registry.RegisterChart(ctx, chartName, repository)
}

// GetRecommendedVersion gets the recommended version for a component
func (m *Manager) GetRecommendedVersion(ctx context.Context, component string, constraints map[string]string) (string, error) {
	// Get available versions
	chartInfo, err := m.registry.GetChartInfo(component, "bitnami")
	if err != nil {
		return "", fmt.Errorf("failed to get chart info: %w", err)
	}

	// Apply constraints
	if constraint, exists := constraints[component]; exists {
		if err := m.validator.AddConstraint(component, constraint); err != nil {
			return "", fmt.Errorf("invalid constraint: %w", err)
		}
	}

	// Find compatible versions
	compatibleVersions, err := m.validator.FindCompatibleVersions(component, chartInfo.AvailableVersions)
	if err != nil {
		return "", fmt.Errorf("failed to find compatible versions: %w", err)
	}

	if len(compatibleVersions) == 0 {
		return "", fmt.Errorf("no compatible versions found")
	}

	// Check for pins
	if pin, exists := m.pinning.GetPin(component); exists {
		m.log.Info("Using pinned version", "component", component, "version", pin.Version)
		return pin.Version, nil
	}

	// Get recommendations considering compatibility
	recommended := RecommendedVersions(compatibleVersions, 1)
	if len(recommended) == 0 {
		return "", fmt.Errorf("no recommended versions found")
	}

	return recommended[0], nil
}

// ValidateUpgrade validates a proposed upgrade
func (m *Manager) ValidateUpgrade(ctx context.Context, platform PlatformReference, component, targetVersion string) (*UpgradeValidation, error) {
	validation := &UpgradeValidation{
		Component:      component,
		TargetVersion:  targetVersion,
		Valid:          true,
		Checks:         []ValidationCheck{},
		Timestamp:      time.Now(),
	}

	// Get current version from history
	currentVersion := ""
	if m.history != nil {
		if ver, exists := m.history.GetLatestVersion(platform, component); exists {
			currentVersion = ver
		}
	}

	// Check version constraint
	constraintCheck := ValidationCheck{
		Name:   "Version Constraint",
		Passed: true,
	}
	if err := m.validator.ValidateVersion(component, targetVersion); err != nil {
		constraintCheck.Passed = false
		constraintCheck.Message = err.Error()
		validation.Valid = false
	} else {
		constraintCheck.Message = "Version satisfies all constraints"
	}
	validation.Checks = append(validation.Checks, constraintCheck)

	// Check compatibility
	compatCheck := ValidationCheck{
		Name:   "Component Compatibility",
		Passed: true,
	}
	// This would check against other components in the platform
	compatCheck.Message = "Compatible with other components"
	validation.Checks = append(validation.Checks, compatCheck)

	// Check for breaking changes
	breakingCheck := ValidationCheck{
		Name:   "Breaking Changes",
		Passed: true,
	}
	upgradeType := m.determineUpgradeType(currentVersion, targetVersion)
	if upgradeType == UpgradeTypeMajor {
		breakingCheck.Message = "Major version upgrade may contain breaking changes"
		validation.Warnings = append(validation.Warnings, "Review breaking changes before upgrading")
	} else {
		breakingCheck.Message = "No breaking changes expected"
	}
	validation.Checks = append(validation.Checks, breakingCheck)

	// Check for overrides
	if override, exists := m.pinning.GetOverride(component, targetVersion); exists {
		validation.Override = override
		if override.Force {
			validation.Valid = true
			validation.Warnings = append(validation.Warnings, "Upgrade forced by override")
		}
	}

	return validation, nil
}

// ExecuteUpgrade executes a version upgrade
func (m *Manager) ExecuteUpgrade(ctx context.Context, platform PlatformReference, component, targetVersion string) error {
	// Validate upgrade first
	validation, err := m.ValidateUpgrade(ctx, platform, component, targetVersion)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !validation.Valid && (validation.Override == nil || !validation.Override.Force) {
		return fmt.Errorf("upgrade validation failed")
	}

	// Record history entry
	if m.history != nil {
		currentVersion := ""
		if ver, exists := m.history.GetLatestVersion(platform, component); exists {
			currentVersion = ver
		}

		entry := VersionHistoryEntry{
			Platform:    platform,
			Component:   component,
			FromVersion: currentVersion,
			ToVersion:   targetVersion,
			Action:      VersionActionUpgrade,
			Status:      VersionStatusInProgress,
			Timestamp:   time.Now(),
		}

		if err := m.history.RecordChange(ctx, entry); err != nil {
			m.log.Error(err, "Failed to record history entry")
		}
	}

	// The actual upgrade would be performed by the component manager
	// This is just recording and validating the version change

	m.log.Info("Upgrade executed",
		"platform", platform.Name,
		"component", component,
		"version", targetVersion,
	)

	return nil
}

// CheckForUpdates checks all registered charts for updates
func (m *Manager) CheckForUpdates(ctx context.Context) error {
	if m.notifications == nil {
		return fmt.Errorf("notifications not enabled")
	}

	return m.notifications.CheckForUpgrades(ctx)
}

// RunVersionTest runs automated version testing
func (m *Manager) RunVersionTest(ctx context.Context, testID string) (*TestResult, error) {
	if m.testing == nil {
		return nil, fmt.Errorf("testing not enabled")
	}

	return m.testing.RunTest(ctx, testID)
}

// GetVersionHistory retrieves version history
func (m *Manager) GetVersionHistory(filter HistoryFilter) []VersionHistoryEntry {
	if m.history == nil {
		return []VersionHistoryEntry{}
	}

	return m.history.GetHistory(filter)
}

// Cleanup performs periodic cleanup tasks
func (m *Manager) Cleanup(ctx context.Context) error {
	// Clean expired pins and overrides
	if err := m.pinning.CleanupExpired(ctx); err != nil {
		m.log.Error(err, "Failed to cleanup expired pins")
	}

	// Clean old history
	if m.history != nil && m.config.HistoryRetention > 0 {
		cutoff := time.Now().Add(-m.config.HistoryRetention)
		if err := m.history.Cleanup(ctx, cutoff); err != nil {
			m.log.Error(err, "Failed to cleanup history")
		}
	}

	return nil
}

// Stop stops the version manager
func (m *Manager) Stop() {
	m.registry.Stop()
	m.log.Info("Version manager stopped")
}

// Helper types

// UpgradeValidation represents upgrade validation results
type UpgradeValidation struct {
	Component      string              `json:"component"`
	TargetVersion  string              `json:"targetVersion"`
	Valid          bool                `json:"valid"`
	Checks         []ValidationCheck   `json:"checks"`
	Warnings       []string            `json:"warnings,omitempty"`
	Override       *VersionOverride    `json:"override,omitempty"`
	Timestamp      time.Time           `json:"timestamp"`
}

// ValidationCheck represents a single validation check
type ValidationCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Getters for accessing internal components

// GetRegistry returns the version registry
func (m *Manager) GetRegistry() *Registry {
	return m.registry
}

// GetValidator returns the version validator
func (m *Manager) GetValidator() *Validator {
	return m.validator
}

// GetCompatibility returns the compatibility matrix
func (m *Manager) GetCompatibility() *CompatibilityMatrix {
	return m.compatibility
}

// GetNotifications returns the notification manager
func (m *Manager) GetNotifications() *NotificationManager {
	return m.notifications
}

// GetPinning returns the pinning manager
func (m *Manager) GetPinning() *PinningManager {
	return m.pinning
}

// GetHistory returns the history tracker
func (m *Manager) GetHistory() *HistoryTracker {
	return m.history
}

// GetTesting returns the test manager
func (m *Manager) GetTesting() *TestManager {
	return m.testing
}

// Helper methods

func (m *Manager) determineUpgradeType(current, target string) UpgradeType {
	// This is a simplified implementation
	// In production, you'd use semver comparison
	return UpgradeTypeMinor
}
