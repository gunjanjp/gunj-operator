/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"fmt"
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
var alertingrulelog = logf.Log.WithName("alertingrule-resource")

// SetupWebhookWithManager sets up the webhook with the Manager.
func (r *AlertingRule) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-observability-io-v1beta1-alertingrule,mutating=true,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=alertingrules,verbs=create;update,versions=v1beta1,name=malertingrule.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &AlertingRule{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AlertingRule) Default() {
	alertingrulelog.Info("default", "name", r.Name)

	// Set default rule type if not specified
	if r.Spec.RuleType == "" {
		r.Spec.RuleType = "prometheus"
	}

	// Default global annotations
	if r.Spec.GlobalAnnotations == nil {
		r.Spec.GlobalAnnotations = make(map[string]string)
	}
	
	// Add default annotations if not present
	if _, ok := r.Spec.GlobalAnnotations["managed_by"]; !ok {
		r.Spec.GlobalAnnotations["managed_by"] = "gunj-operator"
	}

	// Default global labels
	if r.Spec.GlobalLabels == nil {
		r.Spec.GlobalLabels = make(map[string]string)
	}

	// Set defaults for Prometheus rules
	for i := range r.Spec.PrometheusRules {
		group := &r.Spec.PrometheusRules[i]
		
		// Default interval
		if group.Interval == "" {
			group.Interval = "1m"
		}

		// Default partial response strategy
		if group.PartialResponseStrategy == "" {
			group.PartialResponseStrategy = "warn"
		}

		// Set defaults for individual rules
		for j := range group.Rules {
			rule := &group.Rules[j]
			
			// Default enabled state
			if !rule.Enabled {
				rule.Enabled = true
			}

			// Add severity label if not present
			if rule.Severity != "" && rule.Labels == nil {
				rule.Labels = make(map[string]string)
			}
			if rule.Severity != "" {
				rule.Labels["severity"] = rule.Severity
			}

			// Merge global labels
			if rule.Labels == nil {
				rule.Labels = make(map[string]string)
			}
			for k, v := range r.Spec.GlobalLabels {
				if _, exists := rule.Labels[k]; !exists {
					rule.Labels[k] = v
				}
			}

			// Merge global annotations
			if rule.Annotations == nil {
				rule.Annotations = make(map[string]string)
			}
			for k, v := range r.Spec.GlobalAnnotations {
				if _, exists := rule.Annotations[k]; !exists {
					rule.Annotations[k] = v
				}
			}

			// Add default annotations
			if rule.DocumentationURL != "" && rule.Annotations["documentation"] == "" {
				rule.Annotations["documentation"] = rule.DocumentationURL
			}
			if rule.RunbookURL != "" && rule.Annotations["runbook_url"] == "" {
				rule.Annotations["runbook_url"] = rule.RunbookURL
			}
		}
	}

	// Set defaults for Loki rules
	for i := range r.Spec.LokiRules {
		group := &r.Spec.LokiRules[i]
		
		// Default interval
		if group.Interval == "" {
			group.Interval = "1m"
		}

		// Set defaults for individual rules
		for j := range group.Rules {
			rule := &group.Rules[j]
			
			// Default enabled state
			if !rule.Enabled {
				rule.Enabled = true
			}

			// Add severity label if not present
			if rule.Severity != "" && rule.Labels == nil {
				rule.Labels = make(map[string]string)
			}
			if rule.Severity != "" {
				rule.Labels["severity"] = rule.Severity
			}

			// Merge global labels
			if rule.Labels == nil {
				rule.Labels = make(map[string]string)
			}
			for k, v := range r.Spec.GlobalLabels {
				if _, exists := rule.Labels[k]; !exists {
					rule.Labels[k] = v
				}
			}

			// Merge global annotations
			if rule.Annotations == nil {
				rule.Annotations = make(map[string]string)
			}
			for k, v := range r.Spec.GlobalAnnotations {
				if _, exists := rule.Annotations[k]; !exists {
					rule.Annotations[k] = v
				}
			}
		}
	}

	// Set defaults for routing config
	if r.Spec.RoutingConfig != nil {
		if r.Spec.RoutingConfig.GroupWait == "" {
			r.Spec.RoutingConfig.GroupWait = "30s"
		}
		if r.Spec.RoutingConfig.GroupInterval == "" {
			r.Spec.RoutingConfig.GroupInterval = "5m"
		}
		if r.Spec.RoutingConfig.RepeatInterval == "" {
			r.Spec.RoutingConfig.RepeatInterval = "4h"
		}
	}

	// Set defaults for validation config
	if r.Spec.ValidationConfig != nil {
		if r.Spec.ValidationConfig.ValidationTimeout == "" {
			r.Spec.ValidationConfig.ValidationTimeout = "5m"
		}
	}

	// Set defaults for notification receivers
	if r.Spec.NotificationConfig != nil {
		for i := range r.Spec.NotificationConfig.Receivers {
			receiver := &r.Spec.NotificationConfig.Receivers[i]

			// Set defaults for webhook configs
			for j := range receiver.WebhookConfigs {
				webhook := &receiver.WebhookConfigs[j]
				if webhook.MaxAlerts == 0 {
					webhook.MaxAlerts = 1
				}
			}

			// Set defaults for PagerDuty configs
			for j := range receiver.PagerDutyConfigs {
				pd := &receiver.PagerDutyConfigs[j]
				if pd.URL == "" {
					pd.URL = "https://events.pagerduty.com/v2/enqueue"
				}
			}
		}

		// Set defaults for global notification config
		if r.Spec.NotificationConfig.GlobalConfig != nil {
			if r.Spec.NotificationConfig.GlobalConfig.ResolveTimeout == "" {
				r.Spec.NotificationConfig.GlobalConfig.ResolveTimeout = "5m"
			}
			if r.Spec.NotificationConfig.GlobalConfig.PagerDutyURL == "" {
				r.Spec.NotificationConfig.GlobalConfig.PagerDutyURL = "https://events.pagerduty.com/v2/enqueue"
			}
			if r.Spec.NotificationConfig.GlobalConfig.OpsGenieAPIURL == "" {
				r.Spec.NotificationConfig.GlobalConfig.OpsGenieAPIURL = "https://api.opsgenie.com/"
			}
		}
	}

	// Add tenant ID to labels if specified
	if r.Spec.TenantID != "" {
		if r.Spec.GlobalLabels == nil {
			r.Spec.GlobalLabels = make(map[string]string)
		}
		r.Spec.GlobalLabels["tenant_id"] = r.Spec.TenantID
	}
}

