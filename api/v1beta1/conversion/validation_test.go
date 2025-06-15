/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
)

func TestFieldValidator_ValidateConversion(t *testing.T) {
	logger := log.Log.WithName("test")
	validator := conversion.NewFieldValidator(logger)

	tests := []struct {
		name           string
		srcVersion     string
		dstVersion     string
		src            interface{}
		dst            interface{}
		expectedErrors int
		expectedWarnings int
		expectedDataLoss bool
	}{
		{
			name:       "v1alpha1 to v1beta1 - valid conversion",
			srcVersion: "v1alpha1",
			dstVersion: "v1beta1",
			src: &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 3,
						},
					},
				},
			},
			dst: &v1beta1.ObservabilityPlatform{
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 3,
						},
					},
				},
			},
			expectedErrors:   0,
			expectedWarnings: 0,
			expectedDataLoss: false,
		},
		{
			name:       "v1alpha1 to v1beta1 - deprecated field usage",
			srcVersion: "v1alpha1",
			dstVersion: "v1beta1",
			src: &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							CustomConfig: map[string]string{
								"externalLabels": `{"cluster": "prod"}`,
							},
						},
					},
				},
			},
			dst: &v1beta1.ObservabilityPlatform{},
			expectedErrors:   0,
			expectedWarnings: 1, // Deprecated field warning
			expectedDataLoss: false,
		},
		{
			name:       "v1beta1 to v1alpha1 - data loss",
			srcVersion: "v1beta1",
			dstVersion: "v1alpha1",
			src: &v1beta1.ObservabilityPlatform{
				Spec: v1beta1.ObservabilityPlatformSpec{
					Security: &v1beta1.SecurityConfig{
						TLS: v1beta1.TLSConfig{
							Enabled: true,
						},
					},
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							AdditionalScrapeConfigs: []string{"job1", "job2"},
						},
					},
				},
			},
			dst:              &v1alpha1.ObservabilityPlatform{},
			expectedErrors:   0,
			expectedWarnings: 2, // Security and scrape configs will be lost
			expectedDataLoss: true,
		},
		{
			name:       "Invalid resource quantity",
			srcVersion: "v1alpha1",
			dstVersion: "v1beta1",
			src: &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									Memory: "invalid-quantity",
								},
							},
						},
					},
				},
			},
			dst:              &v1beta1.ObservabilityPlatform{},
			expectedErrors:   1,
			expectedWarnings: 0,
			expectedDataLoss: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateConversion(tt.srcVersion, tt.dstVersion, tt.src, tt.dst)

			assert.Equal(t, tt.expectedErrors, len(result.Errors), "unexpected number of errors")
			assert.Equal(t, tt.expectedWarnings, len(result.Warnings), "unexpected number of warnings")
			assert.Equal(t, tt.expectedDataLoss, result.Metrics.DataLossFields > 0, "unexpected data loss")
		})
	}
}

func TestFieldValidator_ResourceValidation(t *testing.T) {
	logger := log.Log.WithName("test")
	validator := conversion.NewFieldValidator(logger)

	tests := []struct {
		name          string
		resources     v1alpha1.ResourceRequirements
		expectedError bool
		errorContains string
	}{
		{
			name: "valid resources",
			resources: v1alpha1.ResourceRequirements{
				Requests: v1alpha1.ResourceList{
					Memory: "4Gi",
					CPU:    "1",
				},
				Limits: v1alpha1.ResourceList{
					Memory: "8Gi",
					CPU:    "2",
				},
			},
			expectedError: false,
		},
		{
			name: "invalid memory quantity",
			resources: v1alpha1.ResourceRequirements{
				Requests: v1alpha1.ResourceList{
					Memory: "4GiB", // Invalid unit
				},
			},
			expectedError: true,
			errorContains: "invalid memory quantity",
		},
		{
			name: "invalid CPU quantity",
			resources: v1alpha1.ResourceRequirements{
				Requests: v1alpha1.ResourceList{
					CPU: "1core", // Invalid format
				},
			},
			expectedError: true,
			errorContains: "invalid CPU quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldPath := field.NewPath("spec", "resources")
			errors := validator.ValidateResources(fieldPath, tt.resources, nil)

			if tt.expectedError {
				assert.NotEmpty(t, errors, "expected validation error")
				if tt.errorContains != "" {
					found := false
					for _, err := range errors {
						if contains(err.Detail, tt.errorContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "expected error containing: %s", tt.errorContains)
				}
			} else {
				assert.Empty(t, errors, "unexpected validation error")
			}
		})
	}
}

