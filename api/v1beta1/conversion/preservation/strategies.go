/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package preservation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// StrategyType defines the type of preservation strategy
type StrategyType string

const (
	// StrategyTypeDeepCopy performs a deep copy of the field
	StrategyTypeDeepCopy StrategyType = "deep-copy"

	// StrategyTypeTransform applies a transformation function
	StrategyTypeTransform StrategyType = "transform"

	// StrategyTypeMerge merges with existing data
	StrategyTypeMerge StrategyType = "merge"

	// StrategyTypeAnnotation stores in annotations
	StrategyTypeAnnotation StrategyType = "annotation"

	// StrategyTypeConditional applies based on conditions
	StrategyTypeConditional StrategyType = "conditional"

	// StrategyTypeComposite combines multiple strategies
	StrategyTypeComposite StrategyType = "composite"

	// StrategyTypeVersioned handles version-specific preservation
	StrategyTypeVersioned StrategyType = "versioned"

	// StrategyTypeEncrypted encrypts sensitive data
	StrategyTypeEncrypted StrategyType = "encrypted"
)

// PreservationStrategy defines how to preserve a field or set of fields
type PreservationStrategy interface {
	// Apply applies the preservation strategy
	Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error)

	// Restore restores the preserved field
	Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error

	// Validate validates the strategy configuration
	Validate() error

	// Type returns the strategy type
	Type() StrategyType
}

// FieldDescriptor describes a field to be preserved
type FieldDescriptor struct {
	Path        string                 // Dot-separated path to the field
	Value       interface{}            // Current value
	Type        reflect.Type           // Field type
	Tags        map[string]string      // Field tags (json, yaml, etc.)
	Metadata    map[string]interface{} // Additional metadata
	Annotations map[string]string      // Field-specific annotations
}

