// Package version provides version pinning and override capabilities
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// PinningManager manages version pins and overrides
type PinningManager struct {
	mu         sync.RWMutex
	pins       map[string]*VersionPin
	overrides  map[string]*VersionOverride
	policies   []PinningPolicy
	client     client.Client
	configPath string
	log        logr.Logger
}

// VersionPin represents a pinned version
type VersionPin struct {
	Component  string            `json:"component"`
	Version    string            `json:"version"`
	Reason     string            `json:"reason"`
	PinnedBy   string            `json:"pinnedBy"`
	PinnedAt   time.Time         `json:"pinnedAt"`
	ExpiresAt  *time.Time        `json:"expiresAt,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// VersionOverride allows overriding version constraints
type VersionOverride struct {
	Component    string            `json:"component"`
	FromVersion  string            `json:"fromVersion"`
	ToVersion    string            `json:"toVersion"`
	Reason       string            `json:"reason"`
	ApprovedBy   string            `json:"approvedBy"`
	ApprovedAt   time.Time         `json:"approvedAt"`
	ExpiresAt    *time.Time        `json:"expiresAt,omitempty"`
	Force        bool              `json:"force"`
	SkipChecks   []string          `json:"skipChecks,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// PinningPolicy defines rules for version pinning
type PinningPolicy struct {
	Name        string
	Description string
	Evaluate    func(pin *VersionPin) error
}

// PinningConfig stores pinning configuration
type PinningConfig struct {
	Version   string                      `json:"version"`
	Pins      map[string]*VersionPin      `json:"pins"`
	Overrides map[string]*VersionOverride `json:"overrides"`
}

// NewPinningManager creates a new pinning manager
func NewPinningManager(client client.Client, configPath string) *PinningManager {
	pm := &PinningManager{
		pins:       make(map[string]*VersionPin),
		overrides:  make(map[string]*VersionOverride),
		policies:   []PinningPolicy{},
		client:     client,
		configPath: configPath,
		log:        log.Log.WithName("pinning-manager"),
	}

	// Add default policies
	pm.addDefaultPolicies()

	// Load configuration if exists
	if err := pm.LoadConfig(); err != nil {
		pm.log.V(1).Info("No existing pin configuration found", "error", err)
	}

	return pm
}

// PinVersion pins a component to a specific version
func (pm *PinningManager) PinVersion(ctx context.Context, pin *VersionPin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Validate pin
	for _, policy := range pm.policies {
		if err := policy.Evaluate(pin); err != nil {
			return fmt.Errorf("policy %s failed: %w", policy.Name, err)
		}
	}

	// Check for existing pin
	key := pm.getPinKey(pin.Component)
	if existing, exists := pm.pins[key]; exists {
		return fmt.Errorf("component %s already pinned to version %s", 
			pin.Component, existing.Version)
	}

	// Set timestamp
	pin.PinnedAt = time.Now()

	// Store pin
	pm.pins[key] = pin

	// Save configuration
	if err := pm.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save pin configuration: %w", err)
	}

	// Create audit event
	pm.createAuditEvent(ctx, "version_pinned", pin)

	pm.log.Info("Version pinned", 
		"component", pin.Component,
		"version", pin.Version,
		"reason", pin.Reason,
		"pinnedBy", pin.PinnedBy,
	)

	return nil
}

// UnpinVersion removes a version pin
func (pm *PinningManager) UnpinVersion(ctx context.Context, component, unpinnedBy string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.getPinKey(component)
	pin, exists := pm.pins[key]
	if !exists {
		return fmt.Errorf("no pin found for component %s", component)
	}

	// Remove pin
	delete(pm.pins, key)

	// Save configuration
	if err := pm.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create audit event
	pm.createAuditEvent(ctx, "version_unpinned", map[string]interface{}{
		"component":  component,
		"version":    pin.Version,
		"unpinnedBy": unpinnedBy,
		"pinnedAt":   pin.PinnedAt,
		"unpinnedAt": time.Now(),
	})

	pm.log.Info("Version unpinned",
		"component", component,
		"version", pin.Version,
		"unpinnedBy", unpinnedBy,
	)

	return nil
}