// +kubebuilder:webhook:path=/validate-observability-io-v1beta1-alertingrule,mutating=false,failurePolicy=fail,sideEffects=None,groups=observability.io,resources=alertingrules,verbs=create;update,versions=v1beta1,name=valertingrule.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &AlertingRule{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertingRule) ValidateCreate() (admission.Warnings, error) {
	alertingrulelog.Info("validate create", "name", r.Name)

	return r.validateAlertingRule()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertingRule) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	alertingrulelog.Info("validate update", "name", r.Name)

	return r.validateAlertingRule()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AlertingRule) ValidateDelete() (admission.Warnings, error) {
	alertingrulelog.Info("validate delete", "name", r.Name)

	// No validation needed for delete
	return nil, nil
}

// validateAlertingRule validates the AlertingRule resource
func (r *AlertingRule) validateAlertingRule() (admission.Warnings, error) {
	var allErrs field.ErrorList
	var warnings admission.Warnings

	// Validate target platform
	if r.Spec.TargetPlatform.Name == "" {
		allErrs = append(allErrs, field.Required(
			field.NewPath("spec").Child("targetPlatform").Child("name"),
			"target platform name is required"))
	}

	// Validate rule type
	switch r.Spec.RuleType {
	case "prometheus":
		if len(r.Spec.PrometheusRules) == 0 {
			allErrs = append(allErrs, field.Required(
				field.NewPath("spec").Child("prometheusRules"),
				"at least one Prometheus rule group is required when ruleType is 'prometheus'"))
		}
	case "loki":
		if len(r.Spec.LokiRules) == 0 {
			allErrs = append(allErrs, field.Required(
				field.NewPath("spec").Child("lokiRules"),
				"at least one Loki rule group is required when ruleType is 'loki'"))
		}
	case "multi":
		if len(r.Spec.PrometheusRules) == 0 && len(r.Spec.LokiRules) == 0 {
			allErrs = append(allErrs, field.Required(
				field.NewPath("spec").Child("prometheusRules", "lokiRules"),
				"at least one rule group is required when ruleType is 'multi'"))
		}
	}

	// Validate Prometheus rules
	for i, group := range r.Spec.PrometheusRules {
		groupPath := field.NewPath("spec").Child("prometheusRules").Index(i)

		// Validate group name
		if group.Name == "" {
			allErrs = append(allErrs, field.Required(
				groupPath.Child("name"),
				"rule group name is required"))
		}

		// Validate interval
		if err := validateDuration(group.Interval); err != nil {
			allErrs = append(allErrs, field.Invalid(
				groupPath.Child("interval"),
				group.Interval,
				"invalid duration format"))
		}

		// Validate rules
		if len(group.Rules) == 0 {
			allErrs = append(allErrs, field.Required(
				groupPath.Child("rules"),
				"at least one rule is required in the group"))
		}

		for j, rule := range group.Rules {
			rulePath := groupPath.Child("rules").Index(j)

			// Validate alert name
			if rule.Alert == "" {
				allErrs = append(allErrs, field.Required(
					rulePath.Child("alert"),
					"alert name is required"))
			} else if !isValidMetricName(rule.Alert) {
				allErrs = append(allErrs, field.Invalid(
					rulePath.Child("alert"),
					rule.Alert,
					"alert name must be a valid metric name"))
			}

			// Validate expression
			if rule.Expr == "" {
				allErrs = append(allErrs, field.Required(
					rulePath.Child("expr"),
					"expression is required"))
			}

			// Validate for duration
			if rule.For != "" {
				if err := validateDuration(rule.For); err != nil {
					allErrs = append(allErrs, field.Invalid(
						rulePath.Child("for"),
						rule.For,
						"invalid duration format"))
				}
			}

			// Validate keepFiringFor duration
			if rule.KeepFiringFor != "" {
				if err := validateDuration(rule.KeepFiringFor); err != nil {
					allErrs = append(allErrs, field.Invalid(
						rulePath.Child("keepFiringFor"),
						rule.KeepFiringFor,
						"invalid duration format"))
				}
			}

			// Validate labels
			for k, v := range rule.Labels {
				if !isValidLabelName(k) {
					allErrs = append(allErrs, field.Invalid(
						rulePath.Child("labels").Key(k),
						k,
						"invalid label name"))
				}
				if v == "" {
					warnings = append(warnings, fmt.Sprintf(
						"empty label value for key '%s' in rule '%s'", k, rule.Alert))
				}
			}

			// Validate severity
			if rule.Severity != "" {
				switch rule.Severity {
				case "critical", "warning", "info", "debug":
					// Valid severity
				default:
					allErrs = append(allErrs, field.Invalid(
						rulePath.Child("severity"),
						rule.Severity,
						"severity must be one of: critical, warning, info, debug"))
				}
			}

			// Validate priority
			if rule.Priority < 0 || rule.Priority > 999 {
				allErrs = append(allErrs, field.Invalid(
					rulePath.Child("priority"),
					rule.Priority,
					"priority must be between 0 and 999"))
			}

			// Check for template references in annotations
			for k, v := range rule.Annotations {
				if strings.Contains(v, "{{") && strings.Contains(v, "}}") {
					warnings = append(warnings, fmt.Sprintf(
						"annotation '%s' in rule '%s' contains template syntax - ensure templates are defined",
						k, rule.Alert))
				}
			}
		}
	}

	// Validate Loki rules
	for i, group := range r.Spec.LokiRules {
		groupPath := field.NewPath("spec").Child("lokiRules").Index(i)

		// Validate group name
		if group.Name == "" {
			allErrs = append(allErrs, field.Required(
				groupPath.Child("name"),
				"rule group name is required"))
		}

		// Validate interval
		if err := validateDuration(group.Interval); err != nil {
			allErrs = append(allErrs, field.Invalid(
				groupPath.Child("interval"),
				group.Interval,
				"invalid duration format"))
		}

		// Validate rules
		if len(group.Rules) == 0 {
			allErrs = append(allErrs, field.Required(
				groupPath.Child("rules"),
				"at least one rule is required in the group"))
		}

		for j, rule := range group.Rules {
			rulePath := groupPath.Child("rules").Index(j)

			// Validate alert name
			if rule.Alert == "" {
				allErrs = append(allErrs, field.Required(
					rulePath.Child("alert"),
					"alert name is required"))
			}

			// Validate expression
			if rule.Expr == "" {
				allErrs = append(allErrs, field.Required(
					rulePath.Child("expr"),
					"expression is required"))
			}

			// Validate for duration
			if rule.For != "" {
				if err := validateDuration(rule.For); err != nil {
					allErrs = append(allErrs, field.Invalid(
						rulePath.Child("for"),
						rule.For,
						"invalid duration format"))
				}
			}
		}
	}

	// Validate routing configuration
	if r.Spec.RoutingConfig != nil {
		routingPath := field.NewPath("spec").Child("routingConfig")

		// Validate durations
		if err := validateDuration(r.Spec.RoutingConfig.GroupWait); err != nil {
			allErrs = append(allErrs, field.Invalid(
				routingPath.Child("groupWait"),
				r.Spec.RoutingConfig.GroupWait,
				"invalid duration format"))
		}

		if err := validateDuration(r.Spec.RoutingConfig.GroupInterval); err != nil {
			allErrs = append(allErrs, field.Invalid(
				routingPath.Child("groupInterval"),
				r.Spec.RoutingConfig.GroupInterval,
				"invalid duration format"))
		}

		if err := validateDuration(r.Spec.RoutingConfig.RepeatInterval); err != nil {
			allErrs = append(allErrs, field.Invalid(
				routingPath.Child("repeatInterval"),
				r.Spec.RoutingConfig.RepeatInterval,
				"invalid duration format"))
		}

		// Validate routes
		for i, route := range r.Spec.RoutingConfig.Routes {
			routePath := routingPath.Child("routes").Index(i)

			// Validate route has either receiver or sub-routes
			if route.Receiver == "" && len(route.Routes) == 0 {
				allErrs = append(allErrs, field.Required(
					routePath,
					"route must have either a receiver or sub-routes"))
			}

			// Validate match conditions
			for k := range route.Match {
				if !isValidLabelName(k) {
					allErrs = append(allErrs, field.Invalid(
						routePath.Child("match").Key(k),
						k,
						"invalid label name in match condition"))
				}
			}

			// Validate regex conditions
			for k, v := range route.MatchRe {
				if !isValidLabelName(k) {
					allErrs = append(allErrs, field.Invalid(
						routePath.Child("matchRe").Key(k),
						k,
						"invalid label name in matchRe condition"))
				}
				if _, err := regexp.Compile(v); err != nil {
					allErrs = append(allErrs, field.Invalid(
						routePath.Child("matchRe").Key(k),
						v,
						fmt.Sprintf("invalid regex: %v", err)))
				}
			}
		}
	}

	// Validate templates
	for i, tmpl := range r.Spec.Templates {
		tmplPath := field.NewPath("spec").Child("templates").Index(i)

		if tmpl.Name == "" {
			allErrs = append(allErrs, field.Required(
				tmplPath.Child("name"),
				"template name is required"))
		}

		if tmpl.Template == "" {
			allErrs = append(allErrs, field.Required(
				tmplPath.Child("template"),
				"template content is required"))
		}
	}

	// Validate notification configuration
	if r.Spec.NotificationConfig != nil {
		notifPath := field.NewPath("spec").Child("notificationConfig")

		// Validate receivers
		for i, receiver := range r.Spec.NotificationConfig.Receivers {
			receiverPath := notifPath.Child("receivers").Index(i)

			if receiver.Name == "" {
				allErrs = append(allErrs, field.Required(
					receiverPath.Child("name"),
					"receiver name is required"))
			}

			// Validate email configs
			for j, email := range receiver.EmailConfigs {
				emailPath := receiverPath.Child("emailConfigs").Index(j)

				if len(email.To) == 0 {
					allErrs = append(allErrs, field.Required(
						emailPath.Child("to"),
						"at least one recipient email is required"))
				}

				for k, to := range email.To {
					if !isValidEmail(to) {
						allErrs = append(allErrs, field.Invalid(
							emailPath.Child("to").Index(k),
							to,
							"invalid email address"))
					}
				}
			}

			// Validate Slack configs
			for j, slack := range receiver.SlackConfigs {
				slackPath := receiverPath.Child("slackConfigs").Index(j)

				if slack.Channel == "" {
					allErrs = append(allErrs, field.Required(
						slackPath.Child("channel"),
						"Slack channel is required"))
				}

				if slack.APIURL.Name == "" && slack.APIURL.Key == "" {
					allErrs = append(allErrs, field.Required(
						slackPath.Child("apiUrl"),
						"Slack API URL secret reference is required"))
				}
			}

			// Validate webhook configs
			for j, webhook := range receiver.WebhookConfigs {
				webhookPath := receiverPath.Child("webhookConfigs").Index(j)

				if webhook.URL == "" {
					allErrs = append(allErrs, field.Required(
						webhookPath.Child("url"),
						"webhook URL is required"))
				}

				if webhook.MaxAlerts < 1 {
					allErrs = append(allErrs, field.Invalid(
						webhookPath.Child("maxAlerts"),
						webhook.MaxAlerts,
						"maxAlerts must be at least 1"))
				}
			}
		}

		// Validate time intervals
		for i, interval := range r.Spec.NotificationConfig.TimeIntervals {
			intervalPath := notifPath.Child("timeIntervals").Index(i)

			if interval.Name == "" {
				allErrs = append(allErrs, field.Required(
					intervalPath.Child("name"),
					"time interval name is required"))
			}

			for j, timeRange := range interval.TimeIntervals {
				rangePath := intervalPath.Child("timeIntervals").Index(j)

				// Validate time of day
				for k, tod := range timeRange.Times {
					todPath := rangePath.Child("times").Index(k)

					if !isValidTimeOfDay(tod.StartTime) {
						allErrs = append(allErrs, field.Invalid(
							todPath.Child("startTime"),
							tod.StartTime,
							"invalid time format, expected HH:MM"))
					}

					if !isValidTimeOfDay(tod.EndTime) {
						allErrs = append(allErrs, field.Invalid(
							todPath.Child("endTime"),
							tod.EndTime,
							"invalid time format, expected HH:MM"))
					}
				}
			}
		}
	}

	// Validate metadata
	if r.Spec.Metadata != nil {
		metadataPath := field.NewPath("spec").Child("metadata")

		// Validate email in owner field
		if r.Spec.Metadata.Owner != "" && strings.Contains(r.Spec.Metadata.Owner, "@") {
			if !isValidEmail(r.Spec.Metadata.Owner) {
				warnings = append(warnings, fmt.Sprintf(
					"owner field contains invalid email address: %s", r.Spec.Metadata.Owner))
			}
		}

		// Validate URLs
		if r.Spec.Metadata.DocumentationURL != "" && !isValidURL(r.Spec.Metadata.DocumentationURL) {
			allErrs = append(allErrs, field.Invalid(
				metadataPath.Child("documentationUrl"),
				r.Spec.Metadata.DocumentationURL,
				"invalid URL format"))
		}

		if r.Spec.Metadata.Repository != "" && !isValidURL(r.Spec.Metadata.Repository) {
			warnings = append(warnings, fmt.Sprintf(
				"repository field may not be a valid URL: %s", r.Spec.Metadata.Repository))
		}
	}

	if len(allErrs) == 0 {
		return warnings, nil
	}

	return warnings, apierrors.NewInvalid(
		schema.GroupKind{Group: "observability.io", Kind: "AlertingRule"},
		r.Name, allErrs)
}

// Helper functions

// validateDuration validates a duration string
func validateDuration(duration string) error {
	if duration == "" {
		return nil
	}

	// Check for valid duration pattern
	validDuration := regexp.MustCompile(`^(\d+)(s|m|h|d|w|y)$`)
	if !validDuration.MatchString(duration) {
		return fmt.Errorf("invalid duration format")
	}

	return nil
}

// isValidMetricName checks if a string is a valid Prometheus metric name
func isValidMetricName(name string) bool {
	validMetric := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validMetric.MatchString(name)
}

// isValidLabelName checks if a string is a valid label name
func isValidLabelName(name string) bool {
	validLabel := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validLabel.MatchString(name)
}

// isValidEmail checks if a string is a valid email address
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isValidURL checks if a string is a valid URL
func isValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^(https?://)?[a-zA-Z0-9-._]+(\.[a-zA-Z]{2,})+(/.*)?$`)
	return urlRegex.MatchString(url)
}

// isValidTimeOfDay checks if a string is a valid time of day (HH:MM)
func isValidTimeOfDay(time string) bool {
	timeRegex := regexp.MustCompile(`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`)
	return timeRegex.MatchString(time)
}
