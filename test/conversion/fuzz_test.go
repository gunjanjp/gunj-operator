/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// FuzzConversion implements fuzzing for v1alpha1 to v1beta1 conversion
func FuzzV1Alpha1ToV1Beta1Conversion(f *testing.F) {
	// Add seed corpus
	seedPlatforms := [][]byte{
		mustMarshalJSON(createMinimalV1Alpha1Platform()),
		mustMarshalJSON(createTypicalV1Alpha1Platform()),
		mustMarshalJSON(createComplexV1Alpha1Platform()),
	}

	for _, seed := range seedPlatforms {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Try to unmarshal the fuzzed data
		var platform v1alpha1.ObservabilityPlatform
		err := json.Unmarshal(data, &platform)
		if err != nil {
			// Invalid JSON is expected in fuzzing
			return
		}

		// Try to convert
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err = platform.ConvertTo(v1beta1Platform)
		
		// If conversion succeeds, verify round-trip
		if err == nil {
			// Convert back
			roundtrip := &v1alpha1.ObservabilityPlatform{}
			err = roundtrip.ConvertFrom(v1beta1Platform)
			
			// Round-trip should not fail if forward conversion succeeded
			assert.NoError(t, err)
		}
	})
}

// FuzzResourceQuantityConversion fuzzes resource quantity conversion
func FuzzResourceQuantityConversion(f *testing.F) {
	// Add seed values
	seeds := []string{
		"100m",
		"1Gi",
		"500Mi",
		"2",
		"0.5",
		"1000000000n",
		"1k",
		"1M",
		"1G",
		"1T",
		"1P",
		"1E",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, quantity string) {
		// Create a platform with the fuzzed resource quantity
		platform := &v1alpha1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fuzz-resources",
				Namespace: "default",
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled: true,
						Resources: v1alpha1.ResourceRequirements{
							Requests: v1alpha1.ResourceList{
								CPU:    quantity,
								Memory: quantity,
							},
						},
					},
				},
			},
		}

		// Try to convert
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err := platform.ConvertTo(v1beta1Platform)
		
		// If the quantity is valid, conversion should succeed
		// If it's invalid, we expect an error
		if err == nil {
			// Verify the quantities were converted
			assert.NotNil(t, v1beta1Platform.Spec.Components.Prometheus.Resources.Requests)
		}
	})
}

// FuzzMapConversion fuzzes map field conversion
func FuzzMapConversion(f *testing.F) {
	// Add seed maps
	f.Add(0, "") // num entries, key prefix
	f.Add(10, "key")
	f.Add(100, "label")
	f.Add(1000, "annotation")

	f.Fuzz(func(t *testing.T, numEntries int, keyPrefix string) {
		// Limit the number of entries to prevent OOM
		if numEntries < 0 || numEntries > 10000 {
			return
		}

		// Create a platform with fuzzed maps
		platform := &v1alpha1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "fuzz-maps",
				Namespace:   "default",
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled: true,
					},
				},
				Global: v1alpha1.GlobalConfig{
					ExternalLabels: make(map[string]string),
					NodeSelector:   make(map[string]string),
				},
			},
		}

		// Fill maps with fuzzed data
		for i := 0; i < numEntries; i++ {
			key := fmt.Sprintf("%s-%d", keyPrefix, i)
			value := fmt.Sprintf("value-%d", i)
			
			if i%4 == 0 {
				platform.Labels[key] = value
			} else if i%4 == 1 {
				platform.Annotations[key] = value
			} else if i%4 == 2 {
				platform.Spec.Global.ExternalLabels[key] = value
			} else {
				platform.Spec.Global.NodeSelector[key] = value
			}
		}

		// Try to convert
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err := platform.ConvertTo(v1beta1Platform)
		
		// Conversion should always succeed for valid maps
		assert.NoError(t, err)
		
		// Verify map sizes (annotations will have +1 for conversion annotation)
		if numEntries > 0 {
			assert.GreaterOrEqual(t, len(v1beta1Platform.Annotations), len(platform.Annotations))
		}
	})
}

