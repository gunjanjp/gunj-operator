// Package version provides upgrade notification capabilities
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// NotificationManager manages upgrade notifications
type NotificationManager struct {
	mu            sync.RWMutex
	client        client.Client
	registry      *Registry
	compatibility *CompatibilityMatrix
	handlers      []NotificationHandler
	history       []NotificationRecord
	log           logr.Logger
}

// NotificationHandler processes upgrade notifications
type NotificationHandler interface {
	Handle(ctx context.Context, notification *UpgradeNotification) error
	Name() string
}

// UpgradeNotification represents an available upgrade
type UpgradeNotification struct {
	ID               string                 `json:"id"`
	Component        string                 `json:"component"`
	CurrentVersion   string                 `json:"currentVersion"`
	AvailableVersion string                 `json:"availableVersion"`
	Type             UpgradeType           `json:"type"`
	Priority         UpgradePriority       `json:"priority"`
	ReleaseNotes     string                 `json:"releaseNotes,omitempty"`
	SecurityFixes    []string              `json:"securityFixes,omitempty"`
	BugFixes         []string              `json:"bugFixes,omitempty"`
	NewFeatures      []string              `json:"newFeatures,omitempty"`
	BreakingChanges  []string              `json:"breakingChanges,omitempty"`
	Compatibility    CompatibilityInfo     `json:"compatibility"`
	Timestamp        time.Time             `json:"timestamp"`
}

// UpgradeType categorizes the upgrade
type UpgradeType string

const (
	UpgradeTypePatch    UpgradeType = "patch"
	UpgradeTypeMinor    UpgradeType = "minor"
	UpgradeTypeMajor    UpgradeType = "major"
	UpgradeTypeSecurity UpgradeType = "security"
)

// UpgradePriority indicates upgrade urgency
type UpgradePriority string

const (
	UpgradePriorityLow      UpgradePriority = "low"
	UpgradePriorityMedium   UpgradePriority = "medium"
	UpgradePriorityHigh     UpgradePriority = "high"
	UpgradePriorityCritical UpgradePriority = "critical"
)

// CompatibilityInfo provides compatibility details
type CompatibilityInfo struct {
	Compatible bool     `json:"compatible"`
	Issues     []string `json:"issues,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// NotificationRecord tracks notification history
type NotificationRecord struct {
	Notification *UpgradeNotification `json:"notification"`
	Delivered    time.Time            `json:"delivered"`
	Acknowledged bool                 `json:"acknowledged"`
	AcknowledgedAt time.Time          `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy string             `json:"acknowledgedBy,omitempty"`
}

// NotificationManagerOptions configures the notification manager
type NotificationManagerOptions struct {
	Client        client.Client
	Registry      *Registry
	Compatibility *CompatibilityMatrix
	HistoryLimit  int
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(opts NotificationManagerOptions) (*NotificationManager, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}
	if opts.Registry == nil {
		return nil, fmt.Errorf("version registry is required")
	}
	if opts.Compatibility == nil {
		return nil, fmt.Errorf("compatibility matrix is required")
	}
	if opts.HistoryLimit == 0 {
		opts.HistoryLimit = 100
	}

	nm := &NotificationManager{
		client:        opts.Client,
		registry:      opts.Registry,
		compatibility: opts.Compatibility,
		handlers:      []NotificationHandler{},
		history:       []NotificationRecord{},
		log:           log.Log.WithName("notification-manager"),
	}

	// Add default handlers
	nm.addDefaultHandlers()

	return nm, nil
}

// RegisterHandler adds a notification handler
func (nm *NotificationManager) RegisterHandler(handler NotificationHandler) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	nm.handlers = append(nm.handlers, handler)
	nm.log.Info("Registered notification handler", "handler", handler.Name())
}

// CheckForUpgrades checks all platforms for available upgrades
func (nm *NotificationManager) CheckForUpgrades(ctx context.Context) error {
	// List all ObservabilityPlatforms
	platformList := &observabilityv1.ObservabilityPlatformList{}
	if err := nm.client.List(ctx, platformList); err != nil {
		return fmt.Errorf("failed to list platforms: %w", err)
	}

	for _, platform := range platformList.Items {
		if err := nm.checkPlatformUpgrades(ctx, &platform); err != nil {
			nm.log.Error(err, "Failed to check upgrades for platform", 
				"platform", platform.Name, "namespace", platform.Namespace)
		}
	}

	return nil
}

