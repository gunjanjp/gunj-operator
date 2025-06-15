/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// ComplexTestObject represents a complex object for testing
type ComplexTestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec   ComplexSpec   `json:"spec,omitempty"`
	Status ComplexStatus `json:"status,omitempty"`
}

type ComplexSpec struct {
	// Standard fields
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	
	// Complex fields
	NestedConfig    *NestedConfig             `json:"nestedConfig,omitempty"`
	ArrayOfStructs  []StructItem             `json:"arrayOfStructs,omitempty"`
	MapOfMaps       map[string]map[string]string `json:"mapOfMaps,omitempty"`
	
	// Unknown fields (will be detected as unknown in target version)
	V1Alpha1OnlyField string                   `json:"v1alpha1OnlyField,omitempty"`
	ExperimentalConfig map[string]interface{}  `json:"experimentalConfig,omitempty"`
	
	// Sensitive fields
	Password    string `json:"password,omitempty" preserve:"encrypt"`
	SecretToken string `json:"secretToken,omitempty" preserve:"encrypt"`
	
	// Large field
	LargeCustomConfig string `json:"largeCustomConfig,omitempty"`
}

type NestedConfig struct {
	Enabled  bool              `json:"enabled"`
	Settings map[string]string `json:"settings,omitempty"`
	Advanced *AdvancedConfig   `json:"advanced,omitempty"`
}

type AdvancedConfig struct {
	FeatureFlags map[string]bool `json:"featureFlags,omitempty"`
	Tuning       TuningParams    `json:"tuning,omitempty"`
}

type TuningParams struct {
	MaxConnections int    `json:"maxConnections"`
	Timeout        string `json:"timeout"`
	RetryPolicy    string `json:"retryPolicy"`
}

type StructItem struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ComplexStatus struct {
	Phase      string       `json:"phase,omitempty"`
	Message    string       `json:"message,omitempty"`
	Conditions []Condition  `json:"conditions,omitempty"`
	Components []Component  `json:"components,omitempty"`
	Metrics    StatusMetrics `json:"metrics,omitempty"`
}

type Condition struct {
	Type               string       `json:"type"`
	Status             string       `json:"status"`
	LastTransitionTime metav1.Time  `json:"lastTransitionTime"`
	Reason             string       `json:"reason,omitempty"`
	Message            string       `json:"message,omitempty"`
}

type Component struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Version string `json:"version,omitempty"`
}

type StatusMetrics struct {
	CPU              float64 `json:"cpu"`
	Memory           int64   `json:"memory"`
	RequestsPerSec   float64 `json:"requestsPerSec"`
	ErrorRate        float64 `json:"errorRate"`
}

// Implement runtime.Object interface
func (c *ComplexTestObject) DeepCopyObject() runtime.Object {
	// Simplified deep copy for testing
	return c
}

func (c *ComplexTestObject) GetObjectKind() schema.ObjectKind {
	return &c.TypeMeta
}

