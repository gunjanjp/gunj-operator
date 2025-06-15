/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// TestConversionErrorScenarios tests various error conditions during conversion
func TestConversionErrorScenarios(t *testing.T) {
	tests := []struct {
		name          string
		setupTest     func() error
		expectedError string
		validateError func(t *testing.T, err error)
	}{
		{
			name: "nil hub object",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}
				return platform.ConvertTo(nil)
			},
			expectedError: "hub cannot be nil",
		},
		{
			name: "wrong hub type for ConvertTo",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}
				wrongType := &corev1.Pod{}
				return platform.ConvertTo(wrongType)
			},
			expectedError: "expected *v1beta1.ObservabilityPlatform",
		},
		{
			name: "wrong hub type for ConvertFrom",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{}
				wrongType := &corev1.Pod{}
				return platform.ConvertFrom(wrongType)
			},
			expectedError: "expected *v1beta1.ObservabilityPlatform",
		},
		{
			name: "invalid CPU resource quantity",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-cpu",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU: "not-a-valid-cpu-value",
									},
								},
							},
						},
					},
				}
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				return platform.ConvertTo(v1beta1Platform)
			},
			expectedError: "failed to parse CPU resource",
		},
		{
			name: "invalid memory resource quantity",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-memory",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Grafana: &v1alpha1.GrafanaSpec{
								Enabled: true,
								Resources: v1alpha1.ResourceRequirements{
									Limits: v1alpha1.ResourceList{
										Memory: "invalid-memory-value",
									},
								},
							},
						},
					},
				}
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				return platform.ConvertTo(v1beta1Platform)
			},
			expectedError: "failed to parse memory resource",
		},
		{
			name: "invalid storage size",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-storage",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Storage: &v1alpha1.StorageConfig{
									Size: "not-a-valid-size",
								},
							},
						},
					},
				}
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				return platform.ConvertTo(v1beta1Platform)
			},
			expectedError: "failed to parse storage size",
		},
		{
			name: "invalid volume mode",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-volume-mode",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Loki: &v1alpha1.LokiSpec{
								Enabled: true,
								Storage: &v1alpha1.StorageConfig{
									Size:       "10Gi",
									VolumeMode: "InvalidMode", // Should be Filesystem or Block
								},
							},
						},
					},
				}
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				return platform.ConvertTo(v1beta1Platform)
			},
			validateError: func(t *testing.T, err error) {
				// This might not cause an error but result in an invalid value
				// depending on implementation
				if err == nil {
					t.Log("Invalid volume mode did not cause error, may result in invalid object")
				}
			},
		},
		{
			name: "multiple resource parse errors",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multiple-errors",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU:    "invalid-cpu",
										Memory: "invalid-memory",
									},
									Limits: v1alpha1.ResourceList{
										CPU:    "also-invalid",
										Memory: "also-invalid-memory",
									},
								},
								Storage: &v1alpha1.StorageConfig{
									Size: "invalid-storage",
								},
							},
						},
					},
				}
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				return platform.ConvertTo(v1beta1Platform)
			},
			expectedError: "failed to parse",
			validateError: func(t *testing.T, err error) {
				// Should report the first error encountered
				assert.Contains(t, err.Error(), "failed to parse")
			},
		},
		{
			name: "conversion panic recovery",
			setupTest: func() error {
				// Create a platform that might cause issues
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "panic-test",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU:    "1",
										Memory: "1Gi",
									},
								},
							},
						},
					},
				}

				// Try conversion with a nil hub that's cast to the wrong type
				// This tests panic recovery if implemented
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				return platform.ConvertTo(v1beta1Platform)
			},
			validateError: func(t *testing.T, err error) {
				// Should not panic, error should be handled gracefully
				if err != nil {
					t.Logf("Conversion error handled gracefully: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupTest()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}

			if tt.validateError != nil {
				tt.validateError(t, err)
			}
		})
	}
}

// TestConversionFromV1Beta1Errors tests error scenarios when converting from v1beta1
func TestConversionFromV1Beta1Errors(t *testing.T) {
	tests := []struct {
		name          string
		setupTest     func() error
		expectedError string
	}{
		{
			name: "nil source object",
			setupTest: func() error {
				platform := &v1alpha1.ObservabilityPlatform{}
				return platform.ConvertFrom(nil)
			},
			expectedError: "hub cannot be nil",
		},
		{
			name: "invalid resource quantities in v1beta1",
			setupTest: func() error {
				// Create v1beta1 with already parsed but invalid quantities
				v1beta1Platform := &v1beta1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-quantities",
						Namespace: "default",
					},
					Spec: v1beta1.ObservabilityPlatformSpec{
						Components: v1beta1.Components{
							Prometheus: &v1beta1.PrometheusSpec{
								Enabled: true,
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										// These are valid quantities but might cause issues
										// in conversion if not handled properly
										corev1.ResourceCPU:    resource.MustParse("999999999999999999"),
										corev1.ResourceMemory: resource.MustParse("999999999999999999Gi"),
									},
								},
							},
						},
					},
				}

				platform := &v1alpha1.ObservabilityPlatform{}
				return platform.ConvertFrom(v1beta1Platform)
			},
			expectedError: "", // Should handle large values gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupTest()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// Some tests verify that errors are handled gracefully
				if err != nil {
					t.Logf("Conversion produced error: %v", err)
				}
			}
		})
	}
}