// ProcessNotification processes a single upgrade notification
func (nm *NotificationManager) ProcessNotification(ctx context.Context, notification *UpgradeNotification) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Execute all handlers
	for _, handler := range nm.handlers {
		if err := handler.Handle(ctx, notification); err != nil {
			nm.log.Error(err, "Handler failed", "handler", handler.Name())
		}
	}

	// Record in history
	nm.addToHistory(notification)

	return nil
}

// GetHistory returns notification history
func (nm *NotificationManager) GetHistory(limit int) []NotificationRecord {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if limit <= 0 || limit > len(nm.history) {
		limit = len(nm.history)
	}

	// Return most recent first
	result := make([]NotificationRecord, limit)
	for i := 0; i < limit; i++ {
		result[i] = nm.history[len(nm.history)-1-i]
	}

	return result
}

// AcknowledgeNotification marks a notification as acknowledged
func (nm *NotificationManager) AcknowledgeNotification(notificationID, acknowledgedBy string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for i := range nm.history {
		if nm.history[i].Notification.ID == notificationID {
			nm.history[i].Acknowledged = true
			nm.history[i].AcknowledgedAt = time.Now()
			nm.history[i].AcknowledgedBy = acknowledgedBy
			return nil
		}
	}

	return fmt.Errorf("notification %s not found", notificationID)
}

// Private helper methods

func (nm *NotificationManager) checkPlatformUpgrades(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Check each component
	// Helper functions to safely get version
	getPrometheusVersion := func() string {
		if platform.Spec.Components.Prometheus != nil {
			return platform.Spec.Components.Prometheus.Version
		}
		return ""
	}
	getGrafanaVersion := func() string {
		if platform.Spec.Components.Grafana != nil {
			return platform.Spec.Components.Grafana.Version
		}
		return ""
	}
	getLokiVersion := func() string {
		if platform.Spec.Components.Loki != nil {
			return platform.Spec.Components.Loki.Version
		}
		return ""
	}
	getTempoVersion := func() string {
		if platform.Spec.Components.Tempo != nil {
			return platform.Spec.Components.Tempo.Version
		}
		return ""
	}

	components := []struct {
		name    string
		enabled bool
		version string
		spec    interface{}
	}{
		{
			name:    "prometheus",
			enabled: platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled,
			version: getPrometheusVersion(),
			spec:    platform.Spec.Components.Prometheus,
		},
		{
			name:    "grafana",
			enabled: platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled,
			version: getGrafanaVersion(),
			spec:    platform.Spec.Components.Grafana,
		},
		{
			name:    "loki",
			enabled: platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled,
			version: getLokiVersion(),
			spec:    platform.Spec.Components.Loki,
		},
		{
			name:    "tempo",
			enabled: platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled,
			version: getTempoVersion(),
			spec:    platform.Spec.Components.Tempo,
		},
	}

	// Get current component versions
	var currentComponents []ComponentVersion
	for _, comp := range components {
		if comp.enabled {
			currentComponents = append(currentComponents, ComponentVersion{
				Component: comp.name,
				Version:   comp.version,
			})
		}
	}

	// Check each enabled component
	for _, comp := range components {
		if !comp.enabled {
			continue
		}

		// Get chart info from registry
		chartInfo, err := nm.registry.GetChartInfo(comp.name, "bitnami")
		if err != nil {
			nm.log.V(1).Info("Failed to get chart info", "component", comp.name, "error", err)
			continue
		}

		// Check if update is available
		if chartInfo.UpdateAvailable {
			notification := nm.createNotification(comp.name, comp.version, chartInfo.LatestVersion, currentComponents)
			if err := nm.ProcessNotification(ctx, notification); err != nil {
				return err
			}
		}
	}

	return nil
}

func (nm *NotificationManager) createNotification(component, currentVersion, availableVersion string, allComponents []ComponentVersion) *UpgradeNotification {
	notification := &UpgradeNotification{
		ID:               fmt.Sprintf("%s-%s-%d", component, availableVersion, time.Now().Unix()),
		Component:        component,
		CurrentVersion:   currentVersion,
		AvailableVersion: availableVersion,
		Timestamp:        time.Now(),
	}

	// Determine upgrade type and priority
	notification.Type = nm.determineUpgradeType(currentVersion, availableVersion)
	notification.Priority = nm.determineUpgradePriority(notification.Type, component)

	// Check compatibility
	testComponents := make([]ComponentVersion, 0, len(allComponents))
	for _, comp := range allComponents {
		if comp.Component == component {
			testComponents = append(testComponents, ComponentVersion{
				Component: component,
				Version:   availableVersion,
			})
		} else {
			testComponents = append(testComponents, comp)
		}
	}

	compatible, issues := nm.compatibility.CheckCompatibility(testComponents)
	notification.Compatibility = CompatibilityInfo{
		Compatible: compatible,
		Issues:     issues,
	}

	// Add release information (would come from actual release notes in production)
	nm.addReleaseInfo(notification)

	return notification
}

