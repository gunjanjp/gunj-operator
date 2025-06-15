/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// TestEdgeCaseConversions tests conversion edge cases
func TestEdgeCaseConversions(t *testing.T) {
	tests := []struct {
		name        string
		setupTest   func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error)
		validate    func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform)
		expectError bool
	}{
		{
			name: "empty string fields",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "", // Empty name
						Namespace: "", // Empty namespace
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled:   true,
								Version:   "", // Empty version
								Retention: "", // Empty retention
							},
						},
						Global: v1alpha1.GlobalConfig{
							LogLevel: "", // Empty log level
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, "", v1beta1Platform.Name)
				assert.Equal(t, "", v1beta1Platform.Namespace)
				assert.Equal(t, "", v1beta1Platform.Spec.Components.Prometheus.Version)
				assert.Equal(t, "", v1beta1Platform.Spec.Components.Prometheus.Retention)
				assert.Equal(t, "", v1beta1Platform.Spec.Global.LogLevel)
			},
		},
		{
			name: "very long strings",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				longString := strings.Repeat("a", 10000)
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "long-strings",
						Namespace: "default",
						Annotations: map[string]string{
							"long-annotation": longString,
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Version: longString[:253], // Max K8s label length
							},
						},
						Global: v1alpha1.GlobalConfig{
							LogLevel: longString[:100],
						},
					},
					Status: v1alpha1.ObservabilityPlatformStatus{
						Message: longString,
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, strings.Repeat("a", 10000), v1beta1Platform.Annotations["long-annotation"])
				assert.Equal(t, strings.Repeat("a", 253), v1beta1Platform.Spec.Components.Prometheus.Version)
				assert.Equal(t, strings.Repeat("a", 100), v1beta1Platform.Spec.Global.LogLevel)
				assert.Equal(t, strings.Repeat("a", 10000), v1beta1Platform.Status.Message)
			},
		},
		{
			name: "special characters in strings",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "special-chars",
						Namespace: "default",
						Labels: map[string]string{
							"test": "value-with-special-!@#$%^&*()_+-=[]{}|;:,.<>?",
						},
						Annotations: map[string]string{
							"unicode": "Hello 世界 🌍 مرحبا мир",
							"newlines": "Line1\nLine2\rLine3\r\nLine4",
							"tabs": "Tab1\tTab2\t\tTab3",
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								RemoteWrite: []v1alpha1.RemoteWriteSpec{
									{
										URL: "https://example.com/path?query=value&special=!@#$",
										Headers: map[string]string{
											"X-Special-Header": "Value with \"quotes\" and 'apostrophes'",
										},
									},
								},
							},
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, "value-with-special-!@#$%^&*()_+-=[]{}|;:,.<>?", v1beta1Platform.Labels["test"])
				assert.Equal(t, "Hello 世界 🌍 مرحبا мир", v1beta1Platform.Annotations["unicode"])
				assert.Equal(t, "Line1\nLine2\rLine3\r\nLine4", v1beta1Platform.Annotations["newlines"])
				assert.Equal(t, "Tab1\tTab2\t\tTab3", v1beta1Platform.Annotations["tabs"])
				assert.Equal(t, "https://example.com/path?query=value&special=!@#$", v1beta1Platform.Spec.Components.Prometheus.RemoteWrite[0].URL)
				assert.Equal(t, "Value with \"quotes\" and 'apostrophes'", v1beta1Platform.Spec.Components.Prometheus.RemoteWrite[0].Headers["X-Special-Header"])
			},
		},
		{
			name: "zero values",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "zero-values",
						Namespace:  "default",
						Generation: 0,
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Paused: false, // Zero value for bool
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled:  false, // Zero value
								Replicas: 0,     // Zero value
							},
						},
						HighAvailability: &v1alpha1.HighAvailabilityConfig{
							Enabled:     false,
							MinReplicas: 0,
						},
						Backup: &v1alpha1.BackupConfig{
							Enabled:       false,
							RetentionDays: 0,
						},
					},
					Status: v1alpha1.ObservabilityPlatformStatus{
						ObservedGeneration: 0,
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, int64(0), v1beta1Platform.Generation)
				assert.False(t, v1beta1Platform.Spec.Paused)
				assert.False(t, v1beta1Platform.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, int32(0), v1beta1Platform.Spec.Components.Prometheus.Replicas)
				assert.False(t, v1beta1Platform.Spec.HighAvailability.Enabled)
				assert.Equal(t, int32(0), v1beta1Platform.Spec.HighAvailability.MinReplicas)
				assert.False(t, v1beta1Platform.Spec.Backup.Enabled)
				assert.Equal(t, int32(0), v1beta1Platform.Spec.Backup.RetentionDays)
				assert.Equal(t, int64(0), v1beta1Platform.Status.ObservedGeneration)
			},
		},
		{
			name: "maximum values",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "max-values",
						Namespace:  "default",
						Generation: math.MaxInt64,
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled:  true,
								Replicas: math.MaxInt32,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU:    "9999999999",
										Memory: "9999999999Gi",
									},
								},
							},
						},
						HighAvailability: &v1alpha1.HighAvailabilityConfig{
							Enabled:     true,
							MinReplicas: math.MaxInt32,
						},
						Backup: &v1alpha1.BackupConfig{
							Enabled:       true,
							RetentionDays: math.MaxInt32,
						},
					},
					Status: v1alpha1.ObservabilityPlatformStatus{
						ObservedGeneration: math.MaxInt64,
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, int64(math.MaxInt64), v1beta1Platform.Generation)
				assert.Equal(t, int32(math.MaxInt32), v1beta1Platform.Spec.Components.Prometheus.Replicas)
				assert.Equal(t, "9999999999", v1beta1Platform.Spec.Components.Prometheus.Resources.Requests.Cpu().String())
				assert.Equal(t, int32(math.MaxInt32), v1beta1Platform.Spec.HighAvailability.MinReplicas)
				assert.Equal(t, int32(math.MaxInt32), v1beta1Platform.Spec.Backup.RetentionDays)
				assert.Equal(t, int64(math.MaxInt64), v1beta1Platform.Status.ObservedGeneration)
			},
		},
		{
			name: "invalid resource quantities",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
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
										CPU:    "not-a-number",
										Memory: "invalid-quantity",
									},
								},
							},
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			expectError: true,
		},
		{
			name: "nil annotations and labels",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "nil-maps",
						Namespace:   "default",
						Labels:      nil,
						Annotations: nil,
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
						Global: v1alpha1.GlobalConfig{
							ExternalLabels: nil,
							NodeSelector:   nil,
							Tolerations:    nil,
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				// Annotations should be created for conversion metadata
				assert.NotNil(t, v1beta1Platform.Annotations)
				assert.Contains(t, v1beta1Platform.Annotations, "observability.io/converted-from")
				
				// Labels should remain nil if they were nil
				assert.Nil(t, v1beta1Platform.Labels)
				
				// Global config nils should remain nil
				assert.Nil(t, v1beta1Platform.Spec.Global.ExternalLabels)
				assert.Nil(t, v1beta1Platform.Spec.Global.NodeSelector)
				assert.Nil(t, v1beta1Platform.Spec.Global.Tolerations)
			},
		},
		{
			name: "duplicate keys in maps",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "duplicate-keys",
						Namespace: "default",
						Labels: map[string]string{
							"key": "value1",
							// Can't actually have duplicate keys in Go maps
							// But we can test overwriting
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
				}

				// Simulate a second update with the same key
				v1alpha1Platform.Labels["key"] = "value2"

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				// Should have the last value
				assert.Equal(t, "value2", v1beta1Platform.Labels["key"])
			},
		},
		{
			name: "self-referential UID",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				uid := types.UID("12345-67890")
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "self-ref",
						Namespace: "default",
						UID:       uid,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "observability.io/v1alpha1",
								Kind:       "ObservabilityPlatform",
								Name:       "self-ref",
								UID:        uid, // Self-referential
							},
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, v1alpha1Platform.UID, v1beta1Platform.UID)
				assert.Len(t, v1beta1Platform.OwnerReferences, 1)
				assert.Equal(t, v1alpha1Platform.UID, v1beta1Platform.OwnerReferences[0].UID)
			},
		},
		{
			name: "circular references in maps",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "circular",
						Namespace: "default",
						Labels: map[string]string{
							"ref1": "points-to-ref2",
							"ref2": "points-to-ref3",
							"ref3": "points-to-ref1",
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								RemoteWrite: []v1alpha1.RemoteWriteSpec{
									{
										URL: "https://remote1.example.com",
										Headers: map[string]string{
											"X-Forward-To": "remote2",
										},
									},
									{
										URL: "https://remote2.example.com",
										Headers: map[string]string{
											"X-Forward-To": "remote1",
										},
									},
								},
							},
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, "points-to-ref2", v1beta1Platform.Labels["ref1"])
				assert.Equal(t, "points-to-ref3", v1beta1Platform.Labels["ref2"])
				assert.Equal(t, "points-to-ref1", v1beta1Platform.Labels["ref3"])
				assert.Len(t, v1beta1Platform.Spec.Components.Prometheus.RemoteWrite, 2)
			},
		},
		{
			name: "inconsistent array lengths",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "array-test",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled:     true,
								RemoteWrite: []v1alpha1.RemoteWriteSpec{}, // Empty
							},
							Grafana: &v1alpha1.GrafanaSpec{
								Enabled: true,
								DataSources: []v1alpha1.DataSourceConfig{
									{Name: "ds1"},
									{Name: "ds2"},
									{Name: "ds3"},
								}, // 3 items
							},
							Loki: &v1alpha1.LokiSpec{
								Enabled: true,
								Storage: &v1alpha1.StorageConfig{
									AccessModes: []string{"ReadWriteOnce", "ReadWriteMany"}, // 2 items
								},
							},
						},
						Global: v1alpha1.GlobalConfig{
							Tolerations: make([]v1alpha1.Toleration, 100), // 100 empty items
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.Len(t, v1beta1Platform.Spec.Components.Prometheus.RemoteWrite, 0)
				assert.Len(t, v1beta1Platform.Spec.Components.Grafana.DataSources, 3)
				assert.Len(t, v1beta1Platform.Spec.Components.Loki.Storage.AccessModes, 2)
				assert.Len(t, v1beta1Platform.Spec.Global.Tolerations, 100)
			},
		},
		{
			name: "time edge cases",
			setupTest: func() (*v1alpha1.ObservabilityPlatform, *v1beta1.ObservabilityPlatform, error) {
				zeroTime := metav1.Time{}
				farFutureTime := metav1.NewTime(time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC))
				
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "time-test",
						Namespace:         "default",
						CreationTimestamp: zeroTime,
						DeletionTimestamp: &farFutureTime,
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
					Status: v1alpha1.ObservabilityPlatformStatus{
						LastTransitionTime: &zeroTime,
						ComponentStatus: map[string]v1alpha1.ComponentStatus{
							"prometheus": {
								LastUpdateTime: &farFutureTime,
							},
						},
					},
				}

				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			validate: func(t *testing.T, v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) {
				assert.True(t, v1beta1Platform.CreationTimestamp.IsZero())
				assert.NotNil(t, v1beta1Platform.DeletionTimestamp)
				assert.Equal(t, int64(9999), v1beta1Platform.DeletionTimestamp.Year())
				assert.True(t, v1beta1Platform.Status.LastTransitionTime.IsZero())
				assert.NotNil(t, v1beta1Platform.Status.ComponentStatus["prometheus"].LastUpdateTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1alpha1Platform, v1beta1Platform, err := tt.setupTest()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, v1alpha1Platform, v1beta1Platform)
			}
		})
	}
}

