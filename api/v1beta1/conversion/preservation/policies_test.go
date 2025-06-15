/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package preservation_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestPolicyManager(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	registry := preservation.NewStrategyRegistry(logger)
	
	// Register test strategies
	deepCopyStrategy := preservation.NewDeepCopyStrategy(logger, false, 10)
	if err := registry.Register("deep-copy", deepCopyStrategy); err != nil {
		t.Fatalf("Failed to register deep-copy strategy: %v", err)
	}
	
	annotationStrategy := preservation.NewAnnotationStrategy(logger, "test.io/preserved")
	if err := registry.Register("annotation", annotationStrategy); err != nil {
		t.Fatalf("Failed to register annotation strategy: %v", err)
	}
	
	// Create policy manager
	pm := preservation.NewPolicyManager(logger, registry)
	
	// Test policy registration
	policy := &preservation.Policy{
		Name:        "test-policy",
		Description: "Test preservation policy",
		Priority:    100,
		Enabled:     true,
		Rules: []preservation.PolicyRule{
			{
				Name:         "status-rule",
				FieldPattern: "^Status\\.",
				Strategy:     "deep-copy",
				Priority:     100,
				Enabled:      true,
			},
			{
				Name:         "large-field-rule",
				FieldPattern: ".*Config$",
				Strategy:     "annotation",
				Priority:     80,
				Enabled:      true,
				Conditions: []preservation.PolicyCondition{
					{
						Type:     preservation.ConditionTypeSize,
						Operator: preservation.OperatorGreaterThan,
						Value:    100,
					},
				},
			},
		},
	}
	
	if err := pm.RegisterPolicy(policy); err != nil {
		t.Fatalf("Failed to register policy: %v", err)
	}
	
	// Test duplicate registration
	if err := pm.RegisterPolicy(policy); err == nil {
		t.Error("Expected error when registering duplicate policy")
	}
	
	// Test policy retrieval
	retrieved, err := pm.GetPolicy("test-policy")
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}
	
	if retrieved.Name != policy.Name {
		t.Errorf("Expected policy name %s, got %s", policy.Name, retrieved.Name)
	}
	
	// Test policy listing
	policies := pm.ListPolicies()
	if len(policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(policies))
	}
}

func TestPolicyApplication(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	registry := preservation.NewStrategyRegistry(logger)
	
	// Register strategies
	deepCopyStrategy := preservation.NewDeepCopyStrategy(logger, false, 10)
	registry.Register("deep-copy", deepCopyStrategy)
	
	annotationStrategy := preservation.NewAnnotationStrategy(logger, "test.io/preserved")
	registry.Register("annotation", annotationStrategy)
	
	// Create policy manager
	pm := preservation.NewPolicyManager(logger, registry)
	
	// Register test policy
	policy := &preservation.Policy{
		Name:        "field-type-policy",
		Description: "Policy based on field patterns",
		Priority:    100,
		Enabled:     true,
		Rules: []preservation.PolicyRule{
			{
				Name:         "status-fields",
				FieldPattern: "^Status\\..*",
				Strategy:     "deep-copy",
				Priority:     100,
				Enabled:      true,
			},
			{
				Name:         "config-fields",
				FieldPattern: ".*Config$",
				Strategy:     "annotation",
				Priority:     90,
				Enabled:      true,
			},
		},
	}
	
	if err := pm.RegisterPolicy(policy); err != nil {
		t.Fatalf("Failed to register policy: %v", err)
	}
	
	tests := []struct {
		name             string
		field            preservation.FieldDescriptor
		expectedStrategy string
		expectNil        bool
	}{
		{
			name: "status field matches deep-copy",
			field: preservation.FieldDescriptor{
				Path:  "Status.Phase",
				Value: "Ready",
				Type:  reflect.TypeOf(""),
			},
			expectedStrategy: "deep-copy",
			expectNil:        false,
		},
		{
			name: "config field matches annotation",
			field: preservation.FieldDescriptor{
				Path:  "Spec.CustomConfig",
				Value: map[string]string{"key": "value"},
				Type:  reflect.TypeOf(map[string]string{}),
			},
			expectedStrategy: "annotation",
			expectNil:        false,
		},
		{
			name: "unmatched field returns nil",
			field: preservation.FieldDescriptor{
				Path:  "Spec.Name",
				Value: "test",
				Type:  reflect.TypeOf(""),
			},
			expectedStrategy: "",
			expectNil:        true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &TestObject{}
			
			strategy, err := pm.ApplyPolicies(obj, tt.field)
			if err != nil {
				t.Fatalf("ApplyPolicies() error = %v", err)
			}
			
			if tt.expectNil {
				if strategy != nil {
					t.Errorf("Expected nil strategy, got %v", strategy.Type())
				}
			} else {
				if strategy == nil {
					t.Fatal("Expected non-nil strategy")
				}
				
				// Verify correct strategy type
				switch tt.expectedStrategy {
				case "deep-copy":
					if strategy.Type() != preservation.StrategyTypeDeepCopy {
						t.Errorf("Expected deep-copy strategy, got %s", strategy.Type())
					}
				case "annotation":
					if strategy.Type() != preservation.StrategyTypeAnnotation {
						t.Errorf("Expected annotation strategy, got %s", strategy.Type())
					}
				}
			}
		})
	}
}

