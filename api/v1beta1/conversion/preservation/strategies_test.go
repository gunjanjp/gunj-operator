/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package preservation_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// TestObject is a mock runtime.Object for testing
type TestObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec TestSpec `json:"spec,omitempty"`
	Status TestStatus `json:"status,omitempty"`
}

type TestSpec struct {
	SimpleField string `json:"simpleField,omitempty"`
	IntField    int    `json:"intField,omitempty"`
	NestedField *NestedStruct `json:"nestedField,omitempty"`
	ArrayField  []string `json:"arrayField,omitempty"`
	MapField    map[string]string `json:"mapField,omitempty"`
}

type NestedStruct struct {
	Name  string `json:"name,omitempty"`
	Value int    `json:"value,omitempty"`
}

type TestStatus struct {
	Phase     string `json:"phase,omitempty"`
	Message   string `json:"message,omitempty"`
	ReadyTime *metav1.Time `json:"readyTime,omitempty"`
}

// DeepEqual checks if runtime.Object implements DeepCopyObject
func (t *TestObject) DeepCopyObject() runtime.Object {
	return t.DeepCopy()
}

// DeepCopy creates a deep copy of TestObject
func (t *TestObject) DeepCopy() *TestObject {
	if t == nil {
		return nil
	}
	out := new(TestObject)
	t.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out
func (t *TestObject) DeepCopyInto(out *TestObject) {
	*out = *t
	out.TypeMeta = t.TypeMeta
	t.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	t.Spec.DeepCopyInto(&out.Spec)
	t.Status.DeepCopyInto(&out.Status)
}

func (s *TestSpec) DeepCopyInto(out *TestSpec) {
	*out = *s
	if s.NestedField != nil {
		out.NestedField = s.NestedField.DeepCopy()
	}
	if s.ArrayField != nil {
		out.ArrayField = make([]string, len(s.ArrayField))
		copy(out.ArrayField, s.ArrayField)
	}
	if s.MapField != nil {
		out.MapField = make(map[string]string)
		for k, v := range s.MapField {
			out.MapField[k] = v
		}
	}
}

func (n *NestedStruct) DeepCopy() *NestedStruct {
	if n == nil {
		return nil
	}
	out := new(NestedStruct)
	*out = *n
	return out
}

func (s *TestStatus) DeepCopyInto(out *TestStatus) {
	*out = *s
	if s.ReadyTime != nil {
		in, out := &s.ReadyTime, &out.ReadyTime
		*out = (*in).DeepCopy()
	}
}

func TestDeepCopyStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	tests := []struct {
		name          string
		includePrivate bool
		maxDepth      int
		field         preservation.FieldDescriptor
		wantErr       bool
	}{
		{
			name:          "simple string field",
			includePrivate: false,
			maxDepth:      10,
			field: preservation.FieldDescriptor{
				Path:  "spec.simpleField",
				Value: "test-value",
				Type:  reflect.TypeOf(""),
			},
			wantErr: false,
		},
		{
			name:          "nested struct field",
			includePrivate: false,
			maxDepth:      10,
			field: preservation.FieldDescriptor{
				Path: "spec.nestedField",
				Value: &NestedStruct{
					Name:  "nested",
					Value: 42,
				},
				Type: reflect.TypeOf(&NestedStruct{}),
			},
			wantErr: false,
		},
		{
			name:          "array field",
			includePrivate: false,
			maxDepth:      10,
			field: preservation.FieldDescriptor{
				Path:  "spec.arrayField",
				Value: []string{"one", "two", "three"},
				Type:  reflect.TypeOf([]string{}),
			},
			wantErr: false,
		},
		{
			name:          "map field",
			includePrivate: false,
			maxDepth:      10,
			field: preservation.FieldDescriptor{
				Path: "spec.mapField",
				Value: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Type: reflect.TypeOf(map[string]string{}),
			},
			wantErr: false,
		},
		{
			name:          "nil value",
			includePrivate: false,
			maxDepth:      10,
			field: preservation.FieldDescriptor{
				Path:  "spec.nilField",
				Value: nil,
				Type:  reflect.TypeOf((*string)(nil)),
			},
			wantErr: false,
		},
		{
			name:          "max depth exceeded",
			includePrivate: false,
			maxDepth:      1,
			field: preservation.FieldDescriptor{
				Path: "spec.deepNested",
				Value: &NestedStruct{
					Name:  "nested",
					Value: 42,
				},
				Type: reflect.TypeOf(&NestedStruct{}),
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := preservation.NewDeepCopyStrategy(logger, tt.includePrivate, tt.maxDepth)
			
			// Validate strategy
			if err := strategy.Validate(); err != nil {
				t.Fatalf("Strategy validation failed: %v", err)
			}
			
			// Create test object
			obj := &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.io/v1",
					Kind:       "TestObject",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}
			
			// Apply strategy
			preserved, err := strategy.Apply(context.Background(), obj, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}
			
			// Verify preservation
			if preserved == nil {
				t.Fatal("Expected non-nil preserved field")
			}
			
			if preserved.Strategy != preservation.StrategyTypeDeepCopy {
				t.Errorf("Expected strategy type %s, got %s", preservation.StrategyTypeDeepCopy, preserved.Strategy)
			}
			
			// Verify deep copy
			if tt.field.Value != nil && preserved.OriginalValue != nil {
				if !reflect.DeepEqual(tt.field.Value, preserved.OriginalValue) {
					t.Errorf("Deep copy mismatch: expected %v, got %v", tt.field.Value, preserved.OriginalValue)
				}
				
				// Verify it's actually a copy (different pointers for reference types)
				switch v := tt.field.Value.(type) {
				case *NestedStruct:
					if pv, ok := preserved.OriginalValue.(*NestedStruct); ok {
						if v == pv {
							t.Error("Expected deep copy to create new instance")
						}
					}
				case []string:
					if pv, ok := preserved.OriginalValue.([]string); ok {
						if len(v) > 0 && len(pv) > 0 && &v[0] == &pv[0] {
							t.Error("Expected deep copy to create new slice")
						}
					}
				case map[string]string:
					// Maps are reference types, verify they're different
					if pv, ok := preserved.OriginalValue.(map[string]string); ok {
						// Modify original to verify copy
						v["test"] = "modified"
						if pv["test"] == "modified" {
							t.Error("Expected deep copy to create new map")
						}
					}
				}
			}
			
			// Test restoration
			newObj := &TestObject{
				TypeMeta:   obj.TypeMeta,
				ObjectMeta: obj.ObjectMeta,
			}
			
			if err := strategy.Restore(context.Background(), newObj, preserved); err != nil {
				t.Errorf("Restore() error = %v", err)
			}
		})
	}
}

func TestTransformStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	// Test transform functions
	toUpperTransform := func(value interface{}, config map[string]interface{}) (interface{}, error) {
		if str, ok := value.(string); ok {
			return str + "_TRANSFORMED", nil
		}
		return value, nil
	}
	
	fromUpperTransform := func(value interface{}, config map[string]interface{}) (interface{}, error) {
		if str, ok := value.(string); ok {
			return str[:len(str)-12], nil // Remove "_TRANSFORMED"
		}
		return value, nil
	}
	
	validator := func(value interface{}) error {
		if str, ok := value.(string); ok {
			if len(str) == 0 {
				return fmt.Errorf("empty string not allowed")
			}
		}
		return nil
	}
	
	strategy := preservation.NewTransformStrategy(logger, toUpperTransform, fromUpperTransform, validator)
	
	// Validate strategy
	if err := strategy.Validate(); err != nil {
		t.Fatalf("Strategy validation failed: %v", err)
	}
	
	tests := []struct {
		name    string
		field   preservation.FieldDescriptor
		wantErr bool
	}{
		{
			name: "valid string transform",
			field: preservation.FieldDescriptor{
				Path:  "spec.field",
				Value: "test",
				Type:  reflect.TypeOf(""),
			},
			wantErr: false,
		},
		{
			name: "empty string validation failure",
			field: preservation.FieldDescriptor{
				Path:  "spec.field",
				Value: "",
				Type:  reflect.TypeOf(""),
			},
			wantErr: true,
		},
		{
			name: "non-string value",
			field: preservation.FieldDescriptor{
				Path:  "spec.field",
				Value: 123,
				Type:  reflect.TypeOf(123),
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &TestObject{}
			
			// Apply strategy
			preserved, err := strategy.Apply(context.Background(), obj, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}
			
			// Verify transformation
			if str, ok := tt.field.Value.(string); ok {
				expected := str + "_TRANSFORMED"
				if preserved.OriginalValue != expected {
					t.Errorf("Expected transformed value %s, got %v", expected, preserved.OriginalValue)
				}
			}
			
			// Test restoration
			if err := strategy.Restore(context.Background(), obj, preserved); err != nil {
				t.Errorf("Restore() error = %v", err)
			}
		})
	}
}

func TestMergeStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	// Test merge function
	mergeFunc := func(source, target interface{}) (interface{}, error) {
		// Simple map merge
		if srcMap, ok := source.(map[string]string); ok {
			if tgtMap, ok := target.(map[string]string); ok {
				merged := make(map[string]string)
				// Copy target first
				for k, v := range tgtMap {
					merged[k] = v
				}
				// Override with source
				for k, v := range srcMap {
					merged[k] = v
				}
				return merged, nil
			}
		}
		return source, nil
	}
	
	strategy := preservation.NewMergeStrategy(logger, mergeFunc, preservation.ConflictResolutionSourceWins)
	
	// Validate strategy
	if err := strategy.Validate(); err != nil {
		t.Fatalf("Strategy validation failed: %v", err)
	}
	
	field := preservation.FieldDescriptor{
		Path: "spec.labels",
		Value: map[string]string{
			"app":     "test",
			"version": "v1",
		},
		Type: reflect.TypeOf(map[string]string{}),
	}
	
	obj := &TestObject{}
	
	// Apply strategy
	preserved, err := strategy.Apply(context.Background(), obj, field)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	
	if preserved.Strategy != preservation.StrategyTypeMerge {
		t.Errorf("Expected strategy type %s, got %s", preservation.StrategyTypeMerge, preserved.Strategy)
	}
}

func TestAnnotationStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	strategy := preservation.NewAnnotationStrategy(logger, "test.io/preserved")
	
	// Validate strategy
	if err := strategy.Validate(); err != nil {
		t.Fatalf("Strategy validation failed: %v", err)
	}
	
	field := preservation.FieldDescriptor{
		Path: "spec.customConfig",
		Value: map[string]interface{}{
			"key1": "value1",
			"key2": []string{"a", "b", "c"},
		},
		Type: reflect.TypeOf(map[string]interface{}{}),
	}
	
	obj := &TestObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: make(map[string]string),
		},
	}
	
	// Apply strategy
	preserved, err := strategy.Apply(context.Background(), obj, field)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	
	// Verify annotation was created
	expectedKey := "test.io/preserved/spec-customconfig"
	if _, exists := obj.Annotations[expectedKey]; !exists {
		t.Errorf("Expected annotation %s to be created", expectedKey)
	}
	
	// Test restoration
	if err := strategy.Restore(context.Background(), obj, preserved); err != nil {
		t.Errorf("Restore() error = %v", err)
	}
	
	// Verify annotation was removed after restoration
	if _, exists := obj.Annotations[expectedKey]; exists {
		t.Errorf("Expected annotation %s to be removed after restoration", expectedKey)
	}
}

func TestConditionalStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	// Create strategies for conditions
	deepCopyStrategy := preservation.NewDeepCopyStrategy(logger, false, 10)
	annotationStrategy := preservation.NewAnnotationStrategy(logger, "test.io/large")
	
	// Create conditional strategy
	conditionalStrategy := preservation.NewConditionalStrategy(logger, deepCopyStrategy)
	
	// Add condition for large fields
	conditionalStrategy.AddCondition(
		"large-field",
		func(obj runtime.Object, field preservation.FieldDescriptor) bool {
			// Use annotation strategy for large values
			if str, ok := field.Value.(string); ok {
				return len(str) > 100
			}
			return false
		},
		annotationStrategy,
	)
	
	// Validate strategy
	if err := conditionalStrategy.Validate(); err != nil {
		t.Fatalf("Strategy validation failed: %v", err)
	}
	
	tests := []struct {
		name           string
		field          preservation.FieldDescriptor
		expectedStrategy preservation.StrategyType
	}{
		{
			name: "small field uses default strategy",
			field: preservation.FieldDescriptor{
				Path:  "spec.small",
				Value: "small value",
				Type:  reflect.TypeOf(""),
			},
			expectedStrategy: preservation.StrategyTypeDeepCopy,
		},
		{
			name: "large field uses annotation strategy",
			field: preservation.FieldDescriptor{
				Path:  "spec.large",
				Value: strings.Repeat("x", 200),
				Type:  reflect.TypeOf(""),
			},
			expectedStrategy: preservation.StrategyTypeAnnotation,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: make(map[string]string),
				},
			}
			
			// Apply strategy
			preserved, err := conditionalStrategy.Apply(context.Background(), obj, tt.field)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			
			if preserved.Strategy != tt.expectedStrategy {
				t.Errorf("Expected strategy %s, got %s", tt.expectedStrategy, preserved.Strategy)
			}
		})
	}
}

func TestCompositeStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	// Create component strategies
	strategy1 := preservation.NewDeepCopyStrategy(logger, false, 10)
	strategy2 := preservation.NewAnnotationStrategy(logger, "test.io/composite")
	
	tests := []struct {
		name string
		mode preservation.CompositeMode
	}{
		{
			name: "sequential mode",
			mode: preservation.CompositeModeSequential,
		},
		{
			name: "parallel mode",
			mode: preservation.CompositeModeParallel,
		},
		{
			name: "pipeline mode",
			mode: preservation.CompositeModePipeline,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compositeStrategy := preservation.NewCompositeStrategy(logger, tt.mode, strategy1, strategy2)
			
			// Validate strategy
			if err := compositeStrategy.Validate(); err != nil {
				t.Fatalf("Strategy validation failed: %v", err)
			}
			
			field := preservation.FieldDescriptor{
				Path:  "spec.composite",
				Value: "test value",
				Type:  reflect.TypeOf(""),
			}
			
			obj := &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: make(map[string]string),
				},
			}
			
			// Apply strategy
			preserved, err := compositeStrategy.Apply(context.Background(), obj, field)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			
			if preserved.Strategy != preservation.StrategyTypeComposite {
				t.Errorf("Expected strategy type %s, got %s", preservation.StrategyTypeComposite, preserved.Strategy)
			}
			
			// Test restoration
			if err := compositeStrategy.Restore(context.Background(), obj, preserved); err != nil {
				t.Errorf("Restore() error = %v", err)
			}
		})
	}
}

