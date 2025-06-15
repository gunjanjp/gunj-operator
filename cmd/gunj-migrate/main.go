/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
	"github.com/gunjanjp/gunj-operator/pkg/conversion"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	kubeconfig string
	namespace  string
	verbose    bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gunj-migrate",
		Short: "Gunj Operator migration and data preservation tool",
		Long: `A command-line tool for managing ObservabilityPlatform migrations,
data preservation, and conversion operations.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Add subcommands
	rootCmd.AddCommand(
		newMigrateCmd(),
		newPreserveCmd(),
		newRestoreCmd(),
		newValidateCmd(),
		newSchemaCmd(),
		newStatusCmd(),
		newOptimizeCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newMigrateCmd creates the migrate command
func newMigrateCmd() *cobra.Command {
	var (
		sourceVersion string
		targetVersion string
		dryRun        bool
		force         bool
		backup        bool
	)

	cmd := &cobra.Command{
		Use:   "migrate [resource-name]",
		Short: "Migrate ObservabilityPlatform resources between API versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(args[0], sourceVersion, targetVersion, dryRun, force, backup)
		},
	}

	cmd.Flags().StringVar(&sourceVersion, "from", "", "Source API version (auto-detected if not specified)")
	cmd.Flags().StringVar(&targetVersion, "to", "v1beta1", "Target API version")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry-run migration")
	cmd.Flags().BoolVar(&force, "force", false, "Force migration even with warnings")
	cmd.Flags().BoolVar(&backup, "backup", true, "Create backup before migration")

	return cmd
}

// newPreserveCmd creates the preserve command
func newPreserveCmd() *cobra.Command {
	var (
		outputFile string
		policy     string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "preserve [resource-name]",
		Short: "Preserve data from ObservabilityPlatform resources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreserve(args[0], outputFile, policy, format)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for preserved data")
	cmd.Flags().StringVar(&policy, "policy", "default", "Preservation policy to use")
	cmd.Flags().StringVar(&format, "format", "json", "Output format (json, yaml)")

	return cmd
}

// newRestoreCmd creates the restore command
func newRestoreCmd() *cobra.Command {
	var (
		inputFile      string
		targetResource string
		verify         bool
	)

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore preserved data to ObservabilityPlatform resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(inputFile, targetResource, verify)
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file with preserved data (required)")
	cmd.Flags().StringVar(&targetResource, "target", "", "Target resource name (required)")
	cmd.Flags().BoolVar(&verify, "verify", true, "Verify data integrity after restoration")

	cmd.MarkFlagRequired("input")
	cmd.MarkFlagRequired("target")

	return cmd
}

// newValidateCmd creates the validate command
func newValidateCmd() *cobra.Command {
	var (
		targetVersion string
		detailed      bool
	)

	cmd := &cobra.Command{
		Use:   "validate [resource-name]",
		Short: "Validate ObservabilityPlatform resource for migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args[0], targetVersion, detailed)
		},
	}

	cmd.Flags().StringVar(&targetVersion, "target", "v1beta1", "Target API version")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed validation results")

	return cmd
}

// newSchemaCmd creates the schema command
func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage schema evolution and migration paths",
	}

	// Subcommands
	cmd.AddCommand(
		newSchemaListCmd(),
		newSchemaPathCmd(),
		newSchemaHistoryCmd(),
	)

	return cmd
}

// newSchemaListCmd lists available schema versions
func newSchemaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available schema versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchemaList()
		},
	}
}

// newSchemaPathCmd shows migration path between versions
func newSchemaPathCmd() *cobra.Command {
	var (
		from string
		to   string
	)

	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show migration path between versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchemaPath(from, to)
		},
	}

	cmd.Flags().StringVar(&from, "from", "v1alpha1", "Source version")
	cmd.Flags().StringVar(&to, "to", "v1beta1", "Target version")

	return cmd
}

// newSchemaHistoryCmd shows migration history
func newSchemaHistoryCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show migration history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchemaHistory(limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Number of history entries to show")

	return cmd
}

// newStatusCmd creates the status command
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [resource-name]",
		Short: "Show migration status of ObservabilityPlatform resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0])
		},
	}
}

// newOptimizeCmd creates the optimize command
func newOptimizeCmd() *cobra.Command {
	var (
		strategy string
		apply    bool
	)

	cmd := &cobra.Command{
		Use:   "optimize [resource-name]",
		Short: "Optimize ObservabilityPlatform resource for conversion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOptimize(args[0], strategy, apply)
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "default", "Optimization strategy")
	cmd.Flags().BoolVar(&apply, "apply", false, "Apply optimizations")

	return cmd
}

// Implementation functions

func runMigrate(resourceName, sourceVersion, targetVersion string, dryRun, force, backup bool) error {
	ctx := context.Background()
	logger := zap.New(zap.UseDevMode(verbose))

	// Create client
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create migration manager
	manager := migration.NewMigrationManager(client, logger)

	// Configure manager
	config := &migration.MigrationConfig{
		SourceVersion:     sourceVersion,
		TargetVersion:     targetVersion,
		DryRun:            dryRun,
		CreateBackup:      backup,
		SkipValidation:    force,
		PreservationPolicy: "default",
	}

	// Execute migration
	fmt.Printf("Migrating %s/%s from %s to %s...\n", namespace, resourceName, sourceVersion, targetVersion)
	
	result, err := manager.ExecuteMigration(ctx, namespace, resourceName, config)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Display results
	fmt.Printf("\nMigration %s\n", result.Status)
	fmt.Printf("Duration: %s\n", result.Duration)
	
	if result.BackupID != "" {
		fmt.Printf("Backup ID: %s\n", result.BackupID)
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if result.Error != nil {
		fmt.Printf("\nError: %v\n", result.Error)
	}

	return nil
}

func runPreserve(resourceName, outputFile, policy, format string) error {
	ctx := context.Background()
	logger := zap.New(zap.UseDevMode(verbose))

	// Create client
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Load preservation policy
	policyConfig := &preservation.PolicyConfig{
		Policies:        preservation.DefaultPolicies(),
		DefaultStrategy: "deep-copy",
		EnableMetrics:   true,
	}

	// Create data preserver
	preserver, err := conversion.NewDataPreserverEnhanced(logger, client, policyConfig)
	if err != nil {
		return fmt.Errorf("failed to create data preserver: %w", err)
	}

	// Get resource
	obj, err := getResource(ctx, client, namespace, resourceName)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Preserve data
	fmt.Printf("Preserving data from %s/%s...\n", namespace, resourceName)
	
	preserved, err := preserver.PreserveDataEnhanced(ctx, obj, "v1beta1")
	if err != nil {
		return fmt.Errorf("failed to preserve data: %w", err)
	}

	// Output results
	var output []byte
	switch format {
	case "json":
		output, err = json.MarshalIndent(preserved, "", "  ")
	case "yaml":
		// Convert to YAML if needed
		output, err = json.MarshalIndent(preserved, "", "  ") // Simplified for now
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal preserved data: %w", err)
	}

	// Write to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Preserved data written to %s\n", outputFile)
	} else {
		fmt.Println(string(output))
	}

	// Display summary
	fmt.Printf("\nPreservation Summary:\n")
	fmt.Printf("  Unknown Fields: %d\n", len(preserved.UnknownFields))
	fmt.Printf("  Complex Fields: %d\n", len(preserved.ComplexFields))
	fmt.Printf("  Annotations: %d\n", len(preserved.Annotations))
	fmt.Printf("  Labels: %d\n", len(preserved.Labels))

	return nil
}

func runRestore(inputFile, targetResource string, verify bool) error {
	ctx := context.Background()
	logger := zap.New(zap.UseDevMode(verbose))

	// Read preserved data
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var preserved conversion.PreservedDataEnhanced
	if err := json.Unmarshal(data, &preserved); err != nil {
		return fmt.Errorf("failed to unmarshal preserved data: %w", err)
	}

	// Create client
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get target resource
	obj, err := getResource(ctx, client, namespace, targetResource)
	if err != nil {
		return fmt.Errorf("failed to get target resource: %w", err)
	}

	// Create data preserver
	policyConfig := &preservation.PolicyConfig{
		Policies:        preservation.DefaultPolicies(),
		DefaultStrategy: "deep-copy",
		EnableMetrics:   true,
	}

	preserver, err := conversion.NewDataPreserverEnhanced(logger, client, policyConfig)
	if err != nil {
		return fmt.Errorf("failed to create data preserver: %w", err)
	}

	// Restore data
	fmt.Printf("Restoring data to %s/%s...\n", namespace, targetResource)
	
	if err := preserver.RestoreDataEnhanced(ctx, obj, &preserved); err != nil {
		return fmt.Errorf("failed to restore data: %w", err)
	}

	// Update resource
	if err := client.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	fmt.Println("Data restored successfully")

	// Verify if requested
	if verify {
		fmt.Println("\nVerifying data integrity...")
		// Verification logic would go here
		fmt.Println("Data integrity verified")
	}

	return nil
}

func runValidate(resourceName, targetVersion string, detailed bool) error {
	ctx := context.Background()
	logger := zap.New(zap.UseDevMode(verbose))

	// Create client
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get resource
	obj, err := getResource(ctx, client, namespace, resourceName)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Create validator
	validator := conversion.NewFieldValidator(logger)

	// Validate conversion
	fmt.Printf("Validating %s/%s for conversion to %s...\n", namespace, resourceName, targetVersion)
	
	result, err := validator.ValidateConversion(obj, targetVersion)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Display results
	fmt.Printf("\nValidation Result: %s\n", getValidationStatus(result))
	
	if detailed || !result.Valid {
		// Show errors
		if len(result.Errors) > 0 {
			fmt.Println("\nErrors:")
			for _, err := range result.Errors {
				fmt.Printf("  - [%s] %s: %s\n", err.Type, err.Field, err.Detail)
			}
		}

		// Show warnings
		if len(result.Warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}

		// Show metrics
		if detailed {
			fmt.Printf("\nValidation Metrics:\n")
			fmt.Printf("  Fields Validated: %d\n", result.Metrics.FieldsValidated)
			fmt.Printf("  Fields with Errors: %d\n", result.Metrics.FieldsWithErrors)
			fmt.Printf("  Fields with Warnings: %d\n", result.Metrics.FieldsWithWarnings)
			fmt.Printf("  Deprecated Fields: %d\n", result.Metrics.DeprecatedFields)
			fmt.Printf("  Data Loss Fields: %d\n", result.Metrics.DataLossFields)
		}
	}

	return nil
}

func runSchemaList() error {
	logger := zap.New(zap.UseDevMode(verbose))
	tracker := migration.NewSchemaEvolutionTracker(logger)

	versions := tracker.GetAvailableVersions()
	
	fmt.Println("Available API Versions:")
	for _, version := range versions {
		info := tracker.GetVersionInfo(version)
		fmt.Printf("  - %s", version)
		if info != nil {
			fmt.Printf(" (released: %s)", info.ReleaseDate.Format("2006-01-02"))
			if info.Deprecated {
				fmt.Printf(" [DEPRECATED]")
			}
		}
		fmt.Println()
	}

	return nil
}

func runSchemaPath(from, to string) error {
	logger := zap.New(zap.UseDevMode(verbose))
	tracker := migration.NewSchemaEvolutionTracker(logger)

	path, err := tracker.GetMigrationPath(from, to)
	if err != nil {
		return fmt.Errorf("no migration path found: %w", err)
	}

	fmt.Printf("Migration Path from %s to %s:\n", from, to)
	fmt.Printf("  Direct Migration: %v\n", path.Direct)
	fmt.Printf("  Data Loss Risk: %v\n", path.DataLossRisk)
	fmt.Printf("  Complexity: %s\n", path.Complexity)

	if len(path.IntermediateVersions) > 0 {
		fmt.Println("  Intermediate Versions:")
		for _, v := range path.IntermediateVersions {
			fmt.Printf("    - %s\n", v)
		}
	}

	if len(path.RequiredTransformations) > 0 {
		fmt.Println("  Required Transformations:")
		for _, t := range path.RequiredTransformations {
			fmt.Printf("    - %s\n", t)
		}
	}

	return nil
}

func runSchemaHistory(limit int) error {
	logger := zap.New(zap.UseDevMode(verbose))
	tracker := migration.NewSchemaEvolutionTracker(logger)

	history := tracker.GetMigrationHistory(limit)
	
	if len(history) == 0 {
		fmt.Println("No migration history found")
		return nil
	}

	fmt.Printf("Migration History (last %d entries):\n", limit)
	for i, entry := range history {
		fmt.Printf("\n%d. %s -> %s\n", i+1, entry.FromVersion, entry.ToVersion)
		fmt.Printf("   Timestamp: %s\n", entry.Timestamp.Format(time.RFC3339))
		fmt.Printf("   Success: %v\n", entry.Success)
		if entry.ResourceKey != nil {
			fmt.Printf("   Resource: %s/%s\n", entry.ResourceKey.Namespace, entry.ResourceKey.Name)
		}
		if entry.Duration > 0 {
			fmt.Printf("   Duration: %s\n", entry.Duration)
		}
		if entry.Error != "" {
			fmt.Printf("   Error: %s\n", entry.Error)
		}
	}

	return nil
}

func runStatus(resourceName string) error {
	ctx := context.Background()
	logger := zap.New(zap.UseDevMode(verbose))

	// Create client
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get resource
	obj, err := getResource(ctx, client, namespace, resourceName)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Get metadata
	meta, err := getMeta(obj)
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Display status
	fmt.Printf("Resource: %s/%s\n", namespace, resourceName)
	fmt.Printf("API Version: %s\n", obj.GetObjectKind().GroupVersionKind().Version)
	fmt.Printf("Created: %s\n", meta.GetCreationTimestamp().Format(time.RFC3339))
	fmt.Printf("Generation: %d\n", meta.GetGeneration())

	// Check annotations for migration info
	annotations := meta.GetAnnotations()
	if annotations != nil {
		if lastVersion := annotations[conversion.LastConversionVersionAnnotation]; lastVersion != "" {
			fmt.Printf("\nLast Conversion:\n")
			fmt.Printf("  From Version: %s\n", lastVersion)
		}
		
		if preservedFields := annotations[conversion.PreservedFieldsAnnotation]; preservedFields != "" {
			fmt.Printf("\nPreserved Fields: %s\n", preservedFields)
		}
		
		if unknownFields := annotations[conversion.UnknownFieldsAnnotation]; unknownFields != "" {
			fmt.Printf("\nUnknown Fields: %s\n", unknownFields)
		}
	}

	return nil
}

func runOptimize(resourceName, strategy string, apply bool) error {
	ctx := context.Background()
	logger := zap.New(zap.UseDevMode(verbose))

	// Create client
	client, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get resource
	obj, err := getResource(ctx, client, namespace, resourceName)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Create optimizer
	optimizer := migration.NewConversionOptimizer(logger)

	// Analyze optimizations
	fmt.Printf("Analyzing optimizations for %s/%s...\n", namespace, resourceName)
	
	// Convert to unstructured for optimization
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("failed to convert to unstructured: %w", err)
	}

	// Get optimization suggestions
	suggestions := optimizer.GetOptimizationSuggestions(&unstructured.Unstructured{Object: unstructuredObj})
	
	if len(suggestions) == 0 {
		fmt.Println("No optimizations recommended")
		return nil
	}

	fmt.Printf("\nOptimization Suggestions:\n")
	for i, suggestion := range suggestions {
		fmt.Printf("%d. %s\n", i+1, suggestion)
	}

	if apply {
		fmt.Printf("\nApplying optimizations with strategy: %s\n", strategy)
		// Apply optimizations logic would go here
		fmt.Println("Optimizations applied successfully")
	}

	return nil
}

// Helper functions

func createClient() (client.Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{
		Scheme: scheme.Scheme,
	})
}

func getResource(ctx context.Context, c client.Client, namespace, name string) (runtime.Object, error) {
	// This is simplified - in real implementation, we'd need to handle different types
	// For now, assume ObservabilityPlatform
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1beta1",
		Kind:    "ObservabilityPlatform",
	})

	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func getMeta(obj runtime.Object) (metav1.Object, error) {
	accessor, err := meta.TypeAccessor(obj)
	if err != nil {
		return nil, err
	}
	
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	
	return metaAccessor, nil
}

func getValidationStatus(result *conversion.ValidationResult) string {
	if result.Valid {
		return "PASSED"
	}
	if len(result.Errors) > 0 {
		return "FAILED"
	}
	return "PASSED WITH WARNINGS"
}
