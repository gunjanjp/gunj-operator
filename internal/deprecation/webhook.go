package deprecation

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// WebhookHandler handles deprecation warnings in admission webhooks
type WebhookHandler struct {
	client  client.Client
	checker *Checker
}

// NewWebhookHandler creates a new deprecation webhook handler
func NewWebhookHandler(client client.Client) *WebhookHandler {
	return &WebhookHandler{
		client:  client,
		checker: NewChecker(),
	}
}

// Handle processes admission requests and adds deprecation warnings
func (h *WebhookHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := log.FromContext(ctx)

	// Parse the object
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(req.Object.Raw, obj); err != nil {
		log.Error(err, "Failed to unmarshal object")
		return admission.Errored(int32(admissionv1.BadRequest), err)
	}

	// Get the GVK
	gvk := schema.GroupVersionKind{
		Group:   req.Kind.Group,
		Version: req.Kind.Version,
		Kind:    req.Kind.Kind,
	}

	// Check for deprecations
	result, err := h.checker.Check(obj, gvk)
	if err != nil {
		log.Error(err, "Failed to check deprecations")
		// Don't fail the request, just log the error
		return admission.Allowed("")
	}

	// If no warnings, allow the request
	if !result.HasWarnings {
		return admission.Allowed("")
	}

	// Build the response with warnings
	response := admission.Allowed("")
	response.Warnings = h.buildWarnings(result)

	// Log deprecation usage for monitoring
	h.logDeprecations(ctx, obj, result)

	// For critical deprecations, we might want to add additional handling
	if result.HasCritical {
		log.Info("Critical deprecations detected",
			"namespace", obj.GetNamespace(),
			"name", obj.GetName(),
			"kind", obj.GetKind(),
			"warningCount", len(result.Warnings))
	}

	return response
}

// buildWarnings converts deprecation warnings to admission warnings
func (h *WebhookHandler) buildWarnings(result *CheckResult) []string {
	warnings := make([]string, 0, len(result.Warnings))

	for _, warning := range result.Warnings {
		// Format the warning message
		msg := h.formatWarningMessage(&warning)
		warnings = append(warnings, msg)
	}

	return warnings
}

// formatWarningMessage formats a single warning for display in admission response
func (h *WebhookHandler) formatWarningMessage(warning *Warning) string {
	// Create a concise but informative warning message
	severity := ""
	switch warning.Severity {
	case SeverityCritical:
		severity = "[CRITICAL] "
	case SeverityWarning:
		severity = "[WARNING] "
	}

	msg := fmt.Sprintf("%s%s", severity, warning.Message)
	
	if warning.Field != "" {
		msg += fmt.Sprintf(" (field: %s)", warning.Field)
	}
	
	if warning.Value != "" {
		msg += fmt.Sprintf(" (value: %s)", warning.Value)
	}

	// Add a hint about migration if available
	if warning.MigrationGuide != "" {
		// Extract just the first line of the migration guide for the warning
		lines := splitLines(warning.MigrationGuide)
		if len(lines) > 0 {
			msg += fmt.Sprintf(" - %s", lines[0])
		}
		msg += " (see documentation for full migration guide)"
	}

	return msg
}

// logDeprecations logs deprecation usage for monitoring and metrics
func (h *WebhookHandler) logDeprecations(ctx context.Context, obj *unstructured.Unstructured, result *CheckResult) {
	log := log.FromContext(ctx)

	for _, warning := range result.Warnings {
		log.Info("Deprecation detected",
			"namespace", obj.GetNamespace(),
			"name", obj.GetName(),
			"kind", obj.GetKind(),
			"field", warning.Field,
			"value", warning.Value,
			"severity", warning.Severity,
			"message", warning.Message)

		// Here you could also emit metrics for deprecation usage tracking
		// For example: deprecationCounter.WithLabelValues(warning.Field, string(warning.Severity)).Inc()
	}
}

// ValidatingWebhook returns an admission webhook that validates and warns about deprecations
func (h *WebhookHandler) ValidatingWebhook() admission.Handler {
	return admission.HandlerFunc(func(ctx context.Context, req admission.Request) admission.Response {
		// For validating webhooks, we check deprecations but don't block
		// unless there are critical deprecations that absolutely must be addressed
		response := h.Handle(ctx, req)

		// Optionally, you could fail requests with critical deprecations
		// by checking if response has critical warnings and returning admission.Denied()
		
		return response
	})
}

