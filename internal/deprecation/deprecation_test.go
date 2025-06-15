package deprecation

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRegistry(t *testing.T) {
	// Reset the global registry for testing
	globalRegistry = nil
	once = sync.Once{}

	registry := GetRegistry()

	// Test that initial deprecations are loaded
	deprecations := registry.GetDeprecations(schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1alpha1",
		Kind:    "ObservabilityPlatform",
	})

	if len(deprecations) == 0 {
		t.Error("Expected initial deprecations to be loaded")
	}

	// Test registering a new deprecation
	testDep := &DeprecationInfo{
		Type:    FieldDeprecation,
		Path:    "spec.test.field",
		Message: "Test deprecation",
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v1",
			DeprecatedSince:     time.Now(),
		},
		Severity:         SeverityWarning,
		AffectedVersions: []string{"v1alpha1"},
	}

	registry.Register(testDep)

	// Verify it was registered
	found := registry.GetDeprecationByPath("spec.test.field", schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1alpha1",
		Kind:    "ObservabilityPlatform",
	})

	if found == nil {
		t.Error("Expected to find registered deprecation")
	}
}

func TestChecker(t *testing.T) {
	// Reset the global registry for testing
	globalRegistry = nil
	once = sync.Once{}

	checker := NewChecker()

	tests := []struct {
		name        string
		obj         map[string]interface{}
		gvk         schema.GroupVersionKind
		expectWarn  bool
		expectField string
	}{
		{
			name: "deprecated monitoring field",
			obj: map[string]interface{}{
				"apiVersion": "observability.io/v1alpha1",
				"kind":       "ObservabilityPlatform",
				"spec": map[string]interface{}{
					"monitoring": map[string]interface{}{
						"prometheus": map[string]interface{}{
							"enabled": true,
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{
				Group:   "observability.io",
				Version: "v1alpha1",
				Kind:    "ObservabilityPlatform",
			},
			expectWarn:  true,
			expectField: "spec.monitoring",
		},
		{
			name: "no deprecations",
			obj: map[string]interface{}{
				"apiVersion": "observability.io/v1beta1",
				"kind":       "ObservabilityPlatform",
				"spec": map[string]interface{}{
					"components": map[string]interface{}{
						"prometheus": map[string]interface{}{
							"enabled": true,
							"version": "v2.48.0",
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{
				Group:   "observability.io",
				Version: "v1beta1",
				Kind:    "ObservabilityPlatform",
			},
			expectWarn: false,
		},
		{
			name: "deprecated value",
			obj: map[string]interface{}{
				"apiVersion": "observability.io/v1beta1",
				"kind":       "ObservabilityPlatform",
				"spec": map[string]interface{}{
					"components": map[string]interface{}{
						"prometheus": map[string]interface{}{
							"enabled": true,
							"version": "v2.30.0",
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{
				Group:   "observability.io",
				Version: "v1beta1",
				Kind:    "ObservabilityPlatform",
			},
			expectWarn:  true,
			expectField: "spec.components.prometheus.version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{Object: tt.obj}
			result, err := checker.Check(obj, tt.gvk)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectWarn && !result.HasWarnings {
				t.Error("expected warnings but got none")
			}

			if !tt.expectWarn && result.HasWarnings {
				t.Errorf("expected no warnings but got %d", len(result.Warnings))
			}

			if tt.expectWarn && tt.expectField != "" {
				found := false
				for _, w := range result.Warnings {
					if w.Field == tt.expectField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning for field %s but not found", tt.expectField)
				}
			}
		})
	}
}

func TestDeprecationSeverity(t *testing.T) {
	tests := []struct {
		name             string
		deprecatedSince  time.Time
		removalDate      *time.Time
		expectedSeverity DeprecationSeverity
	}{
		{
			name:             "recent deprecation",
			deprecatedSince:  time.Now().AddDate(0, -1, 0), // 1 month ago
			removalDate:      nil,
			expectedSeverity: SeverityInfo,
		},
		{
			name:             "old deprecation",
			deprecatedSince:  time.Now().AddDate(0, -7, 0), // 7 months ago
			removalDate:      nil,
			expectedSeverity: SeverityWarning,
		},
		{
			name:             "removal soon",
			deprecatedSince:  time.Now().AddDate(0, -3, 0), // 3 months ago
			removalDate:      &[]time.Time{time.Now().AddDate(0, 2, 0)}[0], // 2 months from now
			expectedSeverity: SeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &DeprecationInfo{
				Policy: DeprecationPolicy{
					DeprecatedSince: tt.deprecatedSince,
					RemovalDate:     tt.removalDate,
				},
				Severity: SeverityInfo, // default
			}

			severity := dep.GetSeverity()
			if severity != tt.expectedSeverity {
				t.Errorf("expected severity %s, got %s", tt.expectedSeverity, severity)
			}
		})
	}
}

func TestDocumentationGenerator(t *testing.T) {
	// Reset the global registry for testing
	globalRegistry = nil
	once = sync.Once{}

	generator := NewDocumentationGenerator()
	
	// Generate markdown
	var buf bytes.Buffer
	err := generator.GenerateMarkdown(&buf)
	if err != nil {
		t.Fatalf("failed to generate markdown: %v", err)
	}

	markdown := buf.String()

	// Check that expected sections are present
	expectedSections := []string{
		"# Deprecation Guide for Gunj Operator",
		"## Table of Contents",
		"## Overview",
		"## Deprecation Timeline",
		"## API Version Deprecations",
		"## Field Deprecations",
		"## Value Deprecations",
		"## Feature Deprecations",
		"## Migration Examples",
		"## Deprecation Policy",
	}

	for _, section := range expectedSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("expected markdown to contain section: %s", section)
		}
	}

	// Test YAML generation
	buf.Reset()
	err = generator.GenerateYAML(&buf)
	if err != nil {
		t.Fatalf("failed to generate YAML: %v", err)
	}

	yaml := buf.String()
	if !strings.Contains(yaml, "deprecations:") {
		t.Error("expected YAML to contain deprecations section")
	}
}

func TestFormatWarning(t *testing.T) {
	registry := GetRegistry()

	dep := &DeprecationInfo{
		Type:    FieldDeprecation,
		Path:    "spec.test",
		Message: "Test field is deprecated",
		MigrationGuide: `Use spec.newTest instead:
  spec:
    newTest: value`,
		AlternativePath: "spec.newTest",
		Policy: DeprecationPolicy{
			DeprecatedInVersion: "v1beta1",
			RemovedInVersion:    "v1",
			DeprecatedSince:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Severity: SeverityWarning,
	}

	formatted := registry.FormatWarning(dep)

	// Check that formatted warning contains expected elements
	expectedElements := []string{
		"DEPRECATION WARNING",
		"Test field is deprecated",
		"Field: spec.test",
		"Deprecated Since: 2025-01-01",
		"Will be removed in: v1",
		"Migration Guide:",
		"Use instead: spec.newTest",
	}

	for _, element := range expectedElements {
		if !strings.Contains(formatted, element) {
			t.Errorf("expected formatted warning to contain: %s", element)
		}
	}
}

func TestWebhookHandler(t *testing.T) {
	// This is a simplified test - in a real implementation you would
	// test with actual admission requests
	
	handler := NewWebhookHandler(nil) // nil client for testing
	
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "observability.io/v1alpha1",
			"kind":       "ObservabilityPlatform",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"monitoring": map[string]interface{}{
					"prometheus": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
	}

	gvk := schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1alpha1",
		Kind:    "ObservabilityPlatform",
	}

	response, err := handler.CheckAndRespond(nil, obj, gvk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !response.Allowed {
		t.Error("expected request to be allowed")
	}

	if len(response.Warnings) == 0 {
		t.Error("expected deprecation warnings")
	}
}
