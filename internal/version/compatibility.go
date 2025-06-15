// Package version provides component compatibility management
package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CompatibilityMatrix manages version compatibility between components
type CompatibilityMatrix struct {
	mu           sync.RWMutex
	rules        map[string][]CompatibilityRule
	cache        map[string]bool
	log          logr.Logger
}

// CompatibilityRule defines compatibility between components
type CompatibilityRule struct {
	Component    string   `json:"component"`
	Version      string   `json:"version"`
	Compatible   []string `json:"compatible"`
	Incompatible []string `json:"incompatible,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// ComponentVersion represents a component and its version
type ComponentVersion struct {
	Component string `json:"component"`
	Version   string `json:"version"`
}

// CompatibilityConfig stores the full compatibility configuration
type CompatibilityConfig struct {
	Version string                        `json:"version"`
	Rules   map[string][]CompatibilityRule `json:"rules"`
	Tested  []TestedCombination           `json:"tested"`
}

// TestedCombination represents a tested component combination
type TestedCombination struct {
	Components []ComponentVersion `json:"components"`
	Status     string            `json:"status"`
	TestedOn   string            `json:"testedOn"`
	Notes      string            `json:"notes,omitempty"`
}

// NewCompatibilityMatrix creates a new compatibility matrix
func NewCompatibilityMatrix() *CompatibilityMatrix {
	return &CompatibilityMatrix{
		rules: make(map[string][]CompatibilityRule),
		cache: make(map[string]bool),
		log:   log.Log.WithName("compatibility-matrix"),
	}
}

// LoadFromFile loads compatibility rules from a file
func (cm *CompatibilityMatrix) LoadFromFile(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read compatibility file: %w", err)
	}

	var config CompatibilityConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse compatibility config: %w", err)
	}

	cm.rules = config.Rules
	cm.cache = make(map[string]bool) // Clear cache
	
	cm.log.Info("Loaded compatibility matrix", "version", config.Version, "rules", len(cm.rules))
	
	return nil
}

// LoadDefault loads the default compatibility matrix
func (cm *CompatibilityMatrix) LoadDefault() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Default compatibility rules
	cm.rules = map[string][]CompatibilityRule{
		"prometheus": {
			{
				Component: "prometheus",
				Version:   ">=2.45.0",
				Compatible: []string{
					"grafana:>=9.0.0",
					"loki:>=2.8.0",
					"tempo:>=2.0.0",
				},
			},
			{
				Component: "prometheus",
				Version:   ">=2.40.0 <2.45.0",
				Compatible: []string{
					"grafana:>=8.5.0",
					"loki:>=2.6.0",
					"tempo:>=1.5.0",
				},
			},
		},
		"grafana": {
			{
				Component: "grafana",
				Version:   ">=10.0.0",
				Compatible: []string{
					"prometheus:>=2.40.0",
					"loki:>=2.8.0",
					"tempo:>=2.0.0",
				},
				Notes: "Grafana 10.x requires newer datasource plugins",
			},
			{
				Component: "grafana",
				Version:   ">=9.0.0 <10.0.0",
				Compatible: []string{
					"prometheus:>=2.30.0",
					"loki:>=2.4.0",
					"tempo:>=1.0.0",
				},
			},
		},
		"loki": {
			{
				Component: "loki",
				Version:   ">=2.9.0",
				Compatible: []string{
					"prometheus:>=2.40.0",
					"grafana:>=9.0.0",
					"tempo:>=2.0.0",
				},
			},
			{
				Component: "loki",
				Version:   ">=2.6.0 <2.9.0",
				Compatible: []string{
					"prometheus:>=2.30.0",
					"grafana:>=8.0.0",
					"tempo:>=1.0.0",
				},
			},
		},
		"tempo": {
			{
				Component: "tempo",
				Version:   ">=2.0.0",
				Compatible: []string{
					"prometheus:>=2.40.0",
					"grafana:>=9.0.0",
					"loki:>=2.8.0",
				},
			},
			{
				Component: "tempo",
				Version:   ">=1.5.0 <2.0.0",
				Compatible: []string{
					"prometheus:>=2.30.0",
					"grafana:>=8.5.0",
					"loki:>=2.6.0",
				},
			},
		},
	}

	cm.cache = make(map[string]bool)
	cm.log.Info("Loaded default compatibility matrix")
}

// CheckCompatibility checks if a set of component versions are compatible
func (cm *CompatibilityMatrix) CheckCompatibility(components []ComponentVersion) (bool, []string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check cache first
	cacheKey := cm.getCacheKey(components)
	if result, exists := cm.cache[cacheKey]; exists {
		return result, nil
	}

	var issues []string
	compatible := true

	// Check each component against others
	for i, comp := range components {
		rules := cm.getRulesForComponent(comp.Component, comp.Version)
		
		for j, other := range components {
			if i == j {
				continue
			}

			// Check if other component is compatible
			if !cm.isCompatibleWith(rules, other) {
				compatible = false
				issues = append(issues, fmt.Sprintf(
					"%s %s is not compatible with %s %s",
					comp.Component, comp.Version,
					other.Component, other.Version,
				))
			}
		}
	}

	// Cache the result
	cm.cache[cacheKey] = compatible

	return compatible, issues
}

// GetCompatibleVersions returns compatible versions for a component given other components
func (cm *CompatibilityMatrix) GetCompatibleVersions(component string, otherComponents []ComponentVersion, availableVersions []string) []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var compatible []string

	for _, version := range availableVersions {
		// Create a test combination
		testComponents := append([]ComponentVersion{}, otherComponents...)
		testComponents = append(testComponents, ComponentVersion{
			Component: component,
			Version:   version,
		})

		// Check if this combination is compatible
		if isCompat, _ := cm.CheckCompatibility(testComponents); isCompat {
			compatible = append(compatible, version)
		}
	}

	return compatible
}

// AddRule adds a compatibility rule
func (cm *CompatibilityMatrix) AddRule(rule CompatibilityRule) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.rules[rule.Component] = append(cm.rules[rule.Component], rule)
	cm.cache = make(map[string]bool) // Clear cache
}

// GetRecommendedVersions suggests the best version combination
func (cm *CompatibilityMatrix) GetRecommendedVersions(components []string, availableVersions map[string][]string) (map[string]string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	recommended := make(map[string]string)

	// Start with the latest versions
	for _, comp := range components {
		if versions, ok := availableVersions[comp]; ok && len(versions) > 0 {
			recommended[comp] = versions[0] // Assume sorted, latest first
		} else {
			return nil, fmt.Errorf("no available versions for component %s", comp)
		}
	}

	// Check if the latest versions are compatible
	var testComponents []ComponentVersion
	for comp, ver := range recommended {
		testComponents = append(testComponents, ComponentVersion{
			Component: comp,
			Version:   ver,
		})
	}

	if compatible, _ := cm.CheckCompatibility(testComponents); compatible {
		return recommended, nil
	}

	// If not compatible, try to find a compatible combination
	// This is a simplified algorithm - in production, you'd want something more sophisticated
	return cm.findCompatibleCombination(components, availableVersions)
}

// Private helper methods

func (cm *CompatibilityMatrix) getRulesForComponent(component, version string) []CompatibilityRule {
	rules, exists := cm.rules[component]
	if !exists {
		return nil
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		cm.log.V(1).Info("Invalid version", "component", component, "version", version)
		return nil
	}

	var applicable []CompatibilityRule
	for _, rule := range rules {
		constraint, err := semver.NewConstraint(rule.Version)
		if err != nil {
			cm.log.V(1).Info("Invalid constraint", "constraint", rule.Version)
			continue
		}

		if constraint.Check(v) {
			applicable = append(applicable, rule)
		}
	}

	return applicable
}

func (cm *CompatibilityMatrix) isCompatibleWith(rules []CompatibilityRule, other ComponentVersion) bool {
	otherVer, err := semver.NewVersion(other.Version)
	if err != nil {
		return false
	}

	for _, rule := range rules {
		// Check compatible list
		for _, compat := range rule.Compatible {
			parts := splitComponentVersion(compat)
			if len(parts) != 2 {
				continue
			}

			if parts[0] == other.Component {
				constraint, err := semver.NewConstraint(parts[1])
				if err != nil {
					continue
				}

				if constraint.Check(otherVer) {
					return true
				}
			}
		}

		// Check incompatible list
		for _, incompat := range rule.Incompatible {
			parts := splitComponentVersion(incompat)
			if len(parts) != 2 {
				continue
			}

			if parts[0] == other.Component {
				constraint, err := semver.NewConstraint(parts[1])
				if err != nil {
					continue
				}

				if constraint.Check(otherVer) {
					return false
				}
			}
		}
	}

	// If no explicit rule, assume compatible
	return true
}

func (cm *CompatibilityMatrix) getCacheKey(components []ComponentVersion) string {
	key := ""
	for _, comp := range components {
		key += fmt.Sprintf("%s:%s;", comp.Component, comp.Version)
	}
	return key
}

func (cm *CompatibilityMatrix) findCompatibleCombination(components []string, availableVersions map[string][]string) (map[string]string, error) {
	// This is a simplified brute-force approach
	// In production, you'd want a more efficient algorithm
	
	// Try different combinations
	indices := make(map[string]int)
	for _, comp := range components {
		indices[comp] = 0
	}

	for {
		// Build current combination
		current := make(map[string]string)
		testComponents := []ComponentVersion{}
		
		for _, comp := range components {
			if idx := indices[comp]; idx < len(availableVersions[comp]) {
				version := availableVersions[comp][idx]
				current[comp] = version
				testComponents = append(testComponents, ComponentVersion{
					Component: comp,
					Version:   version,
				})
			}
		}

		// Check if this combination is compatible
		if compatible, _ := cm.CheckCompatibility(testComponents); compatible {
			return current, nil
		}

		// Increment indices
		done := true
		for i := len(components) - 1; i >= 0; i-- {
			comp := components[i]
			indices[comp]++
			if indices[comp] < len(availableVersions[comp]) {
				done = false
				break
			}
			indices[comp] = 0
		}

		if done {
			break
		}
	}

	return nil, fmt.Errorf("no compatible combination found")
}

// splitComponentVersion splits "component:version" string
func splitComponentVersion(s string) []string {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}

// ExportMatrix exports the compatibility matrix to JSON
func (cm *CompatibilityMatrix) ExportMatrix() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config := CompatibilityConfig{
		Version: "1.0.0",
		Rules:   cm.rules,
		Tested:  []TestedCombination{}, // Would be populated from testing
	}

	return json.MarshalIndent(config, "", "  ")
}
