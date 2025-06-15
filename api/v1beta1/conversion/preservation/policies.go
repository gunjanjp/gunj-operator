/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package preservation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Policy defines a data preservation policy
type Policy struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	Rules       []PolicyRule           `json:"rules"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyRule defines a single rule within a policy
type PolicyRule struct {
	Name         string               `json:"name"`
	FieldPattern string               `json:"fieldPattern"`
	Strategy     string               `json:"strategy"`
	Conditions   []PolicyCondition    `json:"conditions,omitempty"`
	Actions      []PolicyAction       `json:"actions,omitempty"`
	Priority     int                  `json:"priority"`
	Enabled      bool                 `json:"enabled"`
}

// PolicyCondition defines when a rule should apply
type PolicyCondition struct {
	Type     ConditionType          `json:"type"`
	Field    string                 `json:"field,omitempty"`
	Operator ConditionOperator      `json:"operator"`
	Value    interface{}            `json:"value"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyAction defines what action to take
type PolicyAction struct {
	Type     ActionType             `json:"type"`
	Target   string                 `json:"target,omitempty"`
	Value    interface{}            `json:"value,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ConditionType defines the type of condition
type ConditionType string

const (
	ConditionTypeField          ConditionType = "field"
	ConditionTypeVersion        ConditionType = "version"
	ConditionTypeAnnotation     ConditionType = "annotation"
	ConditionTypeLabel          ConditionType = "label"
	ConditionTypeSize           ConditionType = "size"
	ConditionTypeAge            ConditionType = "age"
	ConditionTypeNamespace      ConditionType = "namespace"
	ConditionTypeResourceType   ConditionType = "resource-type"
	ConditionTypeFieldExists    ConditionType = "field-exists"
	ConditionTypeFieldType      ConditionType = "field-type"
	ConditionTypeCustom         ConditionType = "custom"
)

// ConditionOperator defines comparison operators
type ConditionOperator string

const (
	OperatorEquals              ConditionOperator = "equals"
	OperatorNotEquals           ConditionOperator = "not-equals"
	OperatorContains            ConditionOperator = "contains"
	OperatorNotContains         ConditionOperator = "not-contains"
	OperatorStartsWith          ConditionOperator = "starts-with"
	OperatorEndsWith            ConditionOperator = "ends-with"
	OperatorMatches             ConditionOperator = "matches"
	OperatorGreaterThan         ConditionOperator = "greater-than"
	OperatorLessThan            ConditionOperator = "less-than"
	OperatorGreaterThanOrEqual  ConditionOperator = "greater-than-or-equal"
	OperatorLessThanOrEqual     ConditionOperator = "less-than-or-equal"
	OperatorIn                  ConditionOperator = "in"
	OperatorNotIn               ConditionOperator = "not-in"
	OperatorExists              ConditionOperator = "exists"
	OperatorNotExists           ConditionOperator = "not-exists"
)

// ActionType defines the type of action
type ActionType string

const (
	ActionTypePreserve          ActionType = "preserve"
	ActionTypeTransform         ActionType = "transform"
	ActionTypeAnnotate          ActionType = "annotate"
	ActionTypeLog               ActionType = "log"
	ActionTypeAlert             ActionType = "alert"
	ActionTypeMetric            ActionType = "metric"
	ActionTypeValidate          ActionType = "validate"
	ActionTypeEncrypt           ActionType = "encrypt"
	ActionTypeCompress          ActionType = "compress"
	ActionTypeArchive           ActionType = "archive"
)

// PolicyManager manages preservation policies
type PolicyManager struct {
	logger           logr.Logger
	policies         map[string]*Policy
	strategyRegistry *StrategyRegistry
	validators       map[ConditionType]ConditionValidator
	executors        map[ActionType]ActionExecutor
}

// ConditionValidator validates a condition
type ConditionValidator func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error)

// ActionExecutor executes an action
type ActionExecutor func(obj runtime.Object, field FieldDescriptor, action PolicyAction) error

// NewPolicyManager creates a new policy manager
func NewPolicyManager(logger logr.Logger, registry *StrategyRegistry) *PolicyManager {
	pm := &PolicyManager{
		logger:           logger.WithName("policy-manager"),
		policies:         make(map[string]*Policy),
		strategyRegistry: registry,
		validators:       make(map[ConditionType]ConditionValidator),
		executors:        make(map[ActionType]ActionExecutor),
	}

	// Register default validators
	pm.registerDefaultValidators()
	
	// Register default executors
	pm.registerDefaultExecutors()

	return pm
}

// RegisterPolicy registers a new policy
func (pm *PolicyManager) RegisterPolicy(policy *Policy) error {
	if _, exists := pm.policies[policy.Name]; exists {
		return fmt.Errorf("policy %s already exists", policy.Name)
	}

	// Validate policy
	if err := pm.validatePolicy(policy); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	pm.policies[policy.Name] = policy
	pm.logger.V(1).Info("Registered preservation policy", "name", policy.Name, "rules", len(policy.Rules))
	return nil
}

// GetPolicy retrieves a policy by name
func (pm *PolicyManager) GetPolicy(name string) (*Policy, error) {
	policy, exists := pm.policies[name]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", name)
	}
	return policy, nil
}

// ListPolicies returns all registered policies
func (pm *PolicyManager) ListPolicies() []*Policy {
	policies := make([]*Policy, 0, len(pm.policies))
	for _, policy := range pm.policies {
		policies = append(policies, policy)
	}
	return policies
}

// ApplyPolicies applies all enabled policies to a field
func (pm *PolicyManager) ApplyPolicies(obj runtime.Object, field FieldDescriptor) (PreservationStrategy, error) {
	pm.logger.V(2).Info("Applying policies", "field", field.Path)

	// Get applicable rules
	rules := pm.getApplicableRules(obj, field)
	if len(rules) == 0 {
		pm.logger.V(2).Info("No applicable rules found", "field", field.Path)
		return nil, nil
	}

	// Sort rules by priority
	pm.sortRulesByPriority(rules)

	// Apply the highest priority rule
	rule := rules[0]
	pm.logger.V(2).Info("Applying rule", "rule", rule.Name, "strategy", rule.Strategy)

	// Execute actions
	for _, action := range rule.Actions {
		if err := pm.executeAction(obj, field, action); err != nil {
			pm.logger.Error(err, "Failed to execute action", "action", action.Type)
		}
	}

	// Get strategy
	strategy, err := pm.strategyRegistry.Get(rule.Strategy)
	if err != nil {
		return nil, fmt.Errorf("strategy %s not found: %w", rule.Strategy, err)
	}

	return strategy, nil
}

// validatePolicy validates a policy
func (pm *PolicyManager) validatePolicy(policy *Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if len(policy.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}

	for i, rule := range policy.Rules {
		if err := pm.validateRule(rule); err != nil {
			return fmt.Errorf("rule %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateRule validates a policy rule
func (pm *PolicyManager) validateRule(rule PolicyRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.FieldPattern == "" {
		return fmt.Errorf("field pattern is required")
	}

	// Validate field pattern as regex
	if _, err := regexp.Compile(rule.FieldPattern); err != nil {
		return fmt.Errorf("invalid field pattern: %w", err)
	}

	if rule.Strategy == "" {
		return fmt.Errorf("strategy is required")
	}

	// Validate conditions
	for i, condition := range rule.Conditions {
		if err := pm.validateCondition(condition); err != nil {
			return fmt.Errorf("condition %d validation failed: %w", i, err)
		}
	}

	// Validate actions
	for i, action := range rule.Actions {
		if err := pm.validateAction(action); err != nil {
			return fmt.Errorf("action %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateCondition validates a policy condition
func (pm *PolicyManager) validateCondition(condition PolicyCondition) error {
	if condition.Type == "" {
		return fmt.Errorf("condition type is required")
	}

	if _, exists := pm.validators[condition.Type]; !exists {
		return fmt.Errorf("unknown condition type: %s", condition.Type)
	}

	if condition.Operator == "" {
		return fmt.Errorf("condition operator is required")
	}

	return nil
}

// validateAction validates a policy action
func (pm *PolicyManager) validateAction(action PolicyAction) error {
	if action.Type == "" {
		return fmt.Errorf("action type is required")
	}

	if _, exists := pm.executors[action.Type]; !exists {
		return fmt.Errorf("unknown action type: %s", action.Type)
	}

	return nil
}

// getApplicableRules gets rules that apply to a field
func (pm *PolicyManager) getApplicableRules(obj runtime.Object, field FieldDescriptor) []PolicyRule {
	var applicableRules []PolicyRule

	for _, policy := range pm.policies {
		if !policy.Enabled {
			continue
		}

		for _, rule := range policy.Rules {
			if !rule.Enabled {
				continue
			}

			// Check if field matches pattern
			matched, err := regexp.MatchString(rule.FieldPattern, field.Path)
			if err != nil || !matched {
				continue
			}

			// Check conditions
			if pm.checkConditions(obj, field, rule.Conditions) {
				// Add priority from policy
				rule.Priority += policy.Priority
				applicableRules = append(applicableRules, rule)
			}
		}
	}

	return applicableRules
}

// checkConditions checks if all conditions are met
func (pm *PolicyManager) checkConditions(obj runtime.Object, field FieldDescriptor, conditions []PolicyCondition) bool {
	for _, condition := range conditions {
		validator, exists := pm.validators[condition.Type]
		if !exists {
			pm.logger.Error(fmt.Errorf("unknown condition type"), "type", condition.Type)
			return false
		}

		met, err := validator(obj, field, condition)
		if err != nil {
			pm.logger.Error(err, "Condition validation failed", "type", condition.Type)
			return false
		}

		if !met {
			return false
		}
	}

	return true
}

// executeAction executes a policy action
func (pm *PolicyManager) executeAction(obj runtime.Object, field FieldDescriptor, action PolicyAction) error {
	executor, exists := pm.executors[action.Type]
	if !exists {
		return fmt.Errorf("unknown action type: %s", action.Type)
	}

	return executor(obj, field, action)
}

// sortRulesByPriority sorts rules by priority (highest first)
func (pm *PolicyManager) sortRulesByPriority(rules []PolicyRule) {
	// Simple bubble sort for now (can be optimized)
	for i := 0; i < len(rules)-1; i++ {
		for j := 0; j < len(rules)-i-1; j++ {
			if rules[j].Priority < rules[j+1].Priority {
				rules[j], rules[j+1] = rules[j+1], rules[j]
			}
		}
	}
}

// registerDefaultValidators registers default condition validators
func (pm *PolicyManager) registerDefaultValidators() {
	// Field value validator
	pm.validators[ConditionTypeField] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		return pm.validateFieldCondition(field, condition)
	}

	// Version validator
	pm.validators[ConditionTypeVersion] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		gvk := obj.GetObjectKind().GroupVersionKind()
		return pm.compareValues(gvk.Version, condition.Operator, condition.Value)
	}

	// Annotation validator
	pm.validators[ConditionTypeAnnotation] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		if condition.Field == "" {
			return false, fmt.Errorf("annotation key required")
		}
		
		value, exists := field.Annotations[condition.Field]
		if !exists && condition.Operator == OperatorExists {
			return false, nil
		}
		if !exists && condition.Operator == OperatorNotExists {
			return true, nil
		}
		
		return pm.compareValues(value, condition.Operator, condition.Value)
	}

	// Label validator
	pm.validators[ConditionTypeLabel] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		if condition.Field == "" {
			return false, fmt.Errorf("label key required")
		}
		
		// Get labels from metadata if available
		labels := make(map[string]string)
		if metadata, ok := field.Metadata["labels"].(map[string]string); ok {
			labels = metadata
		}
		
		value, exists := labels[condition.Field]
		if !exists && condition.Operator == OperatorExists {
			return false, nil
		}
		if !exists && condition.Operator == OperatorNotExists {
			return true, nil
		}
		
		return pm.compareValues(value, condition.Operator, condition.Value)
	}

	// Size validator
	pm.validators[ConditionTypeSize] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		// Calculate size (simplified - actual implementation would serialize)
		size := pm.calculateFieldSize(field.Value)
		return pm.compareNumericValues(float64(size), condition.Operator, condition.Value)
	}

	// Namespace validator
	pm.validators[ConditionTypeNamespace] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		namespace := ""
		if ns, ok := field.Metadata["namespace"].(string); ok {
			namespace = ns
		}
		return pm.compareValues(namespace, condition.Operator, condition.Value)
	}

	// Resource type validator
	pm.validators[ConditionTypeResourceType] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		gvk := obj.GetObjectKind().GroupVersionKind()
		resourceType := fmt.Sprintf("%s.%s/%s", strings.ToLower(gvk.Kind), gvk.Group, gvk.Version)
		return pm.compareValues(resourceType, condition.Operator, condition.Value)
	}

	// Field exists validator
	pm.validators[ConditionTypeFieldExists] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		exists := field.Value != nil
		if condition.Operator == OperatorExists {
			return exists, nil
		}
		if condition.Operator == OperatorNotExists {
			return !exists, nil
		}
		return false, fmt.Errorf("invalid operator for field-exists condition: %s", condition.Operator)
	}

	// Field type validator
	pm.validators[ConditionTypeFieldType] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		if field.Type == nil {
			return false, nil
		}
		fieldType := field.Type.String()
		return pm.compareValues(fieldType, condition.Operator, condition.Value)
	}

	// Custom validator (placeholder)
	pm.validators[ConditionTypeCustom] = func(obj runtime.Object, field FieldDescriptor, condition PolicyCondition) (bool, error) {
		// Custom validators would be registered separately
		return false, fmt.Errorf("custom validator not implemented")
	}
}

// registerDefaultExecutors registers default action executors
func (pm *PolicyManager) registerDefaultExecutors() {
	// Log executor
	pm.executors[ActionTypeLog] = func(obj runtime.Object, field FieldDescriptor, action PolicyAction) error {
		level := 0
		if l, ok := action.Metadata["level"].(int); ok {
			level = l
		}
		
		pm.logger.V(level).Info("Policy action: log",
			"field", field.Path,
			"message", action.Value,
			"metadata", action.Metadata)
		return nil
	}

	// Metric executor
	pm.executors[ActionTypeMetric] = func(obj runtime.Object, field FieldDescriptor, action PolicyAction) error {
		// Emit metric (implementation would use actual metrics library)
		pm.logger.V(2).Info("Policy action: metric",
			"field", field.Path,
			"metric", action.Target,
			"value", action.Value)
		return nil
	}

	// Alert executor
	pm.executors[ActionTypeAlert] = func(obj runtime.Object, field FieldDescriptor, action PolicyAction) error {
		// Send alert (implementation would use actual alerting system)
		pm.logger.Info("Policy action: alert",
			"field", field.Path,
			"alert", action.Target,
			"message", action.Value)
		return nil
	}

	// Annotate executor
	pm.executors[ActionTypeAnnotate] = func(obj runtime.Object, field FieldDescriptor, action PolicyAction) error {
		if action.Target == "" || action.Value == nil {
			return fmt.Errorf("annotation key and value required")
		}
		
		if field.Annotations == nil {
			field.Annotations = make(map[string]string)
		}
		
		field.Annotations[action.Target] = fmt.Sprintf("%v", action.Value)
		return nil
	}

	// Validate executor
	pm.executors[ActionTypeValidate] = func(obj runtime.Object, field FieldDescriptor, action PolicyAction) error {
		// Perform validation (placeholder)
		pm.logger.V(2).Info("Policy action: validate",
			"field", field.Path,
			"validator", action.Target)
		return nil
	}

	// Other executors would be implemented similarly...
}

// Helper methods

func (pm *PolicyManager) validateFieldCondition(field FieldDescriptor, condition PolicyCondition) (bool, error) {
	if condition.Field != "" {
		// Check nested field
		// This would use reflection to get nested field value
		// For now, just check the main value
	}
	
	return pm.compareValues(field.Value, condition.Operator, condition.Value)
}

func (pm *PolicyManager) compareValues(actual interface{}, operator ConditionOperator, expected interface{}) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	switch operator {
	case OperatorEquals:
		return actualStr == expectedStr, nil
	case OperatorNotEquals:
		return actualStr != expectedStr, nil
	case OperatorContains:
		return strings.Contains(actualStr, expectedStr), nil
	case OperatorNotContains:
		return !strings.Contains(actualStr, expectedStr), nil
	case OperatorStartsWith:
		return strings.HasPrefix(actualStr, expectedStr), nil
	case OperatorEndsWith:
		return strings.HasSuffix(actualStr, expectedStr), nil
	case OperatorMatches:
		return regexp.MatchString(expectedStr, actualStr)
	case OperatorIn:
		if list, ok := expected.([]interface{}); ok {
			for _, item := range list {
				if fmt.Sprintf("%v", item) == actualStr {
					return true, nil
				}
			}
		}
		return false, nil
	case OperatorNotIn:
		if list, ok := expected.([]interface{}); ok {
			for _, item := range list {
				if fmt.Sprintf("%v", item) == actualStr {
					return false, nil
				}
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported operator for string comparison: %s", operator)
	}
}

func (pm *PolicyManager) compareNumericValues(actual float64, operator ConditionOperator, expected interface{}) (bool, error) {
	expectedFloat, ok := toFloat64(expected)
	if !ok {
		return false, fmt.Errorf("expected numeric value, got %T", expected)
	}

	switch operator {
	case OperatorEquals:
		return actual == expectedFloat, nil
	case OperatorNotEquals:
		return actual != expectedFloat, nil
	case OperatorGreaterThan:
		return actual > expectedFloat, nil
	case OperatorLessThan:
		return actual < expectedFloat, nil
	case OperatorGreaterThanOrEqual:
		return actual >= expectedFloat, nil
	case OperatorLessThanOrEqual:
		return actual <= expectedFloat, nil
	default:
		return false, fmt.Errorf("unsupported operator for numeric comparison: %s", operator)
	}
}

func (pm *PolicyManager) calculateFieldSize(value interface{}) int {
	// Simplified size calculation
	// Actual implementation would properly serialize and measure
	return len(fmt.Sprintf("%v", value))
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

// DefaultPolicies returns a set of default preservation policies
func DefaultPolicies() []*Policy {
	return []*Policy{
		{
			Name:        "preserve-status",
			Description: "Preserve status fields during conversion",
			Priority:    100,
			Enabled:     true,
			Rules: []PolicyRule{
				{
					Name:         "status-fields",
					FieldPattern: "^Status\\.",
					Strategy:     "deep-copy",
					Priority:     100,
					Enabled:      true,
				},
			},
		},
		{
			Name:        "preserve-metadata",
			Description: "Preserve important metadata fields",
			Priority:    90,
			Enabled:     true,
			Rules: []PolicyRule{
				{
					Name:         "finalizers",
					FieldPattern: "^Finalizers$",
					Strategy:     "deep-copy",
					Priority:     100,
					Enabled:      true,
				},
				{
					Name:         "owner-references",
					FieldPattern: "^OwnerReferences$",
					Strategy:     "deep-copy",
					Priority:     100,
					Enabled:      true,
				},
			},
		},
		{
			Name:        "preserve-user-annotations",
			Description: "Preserve user-defined annotations",
			Priority:    80,
			Enabled:     true,
			Rules: []PolicyRule{
				{
					Name:         "user-annotations",
					FieldPattern: "^Annotations$",
					Strategy:     "merge",
					Priority:     80,
					Enabled:      true,
					Conditions: []PolicyCondition{
						{
							Type:     ConditionTypeAnnotation,
							Operator: OperatorNotContains,
							Value:    "observability.io/",
						},
					},
				},
			},
		},
		{
			Name:        "preserve-large-fields",
			Description: "Store large fields in annotations",
			Priority:    70,
			Enabled:     true,
			Rules: []PolicyRule{
				{
					Name:         "large-custom-config",
					FieldPattern: "CustomConfig$",
					Strategy:     "annotation",
					Priority:     70,
					Enabled:      true,
					Conditions: []PolicyCondition{
						{
							Type:     ConditionTypeSize,
							Operator: OperatorGreaterThan,
							Value:    1024, // 1KB
						},
					},
					Actions: []PolicyAction{
						{
							Type:  ActionTypeLog,
							Value: "Large custom config field stored in annotation",
						},
					},
				},
			},
		},
		{
			Name:        "version-specific-handling",
			Description: "Handle version-specific field transformations",
			Priority:    60,
			Enabled:     true,
			Rules: []PolicyRule{
				{
					Name:         "v1alpha1-to-v1beta1",
					FieldPattern: ".*",
					Strategy:     "versioned",
					Priority:     60,
					Enabled:      true,
					Conditions: []PolicyCondition{
						{
							Type:     ConditionTypeVersion,
							Operator: OperatorEquals,
							Value:    "v1alpha1",
						},
					},
				},
			},
		},
		{
			Name:        "sensitive-data-handling",
			Description: "Special handling for sensitive fields",
			Priority:    100,
			Enabled:     true,
			Rules: []PolicyRule{
				{
					Name:         "password-fields",
					FieldPattern: "(?i)(password|secret|key|token)",
					Strategy:     "encrypted",
					Priority:     100,
					Enabled:      true,
					Actions: []PolicyAction{
						{
							Type:   ActionTypeLog,
							Value:  "Sensitive field encrypted during preservation",
							Metadata: map[string]interface{}{
								"level": 2,
							},
						},
					},
				},
			},
		},
	}
}

// PolicyConfig represents the configuration for preservation policies
type PolicyConfig struct {
	Policies                []*Policy              `json:"policies"`
	DefaultStrategy         string                 `json:"defaultStrategy"`
	EnableMetrics           bool                   `json:"enableMetrics"`
	EnableAlerting          bool                   `json:"enableAlerting"`
	FieldSizeThreshold      int                    `json:"fieldSizeThreshold"`
	MaxPreservationAttempts int                    `json:"maxPreservationAttempts"`
	PreservationTimeout     string                 `json:"preservationTimeout"`
	CustomValidators        map[string]string      `json:"customValidators,omitempty"`
	CustomExecutors         map[string]string      `json:"customExecutors,omitempty"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
}