// FuzzSliceConversion fuzzes slice field conversion
func FuzzSliceConversion(f *testing.F) {
	f.Add(0)   // empty slice
	f.Add(1)   // single element
	f.Add(10)  // moderate size
	f.Add(100) // large size

	f.Fuzz(func(t *testing.T, numElements int) {
		// Limit to prevent OOM
		if numElements < 0 || numElements > 1000 {
			return
		}

		// Create platform with fuzzed slices
		platform := &v1alpha1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fuzz-slices",
				Namespace: "default",
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled:     true,
						RemoteWrite: make([]v1alpha1.RemoteWriteSpec, 0, numElements),
					},
					Grafana: &v1alpha1.GrafanaSpec{
						Enabled:     true,
						DataSources: make([]v1alpha1.DataSourceConfig, 0, numElements),
					},
				},
				Global: v1alpha1.GlobalConfig{
					Tolerations: make([]v1alpha1.Toleration, 0, numElements),
				},
			},
		}

		// Fill slices
		for i := 0; i < numElements; i++ {
			if i%3 == 0 {
				platform.Spec.Components.Prometheus.RemoteWrite = append(
					platform.Spec.Components.Prometheus.RemoteWrite,
					v1alpha1.RemoteWriteSpec{
						URL: fmt.Sprintf("https://remote%d.example.com/write", i),
					},
				)
			} else if i%3 == 1 {
				platform.Spec.Components.Grafana.DataSources = append(
					platform.Spec.Components.Grafana.DataSources,
					v1alpha1.DataSourceConfig{
						Name: fmt.Sprintf("ds-%d", i),
						Type: "prometheus",
						URL:  fmt.Sprintf("http://prometheus%d:9090", i),
					},
				)
			} else {
				platform.Spec.Global.Tolerations = append(
					platform.Spec.Global.Tolerations,
					v1alpha1.Toleration{
						Key:      fmt.Sprintf("key-%d", i),
						Operator: "Equal",
						Value:    fmt.Sprintf("value-%d", i),
						Effect:   "NoSchedule",
					},
				)
			}
		}

		// Convert
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err := platform.ConvertTo(v1beta1Platform)
		
		// Should always succeed
		assert.NoError(t, err)
		
		// Verify slice lengths
		assert.Len(t, v1beta1Platform.Spec.Components.Prometheus.RemoteWrite, len(platform.Spec.Components.Prometheus.RemoteWrite))
		assert.Len(t, v1beta1Platform.Spec.Components.Grafana.DataSources, len(platform.Spec.Components.Grafana.DataSources))
		assert.Len(t, v1beta1Platform.Spec.Global.Tolerations, len(platform.Spec.Global.Tolerations))
	})
}

// FuzzStringConversion fuzzes string field conversion
func FuzzStringConversion(f *testing.F) {
	// Add seed strings
	seeds := []string{
		"",
		"simple",
		"with-dash",
		"with_underscore",
		"with.dot",
		"with spaces",
		"unicode: 世界 🌍",
		"special: !@#$%^&*()",
		strings.Repeat("a", 1000),
		"\n\r\t",
		"null\x00byte",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, str string) {
		// Create platform with fuzzed string
		platform := &v1alpha1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "fuzz-string",
				Namespace: "default",
				Annotations: map[string]string{
					"fuzzed": str,
				},
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled:   true,
						Version:   str[:min(len(str), 253)], // Limit to max version length
						Retention: str,
					},
				},
				Global: v1alpha1.GlobalConfig{
					LogLevel: str,
				},
			},
			Status: v1alpha1.ObservabilityPlatformStatus{
				Phase:   str,
				Message: str,
			},
		}

		// Try to convert
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err := platform.ConvertTo(v1beta1Platform)
		
		// Conversion should handle any string
		assert.NoError(t, err)
		
		// Verify strings are preserved
		assert.Equal(t, str, v1beta1Platform.Annotations["fuzzed"])
		assert.Equal(t, str, v1beta1Platform.Spec.Global.LogLevel)
		assert.Equal(t, str, v1beta1Platform.Status.Phase)
		assert.Equal(t, str, v1beta1Platform.Status.Message)
	})
}

// FuzzComplexObjectConversion fuzzes conversion of complete objects
func FuzzComplexObjectConversion(f *testing.F) {
	// Initialize random generator
	rand.Seed(time.Now().UnixNano())

	// Add seed complexity levels
	f.Add(1)   // minimal
	f.Add(5)   // simple
	f.Add(10)  // moderate
	f.Add(20)  // complex
	f.Add(50)  // very complex

	f.Fuzz(func(t *testing.T, complexity int) {
		if complexity < 1 || complexity > 100 {
			return
		}

		// Generate a random platform based on complexity
		platform := generateRandomPlatform(complexity)

		// Try to convert
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err := platform.ConvertTo(v1beta1Platform)
		
		// Should not panic
		if err != nil {
			// Some random combinations might produce invalid resources
			return
		}

		// Try round-trip
		roundtrip := &v1alpha1.ObservabilityPlatform{}
		err = roundtrip.ConvertFrom(v1beta1Platform)
		
		// Round-trip should succeed if forward conversion succeeded
		assert.NoError(t, err)
		
		// Basic validation
		assert.Equal(t, platform.Name, roundtrip.Name)
		assert.Equal(t, platform.Namespace, roundtrip.Namespace)
	})
}

