// Package version provides version constraint validation
package version

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Constraint represents a version constraint
type Constraint struct {
	Raw        string
	constraint *semver.Constraints
}

// NewConstraint creates a new version constraint
func NewConstraint(constraint string) (*Constraint, error) {
	if constraint == "" {
		constraint = "*" // Allow any version by default
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return nil, fmt.Errorf("invalid constraint %s: %w", constraint, err)
	}

	return &Constraint{
		Raw:        constraint,
		constraint: c,
	}, nil
}

// Check validates if a version satisfies the constraint
func (c *Constraint) Check(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("invalid version %s: %w", version, err)
	}

	return c.constraint.Check(v), nil
}

// Validator provides advanced version validation capabilities
type Validator struct {
	constraints map[string]*Constraint
	policies    []ValidationPolicy
}

// ValidationPolicy defines a validation rule
type ValidationPolicy struct {
	Name        string
	Description string
	Validate    func(version string) error
}

// NewValidator creates a new version validator
func NewValidator() *Validator {
	v := &Validator{
		constraints: make(map[string]*Constraint),
		policies:    []ValidationPolicy{},
	}

	// Add default policies
	v.addDefaultPolicies()

	return v
}

// AddConstraint adds a version constraint for a component
func (v *Validator) AddConstraint(component, constraint string) error {
	c, err := NewConstraint(constraint)
	if err != nil {
		return fmt.Errorf("failed to add constraint for %s: %w", component, err)
	}

	v.constraints[component] = c
	return nil
}

// ValidateVersion validates a version against constraints and policies
func (v *Validator) ValidateVersion(component, version string) error {
	// Check constraint
	if constraint, exists := v.constraints[component]; exists {
		valid, err := constraint.Check(version)
		if err != nil {
			return fmt.Errorf("constraint check failed: %w", err)
		}
		if !valid {
			return fmt.Errorf("version %s does not satisfy constraint %s", version, constraint.Raw)
		}
	}

	// Apply policies
	for _, policy := range v.policies {
		if err := policy.Validate(version); err != nil {
			return fmt.Errorf("policy %s failed: %w", policy.Name, err)
		}
	}

	return nil
}

// AddPolicy adds a custom validation policy
func (v *Validator) AddPolicy(policy ValidationPolicy) {
	v.policies = append(v.policies, policy)
}

// FindCompatibleVersions finds versions that satisfy constraints
func (v *Validator) FindCompatibleVersions(component string, availableVersions []string) ([]string, error) {
	var compatible []string

	for _, version := range availableVersions {
		if err := v.ValidateVersion(component, version); err == nil {
			compatible = append(compatible, version)
		}
	}

	return compatible, nil
}

// Private methods

func (v *Validator) addDefaultPolicies() {
	// No pre-release versions in production
	v.AddPolicy(ValidationPolicy{
		Name:        "no-prerelease",
		Description: "Prevent pre-release versions in production",
		Validate: func(version string) error {
			v, err := semver.NewVersion(version)
			if err != nil {
				return err
			}
			if v.Prerelease() != "" {
				return fmt.Errorf("pre-release versions not allowed: %s", version)
			}
			return nil
		},
	})

	// Minimum version policy
	v.AddPolicy(ValidationPolicy{
		Name:        "minimum-version",
		Description: "Enforce minimum version requirements",
		Validate: func(version string) error {
			// This is a placeholder - actual minimum versions would be component-specific
			return nil
		},
	})

	// Security patch policy
	v.AddPolicy(ValidationPolicy{
		Name:        "security-patches",
		Description: "Ensure security patches are applied",
		Validate: func(version string) error {
			// This would check against known vulnerable versions
			// For now, it's a placeholder
			return nil
		},
	})
}

// ConstraintSet manages multiple constraints
type ConstraintSet struct {
	constraints map[string]*Constraint
}

// NewConstraintSet creates a new constraint set
func NewConstraintSet() *ConstraintSet {
	return &ConstraintSet{
		constraints: make(map[string]*Constraint),
	}
}

// Add adds a constraint to the set
func (cs *ConstraintSet) Add(name, constraint string) error {
	c, err := NewConstraint(constraint)
	if err != nil {
		return err
	}

	cs.constraints[name] = c
	return nil
}

// CheckAll validates a version against all constraints
func (cs *ConstraintSet) CheckAll(version string) error {
	for name, constraint := range cs.constraints {
		valid, err := constraint.Check(version)
		if err != nil {
			return fmt.Errorf("constraint %s check failed: %w", name, err)
		}
		if !valid {
			return fmt.Errorf("version %s does not satisfy constraint %s: %s", 
				version, name, constraint.Raw)
		}
	}
	return nil
}

// ParseVersionRequirement parses a version requirement string
func ParseVersionRequirement(requirement string) (*Constraint, error) {
	// Handle different requirement formats
	requirement = strings.TrimSpace(requirement)
	
	// Convert common formats to semver constraints
	switch {
	case strings.HasPrefix(requirement, "v"):
		// Remove 'v' prefix for parsing
		requirement = requirement[1:]
	case requirement == "latest":
		requirement = "*"
	case requirement == "stable":
		// Exclude pre-releases
		requirement = ">0.0.0"
	}

	return NewConstraint(requirement)
}

// RecommendedVersions suggests versions based on stability and compatibility
func RecommendedVersions(availableVersions []string, count int) []string {
	if count <= 0 {
		count = 3
	}

	var stable []*semver.Version
	
	for _, v := range availableVersions {
		sv, err := semver.NewVersion(v)
		if err != nil {
			continue
		}
		
		// Only recommend stable versions (no pre-release)
		if sv.Prerelease() == "" {
			stable = append(stable, sv)
		}
	}

	// Sort by version (newest first)
	semver.Sort(stable)
	
	// Reverse for descending order
	for i := 0; i < len(stable)/2; i++ {
		j := len(stable) - 1 - i
		stable[i], stable[j] = stable[j], stable[i]
	}

	// Return top N versions
	var recommended []string
	for i := 0; i < count && i < len(stable); i++ {
		recommended = append(recommended, stable[i].Original())
	}

	return recommended
}