func (nm *NotificationManager) determineUpgradeType(current, available string) UpgradeType {
	currentVer, err1 := semver.NewVersion(current)
	availableVer, err2 := semver.NewVersion(available)

	if err1 != nil || err2 != nil {
		return UpgradeTypePatch
	}

	if currentVer.Major() != availableVer.Major() {
		return UpgradeTypeMajor
	} else if currentVer.Minor() != availableVer.Minor() {
		return UpgradeTypeMinor
	}

	return UpgradeTypePatch
}

func (nm *NotificationManager) determineUpgradePriority(upgradeType UpgradeType, component string) UpgradePriority {
	// This is a simplified priority determination
	// In production, you'd check CVE databases, release notes, etc.
	
	switch upgradeType {
	case UpgradeTypeMajor:
		return UpgradePriorityHigh
	case UpgradeTypeMinor:
		return UpgradePriorityMedium
	case UpgradeTypePatch:
		return UpgradePriorityLow
	case UpgradeTypeSecurity:
		return UpgradePriorityCritical
	default:
		return UpgradePriorityLow
	}
}

func (nm *NotificationManager) addReleaseInfo(notification *UpgradeNotification) {
	// In production, this would fetch actual release notes
	// For now, we'll add placeholder information
	
	switch notification.Type {
	case UpgradeTypeMajor:
		notification.NewFeatures = []string{"Major new features", "Performance improvements"}
		notification.BreakingChanges = []string{"API changes", "Configuration format updates"}
	case UpgradeTypeMinor:
		notification.NewFeatures = []string{"New minor features", "UI improvements"}
		notification.BugFixes = []string{"Various bug fixes"}
	case UpgradeTypePatch:
		notification.BugFixes = []string{"Bug fixes", "Performance improvements"}
	case UpgradeTypeSecurity:
		notification.SecurityFixes = []string{"Critical security patches"}
		notification.Priority = UpgradePriorityCritical
	}
}

func (nm *NotificationManager) addToHistory(notification *UpgradeNotification) {
	record := NotificationRecord{
		Notification: notification,
		Delivered:    time.Now(),
		Acknowledged: false,
	}

	nm.history = append(nm.history, record)

	// Limit history size
	if len(nm.history) > 100 {
		nm.history = nm.history[len(nm.history)-100:]
	}
}

func (nm *NotificationManager) addDefaultHandlers() {
	// Add event handler
	nm.RegisterHandler(&EventNotificationHandler{
		client: nm.client,
		log:    nm.log.WithName("event-handler"),
	})

	// Add log handler
	nm.RegisterHandler(&LogNotificationHandler{
		log: nm.log.WithName("log-handler"),
	})
}

// Default notification handlers

// EventNotificationHandler creates Kubernetes events for notifications
type EventNotificationHandler struct {
	client client.Client
	log    logr.Logger
}

func (h *EventNotificationHandler) Name() string {
	return "kubernetes-events"
}

func (h *EventNotificationHandler) Handle(ctx context.Context, notification *UpgradeNotification) error {
	// Implementation would create Kubernetes events
	h.log.Info("Upgrade available",
		"component", notification.Component,
		"current", notification.CurrentVersion,
		"available", notification.AvailableVersion,
		"priority", notification.Priority,
	)
	return nil
}

// LogNotificationHandler logs notifications
type LogNotificationHandler struct {
	log logr.Logger
}

func (h *LogNotificationHandler) Name() string {
	return "logger"
}

func (h *LogNotificationHandler) Handle(ctx context.Context, notification *UpgradeNotification) error {
	h.log.Info("Upgrade notification",
		"id", notification.ID,
		"component", notification.Component,
		"currentVersion", notification.CurrentVersion,
		"availableVersion", notification.AvailableVersion,
		"type", notification.Type,
		"priority", notification.Priority,
		"compatible", notification.Compatibility.Compatible,
	)
	return nil
}
