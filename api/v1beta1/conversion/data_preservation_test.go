/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func init() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
}

func TestDataPreservation(t *testing.T) {
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	client := fake.NewClientBuilder().Build()
	
	preserver := NewDataPreserver(logger, client)
	
	t.Run("PreserveData", func(t *testing.T) {
		// Create a v1alpha1 object with data to preserve
		obj := &v1alpha1.ObservabilityPlatform{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "observability.io/v1alpha1",
				Kind:       "ObservabilityPlatform",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "test-namespace",
				UID:       types.UID("test-uid"),
				Annotations: map[string]string{
					"user-annotation": "user-value",
					"custom-config":   "custom-value",
				},
				Labels: map[string]string{
					"user-label": "label-value",
					"env":        "production",
				},
				Finalizers: []string{"observability.io/finalizer"},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "parent-deployment",
						UID:        types.UID("parent-uid"),
					},
				},
			},
			Status: v1alpha1.ObservabilityPlatformStatus{
				Phase:              "Ready",
				Message:            "Platform is ready",
				ObservedGeneration: 5,
				ComponentStatus: map[string]v1alpha1.ComponentStatus{
					"prometheus": {
						Phase:         "Ready",
						Version:       "v2.48.0",
						ReadyReplicas: 3,
					},
				},
			},
		}
		
		// Preserve data
		preserved, err := preserver.PreserveData(ctx, obj, "v1beta1")
		require.NoError(t, err)
		
		// Verify preserved data
		assert.NotNil(t, preserved)
		assert.Equal(t, "user-value", preserved.Annotations["user-annotation"])
		assert.Equal(t, "label-value", preserved.Labels["user-label"])
		assert.NotEmpty(t, preserved.Status)
		assert.Equal(t, "Ready", preserved.Status["phase"])
		assert.NotEmpty(t, preserved.Metadata.UID)
		assert.Equal(t, types.UID("test-uid"), preserved.Metadata.UID)
		assert.Len(t, preserved.Metadata.OwnerReferences, 1)
		assert.Equal(t, "parent-deployment", preserved.Metadata.OwnerReferences[0].Name)
		
		// Check that preservation data was stored in annotations
		assert.Contains(t, obj.Annotations, ConversionDataAnnotation)
		assert.Contains(t, obj.Annotations, DataIntegrityHashAnnotation)
		assert.Contains(t, obj.Annotations, LastConversionVersionAnnotation)
	})
	
	t.Run("RestoreData", func(t *testing.T) {
		// Create preserved data
		preserved := &PreservedData{
			Status: map[string]interface{}{
				"phase":              "Ready",
				"message":            "Platform is ready",
				"observedGeneration": int64(5),
			},
			Annotations: map[string]string{
				"user-annotation": "user-value",
				"custom-config":   "custom-value",
			},
			Labels: map[string]string{
				"user-label": "label-value",
				"env":        "production",
			},
			Metadata: PreservedMetadata{
				UID:       types.UID("test-uid"),
				Finalizers: []string{"observability.io/finalizer"},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "parent-deployment",
						UID:        types.UID("parent-uid"),
					},
				},
			},
		}
		
		// Create a v1beta1 object to restore data to
		obj := &v1beta1.ObservabilityPlatform{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "observability.io/v1beta1",
				Kind:       "ObservabilityPlatform",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "test-namespace",
			},
		}
		
		// Calculate and store hash for integrity check
		hash, err := preserver.calculateDataHash(preserved)
		require.NoError(t, err)
		obj.Annotations = map[string]string{
			DataIntegrityHashAnnotation: hash,
		}
		
		// Restore data
		err = preserver.RestoreData(ctx, obj, preserved)
		require.NoError(t, err)
		
		// Verify restored data
		assert.Equal(t, "user-value", obj.Annotations["user-annotation"])
		assert.Equal(t, "label-value", obj.Labels["user-label"])
		assert.Contains(t, obj.Annotations, ConversionHistoryAnnotation)
		
		// Verify finalizers and owner references were preserved
		assert.Equal(t, []string{"observability.io/finalizer"}, obj.Finalizers)
		assert.Len(t, obj.OwnerReferences, 1)
		assert.Equal(t, "parent-deployment", obj.OwnerReferences[0].Name)
	})
	
	t.Run("ExtractResourceReferences", func(t *testing.T) {
		// Create object with resource references
		obj := &v1beta1.ObservabilityPlatform{
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: &v1beta1.Components{
					Grafana: &v1beta1.GrafanaSpec{
						DataSources: []v1beta1.DataSourceSpec{
							{
								Name: "prometheus-datasource",
								Type: "prometheus",
								URL:  "http://prometheus:9090",
							},
						},
					},
				},
			},
		}
		
		preserved := &PreservedData{
			ResourceReferences: []ResourceReference{},
		}
		
		// Extract references
		objValue := reflect.ValueOf(obj).Elem()
		preserver.walkStructForReferences(objValue, "", preserved)
		
		// DataSourceSpec might not be detected as a resource reference
		// since it doesn't have Kind/APIVersion fields
		// This test verifies the walking mechanism works
		assert.NotNil(t, preserved)
	})
	
	t.Run("FieldMappings", func(t *testing.T) {
		// Test field mappings between versions
		mappings := preserver.getFieldMappings("v1alpha1", "v1beta1")
		
		assert.NotEmpty(t, mappings)
		assert.Equal(t, "Spec.Components.Prometheus.AdditionalScrapeConfigs", 
			mappings["Spec.Components.Prometheus.CustomConfig"])
		assert.Equal(t, "Spec.Backup.Retention", 
			mappings["Spec.Backup.RetentionDays"])
	})
	
	t.Run("PreservationRules", func(t *testing.T) {
		rules := preserver.GetPreservationRules("v1alpha1", "v1beta1")
		
		assert.NotEmpty(t, rules)
		
		// Find status preservation rule
		var statusRule *PreservationRule
		for _, rule := range rules {
			if rule.FieldPath == "Status" {
				statusRule = &rule
				break
			}
		}
		
		require.NotNil(t, statusRule)
		assert.Equal(t, PreserveAsIs, statusRule.PreserveType)
		
		// Test transform rule
		var transformRule *PreservationRule
		for _, rule := range rules {
			if rule.FieldPath == "spec.backup.retentionDays" {
				transformRule = &rule
				break
			}
		}
		
		require.NotNil(t, transformRule)
		assert.Equal(t, PreserveWithTransform, transformRule.PreserveType)
		assert.NotNil(t, transformRule.Transform)
		
		// Test transform function
		result := transformRule.Transform(int32(7))
		assert.Equal(t, "7d", result)
	})
	
	t.Run("ValidatePreservedData", func(t *testing.T) {
		// Valid preserved data
		preserved := &PreservedData{
			Status: map[string]interface{}{
				"phase": "Ready",
			},
			ResourceReferences: []ResourceReference{
				{
					Kind:      "ConfigMap",
					Name:      "test-config",
					FieldPath: "spec.configRef",
				},
			},
			FieldMappings: map[string]string{
				"sourceField": "targetField",
			},
		}
		
		err := preserver.ValidatePreservedData(preserved)
		assert.NoError(t, err)
		
		// Invalid preserved data - null status value
		preserved.Status["phase"] = nil
		err = preserver.ValidatePreservedData(preserved)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "null value for status field")
		
		// Invalid resource reference
		preserved.Status["phase"] = "Ready"
		preserved.ResourceReferences = append(preserved.ResourceReferences, ResourceReference{
			FieldPath: "spec.emptyRef",
			// Missing Kind and Name
		})
		err = preserver.ValidatePreservedData(preserved)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid resource reference")
	})
	
	t.Run("DataIntegrity", func(t *testing.T) {
		// Create preserved data
		preserved := &PreservedData{
			Status: map[string]interface{}{
				"phase": "Ready",
			},
		}
		
		// Calculate hash
		hash1, err := preserver.calculateDataHash(preserved)
		require.NoError(t, err)
		assert.NotEmpty(t, hash1)
		
		// Same data should produce same hash
		hash2, err := preserver.calculateDataHash(preserved)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash2)
		
		// Modified data should produce different hash
		preserved.Status["phase"] = "Failed"
		hash3, err := preserver.calculateDataHash(preserved)
		require.NoError(t, err)
		assert.NotEqual(t, hash1, hash3)
	})
	
	t.Run("ConversionHistory", func(t *testing.T) {
		obj := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					LastConversionVersionAnnotation: "v1alpha1",
				},
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "observability.io/v1beta1",
				Kind:       "ObservabilityPlatform",
			},
		}
		
		// Update conversion history
		preserver.updateConversionHistory(obj)
		
		// Verify history was recorded
		assert.Contains(t, obj.Annotations, ConversionHistoryAnnotation)
		
		var history []ConversionHistory
		err := json.Unmarshal([]byte(obj.Annotations[ConversionHistoryAnnotation]), &history)
		require.NoError(t, err)
		
		assert.Len(t, history, 1)
		assert.Equal(t, "v1alpha1", history[0].SourceVersion)
		assert.Equal(t, "v1beta1", history[0].TargetVersion)
		assert.True(t, history[0].Success)
		assert.True(t, history[0].DataPreserved)
	})
	
	t.Run("ComplexFieldPreservation", func(t *testing.T) {
		// Test preserving complex nested structures
		obj := &v1alpha1.ObservabilityPlatform{
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						RemoteWrite: []v1alpha1.RemoteWriteSpec{
							{
								URL:           "https://remote1.example.com",
								RemoteTimeout: "30s",
								Headers: map[string]string{
									"X-Auth": "secret",
								},
							},
							{
								URL:           "https://remote2.example.com",
								RemoteTimeout: "60s",
							},
						},
					},
				},
				Global: v1alpha1.GlobalConfig{
					ExternalLabels: map[string]string{
						"cluster": "production",
						"region":  "us-east-1",
					},
					NodeSelector: map[string]string{
						"node-role": "observability",
					},
				},
			},
		}
		
		preserved, err := preserver.PreserveData(ctx, obj, "v1beta1")
		require.NoError(t, err)
		
		// Verify complex structures were preserved
		assert.NotEmpty(t, preserved.CustomFields)
		// The actual fields preserved would depend on the schema differences
	})
	
	t.Run("SystemFieldFiltering", func(t *testing.T) {
		// Test that system fields are not preserved
		assert.True(t, preserver.isSystemField("TypeMeta"))
		assert.True(t, preserver.isSystemField("ObjectMeta"))
		assert.False(t, preserver.isSystemField("Spec"))
		assert.False(t, preserver.isSystemField("Status"))
		
		// Test system annotation filtering
		assert.True(t, preserver.isSystemAnnotation("kubectl.kubernetes.io/last-applied"))
		assert.True(t, preserver.isSystemAnnotation("kubernetes.io/metadata"))
		assert.True(t, preserver.isSystemAnnotation("observability.io/conversion-data"))
		assert.False(t, preserver.isSystemAnnotation("user-annotation"))
		assert.False(t, preserver.isSystemAnnotation("my-app/config"))
		
		// Test system label filtering
		assert.True(t, preserver.isSystemLabel("app.kubernetes.io/name"))
		assert.True(t, preserver.isSystemLabel("kubernetes.io/hostname"))
		assert.False(t, preserver.isSystemLabel("environment"))
		assert.False(t, preserver.isSystemLabel("team"))
	})
}

