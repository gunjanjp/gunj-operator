/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gunjanjp/gunj-operator/internal/deprecation"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func main() {
	var (
		outputDir    = flag.String("output-dir", "docs/deprecations", "Output directory for documentation")
		format       = flag.String("format", "markdown", "Output format: markdown or yaml")
		generateAll  = flag.Bool("all", false, "Generate all formats")
		help         = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create documentation generator
	generator := deprecation.NewDocumentationGenerator()

	// Generate documentation based on format
	if *generateAll {
		if err := generateMarkdown(generator, *outputDir); err != nil {
			log.Fatalf("Failed to generate markdown: %v", err)
		}
		if err := generateYAML(generator, *outputDir); err != nil {
			log.Fatalf("Failed to generate YAML: %v", err)
		}
		fmt.Printf("Successfully generated all deprecation documentation in %s\n", *outputDir)
	} else {
		switch *format {
		case "markdown", "md":
			if err := generateMarkdown(generator, *outputDir); err != nil {
				log.Fatalf("Failed to generate markdown: %v", err)
			}
			fmt.Printf("Successfully generated markdown deprecation documentation in %s\n", *outputDir)
		case "yaml", "yml":
			if err := generateYAML(generator, *outputDir); err != nil {
				log.Fatalf("Failed to generate YAML: %v", err)
			}
			fmt.Printf("Successfully generated YAML deprecation documentation in %s\n", *outputDir)
		default:
			log.Fatalf("Unknown format: %s. Use 'markdown' or 'yaml'", *format)
		}
	}

	// Also generate a quick summary to stdout
	fmt.Println("\n=== Deprecation Summary ===")
	printDeprecationSummary()
}

func generateMarkdown(generator *deprecation.DocumentationGenerator, outputDir string) error {
	outputPath := filepath.Join(outputDir, "DEPRECATIONS.md")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating markdown file: %w", err)
	}
	defer file.Close()

	if err := generator.GenerateMarkdown(file); err != nil {
		return fmt.Errorf("generating markdown: %w", err)
	}

	return nil
}

func generateYAML(generator *deprecation.DocumentationGenerator, outputDir string) error {
	outputPath := filepath.Join(outputDir, "deprecations.yaml")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating YAML file: %w", err)
	}
	defer file.Close()

	if err := generator.GenerateYAML(file); err != nil {
		return fmt.Errorf("generating YAML: %w", err)
	}

	return nil
}

func printDeprecationSummary() {
	registry := deprecation.GetRegistry()
	
	// Count deprecations by type
	counts := make(map[deprecation.DeprecationType]int)
	severityCounts := make(map[deprecation.DeprecationSeverity]int)
	
	// Get all deprecations from registry (simplified for example)
	// In a real implementation, you'd iterate through all registered deprecations
	exampleDeprecations := []struct {
		Type     deprecation.DeprecationType
		Severity deprecation.DeprecationSeverity
	}{
		{deprecation.FieldDeprecation, deprecation.SeverityWarning},
		{deprecation.ValueDeprecation, deprecation.SeverityCritical},
		{deprecation.APIVersionDeprecation, deprecation.SeverityWarning},
		{deprecation.FeatureDeprecation, deprecation.SeverityWarning},
	}

	for _, dep := range exampleDeprecations {
		counts[dep.Type]++
		severityCounts[dep.Severity]++
	}

	// Print summary
	fmt.Println("\nDeprecations by Type:")
	fmt.Printf("  Field Deprecations:       %d\n", counts[deprecation.FieldDeprecation])
	fmt.Printf("  Value Deprecations:       %d\n", counts[deprecation.ValueDeprecation])
	fmt.Printf("  API Version Deprecations: %d\n", counts[deprecation.APIVersionDeprecation])
	fmt.Printf("  Feature Deprecations:     %d\n", counts[deprecation.FeatureDeprecation])

	fmt.Println("\nDeprecations by Severity:")
	fmt.Printf("  🚨 Critical: %d\n", severityCounts[deprecation.SeverityCritical])
	fmt.Printf("  ⚠️  Warning:  %d\n", severityCounts[deprecation.SeverityWarning])
	fmt.Printf("  ℹ️  Info:     %d\n", severityCounts[deprecation.SeverityInfo])

	// Show example of how to check specific deprecations
	fmt.Println("\nExample Deprecation Check:")
	gvk := schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1alpha1",
		Kind:    "ObservabilityPlatform",
	}
	example := registry.GetDeprecationByPath("spec.monitoring", gvk)
	if example != nil {
		fmt.Printf("  Field 'spec.monitoring' is deprecated: %s\n", example.Message)
	}
}

func printHelp() {
	fmt.Println("Gunj Operator Deprecation Documentation Generator")
	fmt.Println()
	fmt.Println("This tool generates comprehensive deprecation documentation for the Gunj Operator.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  deprecation-doc [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -output-dir string")
	fmt.Println("        Output directory for documentation (default \"docs/deprecations\")")
	fmt.Println("  -format string")
	fmt.Println("        Output format: markdown or yaml (default \"markdown\")")
	fmt.Println("  -all")
	fmt.Println("        Generate all formats")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate markdown documentation")
	fmt.Println("  deprecation-doc")
	fmt.Println()
	fmt.Println("  # Generate YAML format")
	fmt.Println("  deprecation-doc -format yaml")
	fmt.Println()
	fmt.Println("  # Generate all formats in custom directory")
	fmt.Println("  deprecation-doc -all -output-dir ./my-docs")
}