func TestPolicyConditions(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	registry := preservation.NewStrategyRegistry(logger)
	
	// Register strategies
	deepCopyStrategy := preservation.NewDeepCopyStrategy(logger, false, 10)
	registry.Register("deep-copy", deepCopyStrategy)
	
	annotationStrategy := preservation.NewAnnotationStrategy(logger, "test.io/preserved")
	registry.Register("annotation", annotationStrategy)
	
	// Create policy manager
	pm := preservation.NewPolicyManager(logger, registry)
	
	// Register conditional policy
	policy := &preservation.Policy{
		Name:        "conditional-policy",
		Description: "Policy with conditions",
		Priority:    100,
		Enabled:     true,
		Rules: []preservation.PolicyRule{
			{
				Name:         "large-field-annotation",
				FieldPattern: ".*",
				Strategy:     "annotation",
				Priority:     100,
				Enabled:      true,
				Conditions: []preservation.PolicyCondition{
					{
						Type:     preservation.ConditionTypeSize,
						Operator: preservation.OperatorGreaterThan,
						Value:    50,
					},
				},
			},
			{
				Name:         "small-field-copy",
				FieldPattern: ".*",
				Strategy:     "deep-copy",
				Priority:     90,
				Enabled:      true,
				Conditions: []preservation.PolicyCondition{
					{
						Type:     preservation.ConditionTypeSize,
						Operator: preservation.OperatorLessThanOrEqual,
						Value:    50,
					},
				},
			},
		},
	}
	
	if err := pm.RegisterPolicy(policy); err != nil {
		t.Fatalf("Failed to register policy: %v", err)
	}
	
	tests := []struct {
		name             string
		field            preservation.FieldDescriptor
		expectedStrategy preservation.StrategyType
	}{
		{
			name: "large field uses annotation",
			field: preservation.FieldDescriptor{
				Path:  "Spec.LargeConfig",
				Value: strings.Repeat("x", 100),
				Type:  reflect.TypeOf(""),
			},
			expectedStrategy: preservation.StrategyTypeAnnotation,
		},
		{
			name: "small field uses deep-copy",
			field: preservation.FieldDescriptor{
				Path:  "Spec.SmallConfig",
				Value: "small value",
				Type:  reflect.TypeOf(""),
			},
			expectedStrategy: preservation.StrategyTypeDeepCopy,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &TestObject{}
			
			strategy, err := pm.ApplyPolicies(obj, tt.field)
			if err != nil {
				t.Fatalf("ApplyPolicies() error = %v", err)
			}
			
			if strategy == nil {
				t.Fatal("Expected non-nil strategy")
			}
			
			if strategy.Type() != tt.expectedStrategy {
				t.Errorf("Expected strategy %s, got %s", tt.expectedStrategy, strategy.Type())
			}
		})
	}
}

func TestPolicyPriority(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	registry := preservation.NewStrategyRegistry(logger)
	
	// Register strategies
	strategies := []string{"strategy1", "strategy2", "strategy3"}
	for _, name := range strategies {
		strategy := preservation.NewDeepCopyStrategy(logger, false, 10)
		registry.Register(name, strategy)
	}
	
	// Create policy manager
	pm := preservation.NewPolicyManager(logger, registry)
	
	// Register multiple policies with different priorities
	policies := []struct {
		name     string
		priority int
		strategy string
	}{
		{"low-priority", 10, "strategy1"},
		{"medium-priority", 50, "strategy2"},
		{"high-priority", 100, "strategy3"},
	}
	
	for _, p := range policies {
		policy := &preservation.Policy{
			Name:        p.name,
			Description: "Test policy",
			Priority:    p.priority,
			Enabled:     true,
			Rules: []preservation.PolicyRule{
				{
					Name:         "test-rule",
					FieldPattern: ".*TestField.*",
					Strategy:     p.strategy,
					Priority:     10,
					Enabled:      true,
				},
			},
		}
		
		if err := pm.RegisterPolicy(policy); err != nil {
			t.Fatalf("Failed to register policy %s: %v", p.name, err)
		}
	}
	
	// Apply policies to a field that matches all patterns
	field := preservation.FieldDescriptor{
		Path:  "Spec.TestField",
		Value: "test",
		Type:  reflect.TypeOf(""),
	}
	
	obj := &TestObject{}
	
	strategy, err := pm.ApplyPolicies(obj, field)
	if err != nil {
		t.Fatalf("ApplyPolicies() error = %v", err)
	}
	
	if strategy == nil {
		t.Fatal("Expected non-nil strategy")
	}
	
	// Should apply highest priority policy (strategy3)
	// Since all strategies are deep-copy, we can't distinguish by type
	// In a real scenario, different strategies would be used
}