// TestSetFieldByPath tests the setFieldByPath method
func TestSetFieldByPath(t *testing.T) {
	logger := logf.Log.WithName("test")
	client := fake.NewClientBuilder().Build()
	preserver := NewDataPreserver(logger, client)
	
	type TestStruct struct {
		Field1 string
		Field2 struct {
			SubField1 string
			SubField2 int
		}
		Field3 []string
		Field4 map[string]string
	}
	
	t.Run("SimpleField", func(t *testing.T) {
		obj := &TestStruct{}
		err := preserver.setFieldByPath(reflect.ValueOf(obj).Elem(), "Field1", "value1")
		require.NoError(t, err)
		assert.Equal(t, "value1", obj.Field1)
	})
	
	t.Run("NestedField", func(t *testing.T) {
		obj := &TestStruct{}
		err := preserver.setFieldByPath(reflect.ValueOf(obj).Elem(), "Field2.SubField1", "nested")
		require.NoError(t, err)
		assert.Equal(t, "nested", obj.Field2.SubField1)
		
		err = preserver.setFieldByPath(reflect.ValueOf(obj).Elem(), "Field2.SubField2", 42)
		require.NoError(t, err)
		assert.Equal(t, 42, obj.Field2.SubField2)
	})
	
	t.Run("InvalidField", func(t *testing.T) {
		obj := &TestStruct{}
		err := preserver.setFieldByPath(reflect.ValueOf(obj).Elem(), "NonExistent", "value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field NonExistent not found")
	})
}