func TestVersionedStrategy(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	// Create version-specific strategies
	v1alpha1Strategy := preservation.NewDeepCopyStrategy(logger, false, 10)
	v1beta1Strategy := preservation.NewTransformStrategy(
		logger,
		func(v interface{}, _ map[string]interface{}) (interface{}, error) {
			return fmt.Sprintf("v1beta1:%v", v), nil
		},
		func(v interface{}, _ map[string]interface{}) (interface{}, error) {
			if str, ok := v.(string); ok {
				return strings.TrimPrefix(str, "v1beta1:"), nil
			}
			return v, nil
		},
		nil,
	)
	defaultStrategy := preservation.NewAnnotationStrategy(logger, "test.io/default")
	
	versionedStrategy := preservation.NewVersionedStrategy(logger, defaultStrategy)
	
	// Add version strategies
	if err := versionedStrategy.AddVersionStrategy("v1alpha1", v1alpha1Strategy); err != nil {
		t.Fatalf("Failed to add v1alpha1 strategy: %v", err)
	}
	
	if err := versionedStrategy.AddVersionStrategy("v1beta.*", v1beta1Strategy); err != nil {
		t.Fatalf("Failed to add v1beta1 strategy: %v", err)
	}
	
	// Validate strategy
	if err := versionedStrategy.Validate(); err != nil {
		t.Fatalf("Strategy validation failed: %v", err)
	}
	
	tests := []struct {
		name             string
		version          string
		expectedStrategy preservation.StrategyType
	}{
		{
			name:             "v1alpha1 version",
			version:          "v1alpha1",
			expectedStrategy: preservation.StrategyTypeDeepCopy,
		},
		{
			name:             "v1beta1 version",
			version:          "v1beta1",
			expectedStrategy: preservation.StrategyTypeTransform,
		},
		{
			name:             "v1beta2 version",
			version:          "v1beta2",
			expectedStrategy: preservation.StrategyTypeTransform,
		},
		{
			name:             "unknown version uses default",
			version:          "v2",
			expectedStrategy: preservation.StrategyTypeAnnotation,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.io/" + tt.version,
					Kind:       "TestObject",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test",
					Annotations: make(map[string]string),
				},
			}
			
			field := preservation.FieldDescriptor{
				Path:  "spec.field",
				Value: "test value",
				Type:  reflect.TypeOf(""),
			}
			
			// Apply strategy
			preserved, err := versionedStrategy.Apply(context.Background(), obj, field)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			
			// For versioned strategy, check the actual strategy used
			if tt.expectedStrategy == preservation.StrategyTypeTransform {
				// Check if transform was applied
				if str, ok := preserved.OriginalValue.(string); ok {
					if !strings.HasPrefix(str, "v1beta1:") {
						t.Errorf("Expected transformed value to have v1beta1: prefix, got %s", str)
					}
				}
			}
		})
	}
}

func TestStrategyRegistry(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	registry := preservation.NewStrategyRegistry(logger)
	
	// Register strategies
	strategies := map[string]preservation.PreservationStrategy{
		"deep-copy":  preservation.NewDeepCopyStrategy(logger, false, 10),
		"annotation": preservation.NewAnnotationStrategy(logger, "test.io/preserved"),
		"transform": preservation.NewTransformStrategy(
			logger,
			func(v interface{}, _ map[string]interface{}) (interface{}, error) { return v, nil },
			func(v interface{}, _ map[string]interface{}) (interface{}, error) { return v, nil },
			nil,
		),
	}
	
	for name, strategy := range strategies {
		if err := registry.Register(name, strategy); err != nil {
			t.Errorf("Failed to register strategy %s: %v", name, err)
		}
	}
	
	// Test duplicate registration
	if err := registry.Register("deep-copy", strategies["deep-copy"]); err == nil {
		t.Error("Expected error when registering duplicate strategy")
	}
	
	// Test retrieval
	for name := range strategies {
		retrieved, err := registry.Get(name)
		if err != nil {
			t.Errorf("Failed to get strategy %s: %v", name, err)
		}
		if retrieved == nil {
			t.Errorf("Retrieved nil strategy for %s", name)
		}
	}
	
	// Test non-existent strategy
	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent strategy")
	}
	
	// Test list
	names := registry.List()
	if len(names) != len(strategies) {
		t.Errorf("Expected %d strategies, got %d", len(strategies), len(names))
	}
}