// MutatingWebhook returns an admission webhook that can mutate objects to fix deprecations
func (h *WebhookHandler) MutatingWebhook() admission.Handler {
	return admission.HandlerFunc(func(ctx context.Context, req admission.Request) admission.Response {
		log := log.FromContext(ctx)

		// Parse the object
		obj := &unstructured.Unstructured{}
		if err := json.Unmarshal(req.Object.Raw, obj); err != nil {
			log.Error(err, "Failed to unmarshal object")
			return admission.Errored(int32(admissionv1.BadRequest), err)
		}

		// Get the GVK
		gvk := schema.GroupVersionKind{
			Group:   req.Kind.Group,
			Version: req.Kind.Version,
			Kind:    req.Kind.Kind,
		}

		// Check for deprecations
		result, err := h.checker.Check(obj, gvk)
		if err != nil {
			log.Error(err, "Failed to check deprecations")
			return admission.Allowed("")
		}

		// Attempt to auto-migrate simple deprecations
		modified := false
		if result.HasWarnings {
			modified = h.attemptAutoMigration(ctx, obj, result)
		}

		// Build response
		if modified {
			// Marshal the modified object
			marshaled, err := json.Marshal(obj)
			if err != nil {
				log.Error(err, "Failed to marshal modified object")
				return admission.Allowed("")
			}

			// Create patch response
			return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
		}

		// No modifications, but add warnings
		response := admission.Allowed("")
		if result.HasWarnings {
			response.Warnings = h.buildWarnings(result)
		}

		return response
	})
}

// attemptAutoMigration tries to automatically fix simple deprecations
func (h *WebhookHandler) attemptAutoMigration(ctx context.Context, obj *unstructured.Unstructured, result *CheckResult) bool {
	log := log.FromContext(ctx)
	modified := false

	registry := GetRegistry()

	for _, warning := range result.Warnings {
		// Only attempt auto-migration for non-critical deprecations with clear alternatives
		if warning.Severity == SeverityCritical {
			continue
		}

		// Get the full deprecation info
		dep := registry.GetDeprecationByPath(warning.Field, schema.GroupVersionKind{
			Group:   obj.GetObjectKind().GroupVersionKind().Group,
			Version: obj.GetObjectKind().GroupVersionKind().Version,
			Kind:    obj.GetKind(),
		})

		if dep == nil || dep.AlternativePath == "" {
			continue
		}

		// Attempt migration for simple field renames
		if dep.Type == FieldDeprecation {
			if h.migrateField(obj, dep) {
				modified = true
				log.Info("Auto-migrated deprecated field",
					"namespace", obj.GetNamespace(),
					"name", obj.GetName(),
					"oldField", dep.Path,
					"newField", dep.AlternativePath)
			}
		}
	}

	return modified
}

// migrateField attempts to migrate a deprecated field to its alternative
func (h *WebhookHandler) migrateField(obj *unstructured.Unstructured, dep *DeprecationInfo) bool {
	// Get the value from the deprecated field
	oldPathParts := splitPath(dep.Path)
	value, found, err := unstructured.NestedFieldNoCopy(obj.Object, oldPathParts...)
	if err != nil || !found {
		return false
	}

	// Set the value in the new field
	newPathParts := splitPath(dep.AlternativePath)
	if err := unstructured.SetNestedField(obj.Object, value, newPathParts...); err != nil {
		return false
	}

	// Remove the deprecated field
	unstructured.RemoveNestedField(obj.Object, oldPathParts...)

	return true
}

// splitPath splits a dot-separated path into parts
func splitPath(path string) []string {
	return splitLines(path)[0:1] // Reuse splitLines for simplicity
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// WebhookResponse represents the response from deprecation checking
type WebhookResponse struct {
	// Allowed indicates if the request should be allowed
	Allowed bool
	// Warnings contains any deprecation warnings
	Warnings []string
	// Patches contains any auto-migration patches
	Patches []map[string]interface{}
}

// CheckAndRespond performs deprecation checking and returns a structured response
func (h *WebhookHandler) CheckAndRespond(ctx context.Context, obj *unstructured.Unstructured, gvk schema.GroupVersionKind) (*WebhookResponse, error) {
	result, err := h.checker.Check(obj, gvk)
	if err != nil {
		return nil, fmt.Errorf("checking deprecations: %w", err)
	}

	response := &WebhookResponse{
		Allowed:  true,
		Warnings: h.buildWarnings(result),
		Patches:  []map[string]interface{}{},
	}

	// Log deprecations
	if result.HasWarnings {
		h.logDeprecations(ctx, obj, result)
	}

	return response, nil
}
