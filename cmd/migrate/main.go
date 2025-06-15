/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	observabilityv1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
)

var (
	targetVersion   string
	namespace       string
	allNamespaces   bool
	dryRun          bool
	batchSize       int
	maxConcurrent   int
	reportFormat    string
	outputFile      string
	enableOptimization bool
	progressInterval time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gunj-migrate",
		Short: "Gunj Operator Migration Tool",
		Long: `A tool for migrating ObservabilityPlatform resources between API versions.
		
This tool helps you migrate your ObservabilityPlatform resources from one API version
to another, with support for batch processing, dry-run mode, and detailed reporting.`,
	}
	
	// Add subcommands
	rootCmd.AddCommand(
		newMigrateCmd(),
		newStatusCmd(),
		newReportCmd(),
		newAnalyzeCmd(),
		newRollbackCmd(),
	)
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newMigrateCmd creates the migrate command
func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [resource-name]",
		Short: "Migrate ObservabilityPlatform resources to a new API version",
		Long: `Migrate one or more ObservabilityPlatform resources to a new API version.
		
Examples:
  # Migrate a single resource
  gunj-migrate migrate my-platform --target-version v1beta1 --namespace default
  
  # Migrate all resources in a namespace
  gunj-migrate migrate --target-version v1beta1 --namespace monitoring
  
  # Migrate all resources in all namespaces
  gunj-migrate migrate --target-version v1beta1 --all-namespaces
  
  # Dry-run mode to preview changes
  gunj-migrate migrate --target-version v1beta1 --namespace default --dry-run`,
		RunE: runMigrate,
	}
	
	// Add flags
	cmd.Flags().StringVar(&targetVersion, "target-version", "v1beta1", "Target API version")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to migrate resources from")
	cmd.Flags().BoolVar(&allNamespaces, "all-namespaces", false, "Migrate resources in all namespaces")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration without applying changes")
	cmd.Flags().IntVar(&batchSize, "batch-size", 10, "Number of resources to process in each batch")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 5, "Maximum concurrent migrations")
	cmd.Flags().BoolVar(&enableOptimization, "enable-optimization", true, "Enable conversion optimizations")
	cmd.Flags().DurationVar(&progressInterval, "progress-interval", 5*time.Second, "Progress report interval")
	
	return cmd
}

// newStatusCmd creates the status command
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [task-id]",
		Short: "Check migration status",
		Long: `Check the status of ongoing or completed migrations.
		
Examples:
  # List all active migrations
  gunj-migrate status
  
  # Check specific migration status
  gunj-migrate status migrate-12345`,
		RunE: runStatus,
	}
	
	return cmd
}

// newReportCmd creates the report command
func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report [task-id]",
		Short: "Generate migration report",
		Long: `Generate detailed migration reports in various formats.
		
Examples:
  # Generate JSON report
  gunj-migrate report migrate-12345 --format json --output report.json
  
  # Generate HTML report
  gunj-migrate report migrate-12345 --format html --output report.html`,
		RunE: runReport,
	}
	
	cmd.Flags().StringVar(&reportFormat, "format", "json", "Report format (json, html, text)")
	cmd.Flags().StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	
	return cmd
}

// newAnalyzeCmd creates the analyze command
func newAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze resources for migration readiness",
		Long: `Analyze ObservabilityPlatform resources to determine migration readiness.
		
This command will:
- Identify resources that need migration
- Check for potential compatibility issues
- Estimate migration complexity
- Provide recommendations`,
		RunE: runAnalyze,
	}
	
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to analyze")
	cmd.Flags().BoolVar(&allNamespaces, "all-namespaces", false, "Analyze resources in all namespaces")
	cmd.Flags().StringVar(&targetVersion, "target-version", "v1beta1", "Target API version for analysis")
	
	return cmd
}

// newRollbackCmd creates the rollback command
func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback [task-id]",
		Short: "Rollback a migration",
		Long: `Rollback a failed or in-progress migration to the previous state.
		
Examples:
  # Rollback a specific migration
  gunj-migrate rollback migrate-12345`,
		RunE: runRollback,
	}
	
	return cmd
}