// GetPin retrieves a version pin
func (pm *PinningManager) GetPin(component string) (*VersionPin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := pm.getPinKey(component)
	pin, exists := pm.pins[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if pin.ExpiresAt != nil && time.Now().After(*pin.ExpiresAt) {
		return nil, false
	}

	return pm.copyPin(pin), true
}

// CreateOverride creates a version override
func (pm *PinningManager) CreateOverride(ctx context.Context, override *VersionOverride) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Validate override
	if override.Component == "" || override.ToVersion == "" {
		return fmt.Errorf("component and target version are required")
	}

	// Set timestamp
	override.ApprovedAt = time.Now()

	// Store override
	key := pm.getOverrideKey(override.Component, override.ToVersion)
	pm.overrides[key] = override

	// Save configuration
	if err := pm.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save override configuration: %w", err)
	}

	// Create audit event
	pm.createAuditEvent(ctx, "version_override_created", override)

	pm.log.Info("Version override created",
		"component", override.Component,
		"fromVersion", override.FromVersion,
		"toVersion", override.ToVersion,
		"force", override.Force,
		"approvedBy", override.ApprovedBy,
	)

	return nil
}

// GetOverride retrieves an override for a component upgrade
func (pm *PinningManager) GetOverride(component, targetVersion string) (*VersionOverride, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := pm.getOverrideKey(component, targetVersion)
	override, exists := pm.overrides[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if override.ExpiresAt != nil && time.Now().After(*override.ExpiresAt) {
		return nil, false
	}

	return pm.copyOverride(override), true
}

// ApplyToVersion applies pins and overrides to determine the effective version
func (pm *PinningManager) ApplyToVersion(component, requestedVersion string) (string, *VersionDecision) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	decision := &VersionDecision{
		Component:         component,
		RequestedVersion:  requestedVersion,
		EffectiveVersion:  requestedVersion,
		Applied:           []string{},
		Timestamp:         time.Now(),
	}

	// Check for pin first
	if pin, exists := pm.GetPin(component); exists {
		decision.EffectiveVersion = pin.Version
		decision.Applied = append(decision.Applied, fmt.Sprintf("pin: %s", pin.Reason))
		decision.Pin = pin
		return pin.Version, decision
	}

	// Check for override
	if override, exists := pm.GetOverride(component, requestedVersion); exists {
		if override.FromVersion == "" || override.FromVersion == requestedVersion {
			decision.EffectiveVersion = override.ToVersion
			decision.Applied = append(decision.Applied, fmt.Sprintf("override: %s", override.Reason))
			decision.Override = override
			return override.ToVersion, decision
		}
	}

	// Return requested version if no pin or override
	return requestedVersion, decision
}

// CleanupExpired removes expired pins and overrides
func (pm *PinningManager) CleanupExpired(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	cleaned := 0

	// Clean expired pins
	for key, pin := range pm.pins {
		if pin.ExpiresAt != nil && now.After(*pin.ExpiresAt) {
			delete(pm.pins, key)
			cleaned++
			pm.log.V(1).Info("Removed expired pin", "component", pin.Component, "version", pin.Version)
		}
	}

	// Clean expired overrides
	for key, override := range pm.overrides {
		if override.ExpiresAt != nil && now.After(*override.ExpiresAt) {
			delete(pm.overrides, key)
			cleaned++
			pm.log.V(1).Info("Removed expired override", 
				"component", override.Component, 
				"toVersion", override.ToVersion)
		}
	}

	if cleaned > 0 {
		if err := pm.SaveConfig(); err != nil {
			return fmt.Errorf("failed to save configuration after cleanup: %w", err)
		}
		pm.log.Info("Cleaned up expired entries", "count", cleaned)
	}

	return nil
}