// PreservedField represents a preserved field with its metadata
type PreservedField struct {
	Descriptor    FieldDescriptor        `json:"descriptor"`
	OriginalValue interface{}            `json:"originalValue"`
	PreservedAt   metav1.Time            `json:"preservedAt"`
	Strategy      StrategyType           `json:"strategy"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Checksum      string                 `json:"checksum,omitempty"`
}

// DeepCopyStrategy implements deep copy preservation
type DeepCopyStrategy struct {
	logger         logr.Logger
	includePrivate bool
	maxDepth       int
}

// NewDeepCopyStrategy creates a new deep copy strategy
func NewDeepCopyStrategy(logger logr.Logger, includePrivate bool, maxDepth int) *DeepCopyStrategy {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return &DeepCopyStrategy{
		logger:         logger.WithName("deep-copy-strategy"),
		includePrivate: includePrivate,
		maxDepth:       maxDepth,
	}
}

// Apply implements PreservationStrategy
func (s *DeepCopyStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying deep copy strategy", "field", field.Path)

	// Perform deep copy
	copiedValue, err := s.deepCopy(field.Value, 0)
	if err != nil {
		return nil, fmt.Errorf("deep copy failed: %w", err)
	}

	preserved := &PreservedField{
		Descriptor:    field,
		OriginalValue: copiedValue,
		PreservedAt:   metav1.Now(),
		Strategy:      StrategyTypeDeepCopy,
		Metadata: map[string]interface{}{
			"includePrivate": s.includePrivate,
			"maxDepth":       s.maxDepth,
		},
	}

	return preserved, nil
}

// Restore implements PreservationStrategy
func (s *DeepCopyStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	s.logger.V(2).Info("Restoring with deep copy strategy", "field", preserved.Descriptor.Path)

	// Use reflection to set the field value
	objValue := reflect.ValueOf(obj).Elem()
	if err := setFieldByPath(objValue, preserved.Descriptor.Path, preserved.OriginalValue); err != nil {
		return fmt.Errorf("failed to restore field: %w", err)
	}

	return nil
}

// Validate implements PreservationStrategy
func (s *DeepCopyStrategy) Validate() error {
	if s.maxDepth <= 0 {
		return fmt.Errorf("maxDepth must be positive")
	}
	return nil
}

// Type implements PreservationStrategy
func (s *DeepCopyStrategy) Type() StrategyType {
	return StrategyTypeDeepCopy
}

func (s *DeepCopyStrategy) deepCopy(value interface{}, depth int) (interface{}, error) {
	if depth > s.maxDepth {
		return nil, fmt.Errorf("max depth %d exceeded", s.maxDepth)
	}

	if value == nil {
		return nil, nil
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}
		elem, err := s.deepCopy(v.Elem().Interface(), depth+1)
		if err != nil {
			return nil, err
		}
		ptr := reflect.New(v.Elem().Type())
		ptr.Elem().Set(reflect.ValueOf(elem))
		return ptr.Interface(), nil

	case reflect.Struct:
		copy := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !s.includePrivate && !field.IsExported() {
				continue
			}
			fieldValue := v.Field(i)
			if fieldValue.CanInterface() {
				copiedField, err := s.deepCopy(fieldValue.Interface(), depth+1)
				if err != nil {
					return nil, err
				}
				if copy.Field(i).CanSet() {
					copy.Field(i).Set(reflect.ValueOf(copiedField))
				}
			}
		}
		return copy.Interface(), nil

	case reflect.Slice:
		if v.IsNil() {
			return nil, nil
		}
		copy := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			elem, err := s.deepCopy(v.Index(i).Interface(), depth+1)
			if err != nil {
				return nil, err
			}
			copy.Index(i).Set(reflect.ValueOf(elem))
		}
		return copy.Interface(), nil

	case reflect.Map:
		if v.IsNil() {
			return nil, nil
		}
		copy := reflect.MakeMap(v.Type())
		for _, key := range v.MapKeys() {
			copiedKey, err := s.deepCopy(key.Interface(), depth+1)
			if err != nil {
				return nil, err
			}
			copiedValue, err := s.deepCopy(v.MapIndex(key).Interface(), depth+1)
			if err != nil {
				return nil, err
			}
			copy.SetMapIndex(reflect.ValueOf(copiedKey), reflect.ValueOf(copiedValue))
		}
		return copy.Interface(), nil

	default:
		// For basic types, return as is
		return value, nil
	}
}

// TransformStrategy implements transformation-based preservation
type TransformStrategy struct {
	logger          logr.Logger
	transformFunc   TransformFunc
	reverseFunc     TransformFunc
	validationFunc  ValidationFunc
	transformConfig map[string]interface{}
}

// TransformFunc defines a transformation function
type TransformFunc func(interface{}, map[string]interface{}) (interface{}, error)

// ValidationFunc defines a validation function
type ValidationFunc func(interface{}) error

// NewTransformStrategy creates a new transform strategy
func NewTransformStrategy(logger logr.Logger, transform, reverse TransformFunc, validate ValidationFunc) *TransformStrategy {
	return &TransformStrategy{
		logger:          logger.WithName("transform-strategy"),
		transformFunc:   transform,
		reverseFunc:     reverse,
		validationFunc:  validate,
		transformConfig: make(map[string]interface{}),
	}
}

// Apply implements PreservationStrategy
func (s *TransformStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying transform strategy", "field", field.Path)

	// Validate input if validation function is provided
	if s.validationFunc != nil {
		if err := s.validationFunc(field.Value); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Apply transformation
	transformedValue, err := s.transformFunc(field.Value, s.transformConfig)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	preserved := &PreservedField{
		Descriptor:    field,
		OriginalValue: transformedValue,
		PreservedAt:   metav1.Now(),
		Strategy:      StrategyTypeTransform,
		Metadata: map[string]interface{}{
			"transformConfig": s.transformConfig,
		},
	}

	return preserved, nil
}

// Restore implements PreservationStrategy
func (s *TransformStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	s.logger.V(2).Info("Restoring with transform strategy", "field", preserved.Descriptor.Path)

	// Apply reverse transformation
	originalValue, err := s.reverseFunc(preserved.OriginalValue, s.transformConfig)
	if err != nil {
		return fmt.Errorf("reverse transformation failed: %w", err)
	}

	// Validate restored value
	if s.validationFunc != nil {
		if err := s.validationFunc(originalValue); err != nil {
			return fmt.Errorf("restored value validation failed: %w", err)
		}
	}

	// Set the field value
	objValue := reflect.ValueOf(obj).Elem()
	if err := setFieldByPath(objValue, preserved.Descriptor.Path, originalValue); err != nil {
		return fmt.Errorf("failed to restore field: %w", err)
	}

	return nil
}

// Validate implements PreservationStrategy
func (s *TransformStrategy) Validate() error {
	if s.transformFunc == nil {
		return fmt.Errorf("transform function is required")
	}
	if s.reverseFunc == nil {
		return fmt.Errorf("reverse transform function is required")
	}
	return nil
}

// Type implements PreservationStrategy
func (s *TransformStrategy) Type() StrategyType {
	return StrategyTypeTransform
}

// MergeStrategy implements merge-based preservation
type MergeStrategy struct {
	logger        logr.Logger
	mergeFunc     MergeFunc
	conflictRes   ConflictResolution
	preserveOrder bool
}

// MergeFunc defines a merge function
type MergeFunc func(source, target interface{}) (interface{}, error)

// ConflictResolution defines how to resolve merge conflicts
type ConflictResolution string

const (
	ConflictResolutionSourceWins ConflictResolution = "source-wins"
	ConflictResolutionTargetWins ConflictResolution = "target-wins"
	ConflictResolutionMerge      ConflictResolution = "merge"
	ConflictResolutionError      ConflictResolution = "error"
)

// NewMergeStrategy creates a new merge strategy
func NewMergeStrategy(logger logr.Logger, mergeFunc MergeFunc, conflictRes ConflictResolution) *MergeStrategy {
	return &MergeStrategy{
		logger:      logger.WithName("merge-strategy"),
		mergeFunc:   mergeFunc,
		conflictRes: conflictRes,
	}
}

// Apply implements PreservationStrategy
func (s *MergeStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying merge strategy", "field", field.Path)

	preserved := &PreservedField{
		Descriptor:    field,
		OriginalValue: field.Value,
		PreservedAt:   metav1.Now(),
		Strategy:      StrategyTypeMerge,
		Metadata: map[string]interface{}{
			"conflictResolution": string(s.conflictRes),
			"preserveOrder":      s.preserveOrder,
		},
	}

	return preserved, nil
}

// Restore implements PreservationStrategy
func (s *MergeStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	s.logger.V(2).Info("Restoring with merge strategy", "field", preserved.Descriptor.Path)

	// Get current value
	objValue := reflect.ValueOf(obj).Elem()
	currentValue, err := getFieldByPath(objValue, preserved.Descriptor.Path)
	if err != nil {
		return fmt.Errorf("failed to get current value: %w", err)
	}

	// Merge values
	mergedValue, err := s.mergeFunc(preserved.OriginalValue, currentValue)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	// Set merged value
	if err := setFieldByPath(objValue, preserved.Descriptor.Path, mergedValue); err != nil {
		return fmt.Errorf("failed to set merged value: %w", err)
	}

	return nil
}

// Validate implements PreservationStrategy
func (s *MergeStrategy) Validate() error {
	if s.mergeFunc == nil {
		return fmt.Errorf("merge function is required")
	}
	if s.conflictRes == "" {
		return fmt.Errorf("conflict resolution strategy is required")
	}
	return nil
}

// Type implements PreservationStrategy
func (s *MergeStrategy) Type() StrategyType {
	return StrategyTypeMerge
}

// AnnotationStrategy stores data in annotations
type AnnotationStrategy struct {
	logger         logr.Logger
	annotationKey  string
	compressLarge  bool
	sizeThreshold  int
	encryptionFunc func([]byte) ([]byte, error)
	decryptionFunc func([]byte) ([]byte, error)
}

// NewAnnotationStrategy creates a new annotation strategy
func NewAnnotationStrategy(logger logr.Logger, annotationKey string) *AnnotationStrategy {
	return &AnnotationStrategy{
		logger:        logger.WithName("annotation-strategy"),
		annotationKey: annotationKey,
		sizeThreshold: 1024, // 1KB default
	}
}

// Apply implements PreservationStrategy
func (s *AnnotationStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying annotation strategy", "field", field.Path)

	// Serialize field value
	data, err := json.Marshal(field.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize field: %w", err)
	}

	// Compress if needed
	if s.compressLarge && len(data) > s.sizeThreshold {
		data, err = compress(data)
		if err != nil {
			return nil, fmt.Errorf("compression failed: %w", err)
		}
	}

	// Encrypt if encryption function is provided
	if s.encryptionFunc != nil {
		data, err = s.encryptionFunc(data)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %w", err)
		}
	}

	// Store in annotation
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return nil, err
	}

	meta := accessor.GetObjectMeta()
	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	key := s.getAnnotationKey(field.Path)
	annotations[key] = string(data)
	meta.SetAnnotations(annotations)

	preserved := &PreservedField{
		Descriptor:    field,
		OriginalValue: nil, // Stored in annotation
		PreservedAt:   metav1.Now(),
		Strategy:      StrategyTypeAnnotation,
		Metadata: map[string]interface{}{
			"annotationKey": key,
			"compressed":    s.compressLarge && len(data) > s.sizeThreshold,
			"encrypted":     s.encryptionFunc != nil,
		},
	}

	return preserved, nil
}

// Restore implements PreservationStrategy
func (s *AnnotationStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	s.logger.V(2).Info("Restoring from annotation strategy", "field", preserved.Descriptor.Path)

	// Get annotation
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}

	meta := accessor.GetObjectMeta()
	annotations := meta.GetAnnotations()
	if annotations == nil {
		return fmt.Errorf("no annotations found")
	}

	key := s.getAnnotationKey(preserved.Descriptor.Path)
	data, ok := annotations[key]
	if !ok {
		return fmt.Errorf("annotation %s not found", key)
	}

	// Decrypt if needed
	bytes := []byte(data)
	if encrypted, _ := preserved.Metadata["encrypted"].(bool); encrypted && s.decryptionFunc != nil {
		bytes, err = s.decryptionFunc(bytes)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}
	}

	// Decompress if needed
	if compressed, _ := preserved.Metadata["compressed"].(bool); compressed {
		bytes, err = decompress(bytes)
		if err != nil {
			return fmt.Errorf("decompression failed: %w", err)
		}
	}

	// Deserialize
	var value interface{}
	if err := json.Unmarshal(bytes, &value); err != nil {
		return fmt.Errorf("deserialization failed: %w", err)
	}

	// Set field value
	objValue := reflect.ValueOf(obj).Elem()
	if err := setFieldByPath(objValue, preserved.Descriptor.Path, value); err != nil {
		return fmt.Errorf("failed to restore field: %w", err)
	}

	// Remove annotation after restoration
	delete(annotations, key)
	meta.SetAnnotations(annotations)

	return nil
}

// Validate implements PreservationStrategy
func (s *AnnotationStrategy) Validate() error {
	if s.annotationKey == "" {
		return fmt.Errorf("annotation key is required")
	}
	return nil
}

// Type implements PreservationStrategy
func (s *AnnotationStrategy) Type() StrategyType {
	return StrategyTypeAnnotation
}

func (s *AnnotationStrategy) getAnnotationKey(fieldPath string) string {
	// Convert field path to valid annotation key
	key := strings.ReplaceAll(fieldPath, ".", "-")
	key = strings.ToLower(key)
	return fmt.Sprintf("%s/%s", s.annotationKey, key)
}

// ConditionalStrategy applies different strategies based on conditions
type ConditionalStrategy struct {
	logger      logr.Logger
	conditions  []Condition
	fallback    PreservationStrategy
}

// Condition defines when to apply a strategy
type Condition struct {
	Name      string
	Predicate func(runtime.Object, FieldDescriptor) bool
	Strategy  PreservationStrategy
}

// NewConditionalStrategy creates a new conditional strategy
func NewConditionalStrategy(logger logr.Logger, fallback PreservationStrategy) *ConditionalStrategy {
	return &ConditionalStrategy{
		logger:     logger.WithName("conditional-strategy"),
		conditions: []Condition{},
		fallback:   fallback,
	}
}

// AddCondition adds a condition to the strategy
func (s *ConditionalStrategy) AddCondition(name string, predicate func(runtime.Object, FieldDescriptor) bool, strategy PreservationStrategy) {
	s.conditions = append(s.conditions, Condition{
		Name:      name,
		Predicate: predicate,
		Strategy:  strategy,
	})
}

// Apply implements PreservationStrategy
func (s *ConditionalStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying conditional strategy", "field", field.Path)

	// Check conditions
	for _, condition := range s.conditions {
		if condition.Predicate(obj, field) {
			s.logger.V(2).Info("Condition matched", "condition", condition.Name)
			return condition.Strategy.Apply(ctx, obj, field)
		}
	}

	// Use fallback if no condition matches
	if s.fallback != nil {
		s.logger.V(2).Info("Using fallback strategy")
		return s.fallback.Apply(ctx, obj, field)
	}

	return nil, fmt.Errorf("no condition matched and no fallback strategy provided")
}

// Restore implements PreservationStrategy
func (s *ConditionalStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	// Find the strategy that was used
	strategyType := preserved.Strategy

	// Check conditions to find matching strategy
	for _, condition := range s.conditions {
		if condition.Strategy.Type() == strategyType {
			return condition.Strategy.Restore(ctx, obj, preserved)
		}
	}

	// Use fallback
	if s.fallback != nil && s.fallback.Type() == strategyType {
		return s.fallback.Restore(ctx, obj, preserved)
	}

	return fmt.Errorf("no strategy found for type %s", strategyType)
}

// Validate implements PreservationStrategy
func (s *ConditionalStrategy) Validate() error {
	if len(s.conditions) == 0 && s.fallback == nil {
		return fmt.Errorf("at least one condition or fallback strategy is required")
	}

	// Validate all strategies
	for _, condition := range s.conditions {
		if err := condition.Strategy.Validate(); err != nil {
			return fmt.Errorf("condition %s strategy validation failed: %w", condition.Name, err)
		}
	}

	if s.fallback != nil {
		if err := s.fallback.Validate(); err != nil {
			return fmt.Errorf("fallback strategy validation failed: %w", err)
		}
	}

	return nil
}

// Type implements PreservationStrategy
func (s *ConditionalStrategy) Type() StrategyType {
	return StrategyTypeConditional
}

// CompositeStrategy combines multiple strategies
type CompositeStrategy struct {
	logger     logr.Logger
	strategies []PreservationStrategy
	mode       CompositeMode
}

// CompositeMode defines how strategies are combined
type CompositeMode string

const (
	CompositeModeSequential CompositeMode = "sequential" // Apply in order
	CompositeModeParallel   CompositeMode = "parallel"   // Apply all
	CompositeModePipeline   CompositeMode = "pipeline"   // Output of one is input to next
)

// NewCompositeStrategy creates a new composite strategy
func NewCompositeStrategy(logger logr.Logger, mode CompositeMode, strategies ...PreservationStrategy) *CompositeStrategy {
	return &CompositeStrategy{
		logger:     logger.WithName("composite-strategy"),
		strategies: strategies,
		mode:       mode,
	}
}

// Apply implements PreservationStrategy
func (s *CompositeStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying composite strategy", "field", field.Path, "mode", s.mode)

	switch s.mode {
	case CompositeModeSequential:
		return s.applySequential(ctx, obj, field)
	case CompositeModeParallel:
		return s.applyParallel(ctx, obj, field)
	case CompositeModePipeline:
		return s.applyPipeline(ctx, obj, field)
	default:
		return nil, fmt.Errorf("unknown composite mode: %s", s.mode)
	}
}

func (s *CompositeStrategy) applySequential(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	var lastPreserved *PreservedField
	for i, strategy := range s.strategies {
		preserved, err := strategy.Apply(ctx, obj, field)
		if err != nil {
			return nil, fmt.Errorf("strategy %d failed: %w", i, err)
		}
		lastPreserved = preserved
	}
	return lastPreserved, nil
}

func (s *CompositeStrategy) applyParallel(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	// Apply all strategies and combine results
	preservedFields := make([]*PreservedField, len(s.strategies))
	for i, strategy := range s.strategies {
		preserved, err := strategy.Apply(ctx, obj, field)
		if err != nil {
			return nil, fmt.Errorf("strategy %d failed: %w", i, err)
		}
		preservedFields[i] = preserved
	}

	// Combine results
	combined := &PreservedField{
		Descriptor:  field,
		PreservedAt: metav1.Now(),
		Strategy:    StrategyTypeComposite,
		Metadata: map[string]interface{}{
			"mode":             s.mode,
			"preservedFields": preservedFields,
		},
	}

	return combined, nil
}

func (s *CompositeStrategy) applyPipeline(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	currentField := field
	var lastPreserved *PreservedField

	for i, strategy := range s.strategies {
		preserved, err := strategy.Apply(ctx, obj, currentField)
		if err != nil {
			return nil, fmt.Errorf("pipeline strategy %d failed: %w", i, err)
		}

		// Update field for next strategy
		if preserved.OriginalValue != nil {
			currentField.Value = preserved.OriginalValue
		}

		lastPreserved = preserved
	}

	return lastPreserved, nil
}

// Restore implements PreservationStrategy
func (s *CompositeStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	s.logger.V(2).Info("Restoring with composite strategy", "field", preserved.Descriptor.Path)

	switch s.mode {
	case CompositeModeSequential:
		// Restore in reverse order
		for i := len(s.strategies) - 1; i >= 0; i-- {
			if err := s.strategies[i].Restore(ctx, obj, preserved); err != nil {
				return fmt.Errorf("strategy %d restore failed: %w", i, err)
			}
		}
		return nil

	case CompositeModeParallel:
		// Restore all preserved fields
		if preservedFields, ok := preserved.Metadata["preservedFields"].([]*PreservedField); ok {
			for i, pf := range preservedFields {
				if err := s.strategies[i].Restore(ctx, obj, pf); err != nil {
					return fmt.Errorf("parallel strategy %d restore failed: %w", i, err)
				}
			}
		}
		return nil

	case CompositeModePipeline:
		// Restore in reverse order
		for i := len(s.strategies) - 1; i >= 0; i-- {
			if err := s.strategies[i].Restore(ctx, obj, preserved); err != nil {
				return fmt.Errorf("pipeline strategy %d restore failed: %w", i, err)
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown composite mode: %s", s.mode)
	}
}

// Validate implements PreservationStrategy
func (s *CompositeStrategy) Validate() error {
	if len(s.strategies) == 0 {
		return fmt.Errorf("at least one strategy is required")
	}

	for i, strategy := range s.strategies {
		if err := strategy.Validate(); err != nil {
			return fmt.Errorf("strategy %d validation failed: %w", i, err)
		}
	}

	return nil
}

// Type implements PreservationStrategy
func (s *CompositeStrategy) Type() StrategyType {
	return StrategyTypeComposite
}

// VersionedStrategy handles version-specific preservation
type VersionedStrategy struct {
	logger          logr.Logger
	versionPatterns map[string]*regexp.Regexp
	strategies      map[string]PreservationStrategy
	defaultStrategy PreservationStrategy
}

// NewVersionedStrategy creates a new versioned strategy
func NewVersionedStrategy(logger logr.Logger, defaultStrategy PreservationStrategy) *VersionedStrategy {
	return &VersionedStrategy{
		logger:          logger.WithName("versioned-strategy"),
		versionPatterns: make(map[string]*regexp.Regexp),
		strategies:      make(map[string]PreservationStrategy),
		defaultStrategy: defaultStrategy,
	}
}

// AddVersionStrategy adds a strategy for a specific version pattern
func (s *VersionedStrategy) AddVersionStrategy(versionPattern string, strategy PreservationStrategy) error {
	re, err := regexp.Compile(versionPattern)
	if err != nil {
		return fmt.Errorf("invalid version pattern %s: %w", versionPattern, err)
	}

	s.versionPatterns[versionPattern] = re
	s.strategies[versionPattern] = strategy
	return nil
}

// Apply implements PreservationStrategy
func (s *VersionedStrategy) Apply(ctx context.Context, obj runtime.Object, field FieldDescriptor) (*PreservedField, error) {
	s.logger.V(2).Info("Applying versioned strategy", "field", field.Path)

	// Get object version
	gvk := obj.GetObjectKind().GroupVersionKind()
	version := gvk.Version

	// Find matching strategy
	for pattern, re := range s.versionPatterns {
		if re.MatchString(version) {
			s.logger.V(2).Info("Version pattern matched", "pattern", pattern, "version", version)
			return s.strategies[pattern].Apply(ctx, obj, field)
		}
	}

	// Use default strategy
	if s.defaultStrategy != nil {
		s.logger.V(2).Info("Using default strategy for version", "version", version)
		return s.defaultStrategy.Apply(ctx, obj, field)
	}

	return nil, fmt.Errorf("no strategy found for version %s", version)
}

// Restore implements PreservationStrategy
func (s *VersionedStrategy) Restore(ctx context.Context, obj runtime.Object, preserved *PreservedField) error {
	// Find the strategy that was used based on metadata
	if versionPattern, ok := preserved.Metadata["versionPattern"].(string); ok {
		if strategy, exists := s.strategies[versionPattern]; exists {
			return strategy.Restore(ctx, obj, preserved)
		}
	}

	// Try default strategy
	if s.defaultStrategy != nil {
		return s.defaultStrategy.Restore(ctx, obj, preserved)
	}

	return fmt.Errorf("no strategy found for restoration")
}

// Validate implements PreservationStrategy
func (s *VersionedStrategy) Validate() error {
	// Validate all strategies
	for pattern, strategy := range s.strategies {
		if err := strategy.Validate(); err != nil {
			return fmt.Errorf("strategy for pattern %s validation failed: %w", pattern, err)
		}
	}

	if s.defaultStrategy != nil {
		if err := s.defaultStrategy.Validate(); err != nil {
			return fmt.Errorf("default strategy validation failed: %w", err)
		}
	}

	return nil
}

// Type implements PreservationStrategy
func (s *VersionedStrategy) Type() StrategyType {
	return StrategyTypeVersioned
}

// Helper functions

func setFieldByPath(v reflect.Value, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	current := v

	for i, part := range parts {
		if strings.Contains(part, "[") {
			// Handle array/slice indices
			fieldName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]

			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return fmt.Errorf("field %s not found", fieldName)
			}

			var index int
			fmt.Sscanf(indexStr, "%d", &index)

			if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
				if index >= field.Len() {
					// Extend slice if needed
					if field.Kind() == reflect.Slice {
						newSlice := reflect.MakeSlice(field.Type(), index+1, index+1)
						reflect.Copy(newSlice, field)
						field.Set(newSlice)
					} else {
						return fmt.Errorf("index %d out of bounds for array %s", index, fieldName)
					}
				}
				current = field.Index(index)
			} else {
				return fmt.Errorf("field %s is not a slice or array", fieldName)
			}
		} else {
			// Regular field access
			field := current.FieldByName(part)
			if !field.IsValid() {
				// Try to find by JSON tag
				structType := current.Type()
				for j := 0; j < structType.NumField(); j++ {
					f := structType.Field(j)
					jsonTag := f.Tag.Get("json")
					if jsonTag == part || strings.Split(jsonTag, ",")[0] == part {
						field = current.Field(j)
						break
					}
				}
			}

			if !field.IsValid() {
				return fmt.Errorf("field %s not found", part)
			}

			if i == len(parts)-1 {
				// Last part, set the value
				if !field.CanSet() {
					return fmt.Errorf("field %s cannot be set", part)
				}

				targetValue := reflect.ValueOf(value)
				if targetValue.Type().ConvertibleTo(field.Type()) {
					field.Set(targetValue.Convert(field.Type()))
				} else {
					return fmt.Errorf("cannot convert %T to %v", value, field.Type())
				}
				return nil
			}

			current = field
		}
	}

	return nil
}

func getFieldByPath(v reflect.Value, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := v

	for _, part := range parts {
		if strings.Contains(part, "[") {
			// Handle array/slice indices
			fieldName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]

			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return nil, fmt.Errorf("field %s not found", fieldName)
			}

			var index int
			fmt.Sscanf(indexStr, "%d", &index)

			if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
				if index >= field.Len() {
					return nil, fmt.Errorf("index %d out of bounds for field %s", index, fieldName)
				}
				current = field.Index(index)
			} else {
				return nil, fmt.Errorf("field %s is not a slice or array", fieldName)
			}
		} else {
			// Regular field access
			field := current.FieldByName(part)
			if !field.IsValid() {
				// Try to find by JSON tag
				structType := current.Type()
				for i := 0; i < structType.NumField(); i++ {
					f := structType.Field(i)
					jsonTag := f.Tag.Get("json")
					if jsonTag == part || strings.Split(jsonTag, ",")[0] == part {
						field = current.Field(i)
						break
					}
				}
			}

			if !field.IsValid() {
				return nil, fmt.Errorf("field %s not found", part)
			}

			current = field
		}
	}

	if current.CanInterface() {
		return current.Interface(), nil
	}

	return nil, fmt.Errorf("cannot get interface for field")
}

// Placeholder compression functions (implement with actual compression library)
func compress(data []byte) ([]byte, error) {
	// TODO: Implement actual compression (e.g., gzip)
	return data, nil
}

func decompress(data []byte) ([]byte, error) {
	// TODO: Implement actual decompression
	return data, nil
}

// StrategyRegistry manages available preservation strategies
type StrategyRegistry struct {
	strategies map[string]PreservationStrategy
	logger     logr.Logger
}

// NewStrategyRegistry creates a new strategy registry
func NewStrategyRegistry(logger logr.Logger) *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make(map[string]PreservationStrategy),
		logger:     logger.WithName("strategy-registry"),
	}
}

// Register registers a strategy
func (r *StrategyRegistry) Register(name string, strategy PreservationStrategy) error {
	if _, exists := r.strategies[name]; exists {
		return fmt.Errorf("strategy %s already registered", name)
	}

	if err := strategy.Validate(); err != nil {
		return fmt.Errorf("strategy validation failed: %w", err)
	}

	r.strategies[name] = strategy
	r.logger.V(1).Info("Registered preservation strategy", "name", name, "type", strategy.Type())
	return nil
}

// Get retrieves a strategy by name
func (r *StrategyRegistry) Get(name string) (PreservationStrategy, error) {
	strategy, exists := r.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", name)
	}
	return strategy, nil
}

// List returns all registered strategy names
func (r *StrategyRegistry) List() []string {
	names := make([]string, 0, len(r.strategies))
	for name := range r.strategies {
		names = append(names, name)
	}
	return names
}