// runMigrate executes the migrate command
func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("migrate")
	
	// Create Kubernetes client
	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Register schemes
	if err := observabilityv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add v1alpha1 scheme: %w", err)
	}
	if err := observabilityv1beta1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add v1beta1 scheme: %w", err)
	}
	
	// Create migration manager
	migrationConfig := migration.MigrationConfig{
		MaxConcurrentMigrations: maxConcurrent,
		BatchSize:               batchSize,
		RetryAttempts:           3,
		RetryInterval:           5 * time.Second,
		EnableOptimizations:     enableOptimization,
		DryRun:                  dryRun,
		ProgressReportInterval:  progressInterval,
	}
	
	migrationManager := migration.NewMigrationManager(k8sClient, scheme.Scheme, logger, migrationConfig)
	
	// Determine resources to migrate
	resources, err := getResourcesToMigrate(ctx, k8sClient, args)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}
	
	if len(resources) == 0 {
		fmt.Println("No resources found to migrate")
		return nil
	}
	
	// Display migration plan
	fmt.Printf("Migration Plan:\n")
	fmt.Printf("  Target Version: %s\n", targetVersion)
	fmt.Printf("  Resources: %d\n", len(resources))
	fmt.Printf("  Dry Run: %v\n", dryRun)
	fmt.Printf("  Batch Size: %d\n", batchSize)
	fmt.Println()
	
	// Confirm if not dry-run
	if !dryRun {
		fmt.Print("Proceed with migration? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Migration cancelled")
			return nil
		}
	}
	
	// Perform migration
	if len(resources) == 1 {
		// Single resource migration
		fmt.Printf("Migrating %s/%s...\n", resources[0].Namespace, resources[0].Name)
		if err := migrationManager.MigrateResource(ctx, resources[0], targetVersion); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Migration completed successfully")
	} else {
		// Batch migration
		fmt.Printf("Starting batch migration of %d resources...\n", len(resources))
		task, err := migrationManager.MigrateBatch(ctx, resources, targetVersion)
		if err != nil {
			return fmt.Errorf("batch migration failed: %w", err)
		}
		
		// Monitor progress
		if err := monitorMigration(migrationManager, task.ID); err != nil {
			return fmt.Errorf("error monitoring migration: %w", err)
		}
	}
	
	return nil
}

// runStatus executes the status command
func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("status")
	
	// Create Kubernetes client
	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Register schemes
	if err := observabilityv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add v1alpha1 scheme: %w", err)
	}
	if err := observabilityv1beta1.AddToScheme(scheme.Scheme); err != nil {
		return fmt.Errorf("failed to add v1beta1 scheme: %w", err)
	}
	
	// Create migration manager
	migrationConfig := migration.MigrationConfig{}
	migrationManager := migration.NewMigrationManager(k8sClient, scheme.Scheme, logger, migrationConfig)
	
	if len(args) == 0 {
		// List all active migrations
		tasks := migrationManager.ListActiveMigrations()
		if len(tasks) == 0 {
			fmt.Println("No active migrations found")
			return nil
		}
		
		fmt.Printf("Active Migrations:\n\n")
		for _, task := range tasks {
			fmt.Printf("Task ID: %s\n", task.ID)
			fmt.Printf("  Status: %s\n", task.Status)
			fmt.Printf("  Target Version: %s\n", task.TargetVersion)
			fmt.Printf("  Progress: %d/%d\n", 
				task.Progress.MigratedResources+task.Progress.FailedResources+task.Progress.SkippedResources,
				task.Progress.TotalResources)
			fmt.Printf("  Started: %s\n", task.StartTime.Format(time.RFC3339))
			fmt.Println()
		}
	} else {
		// Get specific migration status
		taskID := args[0]
		task, err := migrationManager.GetMigrationStatus(taskID)
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}
		
		displayMigrationStatus(task)
	}
	
	return nil
}

// runReport executes the report command
func runReport(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}
	
	ctx := context.Background()
	taskID := args[0]
	
	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("report")
	
	// Create status reporter
	reporter := migration.NewMigrationStatusReporter(logger)
	
	// Generate report
	var output string
	var err error
	
	switch reportFormat {
	case "json":
		data, err := reporter.ExportReportJSON(taskID)
		if err != nil {
			return fmt.Errorf("failed to generate JSON report: %w", err)
		}
		output = string(data)
		
	case "html":
		output, err = reporter.GenerateHTMLReport(taskID)
		if err != nil {
			return fmt.Errorf("failed to generate HTML report: %w", err)
		}
		
	case "text":
		report, err := reporter.GetReport(taskID)
		if err != nil {
			return fmt.Errorf("failed to get report: %w", err)
		}
		output = formatTextReport(report)
		
	default:
		return fmt.Errorf("unsupported report format: %s", reportFormat)
	}
	
	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Report written to %s\n", outputFile)
	} else {
		fmt.Println(output)
	}
	
	return nil
}