// LoadConfig loads pinning configuration from file
func (pm *PinningManager) LoadConfig() error {
	if pm.configPath == "" {
		return nil
	}

	data, err := ioutil.ReadFile(pm.configPath)
	if err != nil {
		return err
	}

	var config PinningConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.pins = config.Pins
	pm.overrides = config.Overrides

	pm.log.Info("Loaded pinning configuration", 
		"pins", len(pm.pins), 
		"overrides", len(pm.overrides))

	return nil
}

// SaveConfig saves pinning configuration to file
func (pm *PinningManager) SaveConfig() error {
	if pm.configPath == "" {
		return nil
	}

	config := PinningConfig{
		Version:   "1.0.0",
		Pins:      pm.pins,
		Overrides: pm.overrides,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(pm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Private helper methods

func (pm *PinningManager) addDefaultPolicies() {
	// Expiration policy
	pm.policies = append(pm.policies, PinningPolicy{
		Name:        "expiration",
		Description: "Ensure pins have reasonable expiration",
		Evaluate: func(pin *VersionPin) error {
			if pin.ExpiresAt == nil {
				// Pins without expiration must have approval reason
				if pin.Reason == "" {
					return fmt.Errorf("permanent pins require a reason")
				}
			} else {
				// Check expiration is not too far in the future
				maxExpiration := time.Now().Add(365 * 24 * time.Hour) // 1 year
				if pin.ExpiresAt.After(maxExpiration) {
					return fmt.Errorf("pin expiration cannot exceed 1 year")
				}
			}
			return nil
		},
	})

	// Reason policy
	pm.policies = append(pm.policies, PinningPolicy{
		Name:        "reason-required",
		Description: "Ensure pins have valid reasons",
		Evaluate: func(pin *VersionPin) error {
			if len(pin.Reason) < 10 {
				return fmt.Errorf("pin reason must be at least 10 characters")
			}
			return nil
		},
	})
}

func (pm *PinningManager) getPinKey(component string) string {
	return component
}

func (pm *PinningManager) getOverrideKey(component, toVersion string) string {
	return fmt.Sprintf("%s:%s", component, toVersion)
}

func (pm *PinningManager) copyPin(pin *VersionPin) *VersionPin {
	copy := &VersionPin{
		Component: pin.Component,
		Version:   pin.Version,
		Reason:    pin.Reason,
		PinnedBy:  pin.PinnedBy,
		PinnedAt:  pin.PinnedAt,
	}
	if pin.ExpiresAt != nil {
		expires := *pin.ExpiresAt
		copy.ExpiresAt = &expires
	}
	if pin.Labels != nil {
		copy.Labels = make(map[string]string)
		for k, v := range pin.Labels {
			copy.Labels[k] = v
		}
	}
	return copy
}

func (pm *PinningManager) copyOverride(override *VersionOverride) *VersionOverride {
	copy := &VersionOverride{
		Component:    override.Component,
		FromVersion:  override.FromVersion,
		ToVersion:    override.ToVersion,
		Reason:       override.Reason,
		ApprovedBy:   override.ApprovedBy,
		ApprovedAt:   override.ApprovedAt,
		Force:        override.Force,
		SkipChecks:   append([]string{}, override.SkipChecks...),
	}
	if override.ExpiresAt != nil {
		expires := *override.ExpiresAt
		copy.ExpiresAt = &expires
	}
	if override.Labels != nil {
		copy.Labels = make(map[string]string)
		for k, v := range override.Labels {
			copy.Labels[k] = v
		}
	}
	return copy
}

func (pm *PinningManager) createAuditEvent(ctx context.Context, action string, data interface{}) {
	// In production, this would create actual Kubernetes events or audit logs
	pm.log.V(1).Info("Audit event", "action", action, "data", data)
}

// VersionDecision represents the decision made about a version
type VersionDecision struct {
	Component        string           `json:"component"`
	RequestedVersion string           `json:"requestedVersion"`
	EffectiveVersion string           `json:"effectiveVersion"`
	Applied          []string         `json:"applied"`
	Pin              *VersionPin      `json:"pin,omitempty"`
	Override         *VersionOverride `json:"override,omitempty"`
	Timestamp        time.Time        `json:"timestamp"`
}
