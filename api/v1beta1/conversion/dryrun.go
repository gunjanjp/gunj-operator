/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	
	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// DryRunConverter provides dry-run conversion capabilities
type DryRunConverter struct {
	logger      logr.Logger
	validator   *FieldValidator
	metrics     *ConversionMetrics
}

// DryRunResult contains the results of a dry-run conversion
type DryRunResult struct {
	// Success indicates if the conversion would succeed
	Success bool `json:"success"`
	
	// ValidationResult from field validation
	ValidationResult *ValidationResult `json:"validation_result"`
	
	// Original object (as JSON for easy comparison)
	Original json.RawMessage `json:"original"`
	
	// Converted object (as JSON for easy comparison)
	Converted json.RawMessage `json:"converted,omitempty"`
	
	// Diff between original and converted
	Diff []DiffEntry `json:"diff,omitempty"`
	
	// Recommendations for the conversion
	Recommendations []string `json:"recommendations"`
	
	// EstimatedImpact describes the impact of the conversion
	EstimatedImpact ConversionImpact `json:"estimated_impact"`
}

// DiffEntry represents a difference between original and converted objects
type DiffEntry struct {
	Path      string      `json:"path"`
	Type      string      `json:"type"` // added, removed, modified
	OldValue  interface{} `json:"old_value,omitempty"`
	NewValue  interface{} `json:"new_value,omitempty"`
	Impact    string      `json:"impact,omitempty"`
}

// ConversionImpact describes the estimated impact of a conversion
type ConversionImpact struct {
	DataLoss        bool     `json:"data_loss"`
	LostFields      []string `json:"lost_fields,omitempty"`
	EnhancedFields  []string `json:"enhanced_fields,omitempty"`
	RiskLevel       string   `json:"risk_level"` // low, medium, high
	DowntimeNeeded  bool     `json:"downtime_needed"`
	RollbackPossible bool    `json:"rollback_possible"`
}

// NewDryRunConverter creates a new dry-run converter
func NewDryRunConverter(logger logr.Logger) *DryRunConverter {
	return &DryRunConverter{
		logger:    logger,
		validator: NewFieldValidator(logger),
		metrics:   GetMetrics(),
	}
}

// DryRunConversion performs a dry-run conversion and returns detailed results
func (d *DryRunConverter) DryRunConversion(ctx context.Context, obj runtime.Object, targetVersion string) (*DryRunResult, error) {
	result := &DryRunResult{
		Success:         true,
		Recommendations: []string{},
	}
	
	// Determine source and target versions
	sourceVersion, err := d.getObjectVersion(obj)
	if err != nil {
		return nil, fmt.Errorf("getting source version: %w", err)
	}
	
	d.logger.Info("Starting dry-run conversion",
		"sourceVersion", sourceVersion,
		"targetVersion", targetVersion)
	
	// Serialize original object
	originalJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling original object: %w", err)
	}
	result.Original = originalJSON
	
	// Perform the conversion
	converted, err := d.performConversion(obj, sourceVersion, targetVersion)
	if err != nil {
		result.Success = false
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Conversion would fail: %v", err))
		return result, nil
	}
	
	// Serialize converted object
	convertedJSON, err := json.MarshalIndent(converted, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling converted object: %w", err)
	}
	result.Converted = convertedJSON
	
	// Validate the conversion
	result.ValidationResult = d.validator.ValidateConversion(sourceVersion, targetVersion, obj, converted)
	if len(result.ValidationResult.Errors) > 0 {
		result.Success = false
	}
	
	// Generate diff
	result.Diff = d.generateDiff(obj, converted, sourceVersion, targetVersion)
	
	// Estimate impact
	result.EstimatedImpact = d.estimateImpact(sourceVersion, targetVersion, result.ValidationResult, result.Diff)
	
	// Generate recommendations
	d.generateRecommendations(result, sourceVersion, targetVersion)
	
	// Record metrics
	d.recordDryRunMetrics(sourceVersion, targetVersion, result)
	
	return result, nil
}

// DryRunBatchConversion performs dry-run on multiple objects
func (d *DryRunConverter) DryRunBatchConversion(ctx context.Context, objects []runtime.Object, targetVersion string) ([]*DryRunResult, error) {
	results := make([]*DryRunResult, 0, len(objects))
	
	for _, obj := range objects {
		result, err := d.DryRunConversion(ctx, obj, targetVersion)
		if err != nil {
			return nil, fmt.Errorf("dry-run conversion failed for object: %w", err)
		}
		results = append(results, result)
	}
	
	return results, nil
}