// TestConversionWithMalformedData tests conversion with malformed or corrupted data
func TestConversionWithMalformedData(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectError   bool
		errorContains string
	}{
		{
			name: "malformed JSON",
			jsonData: `{
				"apiVersion": "observability.io/v1alpha1",
				"kind": "ObservabilityPlatform",
				"metadata": {
					"name": "test"
					// Missing closing brace
			`,
			expectError:   true,
			errorContains: "unexpected end of JSON",
		},
		{
			name: "wrong type for field",
			jsonData: `{
				"apiVersion": "observability.io/v1alpha1",
				"kind": "ObservabilityPlatform",
				"metadata": {
					"name": "test",
					"namespace": "default"
				},
				"spec": {
					"components": {
						"prometheus": {
							"enabled": "yes", // Should be boolean
							"replicas": "three" // Should be number
						}
					}
				}
			}`,
			expectError:   true,
			errorContains: "cannot unmarshal",
		},
		{
			name: "missing required fields",
			jsonData: `{
				"apiVersion": "observability.io/v1alpha1",
				"kind": "ObservabilityPlatform",
				"metadata": {
					// Missing name
				},
				"spec": {
					// Missing components
				}
			}`,
			expectError: false, // Should still convert, just with empty values
		},
		{
			name: "extra unknown fields",
			jsonData: `{
				"apiVersion": "observability.io/v1alpha1",
				"kind": "ObservabilityPlatform",
				"metadata": {
					"name": "test",
					"namespace": "default",
					"unknownField": "should be ignored"
				},
				"spec": {
					"unknownSpec": "should also be ignored",
					"components": {
						"prometheus": {
							"enabled": true,
							"unknownPrometheusField": "ignored"
						}
					}
				}
			}`,
			expectError: false, // Unknown fields should be ignored
		},
		{
			name: "null values",
			jsonData: `{
				"apiVersion": "observability.io/v1alpha1",
				"kind": "ObservabilityPlatform",
				"metadata": {
					"name": "test",
					"namespace": "default",
					"labels": null,
					"annotations": null
				},
				"spec": {
					"components": {
						"prometheus": null,
						"grafana": null
					},
					"global": null
				}
			}`,
			expectError: false, // Null values should be handled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v1alpha1Platform v1alpha1.ObservabilityPlatform
			err := json.Unmarshal([]byte(tt.jsonData), &v1alpha1Platform)

			if err != nil {
				if tt.expectError {
					assert.Contains(t, err.Error(), tt.errorContains)
					return
				}
				t.Fatalf("Unexpected unmarshal error: %v", err)
			}

			// Try to convert
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err = v1alpha1Platform.ConvertTo(v1beta1Platform)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConversionStability tests that multiple conversions produce stable results
func TestConversionStability(t *testing.T) {
	platform := createCompleteV1Alpha1Platform()

	// Convert multiple times and ensure stability
	var previous *v1beta1.ObservabilityPlatform
	
	for i := 0; i < 10; i++ {
		current := &v1beta1.ObservabilityPlatform{}
		err := platform.ConvertTo(current)
		require.NoError(t, err)

		if previous != nil {
			// Compare JSON representations
			prevJSON, err := json.Marshal(previous)
			require.NoError(t, err)

			currJSON, err := json.Marshal(current)
			require.NoError(t, err)

			assert.JSONEq(t, string(prevJSON), string(currJSON), "Conversion %d produced different result", i)
		}

		previous = current.DeepCopy()
	}
}

// TestConversionWithResourceConstraints simulates conversion under resource constraints
func TestConversionWithResourceConstraints(t *testing.T) {
	// Create a platform with minimal resources that might cause issues
	platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-constrained",
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "0.001m", // Very small CPU
							Memory: "1Ki",    // Very small memory
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "0.01m",
							Memory: "10Ki",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size: "1Ki", // Unrealistically small storage
					},
				},
			},
		},
	}

	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	err := platform.ConvertTo(v1beta1Platform)
	require.NoError(t, err)

	// Verify the tiny values are preserved
	assert.Equal(t, "0.001m", v1beta1Platform.Spec.Components.Prometheus.Resources.Requests.Cpu().String())
	assert.Equal(t, "1Ki", v1beta1Platform.Spec.Components.Prometheus.Resources.Requests.Memory().String())
	assert.Equal(t, "1Ki", v1beta1Platform.Spec.Components.Prometheus.Storage.Size.String())
}

