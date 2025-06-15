package drift_test

import (
	"context"
	"testing"

	"github.com/gunjanjp/gunj-operator/internal/gitops"
	"github.com/gunjanjp/gunj-operator/internal/gitops/drift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDriftDetector_DetectDrift(t *testing.T) {
	tests := []struct {
		name          string
		expected      runtime.Object
		actual        runtime.Object
		expectDrift   bool
		driftedFields []string
	}{
		{
			name: "No drift - identical ConfigMaps",
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			actual: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
					ResourceVersion: "12345", // Should be ignored
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			expectDrift:   false,
			driftedFields: []string{},
		},
		{
			name: "Drift in data field",
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			actual: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"key1": "value1-modified",
					"key2": "value2",
				},
			},
			expectDrift:   true,
			driftedFields: []string{"data.key1"},
		},
		{
			name: "Drift in labels",
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "test",
						"version": "v1",
					},
				},
			},
			actual: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "test",
						"version": "v2", // Changed
						"env":     "prod", // Added
					},
				},
			},
			expectDrift:   true,
			driftedFields: []string{"metadata.labels.version", "metadata.labels.env"},
		},
		{
			name: "Ignored drift - annotations",
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
			},
			actual: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
					Annotations: map[string]string{
						"kubectl.kubernetes.io/last-applied-configuration": "...",
						"gitops.observability.io/ignore-drift":             "true",
					},
				},
			},
			expectDrift:   false,
			driftedFields: []string{},
		},
		{
			name: "Missing field drift",
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			actual: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"key1": "value1",
					// key2 is missing
				},
			},
			expectDrift:   true,
			driftedFields: []string{"data.key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create drift detector
			detector := drift.NewDetector(nil, ctrl.Log.WithName("test"))

			// Detect drift
			result, err := detector.DetectDrift(context.Background(), tt.expected, tt.actual)
			require.NoError(t, err)

			// Verify drift detection
			assert.Equal(t, tt.expectDrift, result.HasDrift, "Drift detection mismatch")

			// Verify drifted fields
			if tt.expectDrift {
				assert.Len(t, result.DriftedFields, len(tt.driftedFields), "Number of drifted fields mismatch")
				
				for _, field := range tt.driftedFields {
					_, exists := result.DriftedFields[field]
					assert.True(t, exists, "Expected drifted field %s not found", field)
				}
			}
		})
	}
}

func TestDriftDetector_DetectDriftForPlatform(t *testing.T) {
	// Create test resources
	resources := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      "prometheus-config",
					"namespace": "monitoring",
				},
				"data": map[string]interface{}{
					"prometheus.yml": "global:\n  scrape_interval: 15s",
				},
			},
		},
		{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "prometheus",
					"namespace": "monitoring",
				},
				"spec": map[string]interface{}{
					"replicas": 3,
					"selector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"app": "prometheus",
						},
					},
					"template": map[string]interface{}{
						"metadata": map[string]interface{}{
							"labels": map[string]interface{}{
								"app": "prometheus",
							},
						},
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name":  "prometheus",
									"image": "prom/prometheus:v2.48.0",
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name             string
		platformName     string
		namespace        string
		actualResources  []runtime.Object
		expectedDriftCnt int
		driftTypes       []drift.DriftType
	}{
		{
			name:         "No drift - resources match",
			platformName: "test-platform",
			namespace:    "monitoring",
			actualResources: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-config",
						Namespace: "monitoring",
					},
					Data: map[string]string{
						"prometheus.yml": "global:\n  scrape_interval: 15s",
					},
				},
			},
			expectedDriftCnt: 1, // Deployment is missing
			driftTypes:       []drift.DriftType{drift.DriftTypeMissing},
		},
		{
			name:         "Modified resource drift",
			platformName: "test-platform",
			namespace:    "monitoring",
			actualResources: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-config",
						Namespace: "monitoring",
					},
					Data: map[string]string{
						"prometheus.yml": "global:\n  scrape_interval: 30s", // Changed
					},
				},
			},
			expectedDriftCnt: 2, // ConfigMap modified, Deployment missing
			driftTypes:       []drift.DriftType{drift.DriftTypeModified, drift.DriftTypeMissing},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with actual resources
			fakeClient := fake.NewClientBuilder().
				WithRuntimeObjects(tt.actualResources...).
				Build()

			// Create drift detector
			detector := drift.NewDetector(fakeClient, ctrl.Log.WithName("test"))

			// Detect drift for platform
			result, err := detector.DetectDriftForPlatform(
				context.Background(),
				tt.platformName,
				tt.namespace,
				resources,
			)
			require.NoError(t, err)

			// Verify results
			assert.Equal(t, tt.platformName, result.PlatformName)
			assert.Equal(t, tt.namespace, result.Namespace)
			assert.Equal(t, tt.expectedDriftCnt > 0, result.HasDrift)
			assert.Len(t, result.Resources, tt.expectedDriftCnt)

			// Verify drift types
			for i, driftType := range tt.driftTypes {
				if i < len(result.Resources) {
					assert.Equal(t, driftType, result.Resources[i].DriftType)
				}
			}
		})
	}
}

