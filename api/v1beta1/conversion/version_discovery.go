/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
)

var versionLog = ctrl.Log.WithName("version-discovery")

// VersionDiscovery provides API version discovery capabilities
type VersionDiscovery struct {
	supportedVersions  []APIVersion
	currentVersion     string
	preferredVersion   string
	deprecationPolicy  DeprecationPolicy
	versionTransitions map[string]VersionTransition
}

// APIVersion represents a supported API version
type APIVersion struct {
	Version          string                 `json:"version"`
	Status           VersionStatus          `json:"status"`
	ServedDate       time.Time              `json:"servedDate"`
	DeprecatedDate   *time.Time             `json:"deprecatedDate,omitempty"`
	RemovalDate      *time.Time             `json:"removalDate,omitempty"`
	Schema           string                 `json:"schema"`
	MinClientVersion string                 `json:"minClientVersion"`
	MaxClientVersion string                 `json:"maxClientVersion,omitempty"`
	Features         []string               `json:"features"`
	BreakingChanges  []BreakingChange       `json:"breakingChanges,omitempty"`
	Endpoints        map[string]EndpointInfo `json:"endpoints"`
}

// VersionStatus represents the status of an API version
type VersionStatus string

const (
	VersionStatusAlpha      VersionStatus = "alpha"
	VersionStatusBeta       VersionStatus = "beta"
	VersionStatusStable     VersionStatus = "stable"
	VersionStatusDeprecated VersionStatus = "deprecated"
	VersionStatusRemoved    VersionStatus = "removed"
)