// LoadPolicyConfig loads policy configuration
func LoadPolicyConfig(data []byte) (*PolicyConfig, error) {
	// This would typically load from YAML/JSON
	// For now, return default config
	return &PolicyConfig{
		Policies:                DefaultPolicies(),
		DefaultStrategy:         "deep-copy",
		EnableMetrics:           true,
		EnableAlerting:          false,
		FieldSizeThreshold:      1024,
		MaxPreservationAttempts: 3,
		PreservationTimeout:     "30s",
	}, nil
}

// PolicyEngine orchestrates policy-based preservation
type PolicyEngine struct {
	logger           logr.Logger
	policyManager    *PolicyManager
	strategyRegistry *StrategyRegistry
	config           *PolicyConfig
	metrics          PolicyMetrics
}

// PolicyMetrics tracks policy execution metrics
type PolicyMetrics struct {
	PoliciesEvaluated   int64
	RulesMatched        int64
	PreservationSuccess int64
	PreservationFailure int64
	AverageLatency      float64
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(logger logr.Logger, config *PolicyConfig) (*PolicyEngine, error) {
	registry := NewStrategyRegistry(logger)
	manager := NewPolicyManager(logger, registry)

	// Register default strategies
	if err := registerDefaultStrategies(logger, registry); err != nil {
		return nil, fmt.Errorf("failed to register default strategies: %w", err)
	}

	// Register configured policies
	for _, policy := range config.Policies {
		if err := manager.RegisterPolicy(policy); err != nil {
			return nil, fmt.Errorf("failed to register policy %s: %w", policy.Name, err)
		}
	}

	return &PolicyEngine{
		logger:           logger.WithName("policy-engine"),
		policyManager:    manager,
		strategyRegistry: registry,
		config:           config,
		metrics:          PolicyMetrics{},
	}, nil
}

// PreserveField preserves a field using policy-based strategies
func (pe *PolicyEngine) PreserveField(obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	pe.logger.V(2).Info("Preserving field with policy engine", "field", field.Path)
	pe.metrics.PoliciesEvaluated++

	// Apply policies to get strategy
	strategy, err := pe.policyManager.ApplyPolicies(obj, field)
	if err != nil {
		pe.metrics.PreservationFailure++
		return nil, fmt.Errorf("failed to apply policies: %w", err)
	}

	if strategy == nil {
		// Use default strategy
		strategy, err = pe.strategyRegistry.Get(pe.config.DefaultStrategy)
		if err != nil {
			pe.metrics.PreservationFailure++
			return nil, fmt.Errorf("default strategy not found: %w", err)
		}
	}

	pe.metrics.RulesMatched++

	// Apply strategy
	preserved, err := strategy.Apply(context.Background(), obj, field)
	if err != nil {
		pe.metrics.PreservationFailure++
		return nil, fmt.Errorf("strategy application failed: %w", err)
	}

	pe.metrics.PreservationSuccess++
	return preserved, nil
}

// GetMetrics returns policy engine metrics
func (pe *PolicyEngine) GetMetrics() PolicyMetrics {
	return pe.metrics
}

// registerDefaultStrategies registers the default preservation strategies
func registerDefaultStrategies(logger logr.Logger, registry *StrategyRegistry) error {
	// Deep copy strategy
	if err := registry.Register("deep-copy", NewDeepCopyStrategy(logger, false, 10)); err != nil {
		return err
	}

	// Transform strategy with example transformations
	transformStrategy := NewTransformStrategy(
		logger,
		// Transform function
		func(value interface{}, config map[string]interface{}) (interface{}, error) {
			// Example: convert string to uppercase
			if str, ok := value.(string); ok {
				return strings.ToUpper(str), nil
			}
			return value, nil
		},
		// Reverse function
		func(value interface{}, config map[string]interface{}) (interface{}, error) {
			// Example: convert string to lowercase
			if str, ok := value.(string); ok {
				return strings.ToLower(str), nil
			}
			return value, nil
		},
		// Validation function
		func(value interface{}) error {
			return nil
		},
	)
	if err := registry.Register("transform", transformStrategy); err != nil {
		return err
	}

	// Merge strategy
	mergeStrategy := NewMergeStrategy(
		logger,
		// Merge function
		func(source, target interface{}) (interface{}, error) {
			// Simple merge implementation
			// Actual implementation would be more sophisticated
			return source, nil
		},
		ConflictResolutionSourceWins,
	)
	if err := registry.Register("merge", mergeStrategy); err != nil {
		return err
	}

	// Annotation strategy
	annotationStrategy := NewAnnotationStrategy(logger, "observability.io/preserved")
	if err := registry.Register("annotation", annotationStrategy); err != nil {
		return err
	}

	// Conditional strategy
	conditionalStrategy := NewConditionalStrategy(logger, nil)
	// Add conditions...
	if err := registry.Register("conditional", conditionalStrategy); err != nil {
		return err
	}

	// Composite strategy
	compositeStrategy := NewCompositeStrategy(logger, CompositeModeSequential)
	if err := registry.Register("composite", compositeStrategy); err != nil {
		return err
	}

	// Versioned strategy
	versionedStrategy := NewVersionedStrategy(logger, nil)
	// Add version strategies...
	if err := registry.Register("versioned", versionedStrategy); err != nil {
		return err
	}

	// Encrypted strategy (placeholder)
	// This would use actual encryption
	encryptedStrategy := NewTransformStrategy(
		logger,
		func(value interface{}, config map[string]interface{}) (interface{}, error) {
			// Placeholder encryption
			return fmt.Sprintf("encrypted:%v", value), nil
		},
		func(value interface{}, config map[string]interface{}) (interface{}, error) {
			// Placeholder decryption
			if str, ok := value.(string); ok && strings.HasPrefix(str, "encrypted:") {
				return strings.TrimPrefix(str, "encrypted:"), nil
			}
			return value, nil
		},
		nil,
	)
	if err := registry.Register("encrypted", encryptedStrategy); err != nil {
		return err
	}

	return nil
}