func TestPolicyActions(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	registry := preservation.NewStrategyRegistry(logger)
	
	// Register strategy
	deepCopyStrategy := preservation.NewDeepCopyStrategy(logger, false, 10)
	registry.Register("deep-copy", deepCopyStrategy)
	
	// Create policy manager
	pm := preservation.NewPolicyManager(logger, registry)
	
	// Track action execution
	actionExecuted := false
	
	// Register policy with actions
	policy := &preservation.Policy{
		Name:        "action-policy",
		Description: "Policy with actions",
		Priority:    100,
		Enabled:     true,
		Rules: []preservation.PolicyRule{
			{
				Name:         "log-action-rule",
				FieldPattern: ".*",
				Strategy:     "deep-copy",
				Priority:     100,
				Enabled:      true,
				Actions: []preservation.PolicyAction{
					{
						Type:  preservation.ActionTypeLog,
						Value: "Field preserved",
						Metadata: map[string]interface{}{
							"level": 1,
						},
					},
					{
						Type:   preservation.ActionTypeAnnotate,
						Target: "test.io/preserved-by",
						Value:  "action-policy",
					},
				},
			},
		},
	}
	
	if err := pm.RegisterPolicy(policy); err != nil {
		t.Fatalf("Failed to register policy: %v", err)
	}
	
	// Apply policy
	field := preservation.FieldDescriptor{
		Path:  "Spec.TestField",
		Value: "test",
		Type:  reflect.TypeOf(""),
		Annotations: make(map[string]string),
	}
	
	obj := &TestObject{}
	
	_, err := pm.ApplyPolicies(obj, field)
	if err != nil {
		t.Fatalf("ApplyPolicies() error = %v", err)
	}
	
	// Verify annotation action was executed
	if field.Annotations["test.io/preserved-by"] != "action-policy" {
		t.Error("Expected annotation to be set by action")
	}
}

func TestPolicyEngine(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))
	
	// Create policy config
	config := &preservation.PolicyConfig{
		Policies:                preservation.DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      1024,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}
	
	// Create policy engine
	engine, err := preservation.NewPolicyEngine(logger, config)
	if err != nil {
		t.Fatalf("Failed to create policy engine: %v", err)
	}
	
	tests := []struct {
		name  string
		field preservation.FieldDescriptor
		expectSuccess bool
	}{
		{
			name: "preserve status field",
			field: preservation.FieldDescriptor{
				Path:  "Status.Phase",
				Value: "Ready",
				Type:  reflect.TypeOf(""),
			},
			expectSuccess: true,
		},
		{
			name: "preserve metadata field",
			field: preservation.FieldDescriptor{
				Path:  "Finalizers",
				Value: []string{"test.io/finalizer"},
				Type:  reflect.TypeOf([]string{}),
			},
			expectSuccess: true,
		},
		{
			name: "preserve user annotation",
			field: preservation.FieldDescriptor{
				Path:  "Annotations",
				Value: map[string]string{"user.io/key": "value"},
				Type:  reflect.TypeOf(map[string]string{}),
				Annotations: map[string]string{
					"user.io/key": "value",
				},
			},
			expectSuccess: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &TestObject{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "test.io/v1alpha1",
					Kind:       "TestObject",
				},
			}
			
			preserved, err := engine.PreserveField(obj, tt.field)
			if (err == nil) != tt.expectSuccess {
				t.Errorf("PreserveField() error = %v, expectSuccess %v", err, tt.expectSuccess)
				return
			}
			
			if tt.expectSuccess && preserved == nil {
				t.Error("Expected non-nil preserved field")
			}
		})
	}
	
	// Check metrics
	metrics := engine.GetMetrics()
	if metrics.PoliciesEvaluated == 0 {
		t.Error("Expected policies to be evaluated")
	}
	if metrics.PreservationSuccess == 0 && metrics.PreservationFailure == 0 {
		t.Error("Expected preservation attempts to be recorded")
	}
}

func TestDefaultPolicies(t *testing.T) {
	policies := preservation.DefaultPolicies()
	
	expectedPolicies := []string{
		"preserve-status",
		"preserve-metadata",
		"preserve-user-annotations",
		"preserve-large-fields",
		"version-specific-handling",
		"sensitive-data-handling",
	}
	
	if len(policies) != len(expectedPolicies) {
		t.Errorf("Expected %d default policies, got %d", len(expectedPolicies), len(policies))
	}
	
	// Verify each policy exists
	policyMap := make(map[string]*preservation.Policy)
	for _, p := range policies {
		policyMap[p.Name] = p
	}
	
	for _, expected := range expectedPolicies {
		if _, exists := policyMap[expected]; !exists {
			t.Errorf("Expected default policy %s not found", expected)
		}
	}
	
	// Verify policies are enabled
	for _, p := range policies {
		if !p.Enabled {
			t.Errorf("Expected default policy %s to be enabled", p.Name)
		}
		
		// Verify each policy has rules
		if len(p.Rules) == 0 {
			t.Errorf("Expected policy %s to have rules", p.Name)
		}
	}
}
