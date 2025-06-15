/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var dashboardlog = logf.Log.WithName("dashboard-resource")

// SetupWebhookWithManager sets up the webhook with the Manager.
func (r *Dashboard) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-dashboard,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=dashboards,verbs=create;update,versions=v1beta1,name=mdashboard.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Dashboard{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Dashboard) Default() {
	dashboardlog.Info("default", "name", r.Name)

	// Set default title if not specified
	if r.Spec.Title == "" {
		r.Spec.Title = r.Name
	}

	// Set default theme
	if r.Spec.Theme == "" {
		r.Spec.Theme = "auto"
	}

	// Set default time settings
	if r.Spec.TimeSettings == nil {
		r.Spec.TimeSettings = &TimeSettings{}
	}
	if r.Spec.TimeSettings.From == "" {
		r.Spec.TimeSettings.From = "now-6h"
	}
	if r.Spec.TimeSettings.To == "" {
		r.Spec.TimeSettings.To = "now"
	}
	if r.Spec.TimeSettings.RefreshInterval == "" {
		r.Spec.TimeSettings.RefreshInterval = "30s"
	}

	// Set default tags
	if r.Spec.Tags == nil {
		r.Spec.Tags = []string{}
	}
	// Add default tag if not present
	managedByTag := "managed-by-gunj-operator"
	hasTag := false
	for _, tag := range r.Spec.Tags {
		if tag == managedByTag {
			hasTag = true
			break
		}
	}
	if !hasTag {
		r.Spec.Tags = append(r.Spec.Tags, managedByTag)
	}

	// Set defaults for variables
	for i := range r.Spec.Variables {
		variable := &r.Spec.Variables[i]
		
		// Default label to name if not set
		if variable.Label == "" {
			variable.Label = variable.Name
		}

		// Default refresh behavior
		if variable.Refresh == "" {
			variable.Refresh = "disabled"
		}

		// Set defaults for query variables
		if variable.Type == "query" && variable.Query != nil {
			if variable.Query.QueryType == "" {
				// Default query type based on query content
				if strings.Contains(variable.Query.Query, "label_values") {
					variable.Query.QueryType = "label_values"
				} else if strings.Contains(variable.Query.Query, "metrics") {
					variable.Query.QueryType = "metrics"
				} else {
					variable.Query.QueryType = "query_result"
				}
			}
		}

		// Default current value if not set and options available
		if variable.Current == nil && len(variable.Options) > 0 {
			for _, opt := range variable.Options {
				if opt.Selected {
					variable.Current = &opt
					break
				}
			}
			// If no option is selected, select the first one
			if variable.Current == nil && len(variable.Options) > 0 {
				variable.Current = &variable.Options[0]
				variable.Options[0].Selected = true
			}
		}
	}

	// Set defaults for panels
	nextPanelID := int32(1)
	usedIDs := make(map[int32]bool)
	
	// First pass: collect used IDs
	for _, panel := range r.Spec.Panels {
		if panel.ID > 0 {
			usedIDs[panel.ID] = true
		}
	}
	
	// Second pass: assign IDs and set defaults
	for i := range r.Spec.Panels {
		panel := &r.Spec.Panels[i]
		
		// Assign ID if not set
		if panel.ID == 0 {
			// Find next available ID
			for usedIDs[nextPanelID] {
				nextPanelID++
			}
			panel.ID = nextPanelID
			usedIDs[nextPanelID] = true
			nextPanelID++
		}

		// Set default grid position if not set
		if panel.GridPos == nil {
			panel.GridPos = &GridPos{
				X: 0,
				Y: 0,
				W: 12,
				H: 8,
			}
		}

		// Set default transparent
		if panel.Type != "text" && panel.Type != "dashlist" && panel.Type != "news" {
			// Most panels look better with transparent background
			panel.Transparent = true
		}

		// Set defaults for targets
		for j := range panel.Targets {
			target := &panel.Targets[j]
			
			// Set default format based on panel type
			if target.Format == "" {
				switch panel.Type {
				case "table":
					target.Format = "table"
				case "logs":
					target.Format = "logs"
				case "traces":
					target.Format = "traces"
				default:
					target.Format = "time_series"
				}
			}
		}

		// Set default field config
		if panel.FieldConfig == nil {
			panel.FieldConfig = &FieldConfig{
				Defaults: &FieldDefaults{},
			}
		}

		// Set default thresholds for certain panel types
		if panel.Type == "stat" || panel.Type == "gauge" || panel.Type == "bargauge" {
			if panel.FieldConfig.Defaults.Thresholds == nil {
				panel.FieldConfig.Defaults.Thresholds = &ThresholdsConfig{
					Mode: "absolute",
					Steps: []ThresholdStep{
						{Value: nil, Color: "green"},
						{Value: floatPtr(80), Color: "yellow"},
						{Value: floatPtr(90), Color: "red"},
					},
				}
			}
		}

		// Set default no value text
		if panel.FieldConfig.Defaults.NoValue == "" {
			panel.FieldConfig.Defaults.NoValue = "-"
		}
	}

	// Set defaults for annotations
	for i := range r.Spec.Annotations {
		annotation := &r.Spec.Annotations[i]
		
		// Default enable
		if !annotation.Enable {
			annotation.Enable = true
		}

		// Default icon color
		if annotation.IconColor == "" {
			annotation.IconColor = "rgba(0, 211, 255, 1)"
		}

		// Default type
		if annotation.Type == "" {
			if annotation.Query != "" || annotation.Expr != "" {
				annotation.Type = "dashboard"
			} else if len(annotation.Tags) > 0 {
				annotation.Type = "tags"
			}
		}
	}

	// Set defaults for links
	for i := range r.Spec.Links {
		link := &r.Spec.Links[i]
		
		// Default icon based on type
		if link.Icon == "" {
			switch link.Type {
			case "dashboards":
				link.Icon = "dashboard"
			case "link":
				link.Icon = "external link"
			}
		}
	}

	// Set defaults for layout
	if r.Spec.Layout == nil {
		r.Spec.Layout = &LayoutConfig{
			Type: "grid",
		}
	}

	// Set defaults for version info
	if r.Spec.Version == nil {
		r.Spec.Version = &VersionInfo{
			Version: 1,
		}
	}

	// Auto-layout panels if positions overlap
	r.autoLayoutPanels()
}