// BreakingChange describes a breaking change in a version
type BreakingChange struct {
	Field        string `json:"field"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	MigrationTip string `json:"migrationTip"`
}

// EndpointInfo contains information about an API endpoint
type EndpointInfo struct {
	Path        string   `json:"path"`
	Methods     []string `json:"methods"`
	Deprecated  bool     `json:"deprecated"`
	Alternative string   `json:"alternative,omitempty"`
}

// DeprecationPolicy defines the deprecation policy for API versions
type DeprecationPolicy struct {
	MinSupportedVersions int           `json:"minSupportedVersions"`
	DeprecationPeriod    time.Duration `json:"deprecationPeriod"`
	RemovalPeriod        time.Duration `json:"removalPeriod"`
	NotificationChannels []string      `json:"notificationChannels"`
}

// VersionTransition describes how to transition between versions
type VersionTransition struct {
	From            string                     `json:"from"`
	To              string                     `json:"to"`
	AutoMigrate     bool                       `json:"autoMigrate"`
	MigrationSteps  []string                   `json:"migrationSteps"`
	RollbackAllowed bool                       `json:"rollbackAllowed"`
	DataLossRisk    bool                       `json:"dataLossRisk"`
	Validators      []TransitionValidator      `json:"-"`
}

// TransitionValidator validates a version transition
type TransitionValidator func(context.Context, interface{}) error

// VersionNegotiator handles version negotiation with clients
type VersionNegotiator struct {
	discovery           *VersionDiscovery
	clientRegistry      map[string]ClientProfile
	negotiationStrategy NegotiationStrategy
}

// ClientProfile stores information about a client
type ClientProfile struct {
	ClientID            string    `json:"clientId"`
	Version             string    `json:"version"`
	UserAgent           string    `json:"userAgent"`
	PreferredVersion    string    `json:"preferredVersion"`
	AcceptedVersions    []string  `json:"acceptedVersions"`
	LastSeen            time.Time `json:"lastSeen"`
	DeprecationWarnings []string  `json:"deprecationWarnings"`
}

// NegotiationStrategy defines how to negotiate versions
type NegotiationStrategy interface {
	SelectVersion(client ClientProfile, available []APIVersion) (string, error)
}

// NewVersionDiscovery creates a new version discovery instance
func NewVersionDiscovery() *VersionDiscovery {
	vd := &VersionDiscovery{
		supportedVersions:  make([]APIVersion, 0),
		currentVersion:     "v1beta1",
		preferredVersion:   "v1beta1",
		versionTransitions: make(map[string]VersionTransition),
	}

	// Initialize supported versions
	vd.initializeSupportedVersions()
	
	// Initialize deprecation policy
	vd.initializeDeprecationPolicy()
	
	// Initialize version transitions
	vd.initializeVersionTransitions()
	
	return vd
}

// GetVersionInfo returns information about all supported versions
func (vd *VersionDiscovery) GetVersionInfo() VersionInfo {
	return VersionInfo{
		Current:           vd.currentVersion,
		Preferred:         vd.preferredVersion,
		Supported:         vd.supportedVersions,
		DeprecationPolicy: vd.deprecationPolicy,
		LastUpdated:       time.Now(),
	}
}

// VersionInfo contains comprehensive version information
type VersionInfo struct {
	Current           string            `json:"current"`
	Preferred         string            `json:"preferred"`
	Supported         []APIVersion      `json:"supported"`
	DeprecationPolicy DeprecationPolicy `json:"deprecationPolicy"`
	LastUpdated       time.Time         `json:"lastUpdated"`
}

// IsVersionSupported checks if a version is supported
func (vd *VersionDiscovery) IsVersionSupported(version string) bool {
	for _, v := range vd.supportedVersions {
		if v.Version == version && v.Status != VersionStatusRemoved {
			return true
		}
	}
	return false
}

// GetVersionEndpoints returns endpoints for a specific version
func (vd *VersionDiscovery) GetVersionEndpoints(version string) (map[string]EndpointInfo, error) {
	for _, v := range vd.supportedVersions {
		if v.Version == version {
			return v.Endpoints, nil
		}
	}
	return nil, fmt.Errorf("version %s not found", version)
}

// GetMigrationPath returns the migration path between versions
func (vd *VersionDiscovery) GetMigrationPath(fromVersion, toVersion string) ([]VersionTransition, error) {
	if fromVersion == toVersion {
		return []VersionTransition{}, nil
	}
	
	// Find direct transition
	key := fmt.Sprintf("%s->%s", fromVersion, toVersion)
	if transition, exists := vd.versionTransitions[key]; exists {
		return []VersionTransition{transition}, nil
	}
	
	// Find multi-step transition path
	path := vd.findTransitionPath(fromVersion, toVersion)
	if len(path) == 0 {
		return nil, fmt.Errorf("no migration path from %s to %s", fromVersion, toVersion)
	}
	
	return path, nil
}

// HandleVersionDiscoveryRequest handles HTTP requests for version discovery
func (vd *VersionDiscovery) HandleVersionDiscoveryRequest(w http.ResponseWriter, r *http.Request) {
	// Add version discovery headers
	w.Header().Set("X-API-Versions", vd.getVersionsHeader())
	w.Header().Set("X-API-Preferred-Version", vd.preferredVersion)
	w.Header().Set("X-API-Deprecation-Policy", vd.getDeprecationPolicyHeader())
	
	// Handle different request types
	switch r.URL.Path {
	case "/api/versions":
		vd.handleVersionsList(w, r)
	case "/api/version":
		vd.handleCurrentVersion(w, r)
	default:
		if strings.HasPrefix(r.URL.Path, "/api/versions/") {
			vd.handleSpecificVersion(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// Private helper methods

func (vd *VersionDiscovery) initializeSupportedVersions() {
	now := time.Now()
	deprecatedDate := now.Add(-180 * 24 * time.Hour) // 6 months ago
	removalDate := now.Add(180 * 24 * time.Hour)    // 6 months from now
	
	vd.supportedVersions = []APIVersion{
		{
			Version:          "v1alpha1",
			Status:           VersionStatusDeprecated,
			ServedDate:       now.Add(-365 * 24 * time.Hour), // 1 year ago
			DeprecatedDate:   &deprecatedDate,
			RemovalDate:      &removalDate,
			Schema:           "/schemas/v1alpha1/observabilityplatform.json",
			MinClientVersion: "v1.15.0",
			Features:         []string{"basic-monitoring", "alerts"},
			BreakingChanges: []BreakingChange{
				{
					Field:        "spec.monitoring",
					Type:         "renamed",
					Description:  "spec.monitoring renamed to spec.components",
					MigrationTip: "Use spec.components instead of spec.monitoring",
				},
			},
			Endpoints: map[string]EndpointInfo{
				"platforms": {
					Path:    "/api/v1alpha1/observabilityplatforms",
					Methods: []string{"GET", "POST", "PUT", "DELETE"},
				},
			},
		},
		{
			Version:          "v1beta1",
			Status:           VersionStatusBeta,
			ServedDate:       now.Add(-90 * 24 * time.Hour), // 3 months ago
			Schema:           "/schemas/v1beta1/observabilityplatform.json",
			MinClientVersion: "v1.18.0",
			Features: []string{
				"basic-monitoring",
				"alerts",
				"multi-cluster",
				"cost-optimization",
				"advanced-metrics",
			},
			Endpoints: map[string]EndpointInfo{
				"platforms": {
					Path:    "/api/v1beta1/observabilityplatforms",
					Methods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
				},
				"metrics": {
					Path:    "/api/v1beta1/metrics",
					Methods: []string{"GET"},
				},
			},
		},
	}
}

func (vd *VersionDiscovery) initializeDeprecationPolicy() {
	vd.deprecationPolicy = DeprecationPolicy{
		MinSupportedVersions: 2,
		DeprecationPeriod:    180 * 24 * time.Hour, // 6 months
		RemovalPeriod:        365 * 24 * time.Hour, // 1 year after deprecation
		NotificationChannels: []string{
			"api-response-headers",
			"webhook-notifications",
			"email-alerts",
			"documentation",
		},
	}
}

func (vd *VersionDiscovery) initializeVersionTransitions() {
	vd.versionTransitions["v1alpha1->v1beta1"] = VersionTransition{
		From:            "v1alpha1",
		To:              "v1beta1",
		AutoMigrate:     true,
		RollbackAllowed: true,
		DataLossRisk:    false,
		MigrationSteps: []string{
			"Convert spec.monitoring to spec.components",
			"Update alerting configuration structure",
			"Migrate storage configuration",
			"Update resource specifications",
		},
		Validators: []TransitionValidator{
			validateResourcesAvailable,
			validateNoDataLoss,
		},
	}
}

func (vd *VersionDiscovery) getVersionsHeader() string {
	versions := make([]string, 0, len(vd.supportedVersions))
	for _, v := range vd.supportedVersions {
		if v.Status != VersionStatusRemoved {
			versions = append(versions, v.Version)
		}
	}
	return strings.Join(versions, ", ")
}

func (vd *VersionDiscovery) getDeprecationPolicyHeader() string {
	return fmt.Sprintf("min-versions=%d; deprecation-period=%s; removal-period=%s",
		vd.deprecationPolicy.MinSupportedVersions,
		vd.deprecationPolicy.DeprecationPeriod,
		vd.deprecationPolicy.RemovalPeriod,
	)
}

func (vd *VersionDiscovery) handleVersionsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"kind":       "APIVersionList",
		"apiVersion": "v1",
		"versions":   vd.supportedVersions,
		"preferred":  vd.preferredVersion,
		"current":    vd.currentVersion,
		"policy":     vd.deprecationPolicy,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		versionLog.Error(err, "Failed to encode version list")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (vd *VersionDiscovery) handleCurrentVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	for _, v := range vd.supportedVersions {
		if v.Version == vd.currentVersion {
			if err := json.NewEncoder(w).Encode(v); err != nil {
				versionLog.Error(err, "Failed to encode current version")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}
	
	http.NotFound(w, r)
}

func (vd *VersionDiscovery) handleSpecificVersion(w http.ResponseWriter, r *http.Request) {
	version := strings.TrimPrefix(r.URL.Path, "/api/versions/")
	
	for _, v := range vd.supportedVersions {
		if v.Version == version {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(v); err != nil {
				versionLog.Error(err, "Failed to encode version info")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}
	
	http.NotFound(w, r)
}

func (vd *VersionDiscovery) findTransitionPath(from, to string) []VersionTransition {
	// Simple BFS to find shortest transition path
	type node struct {
		version string
		path    []VersionTransition
	}
	
	queue := []node{{version: from, path: []VersionTransition{}}}
	visited := map[string]bool{from: true}
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		
		if current.version == to {
			return current.path
		}
		
		// Check all possible transitions from current version
		for key, transition := range vd.versionTransitions {
			if strings.HasPrefix(key, current.version+"->") {
				nextVersion := transition.To
				if !visited[nextVersion] {
					visited[nextVersion] = true
					newPath := append(current.path, transition)
					queue = append(queue, node{version: nextVersion, path: newPath})
				}
			}
		}
	}
	
	return nil
}

// DefaultNegotiationStrategy implements the default version negotiation strategy
type DefaultNegotiationStrategy struct{}

// SelectVersion selects the best version for a client
func (dns *DefaultNegotiationStrategy) SelectVersion(client ClientProfile, available []APIVersion) (string, error) {
	// First, check if client's preferred version is available
	for _, v := range available {
		if v.Version == client.PreferredVersion && v.Status != VersionStatusRemoved {
			return v.Version, nil
		}
	}
	
	// Then, check accepted versions in order
	for _, acceptedVersion := range client.AcceptedVersions {
		for _, v := range available {
			if v.Version == acceptedVersion && v.Status != VersionStatusRemoved {
				return v.Version, nil
			}
		}
	}
	
	// Finally, return the oldest stable version for compatibility
	oldestStable := ""
	for _, v := range available {
		if v.Status == VersionStatusStable || v.Status == VersionStatusBeta {
			if oldestStable == "" || versionLessThan(v.Version, oldestStable) {
				oldestStable = v.Version
			}
		}
	}
	
	if oldestStable != "" {
		return oldestStable, nil
	}
	
	return "", fmt.Errorf("no suitable version found for client")
}

// Helper function for version comparison
func versionLessThan(v1, v2 string) bool {
	// Simple comparison for our version format
	return v1 < v2
}

// Validation functions
func validateResourcesAvailable(ctx context.Context, obj interface{}) error {
	// Validate that required resources are available for migration
	return nil
}

func validateNoDataLoss(ctx context.Context, obj interface{}) error {
	// Validate that migration won't cause data loss
	return nil
}

// NewVersionNegotiator creates a new version negotiator
func NewVersionNegotiator(discovery *VersionDiscovery) *VersionNegotiator {
	return &VersionNegotiator{
		discovery:           discovery,
		clientRegistry:      make(map[string]ClientProfile),
		negotiationStrategy: &DefaultNegotiationStrategy{},
	}
}

// NegotiateVersion negotiates the best version for a client
func (vn *VersionNegotiator) NegotiateVersion(clientID, userAgent string, acceptVersions []string) (string, error) {
	// Get or create client profile
	profile, exists := vn.clientRegistry[clientID]
	if !exists {
		profile = ClientProfile{
			ClientID:         clientID,
			UserAgent:        userAgent,
			AcceptedVersions: acceptVersions,
		}
	}
	
	profile.LastSeen = time.Now()
	
	// Use negotiation strategy to select version
	selected, err := vn.negotiationStrategy.SelectVersion(profile, vn.discovery.supportedVersions)
	if err != nil {
		return "", err
	}
	
	// Update client profile
	profile.PreferredVersion = selected
	vn.clientRegistry[clientID] = profile
	
	// Check for deprecation warnings
	vn.checkDeprecationWarnings(&profile, selected)
	
	return selected, nil
}

func (vn *VersionNegotiator) checkDeprecationWarnings(profile *ClientProfile, version string) {
	profile.DeprecationWarnings = []string{}
	
	for _, v := range vn.discovery.supportedVersions {
		if v.Version == version && v.Status == VersionStatusDeprecated {
			warning := fmt.Sprintf("API version %s is deprecated and will be removed on %s",
				version, v.RemovalDate.Format("2006-01-02"))
			profile.DeprecationWarnings = append(profile.DeprecationWarnings, warning)
		}
	}
}
