package deprecation

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// DocumentationGenerator generates deprecation documentation
type DocumentationGenerator struct {
	registry *Registry
}

// NewDocumentationGenerator creates a new documentation generator
func NewDocumentationGenerator() *DocumentationGenerator {
	return &DocumentationGenerator{
		registry: GetRegistry(),
	}
}

// GenerateMarkdown generates comprehensive deprecation documentation in Markdown format
func (g *DocumentationGenerator) GenerateMarkdown(w io.Writer) error {
	// Header
	fmt.Fprintf(w, "# Deprecation Guide for Gunj Operator\n\n")
	fmt.Fprintf(w, "Last updated: %s\n\n", time.Now().Format("2006-01-02"))
	
	// Table of contents
	fmt.Fprintf(w, "## Table of Contents\n\n")
	fmt.Fprintf(w, "- [Overview](#overview)\n")
	fmt.Fprintf(w, "- [Deprecation Timeline](#deprecation-timeline)\n")
	fmt.Fprintf(w, "- [API Version Deprecations](#api-version-deprecations)\n")
	fmt.Fprintf(w, "- [Field Deprecations](#field-deprecations)\n")
	fmt.Fprintf(w, "- [Value Deprecations](#value-deprecations)\n")
	fmt.Fprintf(w, "- [Feature Deprecations](#feature-deprecations)\n")
	fmt.Fprintf(w, "- [Migration Examples](#migration-examples)\n")
	fmt.Fprintf(w, "- [Deprecation Policy](#deprecation-policy)\n\n")

	// Overview
	fmt.Fprintf(w, "## Overview\n\n")
	fmt.Fprintf(w, "This guide lists all deprecated features, fields, and values in the Gunj Operator. ")
	fmt.Fprintf(w, "Deprecations are categorized by severity:\n\n")
	fmt.Fprintf(w, "- 🚨 **Critical**: Will be removed in the next major version\n")
	fmt.Fprintf(w, "- ⚠️  **Warning**: Deprecated but supported for at least two more versions\n")
	fmt.Fprintf(w, "- ℹ️  **Info**: Deprecated for better alternatives but no removal planned\n\n")

	// Generate sections for each type
	g.generateTimeline(w)
	g.generateAPIVersionSection(w)
	g.generateFieldSection(w)
	g.generateValueSection(w)
	g.generateFeatureSection(w)
	g.generateMigrationExamples(w)
	g.generateDeprecationPolicy(w)

	return nil
}

// generateTimeline creates a timeline of deprecations
func (g *DocumentationGenerator) generateTimeline(w io.Writer) {
	fmt.Fprintf(w, "## Deprecation Timeline\n\n")

	// Collect all deprecations with removal dates
	type timelineEntry struct {
		date        time.Time
		version     string
		description string
		severity    DeprecationSeverity
	}

	var entries []timelineEntry

	g.registry.mu.RLock()
	for _, dep := range g.registry.deprecations {
		// Add deprecation date
		entries = append(entries, timelineEntry{
			date:        dep.Policy.DeprecatedSince,
			version:     dep.Policy.DeprecatedInVersion,
			description: fmt.Sprintf("Deprecated: %s", dep.Path),
			severity:    dep.Severity,
		})

		// Add removal date if set
		if dep.Policy.RemovalDate != nil {
			entries = append(entries, timelineEntry{
				date:        *dep.Policy.RemovalDate,
				version:     dep.Policy.RemovedInVersion,
				description: fmt.Sprintf("Removal planned: %s", dep.Path),
				severity:    SeverityCritical,
			})
		}
	}
	g.registry.mu.RUnlock()

	// Sort by date
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].date.Before(entries[j].date)
	})

	// Generate timeline table
	fmt.Fprintf(w, "| Date | Version | Event | Severity |\n")
	fmt.Fprintf(w, "|------|---------|-------|----------|\n")
	for _, entry := range entries {
		severity := g.getSeverityEmoji(entry.severity)
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n",
			entry.date.Format("2006-01-02"),
			entry.version,
			entry.description,
			severity)
	}
	fmt.Fprintf(w, "\n")
}

// generateAPIVersionSection generates documentation for API version deprecations
func (g *DocumentationGenerator) generateAPIVersionSection(w io.Writer) {
	fmt.Fprintf(w, "## API Version Deprecations\n\n")

	deprecations := g.getDeprecationsByType(APIVersionDeprecation)
	if len(deprecations) == 0 {
		fmt.Fprintf(w, "No API version deprecations at this time.\n\n")
		return
	}

	for _, dep := range deprecations {
		g.generateDeprecationEntry(w, dep)
	}
}