// runAnalyze executes the analyze command
func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("analyze")
	
	// Create Kubernetes client
	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Get resources to analyze
	resources, err := getResourcesToMigrate(ctx, k8sClient, args)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}
	
	if len(resources) == 0 {
		fmt.Println("No resources found to analyze")
		return nil
	}
	
	// Create schema evolution tracker
	tracker := migration.NewSchemaEvolutionTracker(logger)
	
	fmt.Printf("Migration Analysis Report\n")
	fmt.Printf("========================\n\n")
	fmt.Printf("Target Version: %s\n", targetVersion)
	fmt.Printf("Resources Found: %d\n\n", len(resources))
	
	// Analyze each resource
	var needsMigration int
	var warnings []string
	
	for _, resource := range resources {
		// Get current resource
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "observability.io",
			Kind:    "ObservabilityPlatform",
		})
		
		if err := k8sClient.Get(ctx, resource, u); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to get %s/%s: %v", 
				resource.Namespace, resource.Name, err))
			continue
		}
		
		currentVersion := strings.TrimPrefix(u.GetAPIVersion(), "observability.io/")
		if currentVersion != targetVersion {
			needsMigration++
			
			// Get migration path
			path, err := tracker.GetMigrationPath(currentVersion, targetVersion)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s/%s: No migration path from %s to %s",
					resource.Namespace, resource.Name, currentVersion, targetVersion))
			} else {
				if path.DataLossRisk {
					warnings = append(warnings, fmt.Sprintf("%s/%s: Migration may result in data loss",
						resource.Namespace, resource.Name))
				}
				if path.RequiresManual {
					warnings = append(warnings, fmt.Sprintf("%s/%s: Manual intervention required",
						resource.Namespace, resource.Name))
				}
			}
		}
	}
	
	// Display results
	fmt.Printf("Resources Requiring Migration: %d\n", needsMigration)
	fmt.Printf("Resources Already at Target Version: %d\n\n", len(resources)-needsMigration)
	
	if len(warnings) > 0 {
		fmt.Printf("Warnings:\n")
		for _, warning := range warnings {
			fmt.Printf("  - %s\n", warning)
		}
		fmt.Println()
	}
	
	// Migration complexity assessment
	if needsMigration > 0 {
		fmt.Printf("Migration Complexity Assessment:\n")
		if needsMigration < 10 {
			fmt.Printf("  - Low: Small number of resources\n")
		} else if needsMigration < 50 {
			fmt.Printf("  - Medium: Moderate number of resources\n")
		} else {
			fmt.Printf("  - High: Large number of resources\n")
		}
		
		fmt.Printf("\nRecommendations:\n")
		fmt.Printf("  - Run migration in dry-run mode first\n")
		fmt.Printf("  - Backup resources before migration\n")
		fmt.Printf("  - Monitor migration progress closely\n")
		if needsMigration > 50 {
			fmt.Printf("  - Consider migrating in smaller batches\n")
		}
	}
	
	return nil
}

// runRollback executes the rollback command
func runRollback(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}
	
	ctx := context.Background()
	taskID := args[0]
	
	// Setup logging
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	logger := ctrl.Log.WithName("rollback")
	
	// Create Kubernetes client
	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	// Create migration manager
	migrationConfig := migration.MigrationConfig{}
	migrationManager := migration.NewMigrationManager(k8sClient, scheme.Scheme, logger, migrationConfig)
	
	// Get migration status
	task, err := migrationManager.GetMigrationStatus(taskID)
	if err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}
	
	fmt.Printf("Migration Task: %s\n", task.ID)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Resources: %d\n", len(task.Resources))
	fmt.Println()
	
	// Confirm rollback
	fmt.Print("Are you sure you want to rollback this migration? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Rollback cancelled")
		return nil
	}
	
	// Perform rollback
	fmt.Println("Starting rollback...")
	if err := migrationManager.CancelMigration(taskID); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	fmt.Println("Rollback completed successfully")
	return nil
}