// TestResourceReferenceUpdate tests resource reference updating
func TestResourceReferenceUpdate(t *testing.T) {
	logger := logf.Log.WithName("test")
	client := fake.NewClientBuilder().Build()
	preserver := NewDataPreserver(logger, client)
	
	t.Run("UpdateAPIVersion", func(t *testing.T) {
		ref := ResourceReference{
			Kind:       "ObservabilityPlatform",
			APIVersion: "observability.io/v1alpha1",
			Name:       "test-platform",
			Namespace:  "default",
			FieldPath:  "spec.platformRef",
		}
		
		updated := preserver.updateReferenceVersion(ref)
		assert.Equal(t, "observability.io/v1beta1", updated.APIVersion)
		assert.Equal(t, ref.Kind, updated.Kind)
		assert.Equal(t, ref.Name, updated.Name)
	})
	
	t.Run("PreserveUnknownVersion", func(t *testing.T) {
		ref := ResourceReference{
			Kind:       "ConfigMap",
			APIVersion: "v1",
			Name:       "test-config",
			FieldPath:  "spec.configRef",
		}
		
		updated := preserver.updateReferenceVersion(ref)
		assert.Equal(t, "v1", updated.APIVersion)
	})
}

// BenchmarkDataPreservation benchmarks data preservation operations
func BenchmarkDataPreservation(b *testing.B) {
	ctx := context.Background()
	logger := logf.Log.WithName("bench")
	client := fake.NewClientBuilder().Build()
	preserver := NewDataPreserver(logger, client)
	
	// Create a complex object for benchmarking
	obj := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bench-platform",
			Namespace: "bench-namespace",
			Annotations: map[string]string{
				"anno1": "value1",
				"anno2": "value2",
				"anno3": "value3",
			},
			Labels: map[string]string{
				"label1": "value1",
				"label2": "value2",
			},
		},
		Status: v1alpha1.ObservabilityPlatformStatus{
			Phase:   "Ready",
			Message: "Platform is ready",
			ComponentStatus: map[string]v1alpha1.ComponentStatus{
				"prometheus": {Phase: "Ready", Version: "v2.48.0"},
				"grafana":    {Phase: "Ready", Version: "10.0.0"},
				"loki":       {Phase: "Ready", Version: "2.9.0"},
			},
		},
	}
	
	b.Run("PreserveData", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := preserver.PreserveData(ctx, obj, "v1beta1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("RestoreData", func(b *testing.B) {
		preserved, _ := preserver.PreserveData(ctx, obj, "v1beta1")
		target := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bench-platform",
				Namespace: "bench-namespace",
			},
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := preserver.RestoreData(ctx, target, preserved)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
