/*
Copyright 2025 The Gunj Operator Authors.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

// MigrationAutomation provides automated migration capabilities
type MigrationAutomation struct {
	client          client.Client
	dynamicClient   dynamic.Interface
	scheme          *runtime.Scheme
	log             logr.Logger
	migrationMgr    *MigrationManager
	configFlags     *genericclioptions.ConfigFlags
	outputWriter    io.Writer
}

// AutomationConfig contains configuration for migration automation
type AutomationConfig struct {
	DryRun              bool
	Force               bool
	SkipValidation      bool
	BackupBeforeMigrate bool
	MaxParallel         int
	Timeout             time.Duration
	OutputFormat        string
	LogLevel            string
}

// NewMigrationAutomation creates a new migration automation instance
func NewMigrationAutomation(configFlags *genericclioptions.ConfigFlags) (*MigrationAutomation, error) {
	// Get config
	cfg, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("getting REST config: %w", err)
	}

	// Create clients
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	log := ctrl.Log.WithName("migration-automation")
	migrationMgr := NewMigrationManager(c, scheme.Scheme, log)

	return &MigrationAutomation{
		client:          c,
		dynamicClient:   dynClient,
		scheme:          scheme.Scheme,
		log:             log,
		migrationMgr:    migrationMgr,
		configFlags:     configFlags,
		outputWriter:    os.Stdout,
	}, nil
}

// CreateMigrationCommand creates the migration CLI command
func CreateMigrationCommand() *cobra.Command {
	var configFlags = genericclioptions.NewConfigFlags(true)
	var automationConfig = &AutomationConfig{}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate resources between API versions",
		Long: `Migrate Kubernetes resources from one API version to another.

This command automates the migration process including validation,
conversion, and verification of resources.`,
	}

	// Add subcommands
	cmd.AddCommand(
		createPlanCommand(configFlags, automationConfig),
		createExecuteCommand(configFlags, automationConfig),
		createValidateCommand(configFlags, automationConfig),
		createStatusCommand(configFlags, automationConfig),
		createRollbackCommand(configFlags, automationConfig),
		createReportCommand(configFlags, automationConfig),
	)

	// Add common flags
	configFlags.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().BoolVar(&automationConfig.DryRun, "dry-run", false, "Perform a dry run without making changes")
	cmd.PersistentFlags().BoolVar(&automationConfig.Force, "force", false, "Force migration even with validation warnings")
	cmd.PersistentFlags().IntVar(&automationConfig.MaxParallel, "max-parallel", 5, "Maximum parallel migrations")
	cmd.PersistentFlags().DurationVar(&automationConfig.Timeout, "timeout", 30*time.Minute, "Migration timeout")
	cmd.PersistentFlags().StringVar(&automationConfig.OutputFormat, "output", "table", "Output format (table|json|yaml)")

	return cmd
}

// createPlanCommand creates the plan subcommand
func createPlanCommand(configFlags *genericclioptions.ConfigFlags, config *AutomationConfig) *cobra.Command {
	var sourceGVK, targetGVK string
	var selector string
	var outputFile string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Create a migration plan",
		Long:  "Create a migration plan for resources",
		Example: `  # Plan migration for all ObservabilityPlatforms from v1alpha1 to v1beta1
  gunj-operator migrate plan --from observability.io/v1alpha1/ObservabilityPlatform --to observability.io/v1beta1/ObservabilityPlatform

  # Plan migration with selector
  gunj-operator migrate plan --from apps/v1beta2/Deployment --to apps/v1/Deployment --selector app=myapp

  # Save plan to file
  gunj-operator migrate plan --from v1alpha1 --to v1beta1 --output-file migration-plan.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			automation, err := NewMigrationAutomation(configFlags)
			if err != nil {
				return err
			}

			return automation.CreatePlan(cmd.Context(), sourceGVK, targetGVK, selector, outputFile, config)
		},
	}

	cmd.Flags().StringVar(&sourceGVK, "from", "", "Source GroupVersionKind (required)")
	cmd.Flags().StringVar(&targetGVK, "to", "", "Target GroupVersionKind (required)")
	cmd.Flags().StringVar(&selector, "selector", "", "Label selector to filter resources")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "Save plan to file")
	cmd.MarkFlagRequired("from")
	cmd.MarkFlagRequired("to")

	return cmd
}

// createExecuteCommand creates the execute subcommand
func createExecuteCommand(configFlags *genericclioptions.ConfigFlags, config *AutomationConfig) *cobra.Command {
	var planFile string
	var autoApprove bool

	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute a migration plan",
		Long:  "Execute a previously created migration plan",
		Example: `  # Execute migration plan from file
  gunj-operator migrate execute --plan migration-plan.yaml

  # Execute with auto-approval
  gunj-operator migrate execute --plan migration-plan.yaml --auto-approve

  # Execute in dry-run mode
  gunj-operator migrate execute --plan migration-plan.yaml --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			automation, err := NewMigrationAutomation(configFlags)
			if err != nil {
				return err
			}

			return automation.ExecutePlan(cmd.Context(), planFile, autoApprove, config)
		},
	}

	cmd.Flags().StringVar(&planFile, "plan", "", "Migration plan file (required)")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("plan")

	return cmd
}

// createValidateCommand creates the validate subcommand
func createValidateCommand(configFlags *genericclioptions.ConfigFlags, config *AutomationConfig) *cobra.Command {
	var resourceFile string
	var targetVersion string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate resources for migration",
		Long:  "Validate that resources can be migrated to the target version",
		Example: `  # Validate specific resource file
  gunj-operator migrate validate --file platform.yaml --target-version v1beta1

  # Validate all resources of a type
  gunj-operator migrate validate --from observability.io/v1alpha1/ObservabilityPlatform`,
		RunE: func(cmd *cobra.Command, args []string) error {
			automation, err := NewMigrationAutomation(configFlags)
			if err != nil {
				return err
			}

			return automation.ValidateResources(cmd.Context(), resourceFile, targetVersion, config)
		},
	}

	cmd.Flags().StringVar(&resourceFile, "file", "", "Resource file to validate")
	cmd.Flags().StringVar(&targetVersion, "target-version", "", "Target version to validate against")

	return cmd
}

// CreatePlan creates a migration plan
func (a *MigrationAutomation) CreatePlan(ctx context.Context, sourceGVKStr, targetGVKStr, selector, outputFile string, config *AutomationConfig) error {
	// Parse GVKs
	sourceGVK, err := parseGVK(sourceGVKStr)
	if err != nil {
		return fmt.Errorf("parsing source GVK: %w", err)
	}

	targetGVK, err := parseGVK(targetGVKStr)
	if err != nil {
		return fmt.Errorf("parsing target GVK: %w", err)
	}

	// Create migration options
	opts := []MigrationOption{
		WithBatchSize(config.MaxParallel),
		WithDryRun(config.DryRun),
		WithParallel(true),
		WithMaxConcurrency(config.MaxParallel),
		WithValidation(!config.SkipValidation, !config.SkipValidation),
	}

	// Create plan
	plan, err := a.migrationMgr.PlanMigration(ctx, sourceGVK, targetGVK, opts...)
	if err != nil {
		return fmt.Errorf("creating migration plan: %w", err)
	}

	// Apply selector if provided
	if selector != "" {
		plan = a.filterPlanBySelector(plan, selector)
	}

	// Output plan
	if outputFile != "" {
		return a.savePlanToFile(plan, outputFile)
	}

	return a.outputPlan(plan, config.OutputFormat)
}

// ExecutePlan executes a migration plan
func (a *MigrationAutomation) ExecutePlan(ctx context.Context, planFile string, autoApprove bool, config *AutomationConfig) error {
	// Load plan
	plan, err := a.loadPlanFromFile(planFile)
	if err != nil {
		return fmt.Errorf("loading plan: %w", err)
	}

	// Show plan summary
	if err := a.outputPlan(plan, "table"); err != nil {
		return err
	}

	// Confirm execution
	if !autoApprove && !config.DryRun {
		fmt.Fprintf(a.outputWriter, "\nDo you want to proceed with the migration? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			return fmt.Errorf("migration cancelled by user")
		}
	}

	// Create status reporter that outputs to console
	statusHandler := &ConsoleStatusHandler{writer: a.outputWriter}
	a.migrationMgr.statusReporter.AddStatusHandler(statusHandler)

	// Execute migration
	result, err := a.migrationMgr.ExecuteMigration(ctx, plan)
	if err != nil {
		return fmt.Errorf("executing migration: %w", err)
	}

	// Output results
	return a.outputResult(result, config.OutputFormat)
}

// ValidateResources validates resources for migration
func (a *MigrationAutomation) ValidateResources(ctx context.Context, resourceFile, targetVersion string, config *AutomationConfig) error {
	var resources []*unstructured.Unstructured

	if resourceFile != "" {
		// Load resources from file
		loaded, err := a.loadResourcesFromFile(resourceFile)
		if err != nil {
			return fmt.Errorf("loading resources: %w", err)
		}
		resources = loaded
	} else {
		return fmt.Errorf("resource file is required")
	}

	// Validate each resource
	for _, resource := range resources {
		// Create a temporary plan for validation
		sourceGVK := resource.GroupVersionKind()
		targetGVK := sourceGVK
		targetGVK.Version = targetVersion

		plan := &MigrationPlan{
			SourceGVK: sourceGVK,
			TargetGVK: targetGVK,
			Resources: []types.NamespacedName{{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			}},
		}

		// Run validation
		report, err := a.migrationMgr.validator.ValidatePreMigration(ctx, plan)
		if err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		// Output validation results
		if err := a.outputValidationReport(&report, config.OutputFormat); err != nil {
			return err
		}

		if !report.Valid && !config.Force {
			return fmt.Errorf("validation failed with %d errors", len(report.Errors))
		}
	}

	return nil
}

// Helper methods

// parseGVK parses a GVK string
func parseGVK(gvkStr string) (schema.GroupVersionKind, error) {
	parts := strings.Split(gvkStr, "/")
	if len(parts) != 3 {
		return schema.GroupVersionKind{}, fmt.Errorf("invalid GVK format, expected group/version/kind")
	}

	return schema.GroupVersionKind{
		Group:   parts[0],
		Version: parts[1],
		Kind:    parts[2],
	}, nil
}

// filterPlanBySelector filters plan resources by label selector
func (a *MigrationAutomation) filterPlanBySelector(plan *MigrationPlan, selector string) *MigrationPlan {
	// TODO: Implement label selector filtering
	return plan
}

// savePlanToFile saves a migration plan to file
func (a *MigrationAutomation) savePlanToFile(plan *MigrationPlan, filename string) error {
	data, err := yaml.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshaling plan: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing plan file: %w", err)
	}

	fmt.Fprintf(a.outputWriter, "Migration plan saved to: %s\n", filename)
	return nil
}

// loadPlanFromFile loads a migration plan from file
func (a *MigrationAutomation) loadPlanFromFile(filename string) (*MigrationPlan, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading plan file: %w", err)
	}

	var plan MigrationPlan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("unmarshaling plan: %w", err)
	}

	return &plan, nil
}

// loadResourcesFromFile loads resources from a file
func (a *MigrationAutomation) loadResourcesFromFile(filename string) ([]*unstructured.Unstructured, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Split by document separator for multi-document YAML
	docs := strings.Split(string(data), "\n---\n")
	resources := make([]*unstructured.Unstructured, 0, len(docs))

	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), obj); err != nil {
			return nil, fmt.Errorf("unmarshaling document: %w", err)
		}

		resources = append(resources, obj)
	}

	return resources, nil
}

// outputPlan outputs a migration plan in the specified format
func (a *MigrationAutomation) outputPlan(plan *MigrationPlan, format string) error {
	switch format {
	case "json":
		return a.outputJSON(plan)
	case "yaml":
		return a.outputYAML(plan)
	default:
		return a.outputPlanTable(plan)
	}
}

// outputResult outputs migration results
func (a *MigrationAutomation) outputResult(result *MigrationResult, format string) error {
	switch format {
	case "json":
		return a.outputJSON(result)
	case "yaml":
		return a.outputYAML(result)
	default:
		return a.outputResultTable(result)
	}
}

// outputValidationReport outputs a validation report
func (a *MigrationAutomation) outputValidationReport(report *ValidationReport, format string) error {
	switch format {
	case "json":
		return a.outputJSON(report)
	case "yaml":
		return a.outputYAML(report)
	default:
		return a.outputValidationTable(report)
	}
}

// Output formatting methods

func (a *MigrationAutomation) outputPlanTable(plan *MigrationPlan) error {
	fmt.Fprintf(a.outputWriter, "\nMigration Plan\n")
	fmt.Fprintf(a.outputWriter, "==============\n\n")
	fmt.Fprintf(a.outputWriter, "Source: %s\n", plan.SourceGVK)
	fmt.Fprintf(a.outputWriter, "Target: %s\n", plan.TargetGVK)
	fmt.Fprintf(a.outputWriter, "Resources: %d\n", len(plan.Resources))
	fmt.Fprintf(a.outputWriter, "Batch Size: %d\n", plan.BatchSize)
	fmt.Fprintf(a.outputWriter, "Dry Run: %v\n", plan.DryRun)
	fmt.Fprintf(a.outputWriter, "\nResources to migrate:\n")
	
	for i, res := range plan.Resources {
		fmt.Fprintf(a.outputWriter, "  %d. %s/%s\n", i+1, res.Namespace, res.Name)
		if i >= 10 && len(plan.Resources) > 10 {
			fmt.Fprintf(a.outputWriter, "  ... and %d more\n", len(plan.Resources)-10)
			break
		}
	}
	
	return nil
}

func (a *MigrationAutomation) outputResultTable(result *MigrationResult) error {
	fmt.Fprintf(a.outputWriter, "\nMigration Results\n")
	fmt.Fprintf(a.outputWriter, "=================\n\n")
	fmt.Fprintf(a.outputWriter, "Total Resources: %d\n", result.TotalResources)
	fmt.Fprintf(a.outputWriter, "Successful: %d\n", result.SuccessfulCount)
	fmt.Fprintf(a.outputWriter, "Failed: %d\n", result.FailedCount)
	fmt.Fprintf(a.outputWriter, "Skipped: %d\n", result.SkippedCount)
	fmt.Fprintf(a.outputWriter, "Duration: %s\n", result.Duration)
	
	if len(result.Errors) > 0 {
		fmt.Fprintf(a.outputWriter, "\nErrors:\n")
		for i, err := range result.Errors {
			fmt.Fprintf(a.outputWriter, "  %d. %s/%s: %s\n", 
				i+1, err.Resource.Namespace, err.Resource.Name, err.Error)
		}
	}
	
	return nil
}

func (a *MigrationAutomation) outputValidationTable(report *ValidationReport) error {
	fmt.Fprintf(a.outputWriter, "\nValidation Report\n")
	fmt.Fprintf(a.outputWriter, "=================\n\n")
	fmt.Fprintf(a.outputWriter, "Valid: %v\n", report.Valid)
	fmt.Fprintf(a.outputWriter, "Total Checks: %d\n", report.TotalChecks)
	fmt.Fprintf(a.outputWriter, "Passed: %d\n", report.PassedChecks)
	fmt.Fprintf(a.outputWriter, "Failed: %d\n", report.FailedChecks)
	
	if len(report.Errors) > 0 {
		fmt.Fprintf(a.outputWriter, "\nErrors:\n")
		for i, err := range report.Errors {
			fmt.Fprintf(a.outputWriter, "  %d. [%s] %s/%s - %s: %s\n",
				i+1, err.Severity, err.Resource.Namespace, err.Resource.Name, err.Rule, err.Message)
			if err.Remediation != "" {
				fmt.Fprintf(a.outputWriter, "     Remediation: %s\n", err.Remediation)
			}
		}
	}
	
	if len(report.Warnings) > 0 {
		fmt.Fprintf(a.outputWriter, "\nWarnings:\n")
		for i, warn := range report.Warnings {
			fmt.Fprintf(a.outputWriter, "  %d. %s/%s - %s: %s\n",
				i+1, warn.Resource.Namespace, warn.Resource.Name, warn.Rule, warn.Message)
		}
	}
	
	return nil
}

func (a *MigrationAutomation) outputJSON(v interface{}) error {
	// Implementation would use encoding/json
	return fmt.Errorf("JSON output not implemented")
}

func (a *MigrationAutomation) outputYAML(v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = a.outputWriter.Write(data)
	return err
}

// ConsoleStatusHandler outputs status updates to console
type ConsoleStatusHandler struct {
	writer io.Writer
}

func (h *ConsoleStatusHandler) HandleStatusUpdate(status *MigrationStatus) {
	fmt.Fprintf(h.writer, "\r[%s] Progress: %d/%d (%.1f%%) - Phase: %s",
		time.Now().Format("15:04:05"),
		status.Progress.ProcessedResources,
		status.Progress.TotalResources,
		status.Progress.PercentComplete,
		status.Phase,
	)
}

// createStatusCommand creates the status subcommand
func createStatusCommand(configFlags *genericclioptions.ConfigFlags, config *AutomationConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  "Show the current status of ongoing migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			automation, err := NewMigrationAutomation(configFlags)
			if err != nil {
				return err
			}

			status, err := automation.migrationMgr.GetMigrationStatus(cmd.Context())
			if err != nil {
				return err
			}

			return automation.outputJSON(status) // or use a proper status output format
		},
	}

	return cmd
}

// createRollbackCommand creates the rollback subcommand
func createRollbackCommand(configFlags *genericclioptions.ConfigFlags, config *AutomationConfig) *cobra.Command {
	var planFile string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback a failed migration",
		Long:  "Rollback a migration that failed or needs to be reverted",
		RunE: func(cmd *cobra.Command, args []string) error {
			automation, err := NewMigrationAutomation(configFlags)
			if err != nil {
				return err
			}

			// Load original plan
			plan, err := automation.loadPlanFromFile(planFile)
			if err != nil {
				return fmt.Errorf("loading plan: %w", err)
			}

			// TODO: Load the result from a previous execution
			// For now, creating an empty result
			result := &MigrationResult{}

			return automation.migrationMgr.Rollback(cmd.Context(), plan, result)
		},
	}

	cmd.Flags().StringVar(&planFile, "plan", "", "Original migration plan file")
	cmd.MarkFlagRequired("plan")

	return cmd
}

// createReportCommand creates the report subcommand
func createReportCommand(configFlags *genericclioptions.ConfigFlags, config *AutomationConfig) *cobra.Command {
	var historyLimit int
	var outputFile string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate migration reports",
		Long:  "Generate detailed reports about migration history and performance",
		RunE: func(cmd *cobra.Command, args []string) error {
			automation, err := NewMigrationAutomation(configFlags)
			if err != nil {
				return err
			}

			history, err := automation.migrationMgr.GetMigrationHistory(cmd.Context(), historyLimit)
			if err != nil {
				return err
			}

			// Generate report
			report := generateHistoryReport(history)

			if outputFile != "" {
				return os.WriteFile(outputFile, []byte(report), 0644)
			}

			fmt.Fprint(automation.outputWriter, report)
			return nil
		},
	}

	cmd.Flags().IntVar(&historyLimit, "limit", 10, "Number of history entries to include")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "Save report to file")

	return cmd
}

// generateHistoryReport generates a history report
func generateHistoryReport(history []*MigrationHistoryEntry) string {
	var report strings.Builder

	report.WriteString("Migration History Report\n")
	report.WriteString("=======================\n\n")

	for i, entry := range history {
		report.WriteString(fmt.Sprintf("Migration #%d\n", i+1))
		report.WriteString(fmt.Sprintf("Timestamp: %s\n", entry.Timestamp.Format(time.RFC3339)))
		report.WriteString(fmt.Sprintf("Source: %s\n", entry.Plan.SourceGVK))
		report.WriteString(fmt.Sprintf("Target: %s\n", entry.Plan.TargetGVK))
		report.WriteString(fmt.Sprintf("Resources: %d\n", entry.Result.TotalResources))
		report.WriteString(fmt.Sprintf("Success: %d\n", entry.Result.SuccessfulCount))
		report.WriteString(fmt.Sprintf("Failed: %d\n", entry.Result.FailedCount))
		report.WriteString(fmt.Sprintf("Duration: %s\n", entry.Result.Duration))
		report.WriteString("\n")
	}

	return report.String()
}