func TestCompatibilityMatrix_FieldMappings(t *testing.T) {
	matrix := conversion.NewCompatibilityMatrix()

	tests := []struct {
		name          string
		srcVersion    string
		dstVersion    string
		minMappings   int
		checkMappings []string
	}{
		{
			name:        "v1alpha1 to v1beta1 mappings exist",
			srcVersion:  "v1alpha1",
			dstVersion:  "v1beta1",
			minMappings: 10,
			checkMappings: []string{
				"Spec.Paused",
				"Spec.Components.Prometheus.Enabled",
				"Spec.Components.Prometheus.Version",
			},
		},
		{
			name:        "v1beta1 to v1alpha1 mappings exist",
			srcVersion:  "v1beta1",
			dstVersion:  "v1alpha1",
			minMappings: 5,
			checkMappings: []string{
				"Spec.Security",
				"Spec.CostOptimization",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := matrix.GetFieldMappings(tt.srcVersion, tt.dstVersion)
			assert.GreaterOrEqual(t, len(mappings), tt.minMappings, "insufficient mappings")

			// Check specific mappings exist
			for _, checkPath := range tt.checkMappings {
				found := false
				for _, mapping := range mappings {
					if mapping.SourcePath == checkPath {
						found = true
						break
					}
				}
				assert.True(t, found, "mapping not found for %s", checkPath)
			}
		})
	}
}

func TestCompatibilityMatrix_LostFields(t *testing.T) {
	matrix := conversion.NewCompatibilityMatrix()

	lostFields := matrix.GetLostFields("v1beta1", "v1alpha1")
	
	expectedLost := []string{
		"Spec.Security",
		"Spec.CostOptimization",
		"Spec.ServiceMesh",
		"Spec.Components.Prometheus.AdditionalScrapeConfigs",
		"Spec.Components.Grafana.Plugins",
	}

	for _, expected := range expectedLost {
		assert.Contains(t, lostFields, expected, "expected field %s to be marked as lost", expected)
	}
}

func TestDryRunConverter_DryRunConversion(t *testing.T) {
	logger := log.Log.WithName("test")
	converter := conversion.NewDryRunConverter(logger)
	ctx := context.Background()

	// Test successful conversion
	src := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Storage: &v1alpha1.StorageConfig{
						Size: "100Gi",
					},
				},
			},
		},
	}

	result, err := converter.DryRunConversion(ctx, src, "v1beta1")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Original)
	assert.NotEmpty(t, result.Converted)
	assert.NotEmpty(t, result.Diff)
	assert.Equal(t, "minimal", result.EstimatedImpact.RiskLevel)
	assert.False(t, result.EstimatedImpact.DataLoss)
}

func TestDryRunConverter_DataLossDetection(t *testing.T) {
	logger := log.Log.WithName("test")
	converter := conversion.NewDryRunConverter(logger)
	ctx := context.Background()

	// Test conversion with data loss
	src := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: v1beta1.ObservabilityPlatformSpec{
			Security: &v1beta1.SecurityConfig{
				TLS: v1beta1.TLSConfig{
					Enabled: true,
				},
			},
			Components: v1beta1.Components{
				Grafana: &v1beta1.GrafanaSpec{
					Enabled: true,
					Plugins: []string{"piechart-panel", "worldmap-panel"},
				},
			},
		},
	}

	result, err := converter.DryRunConversion(ctx, src, "v1alpha1")
	require.NoError(t, err)
	assert.True(t, result.Success) // Conversion succeeds but with data loss
	assert.True(t, result.EstimatedImpact.DataLoss)
	assert.NotEmpty(t, result.EstimatedImpact.LostFields)
	assert.Contains(t, result.EstimatedImpact.LostFields, "Spec.Security")
	assert.Contains(t, result.EstimatedImpact.LostFields, "Spec.Components.Grafana.Plugins")
	assert.NotEqual(t, "minimal", result.EstimatedImpact.RiskLevel)
}

