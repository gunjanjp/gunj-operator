/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
)

// versionManager implements the VersionManager interface
type versionManager struct {
	repository Repository
}

// NewVersionManager creates a new version manager
func NewVersionManager(repository Repository) VersionManager {
	return &versionManager{
		repository: repository,
	}
}

// GetLatestVersion gets the latest stable version of a chart
func (vm *versionManager) GetLatestVersion(ctx context.Context, chartName string) (string, error) {
	versions, err := vm.repository.GetChartVersions(ctx, chartName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get chart versions")
	}
	
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found for chart %s", chartName)
	}
	
	// Parse all versions
	semVersions := make([]*semver.Version, 0, len(versions))
	for _, v := range versions {
		// Clean version string
		cleanVersion := strings.TrimPrefix(v, "v")
		
		sv, err := semver.NewVersion(cleanVersion)
		if err != nil {
			// Skip invalid versions
			continue
		}
		
		// Skip pre-release versions for stable
		if sv.Prerelease() != "" {
			continue
		}
		
		semVersions = append(semVersions, sv)
	}
	
	if len(semVersions) == 0 {
		return "", fmt.Errorf("no stable versions found for chart %s", chartName)
	}
	
	// Sort versions
	sorted := semver.Collection(semVersions)
	sorted.Sort()
	
	// Return the latest (last in sorted array)
	latest := sorted[len(sorted)-1]
	return latest.String(), nil
}

// CompareVersions compares two versions
// Returns:
// -1 if v1 < v2
//  0 if v1 == v2
//  1 if v1 > v2
func (vm *versionManager) CompareVersions(v1, v2 string) (int, error) {
	// Clean version strings
	cleanV1 := strings.TrimPrefix(v1, "v")
	cleanV2 := strings.TrimPrefix(v2, "v")
	
	sv1, err := semver.NewVersion(cleanV1)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid version: %s", v1)
	}
	
	sv2, err := semver.NewVersion(cleanV2)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid version: %s", v2)
	}
	
	return sv1.Compare(sv2), nil
}

// IsVersionCompatible checks if a version is compatible
func (vm *versionManager) IsVersionCompatible(current, target string) (bool, error) {
	// Clean version strings
	cleanCurrent := strings.TrimPrefix(current, "v")
	cleanTarget := strings.TrimPrefix(target, "v")
	
	currentVer, err := semver.NewVersion(cleanCurrent)
	if err != nil {
		return false, errors.Wrapf(err, "invalid current version: %s", current)
	}
	
	targetVer, err := semver.NewVersion(cleanTarget)
	if err != nil {
		return false, errors.Wrapf(err, "invalid target version: %s", target)
	}
	
	// Check compatibility rules:
	// 1. Major version must be the same for compatibility
	// 2. Minor version upgrade is allowed
	// 3. Patch version upgrade is always allowed
	// 4. Downgrade within same major version is allowed with warning
	
	if currentVer.Major() != targetVer.Major() {
		// Major version change - not compatible
		return false, nil
	}
	
	// Same major version - compatible
	return true, nil
}

// GetUpgradePath returns the upgrade path between versions
func (vm *versionManager) GetUpgradePath(current, target string) ([]string, error) {
	// Clean version strings
	cleanCurrent := strings.TrimPrefix(current, "v")
	cleanTarget := strings.TrimPrefix(target, "v")
	
	currentVer, err := semver.NewVersion(cleanCurrent)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid current version: %s", current)
	}
	
	targetVer, err := semver.NewVersion(cleanTarget)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid target version: %s", target)
	}
	
	// Check if upgrade is needed
	comparison := currentVer.Compare(targetVer)
	if comparison == 0 {
		// Same version
		return []string{}, nil
	}
	
	if comparison > 0 {
		// Downgrade - return direct path
		return []string{target}, nil
	}
	
	// Build upgrade path
	path := []string{}
	
	// If major version change, we need intermediate steps
	if currentVer.Major() != targetVer.Major() {
		// For each major version, add the latest minor.patch
		for major := currentVer.Major() + 1; major <= targetVer.Major(); major++ {
			if major == targetVer.Major() {
				// Final major version - use target
				path = append(path, target)
			} else {
				// Intermediate major version - use latest available
				intermediateVersion := fmt.Sprintf("%d.0.0", major)
				path = append(path, intermediateVersion)
			}
		}
	} else {
		// Same major version - direct upgrade
		path = append(path, target)
	}
	
	return path, nil
}

// upgradeRules defines version-specific upgrade rules
type upgradeRules struct {
	fromVersion string
	toVersion   string
	preUpgrade  []string // Commands to run before upgrade
	postUpgrade []string // Commands to run after upgrade
	notes       string   // Upgrade notes
}

// getUpgradeRules returns specific upgrade rules for known version transitions
func (vm *versionManager) getUpgradeRules(component, fromVersion, toVersion string) *upgradeRules {
	// Define known upgrade rules for each component
	rules := map[string][]upgradeRules{
		"prometheus": {
			{
				fromVersion: "2.x",
				toVersion:   "3.x",
				notes:       "Major version upgrade: Review breaking changes in Prometheus 3.x",
			},
		},
		"grafana": {
			{
				fromVersion: "9.x",
				toVersion:   "10.x",
				notes:       "Major version upgrade: Database migration will be performed automatically",
			},
		},
		"loki": {
			{
				fromVersion: "2.x",
				toVersion:   "3.x",
				notes:       "Major version upgrade: Schema migration required",
			},
		},
		"tempo": {
			{
				fromVersion: "1.x",
				toVersion:   "2.x",
				notes:       "Major version upgrade: Configuration format has changed",
			},
		},
	}
	
	componentRules, exists := rules[component]
	if !exists {
		return nil
	}
	
	// Find matching rule
	for _, rule := range componentRules {
		if vm.matchesVersionPattern(fromVersion, rule.fromVersion) &&
			vm.matchesVersionPattern(toVersion, rule.toVersion) {
			return &rule
		}
	}
	
	return nil
}

// matchesVersionPattern checks if a version matches a pattern (e.g., "2.x")
func (vm *versionManager) matchesVersionPattern(version, pattern string) bool {
	if pattern == "" {
		return true
	}
	
	// Handle x.x pattern
	if strings.Contains(pattern, "x") {
		parts := strings.Split(pattern, ".")
		versionParts := strings.Split(version, ".")
		
		for i, part := range parts {
			if part == "x" {
				continue
			}
			if i >= len(versionParts) || part != versionParts[i] {
				return false
			}
		}
		return true
	}
	
	// Exact match
	return version == pattern
}