// performConversion performs the actual conversion
func (d *DryRunConverter) performConversion(obj runtime.Object, sourceVersion, targetVersion string) (runtime.Object, error) {
	// Handle v1alpha1 to v1beta1
	if sourceVersion == "v1alpha1" && targetVersion == "v1beta1" {
		src, ok := obj.(*v1alpha1.ObservabilityPlatform)
		if !ok {
			return nil, fmt.Errorf("expected v1alpha1.ObservabilityPlatform, got %T", obj)
		}
		
		dst := &v1beta1.ObservabilityPlatform{}
		if err := src.ConvertTo(dst); err != nil {
			return nil, err
		}
		return dst, nil
	}
	
	// Handle v1beta1 to v1alpha1
	if sourceVersion == "v1beta1" && targetVersion == "v1alpha1" {
		src, ok := obj.(*v1beta1.ObservabilityPlatform)
		if !ok {
			return nil, fmt.Errorf("expected v1beta1.ObservabilityPlatform, got %T", obj)
		}
		
		dst := &v1alpha1.ObservabilityPlatform{}
		if err := dst.ConvertFrom(src); err != nil {
			return nil, err
		}
		return dst, nil
	}
	
	return nil, fmt.Errorf("unsupported conversion from %s to %s", sourceVersion, targetVersion)
}

// generateDiff generates a diff between original and converted objects
func (d *DryRunConverter) generateDiff(original, converted runtime.Object, sourceVersion, targetVersion string) []DiffEntry {
	diff := []DiffEntry{}
	
	// Get field mappings
	matrix := NewCompatibilityMatrix()
	mappings := matrix.GetFieldMappings(sourceVersion, targetVersion)
	lostFields := matrix.GetLostFields(sourceVersion, targetVersion)
	
	// Check for lost fields
	for _, field := range lostFields {
		value := d.validator.getFieldValue(original, field)
		if !d.validator.isZeroValue(value) {
			diff = append(diff, DiffEntry{
				Path:     field,
				Type:     "removed",
				OldValue: value,
				Impact:   "Data will be lost",
			})
		}
	}
	
	// Check for modified fields
	for _, mapping := range mappings {
		if mapping.SourcePath != "" && mapping.DestPath != "" {
			srcValue := d.validator.getFieldValue(original, mapping.SourcePath)
			dstValue := d.validator.getFieldValue(converted, mapping.DestPath)
			
			if mapping.DataTransform != nil || mapping.SemanticChange != nil {
				diff = append(diff, DiffEntry{
					Path:     mapping.SourcePath,
					Type:     "modified",
					OldValue: srcValue,
					NewValue: dstValue,
					Impact:   "Field transformed during conversion",
				})
			}
		}
	}
	
	// Check for added fields (new defaults in target version)
	for _, mapping := range mappings {
		if mapping.SourcePath == "" && mapping.DestPath != "" && mapping.DefaultValue != nil {
			diff = append(diff, DiffEntry{
				Path:     mapping.DestPath,
				Type:     "added",
				NewValue: mapping.DefaultValue,
				Impact:   "New field with default value",
			})
		}
	}
	
	return diff
}

// estimateImpact estimates the impact of the conversion
func (d *DryRunConverter) estimateImpact(sourceVersion, targetVersion string, validation *ValidationResult, diff []DiffEntry) ConversionImpact {
	impact := ConversionImpact{
		DataLoss:         validation.Metrics.DataLossFields > 0,
		LostFields:       []string{},
		EnhancedFields:   []string{},
		RollbackPossible: true,
	}
	
	// Collect lost and enhanced fields from diff
	for _, entry := range diff {
		switch entry.Type {
		case "removed":
			impact.LostFields = append(impact.LostFields, entry.Path)
		case "added":
			if entry.Path != "" {
				impact.EnhancedFields = append(impact.EnhancedFields, entry.Path)
			}
		}
	}
	
	// Determine risk level
	if len(validation.Errors) > 0 {
		impact.RiskLevel = "high"
		impact.DowntimeNeeded = true
	} else if impact.DataLoss {
		impact.RiskLevel = "medium"
		impact.RollbackPossible = false // Can't rollback if data is lost
	} else if len(validation.Warnings) > 0 {
		impact.RiskLevel = "low"
	} else {
		impact.RiskLevel = "minimal"
	}
	
	// Special cases
	if sourceVersion == "v1beta1" && targetVersion == "v1alpha1" {
		impact.RollbackPossible = false // Downgrade loses features
		if impact.RiskLevel == "minimal" {
			impact.RiskLevel = "low" // Downgrade is always at least low risk
		}
	}
	
	return impact
}