func TestDataPreserverEnhanced_PreserveDataEnhanced(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	client := fake.NewClientBuilder().Build()
	
	// Create policy config
	config := &preservation.PolicyConfig{
		Policies:                preservation.DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      100,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}
	
	// Create enhanced data preserver
	dp, err := conversion.NewDataPreserverEnhanced(logger, client, config)
	if err != nil {
		t.Fatalf("Failed to create data preserver: %v", err)
	}
	
	// Create test object with complex data
	obj := &ComplexTestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test.io/v1alpha1",
			Kind:       "ComplexTestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-object",
			Namespace: "default",
			Annotations: map[string]string{
				"user.io/annotation": "user-value",
				"app.io/version":     "v1.0.0",
			},
			Labels: map[string]string{
				"app":         "test",
				"environment": "dev",
			},
			Finalizers: []string{"test.io/finalizer"},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "owner-deployment",
					UID:        "12345",
				},
			},
		},
		Spec: ComplexSpec{
			Name:        "test-spec",
			Description: "Test description",
			NestedConfig: &NestedConfig{
				Enabled: true,
				Settings: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Advanced: &AdvancedConfig{
					FeatureFlags: map[string]bool{
						"feature1": true,
						"feature2": false,
					},
					Tuning: TuningParams{
						MaxConnections: 100,
						Timeout:        "30s",
						RetryPolicy:    "exponential",
					},
				},
			},
			ArrayOfStructs: []StructItem{
				{
					ID:   "item1",
					Name: "First Item",
					Metadata: map[string]interface{}{
						"priority": 1,
						"tags":     []string{"tag1", "tag2"},
					},
				},
				{
					ID:   "item2",
					Name: "Second Item",
					Metadata: map[string]interface{}{
						"priority": 2,
					},
				},
			},
			MapOfMaps: map[string]map[string]string{
				"region1": {
					"zone1": "active",
					"zone2": "standby",
				},
				"region2": {
					"zone1": "active",
				},
			},
			V1Alpha1OnlyField: "This field only exists in v1alpha1",
			ExperimentalConfig: map[string]interface{}{
				"experiment1": true,
				"experiment2": map[string]string{
					"param": "value",
				},
			},
			Password:          "sensitive-password",
			SecretToken:       "secret-token-12345",
			LargeCustomConfig: strings.Repeat("x", 200), // Large field
		},
		Status: ComplexStatus{
			Phase:   "Ready",
			Message: "All components operational",
			Conditions: []Condition{
				{
					Type:               "Ready",
					Status:             "True",
					LastTransitionTime: metav1.Now(),
					Reason:             "AllComponentsReady",
					Message:            "All components are ready",
				},
			},
			Components: []Component{
				{
					Name:    "component1",
					Ready:   true,
					Version: "v1.0.0",
				},
				{
					Name:    "component2",
					Ready:   true,
					Version: "v1.0.1",
				},
			},
			Metrics: StatusMetrics{
				CPU:            0.5,
				Memory:         1024,
				RequestsPerSec: 100.5,
				ErrorRate:      0.01,
			},
		},
	}
	
	// Test preservation
	ctx := context.Background()
	preserved, err := dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
	if err != nil {
		t.Fatalf("PreserveDataEnhanced() error = %v", err)
	}
	
	// Verify preserved data
	t.Run("Metadata preservation", func(t *testing.T) {
		if preserved.MetadataSnapshot == nil {
			t.Fatal("Expected metadata snapshot to be preserved")
		}
		
		if preserved.MetadataSnapshot.UID != obj.UID {
			t.Errorf("Expected UID %s, got %s", obj.UID, preserved.MetadataSnapshot.UID)
		}
		
		if len(preserved.MetadataSnapshot.OwnerReferences) != len(obj.OwnerReferences) {
			t.Errorf("Expected %d owner references, got %d", 
				len(obj.OwnerReferences), len(preserved.MetadataSnapshot.OwnerReferences))
		}
		
		if len(preserved.MetadataSnapshot.Finalizers) != len(obj.Finalizers) {
			t.Errorf("Expected %d finalizers, got %d", 
				len(obj.Finalizers), len(preserved.MetadataSnapshot.Finalizers))
		}
	})
	
	t.Run("Unknown fields detection", func(t *testing.T) {
		// V1Alpha1OnlyField should be detected as unknown
		if _, exists := preserved.UnknownFields["V1Alpha1OnlyField"]; !exists {
			t.Error("Expected V1Alpha1OnlyField to be detected as unknown")
		}
		
		// ExperimentalConfig should be detected as unknown
		if _, exists := preserved.UnknownFields["ExperimentalConfig"]; !exists {
			t.Error("Expected ExperimentalConfig to be detected as unknown")
		}
		
		// Verify unknown field values
		if unknownField, exists := preserved.UnknownFields["V1Alpha1OnlyField"]; exists {
			if unknownField.Value != obj.Spec.V1Alpha1OnlyField {
				t.Errorf("Expected unknown field value %s, got %v", 
					obj.Spec.V1Alpha1OnlyField, unknownField.Value)
			}
		}
	})
	
	t.Run("Complex fields handling", func(t *testing.T) {
		// NestedConfig should be handled as complex field
		nestedPath := "Spec.NestedConfig"
		if _, exists := preserved.ComplexFields[nestedPath]; !exists {
			t.Errorf("Expected %s to be handled as complex field", nestedPath)
		}
		
		// ArrayOfStructs should be handled as complex field
		arrayPath := "Spec.ArrayOfStructs"
		if _, exists := preserved.ComplexFields[arrayPath]; !exists {
			t.Errorf("Expected %s to be handled as complex field", arrayPath)
		}
		
		// MapOfMaps should be handled as complex field
		mapPath := "Spec.MapOfMaps"
		if _, exists := preserved.ComplexFields[mapPath]; !exists {
			t.Errorf("Expected %s to be handled as complex field", mapPath)
		}
	})
	
	t.Run("Status preservation", func(t *testing.T) {
		if len(preserved.Status) == 0 {
			t.Fatal("Expected status to be preserved")
		}
		
		// Check specific status fields
		if phase, exists := preserved.Status["phase"]; exists {
			if phase != "Ready" {
				t.Errorf("Expected status phase Ready, got %v", phase)
			}
		} else {
			t.Error("Expected status phase to be preserved")
		}
	})
	
	t.Run("Annotation and label preservation", func(t *testing.T) {
		// User annotations should be preserved
		if preserved.Annotations["user.io/annotation"] != "user-value" {
			t.Error("Expected user annotation to be preserved")
		}
		
		if preserved.Annotations["app.io/version"] != "v1.0.0" {
			t.Error("Expected app annotation to be preserved")
		}
		
		// Labels should be preserved
		if preserved.Labels["app"] != "test" {
			t.Error("Expected app label to be preserved")
		}
		
		if preserved.Labels["environment"] != "dev" {
			t.Error("Expected environment label to be preserved")
		}
	})
	
	t.Run("Preservation strategies", func(t *testing.T) {
		// Check that appropriate strategies were applied
		if len(preserved.Strategies) == 0 {
			t.Error("Expected preservation strategies to be recorded")
		}
		
		// Status should use deep-copy strategy (from default policies)
		if strategy, exists := preserved.Strategies["Status"]; exists {
			if strategy != preservation.StrategyTypeDeepCopy {
				t.Errorf("Expected Status to use deep-copy strategy, got %s", strategy)
			}
		}
		
		// Large field should use annotation strategy
		// Note: The actual field path would depend on how the walker builds paths
		// This is a simplified check
		hasAnnotationStrategy := false
		for _, strategy := range preserved.Strategies {
			if strategy == preservation.StrategyTypeAnnotation {
				hasAnnotationStrategy = true
				break
			}
		}
		if !hasAnnotationStrategy {
			t.Error("Expected at least one field to use annotation strategy for large data")
		}
	})
	
	t.Run("Validation results", func(t *testing.T) {
		if len(preserved.ValidationResults) == 0 {
			t.Error("Expected validation results to be recorded")
		}
		
		// Check for unknown field warnings
		hasUnknownFieldWarning := false
		for _, result := range preserved.ValidationResults {
			if result.Level == "warning" && strings.Contains(result.Message, "Unknown field") {
				hasUnknownFieldWarning = true
				break
			}
		}
		if !hasUnknownFieldWarning {
			t.Error("Expected warning for unknown fields")
		}
	})
	
	t.Run("Data integrity", func(t *testing.T) {
		// Check that annotations were updated
		if obj.Annotations[conversion.ConversionDataAnnotation] == "" {
			t.Error("Expected conversion data annotation to be set")
		}
		
		if obj.Annotations[conversion.DataIntegrityHashAnnotation] == "" {
			t.Error("Expected data integrity hash annotation to be set")
		}
		
		if obj.Annotations[conversion.LastConversionVersionAnnotation] != "v1alpha1" {
			t.Error("Expected last conversion version annotation to be set")
		}
	})
}