// getResourcesToMigrate returns the list of resources to migrate
func getResourcesToMigrate(ctx context.Context, k8sClient client.Client, args []string) ([]types.NamespacedName, error) {
	var resources []types.NamespacedName
	
	if len(args) > 0 {
		// Specific resource provided
		resources = append(resources, types.NamespacedName{
			Name:      args[0],
			Namespace: namespace,
		})
	} else {
		// List all resources
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "observability.io",
			Kind:    "ObservabilityPlatformList",
		})
		
		opts := []client.ListOption{}
		if !allNamespaces {
			opts = append(opts, client.InNamespace(namespace))
		}
		
		if err := k8sClient.List(ctx, list, opts...); err != nil {
			return nil, fmt.Errorf("failed to list resources: %w", err)
		}
		
		for _, item := range list.Items {
			resources = append(resources, types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			})
		}
	}
	
	return resources, nil
}

// monitorMigration monitors the progress of a migration
func monitorMigration(manager *migration.MigrationManager, taskID string) error {
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			task, err := manager.GetMigrationStatus(taskID)
			if err != nil {
				return err
			}
			
			displayMigrationProgress(task)
			
			if task.Status != migration.MigrationStatusInProgress {
				displayMigrationStatus(task)
				return nil
			}
		}
	}
}

// displayMigrationProgress displays migration progress
func displayMigrationProgress(task *migration.MigrationTask) {
	processed := task.Progress.MigratedResources + task.Progress.FailedResources + task.Progress.SkippedResources
	percentage := float64(processed) / float64(task.Progress.TotalResources) * 100
	
	fmt.Printf("\rProgress: %d/%d (%.1f%%) - Migrated: %d, Failed: %d, Skipped: %d",
		processed, task.Progress.TotalResources, percentage,
		task.Progress.MigratedResources,
		task.Progress.FailedResources,
		task.Progress.SkippedResources)
	
	if task.Progress.EstimatedTimeLeft > 0 {
		fmt.Printf(" - ETA: %s", task.Progress.EstimatedTimeLeft.Round(time.Second))
	}
}

// displayMigrationStatus displays detailed migration status
func displayMigrationStatus(task *migration.MigrationTask) {
	fmt.Printf("\n\nMigration Status\n")
	fmt.Printf("================\n")
	fmt.Printf("Task ID: %s\n", task.ID)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Target Version: %s\n", task.TargetVersion)
	fmt.Printf("Start Time: %s\n", task.StartTime.Format(time.RFC3339))
	
	if task.EndTime != nil {
		fmt.Printf("End Time: %s\n", task.EndTime.Format(time.RFC3339))
		fmt.Printf("Duration: %s\n", task.EndTime.Sub(task.StartTime))
	}
	
	fmt.Printf("\nResults:\n")
	fmt.Printf("  Total Resources: %d\n", task.Progress.TotalResources)
	fmt.Printf("  Migrated: %d\n", task.Progress.MigratedResources)
	fmt.Printf("  Failed: %d\n", task.Progress.FailedResources)
	fmt.Printf("  Skipped: %d\n", task.Progress.SkippedResources)
	
	if task.Error != nil {
		fmt.Printf("\nError: %v\n", task.Error)
	}
}

// formatTextReport formats a migration report as text
func formatTextReport(report *migration.MigrationReport) string {
	var b strings.Builder
	
	b.WriteString("Migration Report\n")
	b.WriteString("================\n\n")
	
	b.WriteString(fmt.Sprintf("Task ID: %s\n", report.TaskID))
	b.WriteString(fmt.Sprintf("Status: %s\n", report.Status))
	b.WriteString(fmt.Sprintf("Start Time: %s\n", report.StartTime.Format(time.RFC3339)))
	
	if report.EndTime != nil {
		b.WriteString(fmt.Sprintf("End Time: %s\n", report.EndTime.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("Duration: %s\n", report.Duration))
	}
	
	b.WriteString(fmt.Sprintf("\nTotal Resources: %d\n", report.TotalResources))
	b.WriteString(fmt.Sprintf("Success: %d\n", report.SuccessCount))
	b.WriteString(fmt.Sprintf("Failed: %d\n", report.FailureCount))
	b.WriteString(fmt.Sprintf("Skipped: %d\n", report.SkippedCount))
	
	if len(report.Events) > 0 {
		b.WriteString("\nEvents:\n")
		for _, event := range report.Events {
			b.WriteString(fmt.Sprintf("  [%s] %s: %s\n",
				event.Timestamp.Format("15:04:05"),
				event.Type,
				event.Message))
		}
	}
	
	if len(report.Recommendations) > 0 {
		b.WriteString("\nRecommendations:\n")
		for _, rec := range report.Recommendations {
			b.WriteString(fmt.Sprintf("  - %s\n", rec))
		}
	}
	
	return b.String()
}