// generateFieldSection generates documentation for field deprecations
func (g *DocumentationGenerator) generateFieldSection(w io.Writer) {
	fmt.Fprintf(w, "## Field Deprecations\n\n")

	deprecations := g.getDeprecationsByType(FieldDeprecation)
	if len(deprecations) == 0 {
		fmt.Fprintf(w, "No field deprecations at this time.\n\n")
		return
	}

	// Group by API version
	byVersion := g.groupByAPIVersion(deprecations)
	for version, deps := range byVersion {
		fmt.Fprintf(w, "### %s\n\n", version)
		for _, dep := range deps {
			g.generateDeprecationEntry(w, dep)
		}
	}
}

// generateValueSection generates documentation for value deprecations
func (g *DocumentationGenerator) generateValueSection(w io.Writer) {
	fmt.Fprintf(w, "## Value Deprecations\n\n")

	deprecations := g.getDeprecationsByType(ValueDeprecation)
	if len(deprecations) == 0 {
		fmt.Fprintf(w, "No value deprecations at this time.\n\n")
		return
	}

	for _, dep := range deprecations {
		g.generateDeprecationEntry(w, dep)
	}
}

// generateFeatureSection generates documentation for feature deprecations
func (g *DocumentationGenerator) generateFeatureSection(w io.Writer) {
	fmt.Fprintf(w, "## Feature Deprecations\n\n")

	deprecations := g.getDeprecationsByType(FeatureDeprecation)
	if len(deprecations) == 0 {
		fmt.Fprintf(w, "No feature deprecations at this time.\n\n")
		return
	}

	for _, dep := range deprecations {
		g.generateDeprecationEntry(w, dep)
	}
}

// generateDeprecationEntry generates a single deprecation entry
func (g *DocumentationGenerator) generateDeprecationEntry(w io.Writer, dep *DeprecationInfo) {
	// Header with severity
	severity := g.getSeverityEmoji(dep.GetSeverity())
	fmt.Fprintf(w, "### %s %s\n\n", severity, dep.Path)

	// Basic info
	fmt.Fprintf(w, "**Status**: %s\n\n", dep.Message)
	
	if dep.Type == ValueDeprecation {
		fmt.Fprintf(w, "**Deprecated Value**: `%s`\n\n", dep.Value)
	}

	// Timeline
	fmt.Fprintf(w, "**Timeline**:\n")
	fmt.Fprintf(w, "- Deprecated in: %s (since %s)\n", 
		dep.Policy.DeprecatedInVersion,
		dep.Policy.DeprecatedSince.Format("2006-01-02"))
	fmt.Fprintf(w, "- Will be removed in: %s\n", dep.Policy.RemovedInVersion)
	if dep.Policy.RemovalDate != nil {
		fmt.Fprintf(w, "- Planned removal date: %s\n", 
			dep.Policy.RemovalDate.Format("2006-01-02"))
	}
	fmt.Fprintf(w, "\n")

	// Alternative
	if dep.AlternativePath != "" {
		fmt.Fprintf(w, "**Alternative**: Use `%s` instead\n\n", dep.AlternativePath)
	}

	// Migration guide
	if dep.MigrationGuide != "" {
		fmt.Fprintf(w, "**Migration Guide**:\n\n")
		fmt.Fprintf(w, "```yaml\n%s\n```\n\n", dep.MigrationGuide)
	}

	// Affected versions
	if len(dep.AffectedVersions) > 0 {
		fmt.Fprintf(w, "**Affected API Versions**: %s\n\n", 
			strings.Join(dep.AffectedVersions, ", "))
	}

	fmt.Fprintf(w, "---\n\n")
}