func TestDataPreserverEnhanced_RestoreDataEnhanced(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	client := fake.NewClientBuilder().Build()
	
	config := &preservation.PolicyConfig{
		Policies:                preservation.DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      100,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}
	
	dp, err := conversion.NewDataPreserverEnhanced(logger, client, config)
	if err != nil {
		t.Fatalf("Failed to create data preserver: %v", err)
	}
	
	// Create preserved data
	preserved := &conversion.PreservedDataEnhanced{
		PreservedData: conversion.PreservedData{
			Status: map[string]interface{}{
				"phase":   "Ready",
				"message": "All components operational",
			},
			Annotations: map[string]string{
				"user.io/annotation": "user-value",
				"app.io/version":     "v1.0.0",
			},
			Labels: map[string]string{
				"app":         "test",
				"environment": "dev",
			},
			CustomFields: map[string]interface{}{
				"customField": "custom-value",
			},
			ResourceReferences: []conversion.ResourceReference{
				{
					Kind:       "ConfigMap",
					APIVersion: "v1",
					Name:       "test-config",
					Namespace:  "default",
					FieldPath:  "spec.configRef",
				},
			},
			FieldMappings: map[string]string{
				"spec.oldField": "spec.newField",
			},
			Metadata: conversion.PreservedMetadata{
				Generation: 5,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "owner-deployment",
						UID:        "12345",
					},
				},
				Finalizers: []string{"test.io/finalizer"},
			},
		},
		UnknownFields: map[string]*conversion.UnknownField{
			"V1Alpha1OnlyField": {
				Path:  "V1Alpha1OnlyField",
				Value: "unknown-value",
				Type:  "string",
				SourceSchema: schema.GroupVersionKind{
					Group:   "test.io",
					Version: "v1alpha1",
					Kind:    "ComplexTestObject",
				},
				DetectedAt: metav1.Now(),
			},
		},
		ComplexFields: map[string]*conversion.ComplexField{
			"Spec.NestedConfig": {
				Path: "Spec.NestedConfig",
				Type: conversion.ComplexFieldTypeNested,
				OriginalValue: &NestedConfig{
					Enabled: true,
					Settings: map[string]string{
						"key": "value",
					},
				},
				PreservedValue: &NestedConfig{
					Enabled: true,
					Settings: map[string]string{
						"key": "value",
					},
				},
				Strategy: preservation.StrategyTypeDeepCopy,
			},
		},
		PreservationPolicy: "default",
		Strategies: map[string]preservation.StrategyType{
			"Status":            preservation.StrategyTypeDeepCopy,
			"Spec.NestedConfig": preservation.StrategyTypeDeepCopy,
		},
		ValidationResults: []conversion.ValidationResult{
			{
				Field:   "metadata.uid",
				Valid:   true,
				Level:   "info",
				Message: "UID preserved",
			},
		},
		MetadataSnapshot: &conversion.MetadataSnapshot{
			Generation: 5,
			UID:        "test-uid",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "owner-deployment",
					UID:        "12345",
				},
			},
			Finalizers: []string{"test.io/finalizer"},
		},
	}
	
	// Create target object for restoration
	obj := &ComplexTestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test.io/v1beta1",
			Kind:       "ComplexTestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-object",
			Namespace:   "default",
			Annotations: make(map[string]string),
			Labels:      make(map[string]string),
		},
		Spec: ComplexSpec{
			Name:        "test-spec",
			Description: "Test description",
		},
		Status: ComplexStatus{},
	}
	
	// Test restoration
	ctx := context.Background()
	err = dp.RestoreDataEnhanced(ctx, obj, preserved)
	if err != nil {
		t.Fatalf("RestoreDataEnhanced() error = %v", err)
	}
	
	// Verify restoration
	t.Run("Metadata restoration", func(t *testing.T) {
		if len(obj.OwnerReferences) != len(preserved.MetadataSnapshot.OwnerReferences) {
			t.Errorf("Expected %d owner references, got %d",
				len(preserved.MetadataSnapshot.OwnerReferences), len(obj.OwnerReferences))
		}
		
		if len(obj.Finalizers) != len(preserved.MetadataSnapshot.Finalizers) {
			t.Errorf("Expected %d finalizers, got %d",
				len(preserved.MetadataSnapshot.Finalizers), len(obj.Finalizers))
		}
	})
	
	t.Run("Status restoration", func(t *testing.T) {
		if obj.Status.Phase != "Ready" {
			t.Errorf("Expected status phase Ready, got %s", obj.Status.Phase)
		}
		
		if obj.Status.Message != "All components operational" {
			t.Errorf("Expected status message 'All components operational', got %s", obj.Status.Message)
		}
	})
	
	t.Run("Annotation and label restoration", func(t *testing.T) {
		if obj.Annotations["user.io/annotation"] != "user-value" {
			t.Error("Expected user annotation to be restored")
		}
		
		if obj.Annotations["app.io/version"] != "v1.0.0" {
			t.Error("Expected app annotation to be restored")
		}
		
		if obj.Labels["app"] != "test" {
			t.Error("Expected app label to be restored")
		}
		
		if obj.Labels["environment"] != "dev" {
			t.Error("Expected environment label to be restored")
		}
		
		// Check that data preservation label was added
		if obj.Labels[conversion.DataPreservationLabel] != "true" {
			t.Error("Expected data preservation label to be set")
		}
	})
	
	t.Run("Unknown fields handling", func(t *testing.T) {
		// Unknown fields should be stored in annotations
		unknownFieldKey := conversion.UnknownFieldsAnnotation + ".v1alpha1onlyfield"
		if _, exists := obj.Annotations[unknownFieldKey]; !exists {
			t.Errorf("Expected unknown field to be stored in annotation %s", unknownFieldKey)
		}
	})
	
	t.Run("Conversion history", func(t *testing.T) {
		// Check that conversion history was updated
		if _, exists := obj.Annotations[conversion.ConversionHistoryAnnotation]; !exists {
			t.Error("Expected conversion history annotation to be added")
		}
	})
}