// TestConversionWithInvalidEnums tests conversion with invalid enum values
func TestConversionWithInvalidEnums(t *testing.T) {
	// Manually create objects with invalid enum values using reflection
	// This simulates what might happen if the validation is bypassed
	
	tests := []struct {
		name        string
		setupTest   func() *v1alpha1.ObservabilityPlatform
		expectError bool
	}{
		{
			name: "invalid phase in status",
			setupTest: func() *v1alpha1.ObservabilityPlatform {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-phase",
						Namespace: "default",
					},
					Status: v1alpha1.ObservabilityPlatformStatus{
						Phase: "InvalidPhase", // Not a valid phase
					},
				}
				return platform
			},
			expectError: false, // Should still convert, just with the invalid value
		},
		{
			name: "invalid toleration operator",
			setupTest: func() *v1alpha1.ObservabilityPlatform {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-toleration",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Global: v1alpha1.GlobalConfig{
							Tolerations: []v1alpha1.Toleration{
								{
									Key:      "test",
									Operator: "InvalidOperator", // Not a valid operator
									Value:    "test",
									Effect:   "NoSchedule",
								},
							},
						},
					},
				}
				return platform
			},
			expectError: false, // Conversion should handle invalid operators
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform := tt.setupTest()
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := platform.ConvertTo(v1beta1Platform)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConversionPreservesUnknownFields tests that unknown fields are handled properly
func TestConversionPreservesUnknownFields(t *testing.T) {
	// Create a platform with standard fields
	platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unknown-fields",
			Namespace: "default",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"extra":"field"}`,
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}

	// Convert to v1beta1
	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	err := platform.ConvertTo(v1beta1Platform)
	require.NoError(t, err)

	// The annotation should be preserved
	assert.Contains(t, v1beta1Platform.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	
	// Convert back to v1alpha1
	roundtrip := &v1alpha1.ObservabilityPlatform{}
	err = roundtrip.ConvertFrom(v1beta1Platform)
	require.NoError(t, err)

	// The annotation should still be there
	assert.Contains(t, roundtrip.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
}