// generateMigrationExamples generates complete migration examples
func (g *DocumentationGenerator) generateMigrationExamples(w io.Writer) {
	fmt.Fprintf(w, "## Migration Examples\n\n")

	// Example 1: Complete v1alpha1 to v1beta1 migration
	fmt.Fprintf(w, "### Complete v1alpha1 to v1beta1 Migration\n\n")
	fmt.Fprintf(w, "Here's a complete example of migrating an ObservabilityPlatform from v1alpha1 to v1beta1:\n\n")
	
	fmt.Fprintf(w, "**Before (v1alpha1)**:\n")
	fmt.Fprintf(w, "```yaml\n")
	fmt.Fprintf(w, "apiVersion: observability.io/v1alpha1\n")
	fmt.Fprintf(w, "kind: ObservabilityPlatform\n")
	fmt.Fprintf(w, "metadata:\n")
	fmt.Fprintf(w, "  name: production\n")
	fmt.Fprintf(w, "spec:\n")
	fmt.Fprintf(w, "  monitoring:\n")
	fmt.Fprintf(w, "    prometheus:\n")
	fmt.Fprintf(w, "      enabled: true\n")
	fmt.Fprintf(w, "      version: v2.30.0\n")
	fmt.Fprintf(w, "  storage:\n")
	fmt.Fprintf(w, "    class: fast-ssd\n")
	fmt.Fprintf(w, "  tls:\n")
	fmt.Fprintf(w, "    manual:\n")
	fmt.Fprintf(w, "      cert: |\n")
	fmt.Fprintf(w, "        -----BEGIN CERTIFICATE-----\n")
	fmt.Fprintf(w, "        ...\n")
	fmt.Fprintf(w, "```\n\n")

	fmt.Fprintf(w, "**After (v1beta1)**:\n")
	fmt.Fprintf(w, "```yaml\n")
	fmt.Fprintf(w, "apiVersion: observability.io/v1beta1\n")
	fmt.Fprintf(w, "kind: ObservabilityPlatform\n")
	fmt.Fprintf(w, "metadata:\n")
	fmt.Fprintf(w, "  name: production\n")
	fmt.Fprintf(w, "spec:\n")
	fmt.Fprintf(w, "  components:  # Changed from 'monitoring'\n")
	fmt.Fprintf(w, "    prometheus:\n")
	fmt.Fprintf(w, "      enabled: true\n")
	fmt.Fprintf(w, "      version: v2.48.0  # Updated to supported version\n")
	fmt.Fprintf(w, "  storage:\n")
	fmt.Fprintf(w, "    storageClassName: fast-ssd  # Changed from 'class'\n")
	fmt.Fprintf(w, "  tls:\n")
	fmt.Fprintf(w, "    certManager:  # Changed from 'manual'\n")
	fmt.Fprintf(w, "      enabled: true\n")
	fmt.Fprintf(w, "      issuerRef:\n")
	fmt.Fprintf(w, "        name: letsencrypt-prod\n")
	fmt.Fprintf(w, "        kind: ClusterIssuer\n")
	fmt.Fprintf(w, "```\n\n")

	// Example 2: Using kubectl to check for deprecations
	fmt.Fprintf(w, "### Checking for Deprecations\n\n")
	fmt.Fprintf(w, "You can check your resources for deprecations using kubectl:\n\n")
	fmt.Fprintf(w, "```bash\n")
	fmt.Fprintf(w, "# Apply with dry-run to see warnings\n")
	fmt.Fprintf(w, "kubectl apply -f my-platform.yaml --dry-run=server\n\n")
	fmt.Fprintf(w, "# The output will include deprecation warnings:\n")
	fmt.Fprintf(w, "# Warning: spec.monitoring is deprecated, use spec.components instead\n")
	fmt.Fprintf(w, "# Warning: Prometheus version v2.30.0 is deprecated due to security vulnerabilities\n")
	fmt.Fprintf(w, "```\n\n")
}