func TestDriftDetector_Remediate(t *testing.T) {
	// Create drift result
	driftResult := &gitops.DriftResult{
		HasDrift: true,
		DriftedFields: map[string]gitops.DriftDetail{
			"data.key1": {
				Path:     "data.key1",
				Expected: "expected-value",
				Actual:   "actual-value",
			},
		},
	}

	// Create fake client
	fakeClient := fake.NewClientBuilder().Build()

	// Create drift detector
	detector := drift.NewDetector(fakeClient, ctrl.Log.WithName("test"))

	// Test remediation
	err := detector.Remediate(context.Background(), driftResult)
	assert.NoError(t, err)
}

func TestDriftResult_GetDriftSummary(t *testing.T) {
	tests := []struct {
		name     string
		result   *drift.PlatformDriftResult
		expected string
	}{
		{
			name: "No drift",
			result: &drift.PlatformDriftResult{
				HasDrift: false,
			},
			expected: "No drift detected",
		},
		{
			name: "Single missing resource",
			result: &drift.PlatformDriftResult{
				HasDrift: true,
				Resources: []drift.ResourceDriftResult{
					{
						DriftType: drift.DriftTypeMissing,
					},
				},
			},
			expected: "Drift detected: 1 missing resources",
		},
		{
			name: "Multiple drift types",
			result: &drift.PlatformDriftResult{
				HasDrift: true,
				Resources: []drift.ResourceDriftResult{
					{DriftType: drift.DriftTypeMissing},
					{DriftType: drift.DriftTypeMissing},
					{DriftType: drift.DriftTypeModified},
					{DriftType: drift.DriftTypeModified},
					{DriftType: drift.DriftTypeModified},
					{DriftType: drift.DriftTypeExtra},
				},
			},
			expected: "Drift detected: 2 missing, 3 modified, 1 extra resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.GetDriftSummary()
			assert.Equal(t, tt.expected, summary)
		})
	}
}

func TestDriftDetector_IgnoreFields(t *testing.T) {
	// Create detector with custom ignore fields
	detector := drift.NewDetector(nil, ctrl.Log.WithName("test"))

	// Create test objects with ignored fields
	expected := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":              "test",
			"namespace":         "default",
			"resourceVersion":   "12345",
			"generation":        1,
			"uid":               "abc-123",
			"creationTimestamp": "2024-01-01T00:00:00Z",
		},
		"data": map[string]interface{}{
			"key": "value",
		},
		"status": map[string]interface{}{
			"phase": "Active",
		},
	}

	actual := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":              "test",
			"namespace":         "default",
			"resourceVersion":   "67890", // Different but should be ignored
			"generation":        2,        // Different but should be ignored
			"uid":               "def-456", // Different but should be ignored
			"creationTimestamp": "2024-01-02T00:00:00Z", // Different but should be ignored
		},
		"data": map[string]interface{}{
			"key": "value",
		},
		"status": map[string]interface{}{
			"phase": "Terminating", // Different but should be ignored
		},
	}

	// Convert to unstructured
	expectedObj := &unstructured.Unstructured{Object: expected}
	actualObj := &unstructured.Unstructured{Object: actual}

	// Detect drift
	result, err := detector.DetectDrift(context.Background(), expectedObj, actualObj)
	require.NoError(t, err)

	// Should not detect drift in ignored fields
	assert.False(t, result.HasDrift, "Should not detect drift in ignored fields")
	assert.Empty(t, result.DriftedFields)
}

func TestDriftDetector_ComplexObjectDrift(t *testing.T) {
	// Test complex nested object drift detection
	expected := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "prometheus",
			"namespace": "monitoring",
			"labels": map[string]interface{}{
				"app":     "prometheus",
				"version": "v2.48.0",
			},
		},
		"spec": map[string]interface{}{
			"replicas": 3,
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "prometheus",
							"image": "prom/prometheus:v2.48.0",
							"ports": []interface{}{
								map[string]interface{}{
									"containerPort": 9090,
									"name":          "http",
								},
							},
							"env": []interface{}{
								map[string]interface{}{
									"name":  "POD_NAME",
									"value": "prometheus",
								},
							},
						},
					},
				},
			},
		},
	}

	actual := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "prometheus",
			"namespace": "monitoring",
			"labels": map[string]interface{}{
				"app":     "prometheus",
				"version": "v2.49.0", // Changed
			},
		},
		"spec": map[string]interface{}{
			"replicas": 5, // Changed
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "prometheus",
							"image": "prom/prometheus:v2.49.0", // Changed
							"ports": []interface{}{
								map[string]interface{}{
									"containerPort": 9090,
									"name":          "http",
								},
							},
							"env": []interface{}{
								map[string]interface{}{
									"name":  "POD_NAME",
									"value": "prometheus",
								},
								map[string]interface{}{ // Added
									"name":  "EXTRA_ENV",
									"value": "extra",
								},
							},
						},
					},
				},
			},
		},
	}

	// Convert to unstructured
	expectedObj := &unstructured.Unstructured{Object: expected}
	actualObj := &unstructured.Unstructured{Object: actual}

	// Create detector
	detector := drift.NewDetector(nil, ctrl.Log.WithName("test"))

	// Detect drift
	result, err := detector.DetectDrift(context.Background(), expectedObj, actualObj)
	require.NoError(t, err)

	// Verify drift detected
	assert.True(t, result.HasDrift)
	
	// Check specific drifted fields
	expectedDrifts := []string{
		"metadata.labels.version",
		"spec.replicas",
		"spec.template.spec.containers",
	}

	for _, field := range expectedDrifts {
		found := false
		for driftField := range result.DriftedFields {
			if driftField == field {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected drift in field %s not detected", field)
	}
}