// FuzzJSONRoundTrip fuzzes JSON serialization round-trip
func FuzzJSONRoundTrip(f *testing.F) {
	// Add seed JSON documents
	seeds := []string{
		`{"apiVersion":"observability.io/v1alpha1","kind":"ObservabilityPlatform","metadata":{"name":"test"}}`,
		`{"apiVersion":"observability.io/v1alpha1","kind":"ObservabilityPlatform","metadata":{"name":"test","labels":{"app":"test"}}}`,
		`{"apiVersion":"observability.io/v1alpha1","kind":"ObservabilityPlatform","metadata":{"name":"test"},"spec":{"components":{"prometheus":{"enabled":true}}}}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, jsonStr string) {
		// Try to unmarshal
		var platform v1alpha1.ObservabilityPlatform
		err := json.Unmarshal([]byte(jsonStr), &platform)
		if err != nil {
			// Invalid JSON is expected
			return
		}

		// Convert to v1beta1
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err = platform.ConvertTo(v1beta1Platform)
		if err != nil {
			// Conversion might fail for invalid data
			return
		}

		// Marshal v1beta1
		v1beta1JSON, err := json.Marshal(v1beta1Platform)
		if err != nil {
			// Should not happen
			t.Fatalf("Failed to marshal converted platform: %v", err)
		}

		// Unmarshal back
		var unmarshaled v1beta1.ObservabilityPlatform
		err = json.Unmarshal(v1beta1JSON, &unmarshaled)
		assert.NoError(t, err)

		// Should be equivalent
		assert.Equal(t, v1beta1Platform.Name, unmarshaled.Name)
	})
}

// Helper functions

func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateRandomPlatform(complexity int) *v1alpha1.ObservabilityPlatform {
	platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("random-%d", rand.Intn(1000)),
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{},
		},
	}

	// Add random components based on complexity
	if rand.Intn(100) < complexity*5 {
		platform.Spec.Components.Prometheus = &v1alpha1.PrometheusSpec{
			Enabled:  true,
			Version:  fmt.Sprintf("v2.%d.0", 40+rand.Intn(10)),
			Replicas: int32(rand.Intn(5) + 1),
		}
	}

	if rand.Intn(100) < complexity*4 {
		platform.Spec.Components.Grafana = &v1alpha1.GrafanaSpec{
			Enabled:  true,
			Version:  fmt.Sprintf("10.%d.0", rand.Intn(5)),
			Replicas: int32(rand.Intn(3) + 1),
		}
	}

	if rand.Intn(100) < complexity*3 {
		platform.Spec.Components.Loki = &v1alpha1.LokiSpec{
			Enabled:   true,
			Version:   fmt.Sprintf("2.%d.0", 8+rand.Intn(3)),
			Retention: fmt.Sprintf("%dd", rand.Intn(30)+1),
		}
	}

	if rand.Intn(100) < complexity*2 {
		platform.Spec.Components.Tempo = &v1alpha1.TempoSpec{
			Enabled:   true,
			Version:   fmt.Sprintf("2.%d.0", rand.Intn(5)),
			Retention: fmt.Sprintf("%dh", rand.Intn(168)+1),
		}
	}

	// Add random labels
	if complexity > 5 {
		platform.Labels = make(map[string]string)
		for i := 0; i < rand.Intn(complexity); i++ {
			platform.Labels[fmt.Sprintf("label%d", i)] = fmt.Sprintf("value%d", i)
		}
	}

	// Add random annotations
	if complexity > 10 {
		platform.Annotations = make(map[string]string)
		for i := 0; i < rand.Intn(complexity/2); i++ {
			platform.Annotations[fmt.Sprintf("ann%d", i)] = fmt.Sprintf("value%d", i)
		}
	}

	// Add global config
	if complexity > 15 {
		platform.Spec.Global = v1alpha1.GlobalConfig{
			LogLevel: []string{"debug", "info", "warn", "error"}[rand.Intn(4)],
		}
		
		if rand.Intn(100) < complexity {
			platform.Spec.Global.ExternalLabels = make(map[string]string)
			for i := 0; i < rand.Intn(5); i++ {
				platform.Spec.Global.ExternalLabels[fmt.Sprintf("label%d", i)] = fmt.Sprintf("value%d", i)
			}
		}
	}

	return platform
}

// TestFuzzingCoverage verifies that our fuzzing covers important scenarios
func TestFuzzingCoverage(t *testing.T) {
	testCases := []struct {
		name     string
		platform *v1alpha1.ObservabilityPlatform
	}{
		{
			name: "empty platform",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty",
				},
			},
		},
		{
			name: "nil components",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nil-components",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: nil,
						Grafana:    nil,
						Loki:       nil,
						Tempo:      nil,
					},
				},
			},
		},
		{
			name: "extremely large maps",
			platform: func() *v1alpha1.ObservabilityPlatform {
				p := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "large-maps",
						Labels: make(map[string]string),
					},
				}
				for i := 0; i < 1000; i++ {
					p.Labels[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
				}
				return p
			}(),
		},
		{
			name: "deeply nested structures",
			platform: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nested",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Ingress: &v1alpha1.IngressConfig{
								Enabled: true,
								TLS: &v1alpha1.IngressTLS{
									Enabled:    true,
									SecretName: "tls-secret",
									Hosts:      []string{"host1", "host2", "host3"},
								},
								Annotations: map[string]string{
									"key1": "value1",
									"key2": "value2",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test conversion
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tc.platform.ConvertTo(v1beta1Platform)
			assert.NoError(t, err)

			// Test round-trip
			roundtrip := &v1alpha1.ObservabilityPlatform{}
			err = roundtrip.ConvertFrom(v1beta1Platform)
			assert.NoError(t, err)
		})
	}
}