// autoLayoutPanels arranges panels to avoid overlaps
func (r *Dashboard) autoLayoutPanels() {
	if len(r.Spec.Panels) == 0 {
		return
	}

	// Create a grid to track occupied spaces
	maxY := int32(0)
	for _, panel := range r.Spec.Panels {
		if panel.GridPos != nil && panel.GridPos.Y+panel.GridPos.H > maxY {
			maxY = panel.GridPos.Y + panel.GridPos.H
		}
	}

	if maxY == 0 {
		maxY = 100 // Default grid height
	}

	grid := make([][]bool, maxY+20)
	for i := range grid {
		grid[i] = make([]bool, 24) // Grafana uses 24 column grid
	}

	// Mark occupied spaces
	validPanels := []*Panel{}
	overlappingPanels := []*Panel{}

	for i := range r.Spec.Panels {
		panel := &r.Spec.Panels[i]
		if panel.GridPos == nil {
			overlappingPanels = append(overlappingPanels, panel)
			continue
		}

		// Check if position is valid and not overlapping
		overlap := false
		for y := panel.GridPos.Y; y < panel.GridPos.Y+panel.GridPos.H && int(y) < len(grid); y++ {
			for x := panel.GridPos.X; x < panel.GridPos.X+panel.GridPos.W && x < 24; x++ {
				if grid[y][x] {
					overlap = true
					break
				}
			}
			if overlap {
				break
			}
		}

		if overlap {
			overlappingPanels = append(overlappingPanels, panel)
		} else {
			// Mark as occupied
			for y := panel.GridPos.Y; y < panel.GridPos.Y+panel.GridPos.H && int(y) < len(grid); y++ {
				for x := panel.GridPos.X; x < panel.GridPos.X+panel.GridPos.W && x < 24; x++ {
					grid[y][x] = true
				}
			}
			validPanels = append(validPanels, panel)
		}
	}

	// Place overlapping panels
	currentY := int32(0)
	for _, panel := range overlappingPanels {
		if panel.GridPos == nil {
			panel.GridPos = &GridPos{
				W: 12,
				H: 8,
			}
		}

		// Find next available position
		placed := false
		for y := currentY; int(y) < len(grid)-int(panel.GridPos.H); y++ {
			for x := int32(0); x <= 24-panel.GridPos.W; x++ {
				// Check if position is available
				available := true
				for dy := int32(0); dy < panel.GridPos.H && available; dy++ {
					for dx := int32(0); dx < panel.GridPos.W && available; dx++ {
						if grid[y+dy][x+dx] {
							available = false
						}
					}
				}

				if available {
					// Place panel
					panel.GridPos.X = x
					panel.GridPos.Y = y
					
					// Mark as occupied
					for dy := int32(0); dy < panel.GridPos.H; dy++ {
						for dx := int32(0); dx < panel.GridPos.W; dx++ {
							grid[y+dy][x+dx] = true
						}
					}
					
					placed = true
					currentY = y
					break
				}
			}
			if placed {
				break
			}
		}

		// If not placed, append to bottom
		if !placed {
			panel.GridPos.X = 0
			panel.GridPos.Y = maxY
			maxY += panel.GridPos.H
		}
	}
}

// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-dashboard,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=dashboards,verbs=create;update,versions=v1beta1,name=vdashboard.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Dashboard{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Dashboard) ValidateCreate() (admission.Warnings, error) {
	dashboardlog.Info("validate create", "name", r.Name)

	return r.validateDashboard()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Dashboard) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	dashboardlog.Info("validate update", "name", r.Name)

	return r.validateDashboard()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Dashboard) ValidateDelete() (admission.Warnings, error) {
	dashboardlog.Info("validate delete", "name", r.Name)

	// No validation needed for delete
	return nil, nil
}

// validateDashboard validates the Dashboard resource
func (r *Dashboard) validateDashboard() (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	// Validate target platform
	if r.Spec.TargetPlatform.Name == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec").Child("targetPlatform").Child("name"),
			"target platform name is required"))
	}

	// Validate title
	if r.Spec.Title == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec").Child("title"),
			"dashboard title is required"))
	} else if len(r.Spec.Title) > 128 {
		allErrs = append(allErrs, field.TooLong(
			field.NewPath("spec").Child("title"),
			r.Spec.Title,
			128))
	}

	// Validate folder name
	if r.Spec.Folder != "" && !isValidFolderName(r.Spec.Folder) {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("folder"),
			r.Spec.Folder,
			"folder name must contain only alphanumeric characters, spaces, hyphens, and underscores"))
	}

	// Validate JSON model if provided
	if r.Spec.JSONModel != "" {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(r.Spec.JSONModel), &jsonData); err != nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("jsonModel"),
				"<json content>",
				fmt.Sprintf("invalid JSON: %v", err)))
		} else {
			warnings = append(warnings, "using jsonModel will override most other settings")
		}
	}

	// Validate time settings
	if r.Spec.TimeSettings != nil {
		timePath := field.NewPath("spec").Child("timeSettings")
		
		// Validate refresh interval
		if r.Spec.TimeSettings.RefreshInterval != "" {
			if err := validateRefreshInterval(r.Spec.TimeSettings.RefreshInterval); err != nil {
				allErrs = append(allErrs, field.Invalid(
					timePath.Child("refreshInterval"),
					r.Spec.TimeSettings.RefreshInterval,
					err.Error()))
			}
		}

		// Validate fiscal year start month
		if r.Spec.TimeSettings.FiscalYearStartMonth < 0 || r.Spec.TimeSettings.FiscalYearStartMonth > 12 {
			allErrs = append(allErrs, field.Invalid(
				timePath.Child("fiscalYearStartMonth"),
				r.Spec.TimeSettings.FiscalYearStartMonth,
				"must be between 1 and 12 (0 for default)"))
		}
	}

	// Validate variables
	variableNames := make(map[string]bool)
	for i, variable := range r.Spec.Variables {
		varPath := field.NewPath("spec").Child("variables").Index(i)
		
		// Validate variable name
		if variable.Name == "" {
			allErrs = append(allErrs, field.Required(
				varPath.Child("name"),
				"variable name is required"))
		} else if !isValidVariableName(variable.Name) {
			allErrs = append(allErrs, field.Invalid(
				varPath.Child("name"),
				variable.Name,
				"variable name must contain only alphanumeric characters and underscores"))
		} else if variableNames[variable.Name] {
			allErrs = append(allErrs, field.Duplicate(
				varPath.Child("name"),
				variable.Name))
		} else {
			variableNames[variable.Name] = true
		}

		// Validate variable type-specific fields
		switch variable.Type {
		case "query":
			if variable.Query == nil || variable.Query.Query == "" {
				allErrs = append(allErrs, field.Required(
					varPath.Child("query").Child("query"),
					"query is required for query-type variables"))
			}
		case "custom":
			if len(variable.Options) == 0 {
				allErrs = append(allErrs, field.Required(
					varPath.Child("options"),
					"options are required for custom variables"))
			}
		case "textbox":
			// No specific validation needed
		case "constant":
			if variable.Current == nil {
				allErrs = append(allErrs, field.Required(
					varPath.Child("current"),
					"current value is required for constant variables"))
			}
		case "datasource":
			// Validate datasource query if present
			if variable.Query != nil && variable.Query.QueryType != "" {
				validTypes := []string{"prometheus", "loki", "tempo", "elasticsearch"}
				valid := false
				for _, t := range validTypes {
					if variable.Query.QueryType == t {
						valid = true
						break
					}
				}
				if !valid {
					allErrs = append(allErrs, field.NotSupported(
						varPath.Child("query").Child("queryType"),
						variable.Query.QueryType,
						validTypes))
				}
			}
		}

		// Validate regex if present
		if variable.Regex != "" {
			if _, err := regexp.Compile(variable.Regex); err != nil {
				allErrs = append(allErrs, field.Invalid(
					varPath.Child("regex"),
					variable.Regex,
					fmt.Sprintf("invalid regex: %v", err)))
			}
		}
	}

	// Validate panels
	panelIDs := make(map[int32]bool)
	for i, panel := range r.Spec.Panels {
		panelPath := field.NewPath("spec").Child("panels").Index(i)
		
		// Validate panel ID uniqueness
		if panel.ID == 0 {
			allErrs = append(allErrs, field.Required(
				panelPath.Child("id"),
				"panel ID is required"))
		} else if panelIDs[panel.ID] {
			allErrs = append(allErrs, field.Duplicate(
				panelPath.Child("id"),
				panel.ID))
		} else {
			panelIDs[panel.ID] = true
		}

		// Validate panel title
		if panel.Title == "" {
			warnings = append(warnings, fmt.Sprintf("panel %d has no title", panel.ID))
		}

		// Validate grid position
		if panel.GridPos != nil {
			gridPath := panelPath.Child("gridPos")
			
			if panel.GridPos.W < 1 || panel.GridPos.W > 24 {
				allErrs = append(allErrs, field.Invalid(
					gridPath.Child("w"),
					panel.GridPos.W,
					"width must be between 1 and 24"))
			}
			
			if panel.GridPos.H < 1 {
				allErrs = append(allErrs, field.Invalid(
					gridPath.Child("h"),
					panel.GridPos.H,
					"height must be at least 1"))
			}
			
			if panel.GridPos.X < 0 || panel.GridPos.X > 23 {
				allErrs = append(allErrs, field.Invalid(
					gridPath.Child("x"),
					panel.GridPos.X,
					"x position must be between 0 and 23"))
			}
			
			if panel.GridPos.Y < 0 {
				allErrs = append(allErrs, field.Invalid(
					gridPath.Child("y"),
					panel.GridPos.Y,
					"y position must be non-negative"))
			}
		}

		// Validate targets
		refIDs := make(map[string]bool)
		for j, target := range panel.Targets {
			targetPath := panelPath.Child("targets").Index(j)
			
			// Validate refId
			if target.RefID == "" {
				allErrs = append(allErrs, field.Required(
					targetPath.Child("refId"),
					"target refId is required"))
			} else if refIDs[target.RefID] {
				allErrs = append(allErrs, field.Duplicate(
					targetPath.Child("refId"),
					target.RefID))
			} else {
				refIDs[target.RefID] = true
			}

			// Validate query expression
			if target.Expr == "" && target.RawQuery == "" && 
			   target.MetricQuery == nil && target.LogQuery == nil && target.TraceQuery == nil {
				warnings = append(warnings, fmt.Sprintf(
					"target %s in panel %d has no query defined", target.RefID, panel.ID))
			}
		}

		// Validate panel type specific options
		switch panel.Type {
		case "graph", "timeseries":
			// These panels typically need targets
			if len(panel.Targets) == 0 {
				warnings = append(warnings, fmt.Sprintf(
					"%s panel %d has no targets", panel.Type, panel.ID))
			}
		case "text":
			// Text panels don't need targets
		case "table":
			// Tables can work with or without targets
		}

		// Validate transformations
		for j, transform := range panel.Transformations {
			transformPath := panelPath.Child("transformations").Index(j)
			
			if transform.ID == "" {
				allErrs = append(allErrs, field.Required(
					transformPath.Child("id"),
					"transformation ID is required"))
			}
			
			// Validate known transformation IDs
			validTransforms := []string{
				"merge", "seriesToColumns", "organize", "rename", 
				"calculateField", "filterFieldsByName", "filterByValue",
				"reduce", "labelsToFields", "groupBy", "sortBy",
			}
			valid := false
			for _, validID := range validTransforms {
				if transform.ID == validID {
					valid = true
					break
				}
			}
			if !valid {
				warnings = append(warnings, fmt.Sprintf(
					"unknown transformation ID '%s' in panel %d", transform.ID, panel.ID))
			}
		}

		// Validate panel links
		for j, link := range panel.Links {
			linkPath := panelPath.Child("links").Index(j)
			
			if link.Title == "" {
				allErrs = append(allErrs, field.Required(
					linkPath.Child("title"),
					"link title is required"))
			}
			
			if link.Type == "" {
				allErrs = append(allErrs, field.Required(
					linkPath.Child("type"),
					"link type is required"))
			}
			
			// Validate URL for external links
			if link.Type == "absolute" && link.URL != "" {
				if _, err := url.Parse(link.URL); err != nil {
					allErrs = append(allErrs, field.Invalid(
						linkPath.Child("url"),
						link.URL,
						"invalid URL"))
				}
			}
		}
	}

	// Validate annotations
	for i, annotation := range r.Spec.Annotations {
		annotationPath := field.NewPath("spec").Child("annotations").Index(i)
		
		if annotation.Name == "" {
			allErrs = append(allErrs, field.Required(
				annotationPath.Child("name"),
				"annotation name is required"))
		}
		
		// Validate query/expr based on type
		if annotation.Type == "dashboard" {
			if annotation.Query == "" && annotation.Expr == "" {
				allErrs = append(allErrs, field.Required(
					annotationPath,
					"query or expr is required for dashboard annotations"))
			}
		}
	}

	// Validate dashboard links
	for i, link := range r.Spec.Links {
		linkPath := field.NewPath("spec").Child("links").Index(i)
		
		if link.Title == "" {
			allErrs = append(allErrs, field.Required(
				linkPath.Child("title"),
				"link title is required"))
		}
		
		if link.Type == "link" && link.URL == "" {
			allErrs = append(allErrs, field.Required(
				linkPath.Child("url"),
				"URL is required for link type"))
		}
		
		if link.Type == "dashboards" && len(link.Tags) == 0 {
			warnings = append(warnings, "dashboard link has no tags specified")
		}
	}

	// Validate access control
	if r.Spec.AccessControl != nil {
		accessPath := field.NewPath("spec").Child("accessControl")
		
		for i, perm := range r.Spec.AccessControl.Permissions {
			permPath := accessPath.Child("permissions").Index(i)
			
			// Validate permission has a target
			if perm.UserID == 0 && perm.TeamID == 0 && 
			   perm.UserLogin == "" && perm.Team == "" && perm.Role == "" {
				allErrs = append(allErrs, field.Required(
					permPath,
					"permission must specify a user, team, or role"))
			}
		}
	}

	// Validate import config
	if r.Spec.ImportConfig != nil {
		importPath := field.NewPath("spec").Child("importConfig")
		
		// Count import sources
		sources := 0
		if r.Spec.ImportConfig.DashboardID > 0 {
			sources++
		}
		if r.Spec.ImportConfig.URL != "" {
			sources++
		}
		if r.Spec.ImportConfig.JSONContent != "" {
			sources++
		}
		if r.Spec.ImportConfig.ConfigMapRef != nil {
			sources++
		}
		if r.Spec.ImportConfig.SecretRef != nil {
			sources++
		}
		
		if sources == 0 {
			allErrs = append(allErrs, field.Required(
				importPath,
				"import config must specify a source"))
		} else if sources > 1 {
			allErrs = append(allErrs, field.Invalid(
				importPath,
				"multiple sources",
				"only one import source can be specified"))
		}
		
		// Validate URL if provided
		if r.Spec.ImportConfig.URL != "" {
			if _, err := url.Parse(r.Spec.ImportConfig.URL); err != nil {
				allErrs = append(allErrs, field.Invalid(
					importPath.Child("url"),
					r.Spec.ImportConfig.URL,
					"invalid URL"))
			}
		}
		
		// Validate JSON content if provided
		if r.Spec.ImportConfig.JSONContent != "" {
			var jsonData interface{}
			if err := json.Unmarshal([]byte(r.Spec.ImportConfig.JSONContent), &jsonData); err != nil {
				allErrs = append(allErrs, field.Invalid(
					importPath.Child("jsonContent"),
					"<json content>",
					fmt.Sprintf("invalid JSON: %v", err)))
			}
		}
	}

	// Validate metadata
	if r.Spec.Metadata != nil {
		metadataPath := field.NewPath("spec").Child("metadata")
		
		// Validate URLs
		if r.Spec.Metadata.DocumentationURL != "" && !isValidURL(r.Spec.Metadata.DocumentationURL) {
			allErrs = append(allErrs, field.Invalid(
				metadataPath.Child("documentationUrl"),
				r.Spec.Metadata.DocumentationURL,
				"invalid URL format"))
		}
		
		if r.Spec.Metadata.RunbookURL != "" && !isValidURL(r.Spec.Metadata.RunbookURL) {
			allErrs = append(allErrs, field.Invalid(
				metadataPath.Child("runbookUrl"),
				r.Spec.Metadata.RunbookURL,
				"invalid URL format"))
		}
	}

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "observability.io", Kind: "Dashboard"},
		r.Name, allErrs)
}

// Helper functions

// floatPtr returns a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}

// isValidFolderName checks if a folder name is valid
func isValidFolderName(name string) bool {
	validFolder := regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)
	return validFolder.MatchString(name)
}

// isValidVariableName checks if a variable name is valid
func isValidVariableName(name string) bool {
	validVar := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validVar.MatchString(name)
}

// validateRefreshInterval validates a refresh interval string
func validateRefreshInterval(interval string) error {
	if interval == "" {
		return nil
	}
	
	// Check for valid refresh intervals
	validIntervals := []string{
		"5s", "10s", "30s", "1m", "5m", "15m", "30m", "1h", "2h", "1d", "off",
	}
	
	for _, valid := range validIntervals {
		if interval == valid {
			return nil
		}
	}
	
	// Also accept custom intervals with valid duration pattern
	validDuration := regexp.MustCompile(`^(\d+)(ms|s|m|h|d)$`)
	if !validDuration.MatchString(interval) {
		return fmt.Errorf("invalid refresh interval format")
	}
	
	return nil
}