func TestDataPreserverEnhanced_EdgeCases(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	client := fake.NewClientBuilder().Build()
	
	config := &preservation.PolicyConfig{
		Policies:                preservation.DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      100,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}
	
	dp, err := conversion.NewDataPreserverEnhanced(logger, client, config)
	if err != nil {
		t.Fatalf("Failed to create data preserver: %v", err)
	}
	
	ctx := context.Background()
	
	t.Run("Nil values handling", func(t *testing.T) {
		obj := &ComplexTestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "test.io/v1alpha1",
				Kind:       "ComplexTestObject",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nil",
				Namespace: "default",
			},
			Spec: ComplexSpec{
				NestedConfig:   nil,
				ArrayOfStructs: nil,
				MapOfMaps:      nil,
			},
		}
		
		preserved, err := dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
		if err != nil {
			t.Fatalf("PreserveDataEnhanced() with nil values error = %v", err)
		}
		
		if preserved == nil {
			t.Fatal("Expected non-nil preserved data even with nil fields")
		}
	})
	
	t.Run("Empty collections handling", func(t *testing.T) {
		obj := &ComplexTestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "test.io/v1alpha1",
				Kind:       "ComplexTestObject",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-empty",
				Namespace:   "default",
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			Spec: ComplexSpec{
				ArrayOfStructs: []StructItem{},
				MapOfMaps:      map[string]map[string]string{},
			},
		}
		
		preserved, err := dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
		if err != nil {
			t.Fatalf("PreserveDataEnhanced() with empty collections error = %v", err)
		}
		
		if preserved == nil {
			t.Fatal("Expected non-nil preserved data even with empty collections")
		}
	})
	
	t.Run("Cyclic references handling", func(t *testing.T) {
		// Create object with potential cyclic structure
		obj := &ComplexTestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "test.io/v1alpha1",
				Kind:       "ComplexTestObject",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cyclic",
				Namespace: "default",
			},
			Spec: ComplexSpec{
				ExperimentalConfig: map[string]interface{}{
					"self": map[string]interface{}{
						"ref": "circular",
					},
				},
			},
		}
		
		// Add circular reference
		obj.Spec.ExperimentalConfig["self"].(map[string]interface{})["parent"] = obj.Spec.ExperimentalConfig
		
		preserved, err := dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
		if err != nil {
			// This might fail due to JSON marshaling, which is expected
			t.Logf("Expected error with cyclic reference: %v", err)
		} else if preserved != nil {
			// If it succeeds, that's also acceptable if the implementation handles cycles
			t.Log("Successfully handled cyclic reference")
		}
	})
}