func TestConversionMetrics_Recording(t *testing.T) {
	metrics := conversion.GetMetrics()

	// Record some test metrics
	timer := metrics.StartTimer("v1alpha1", "v1beta1", "test-resource")
	timer.Complete(true)

	metrics.RecordFieldValidationError("v1alpha1", "v1beta1", "spec.invalidField", "Invalid")
	metrics.RecordDataLoss("v1beta1", "v1alpha1", "spec.security")
	metrics.RecordDeprecatedFieldUsage("spec.oldField", "v1.0.0")
	metrics.RecordEnhancedField("v1alpha1", "v1beta1", "spec.newFeature", "security")

	// Get summary
	summary := conversion.GetConversionMetricsSummary()
	assert.NotNil(t, summary)
	assert.Contains(t, summary, "success_rates")
	assert.Contains(t, summary, "total_conversions")
}

func TestRollbackManager_SnapshotAndRollback(t *testing.T) {
	logger := log.Log.WithName("test")
	store := conversion.NewInMemoryRollbackStore()
	rollbackMgr := conversion.NewRollbackManager(logger, nil, store)
	ctx := context.Background()

	// Create a test object
	obj := &v1alpha1.ObservabilityPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "observability.io/v1alpha1",
			Kind:       "ObservabilityPlatform",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
					Version: "v2.48.0",
				},
			},
		},
	}

	// Create snapshot
	snapshot, err := rollbackMgr.CreateSnapshot(ctx, obj, "v1beta1", "test-conversion")
	require.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, "v1alpha1", snapshot.SourceVersion)
	assert.Equal(t, "v1beta1", snapshot.TargetVersion)

	// List available rollbacks
	rollbacks, err := rollbackMgr.ListAvailableRollbacks(ctx, types.NamespacedName{
		Name:      "test-platform",
		Namespace: "default",
	})
	require.NoError(t, err)
	assert.Len(t, rollbacks, 1)
	assert.True(t, rollbacks[0].CanRollback)
}

func TestConversionTestSuite_ScenarioTesting(t *testing.T) {
	suite := conversion.NewConversionTestSuite(t)

	// Add test scenarios
	suite.AddScenario(conversion.ConversionScenario{
		Name:        "Basic v1alpha1 to v1beta1",
		Description: "Test basic conversion with all supported fields",
		SourceObject: conversion.CreateV1Alpha1Platform(
			conversion.WithPrometheus(true, "v2.48.0"),
			conversion.WithGrafana(true, "10.2.0"),
		),
		TargetVersion: "v1beta1",
		ExpectedResult: conversion.ConversionExpectation{
			ShouldSucceed:    true,
			ExpectedWarnings: []string{},
			ExpectedErrors:   []string{},
			ExpectedRiskLevel: "minimal",
		},
		FieldChecks: []conversion.FieldCheck{
			{
				FieldPath:     "Spec.Components.Prometheus.Version",
				ExpectedValue: "v2.48.0",
				CheckType:     "exact",
			},
			{
				FieldPath:     "Spec.Security.TLS.Enabled",
				ExpectedValue: true,
				CheckType:     "exact",
			},
		},
	})

	// Run all scenarios
	suite.RunAll()
}

func TestPropertyBasedTesting_RequiredFields(t *testing.T) {
	suite := conversion.NewConversionTestSuite(t)
	propertyTester := conversion.NewPropertyBasedTester(suite)

	// Test that required fields are preserved
	propertyTester.TestConversionProperties()
}

func TestRoundTripConversion(t *testing.T) {
	suite := conversion.NewConversionTestSuite(t)
	roundTrip := conversion.NewRoundTripTest(suite)

	// Test round-trip for v1alpha1
	v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "roundtrip-test",
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Paused: true,
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							Memory: "4Gi",
							CPU:    "1",
						},
					},
				},
			},
			Global: &v1alpha1.GlobalConfig{
				LogLevel: "debug",
				ExternalLabels: map[string]string{
					"cluster": "test",
					"env":     "dev",
				},
			},
		},
	}

	roundTrip.TestRoundTrip(v1alpha1Platform)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}