// generateRecommendations generates recommendations based on the dry-run results
func (d *DryRunConverter) generateRecommendations(result *DryRunResult, sourceVersion, targetVersion string) {
	// Add validation suggestions
	result.Recommendations = append(result.Recommendations, result.ValidationResult.Suggestions...)
	
	// Risk-based recommendations
	switch result.EstimatedImpact.RiskLevel {
	case "high":
		result.Recommendations = append(result.Recommendations,
			"High-risk conversion detected. Thoroughly test in a non-production environment first.",
			"Consider creating a backup before proceeding with the conversion.",
			"Review all validation errors and resolve them before converting.")
	case "medium":
		result.Recommendations = append(result.Recommendations,
			"Medium-risk conversion with potential data loss. Document lost fields before proceeding.",
			"Consider implementing a gradual rollout strategy.")
	case "low":
		result.Recommendations = append(result.Recommendations,
			"Low-risk conversion. Review warnings and proceed when ready.")
	}
	
	// Version-specific recommendations
	if sourceVersion == "v1alpha1" && targetVersion == "v1beta1" {
		result.Recommendations = append(result.Recommendations,
			"Upgrading to v1beta1 provides additional features like security configuration and cost optimization.",
			"After conversion, configure the new security and cost optimization settings for best results.")
	}
	
	if sourceVersion == "v1beta1" && targetVersion == "v1alpha1" {
		result.Recommendations = append(result.Recommendations,
			"Downgrading to v1alpha1 will result in feature loss. Ensure this is intentional.",
			"Export any critical configuration from v1beta1-specific fields before downgrading.")
	}
	
	// Data loss recommendations
	if result.EstimatedImpact.DataLoss {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Data loss detected in %d fields. Review the lost fields list carefully.",
				len(result.EstimatedImpact.LostFields)))
	}
}

// getObjectVersion determines the version of the object
func (d *DryRunConverter) getObjectVersion(obj runtime.Object) (string, error) {
	switch obj.(type) {
	case *v1alpha1.ObservabilityPlatform:
		return "v1alpha1", nil
	case *v1beta1.ObservabilityPlatform:
		return "v1beta1", nil
	default:
		return "", fmt.Errorf("unknown object type: %T", obj)
	}
}

// recordDryRunMetrics records metrics for the dry-run
func (d *DryRunConverter) recordDryRunMetrics(sourceVersion, targetVersion string, result *DryRunResult) {
	// Record as a special "dry-run" resource type
	timer := d.metrics.StartTimer(sourceVersion, targetVersion, "dry-run")
	defer timer.Complete(result.Success)
	
	// Report validation metrics
	reporter := NewMetricsReporter(sourceVersion, targetVersion)
	reporter.ReportValidationResult(result.ValidationResult)
}

// GenerateConversionReport generates a detailed conversion report
func (d *DryRunConverter) GenerateConversionReport(results []*DryRunResult) *ConversionReport {
	report := &ConversionReport{
		Timestamp:       time.Now(),
		TotalObjects:    len(results),
		SuccessfulCount: 0,
		FailedCount:     0,
		Warnings:        []string{},
		OverallRisk:     "minimal",
	}
	
	// Analyze results
	for _, result := range results {
		if result.Success {
			report.SuccessfulCount++
		} else {
			report.FailedCount++
		}
		
		// Collect unique warnings
		for _, warning := range result.ValidationResult.Warnings {
			if !contains(report.Warnings, warning) {
				report.Warnings = append(report.Warnings, warning)
			}
		}
		
		// Update overall risk
		if riskLevel := result.EstimatedImpact.RiskLevel; riskLevel != "" {
			if isHigherRisk(riskLevel, report.OverallRisk) {
				report.OverallRisk = riskLevel
			}
		}
	}
	
	// Generate summary
	report.Summary = fmt.Sprintf(
		"Conversion dry-run completed: %d successful, %d failed out of %d total objects. Overall risk: %s",
		report.SuccessfulCount, report.FailedCount, report.TotalObjects, report.OverallRisk)
	
	return report
}

// ConversionReport represents a summary of conversion dry-run results
type ConversionReport struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalObjects    int       `json:"total_objects"`
	SuccessfulCount int       `json:"successful_count"`
	FailedCount     int       `json:"failed_count"`
	Warnings        []string  `json:"warnings"`
	OverallRisk     string    `json:"overall_risk"`
	Summary         string    `json:"summary"`
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isHigherRisk(risk1, risk2 string) bool {
	riskLevels := map[string]int{
		"minimal": 0,
		"low":     1,
		"medium":  2,
		"high":    3,
	}
	
	return riskLevels[risk1] > riskLevels[risk2]
}