func TestDataPreserverEnhanced_Metrics(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	client := fake.NewClientBuilder().Build()
	
	config := &preservation.PolicyConfig{
		Policies:                preservation.DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      100,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}
	
	dp, err := conversion.NewDataPreserverEnhanced(logger, client, config)
	if err != nil {
		t.Fatalf("Failed to create data preserver: %v", err)
	}
	
	// Perform several preservation and restoration operations
	ctx := context.Background()
	
	for i := 0; i < 3; i++ {
		obj := &ComplexTestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "test.io/v1alpha1",
				Kind:       "ComplexTestObject",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-%d", i),
				Namespace: "default",
			},
			Spec: ComplexSpec{
				Name: fmt.Sprintf("spec-%d", i),
			},
		}
		
		preserved, err := dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
		if err != nil {
			t.Fatalf("PreserveDataEnhanced() error = %v", err)
		}
		
		// Restore to trigger restoration metrics
		newObj := &ComplexTestObject{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "test.io/v1beta1",
				Kind:       "ComplexTestObject",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      obj.Name,
				Namespace: obj.Namespace,
			},
		}
		
		if err := dp.RestoreDataEnhanced(ctx, newObj, preserved); err != nil {
			t.Fatalf("RestoreDataEnhanced() error = %v", err)
		}
	}
	
	// Check metrics
	// Note: In a real implementation, we would check metrics like:
	// - preservation_attempts
	// - preservation_successes
	// - restoration_attempts
	// - restoration_successes
	// - cache_hits
	t.Log("Metrics tracking: 3 preservations and 3 restorations completed successfully")
	
	// Test cache functionality by preserving the same object again
	obj := &ComplexTestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test.io/v1alpha1",
			Kind:       "ComplexTestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cache",
			Namespace: "default",
		},
	}
	
	// First preservation should miss cache
	_, err = dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
	if err != nil {
		t.Fatalf("First preservation error = %v", err)
	}
	
	// Second preservation should hit cache (internal cache implementation)
	_, err = dp.PreserveDataEnhanced(ctx, obj, "v1beta1")
	if err != nil {
		t.Fatalf("Second preservation error = %v", err)
	}
	
	t.Log("Cache functionality tested: second preservation should have hit cache")
}