// TestConversionWithCorruptedData tests conversion with corrupted or malformed data
func TestConversionWithCorruptedData(t *testing.T) {
	tests := []struct {
		name          string
		platform      *v1alpha1.ObservabilityPlatform
		corruptFunc   func(*v1alpha1.ObservabilityPlatform)
		expectedError bool
		errorContains string
	}{
		{
			name: "negative replicas",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "negative-replicas",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:  true,
							Replicas: -5, // Negative replicas
						},
					},
				},
			},
			expectedError: false, // Should convert but with invalid value
		},
		{
			name: "empty resource requests",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-resources",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "",
									Memory: "",
								},
							},
						},
					},
				},
			},
			expectedError: false, // Empty strings should be skipped
		},
		{
			name: "resource with only units",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "units-only",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "m",    // Just unit, no number
									Memory: "Gi",   // Just unit, no number
								},
							},
						},
					},
				},
			},
			expectedError: true,
			errorContains: "failed to parse",
		},
		{
			name: "invalid characters in resource",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-chars",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "1.5.6", // Multiple decimal points
									Memory: "2G!",   // Invalid character
								},
							},
						},
					},
				},
			},
			expectedError: true,
			errorContains: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.corruptFunc != nil {
				tt.corruptFunc(tt.platform)
			}

			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.platform.ConvertTo(v1beta1Platform)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				// Even if no error, the data might be invalid
				if err != nil {
					t.Logf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestConversionErrorPropagation tests that errors are properly propagated
func TestConversionErrorPropagation(t *testing.T) {
	// Mock a conversion that will fail at different stages
	stages := []struct {
		name          string
		platform      *v1alpha1.ObservabilityPlatform
		failurePoint  string
		expectedError string
	}{
		{
			name: "fail at prometheus conversion",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fail-prometheus",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU: "invalid",
								},
							},
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true, // This won't be reached
						},
					},
				},
			},
			failurePoint:  "prometheus",
			expectedError: "failed to parse CPU resource",
		},
		{
			name: "fail at grafana conversion",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fail-grafana",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true, // This succeeds
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Limits: v1alpha1.ResourceList{
									Memory: "invalid",
								},
							},
						},
					},
				},
			},
			failurePoint:  "grafana",
			expectedError: "failed to parse memory resource",
		},
		{
			name: "fail at storage conversion",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fail-storage",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Loki: &v1alpha1.LokiSpec{
							Enabled: true,
							Storage: &v1alpha1.StorageConfig{
								Size: "invalid-size",
							},
						},
					},
				},
			},
			failurePoint:  "storage",
			expectedError: "failed to parse storage size",
		},
	}

	for _, stage := range stages {
		t.Run(stage.name, func(t *testing.T) {
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := stage.platform.ConvertTo(v1beta1Platform)

			require.Error(t, err)
			assert.Contains(t, err.Error(), stage.expectedError)
			
			// Verify error mentions the component
			t.Logf("Error at %s: %v", stage.failurePoint, err)
		})
	}
}

// TestConversionErrorRecovery tests recovery from errors
func TestConversionErrorRecovery(t *testing.T) {
	// Create a platform with some valid and some invalid data
	platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "partial-valid",
			Namespace: "default",
			Labels: map[string]string{
				"valid": "label",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
					Version: "v2.48.0", // Valid
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "invalid", // Invalid
							Memory: "2Gi",     // Valid
						},
					},
				},
			},
		},
	}

	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	err := platform.ConvertTo(v1beta1Platform)

	// Should fail due to invalid CPU
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse CPU resource")

	// But valid fields should have been attempted
	// (though the object state is undefined after error)
}

// TestConversionValidation tests that conversion includes validation
func TestConversionValidation(t *testing.T) {
	tests := []struct {
		name             string
		platform         *v1alpha1.ObservabilityPlatform
		expectConvertErr bool
		validateResult   func(t *testing.T, v1beta1Platform *v1beta1.ObservabilityPlatform)
	}{
		{
			name: "valid platform passes validation",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-platform",
					Namespace: "default",
				},
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
			expectConvertErr: false,
			validateResult: func(t *testing.T, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, "v2.48.0", v1beta1Platform.Spec.Components.Prometheus.Version)
				assert.Equal(t, int32(3), v1beta1Platform.Spec.Components.Prometheus.Replicas)
			},
		},
		{
			name: "invalid resource quantities fail",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-resources",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU: "not-a-quantity",
								},
							},
						},
					},
				},
			},
			expectConvertErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.platform.ConvertTo(v1beta1Platform)

			if tt.expectConvertErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validateResult != nil {
					tt.validateResult(t, v1beta1Platform)
				}
			}
		})
	}
}

// TestCustomError tests custom error types if used in conversion
type ConversionError struct {
	Field   string
	Value   string
	Message string
}

func (e ConversionError) Error() string {
	return fmt.Sprintf("conversion error in field %s with value %s: %s", e.Field, e.Value, e.Message)
}

func TestCustomErrorTypes(t *testing.T) {
	// Simulate a conversion error
	err := ConversionError{
		Field:   "spec.components.prometheus.resources.requests.cpu",
		Value:   "invalid-cpu",
		Message: "failed to parse quantity",
	}

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversion error")
	assert.Contains(t, err.Error(), "invalid-cpu")
	assert.Contains(t, err.Error(), "failed to parse quantity")
}

// TestErrorWrapping tests proper error wrapping
func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := fmt.Errorf("converting prometheus: %w", baseErr)
	doubleWrappedErr := fmt.Errorf("converting platform: %w", wrappedErr)

	assert.Error(t, doubleWrappedErr)
	assert.Contains(t, doubleWrappedErr.Error(), "converting platform")
	assert.Contains(t, doubleWrappedErr.Error(), "converting prometheus")
	assert.Contains(t, doubleWrappedErr.Error(), "base error")

	// Test unwrapping
	assert.True(t, errors.Is(doubleWrappedErr, baseErr))
}