// generateDeprecationPolicy explains the deprecation policy
func (g *DocumentationGenerator) generateDeprecationPolicy(w io.Writer) {
	fmt.Fprintf(w, "## Deprecation Policy\n\n")
	
	fmt.Fprintf(w, "The Gunj Operator follows these deprecation guidelines:\n\n")
	
	fmt.Fprintf(w, "### Version Support\n\n")
	fmt.Fprintf(w, "- **Alpha versions** (v1alpha1, v1alpha2, etc.): No compatibility guarantees\n")
	fmt.Fprintf(w, "- **Beta versions** (v1beta1, v1beta2, etc.): Compatible for at least 2 releases\n")
	fmt.Fprintf(w, "- **Stable versions** (v1, v2, etc.): Compatible for at least 3 releases\n\n")
	
	fmt.Fprintf(w, "### Deprecation Process\n\n")
	fmt.Fprintf(w, "1. **Announcement**: Deprecations are announced in release notes\n")
	fmt.Fprintf(w, "2. **Warning Period**: Deprecated features show warnings but continue to work\n")
	fmt.Fprintf(w, "3. **Migration Period**: At least 2 releases for beta, 3 for stable APIs\n")
	fmt.Fprintf(w, "4. **Removal**: Features are removed only in major version updates\n\n")
	
	fmt.Fprintf(w, "### Monitoring Deprecations\n\n")
	fmt.Fprintf(w, "You can monitor deprecation usage in your cluster:\n\n")
	fmt.Fprintf(w, "```bash\n")
	fmt.Fprintf(w, "# Check operator logs for deprecation warnings\n")
	fmt.Fprintf(w, "kubectl logs -n gunj-system deployment/gunj-operator | grep -i deprecat\n\n")
	fmt.Fprintf(w, "# Use kubectl deprecations (if available)\n")
	fmt.Fprintf(w, "kubectl deprecations\n")
	fmt.Fprintf(w, "```\n\n")
	
	fmt.Fprintf(w, "### Getting Help\n\n")
	fmt.Fprintf(w, "If you need help with migrations:\n\n")
	fmt.Fprintf(w, "- Check the [migration examples](#migration-examples) above\n")
	fmt.Fprintf(w, "- Join our [community Slack](https://gunj-operator.slack.com)\n")
	fmt.Fprintf(w, "- Open an issue on [GitHub](https://github.com/gunjanjp/gunj-operator/issues)\n")
	fmt.Fprintf(w, "- Email: gunjanjp@gmail.com\n\n")
}

// Helper methods

func (g *DocumentationGenerator) getDeprecationsByType(depType DeprecationType) []*DeprecationInfo {
	var result []*DeprecationInfo
	
	g.registry.mu.RLock()
	defer g.registry.mu.RUnlock()
	
	for _, dep := range g.registry.deprecations {
		if dep.Type == depType {
			result = append(result, dep)
		}
	}
	
	// Sort by path for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	
	return result
}

func (g *DocumentationGenerator) groupByAPIVersion(deprecations []*DeprecationInfo) map[string][]*DeprecationInfo {
	grouped := make(map[string][]*DeprecationInfo)
	
	for _, dep := range deprecations {
		for _, version := range dep.AffectedVersions {
			grouped[version] = append(grouped[version], dep)
		}
	}
	
	return grouped
}

func (g *DocumentationGenerator) getSeverityEmoji(severity DeprecationSeverity) string {
	switch severity {
	case SeverityCritical:
		return "🚨"
	case SeverityWarning:
		return "⚠️"
	default:
		return "ℹ️"
	}
}

// GenerateYAML generates deprecation information in YAML format
func (g *DocumentationGenerator) GenerateYAML(w io.Writer) error {
	fmt.Fprintf(w, "# Gunj Operator Deprecations\n")
	fmt.Fprintf(w, "# Generated: %s\n\n", time.Now().Format(time.RFC3339))
	
	fmt.Fprintf(w, "deprecations:\n")
	
	g.registry.mu.RLock()
	defer g.registry.mu.RUnlock()
	
	// Sort deprecations for consistent output
	var keys []string
	for key := range g.registry.deprecations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	for _, key := range keys {
		dep := g.registry.deprecations[key]
		fmt.Fprintf(w, "  - type: %s\n", dep.Type)
		fmt.Fprintf(w, "    path: %s\n", dep.Path)
		if dep.Value != "" {
			fmt.Fprintf(w, "    value: %s\n", dep.Value)
		}
		fmt.Fprintf(w, "    message: %s\n", dep.Message)
		if dep.AlternativePath != "" {
			fmt.Fprintf(w, "    alternative: %s\n", dep.AlternativePath)
		}
		fmt.Fprintf(w, "    severity: %s\n", dep.GetSeverity())
		fmt.Fprintf(w, "    policy:\n")
		fmt.Fprintf(w, "      deprecatedIn: %s\n", dep.Policy.DeprecatedInVersion)
		fmt.Fprintf(w, "      removedIn: %s\n", dep.Policy.RemovedInVersion)
		fmt.Fprintf(w, "      deprecatedSince: %s\n", dep.Policy.DeprecatedSince.Format(time.RFC3339))
		if dep.Policy.RemovalDate != nil {
			fmt.Fprintf(w, "      removalDate: %s\n", dep.Policy.RemovalDate.Format(time.RFC3339))
		}
		fmt.Fprintf(w, "    affectedVersions:\n")
		for _, version := range dep.AffectedVersions {
			fmt.Fprintf(w, "      - %s\n", version)
		}
		fmt.Fprintf(w, "\n")
	}
	
	return nil
}
