/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Annotation keys for data preservation
	ConversionDataAnnotation        = "observability.io/conversion-data"
	LastConversionVersionAnnotation = "observability.io/last-conversion-version"
	PreservedFieldsAnnotation       = "observability.io/preserved-fields"
	DataIntegrityHashAnnotation     = "observability.io/data-integrity-hash"
	ConversionHistoryAnnotation     = "observability.io/conversion-history"
	PreservationPolicyAnnotation    = "observability.io/preservation-policy"
	UnknownFieldsAnnotation         = "observability.io/unknown-fields"
	
	// Label keys for conversion tracking
	ConversionSourceVersionLabel = "observability.io/source-version"
	ConversionTargetVersionLabel = "observability.io/target-version"
	ConversionTimestampLabel     = "observability.io/conversion-timestamp"
	DataPreservationLabel        = "observability.io/data-preserved"
)

// DataPreserverEnhanced handles data preservation during API conversions with advanced features
type DataPreserverEnhanced struct {
	logger              logr.Logger
	client              client.Client
	policyEngine        *preservation.PolicyEngine
	strategyRegistry    *preservation.StrategyRegistry
	unknownFieldHandler UnknownFieldHandler
	cache               *preservationCache
	metrics             *PreservationMetrics
	mu                  sync.RWMutex
}

// NewDataPreserverEnhanced creates a new enhanced data preserver instance
func NewDataPreserverEnhanced(logger logr.Logger, client client.Client, config *preservation.PolicyConfig) (*DataPreserverEnhanced, error) {
	// Create policy engine
	policyEngine, err := preservation.NewPolicyEngine(logger, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy engine: %w", err)
	}

	// Create strategy registry
	strategyRegistry := preservation.NewStrategyRegistry(logger)

	return &DataPreserverEnhanced{
		logger:              logger.WithName("data-preserver-enhanced"),
		client:              client,
		policyEngine:        policyEngine,
		strategyRegistry:    strategyRegistry,
		unknownFieldHandler: NewDefaultUnknownFieldHandler(logger),
		cache:               newPreservationCache(),
		metrics:             NewPreservationMetrics(),
	}, nil
}

// PreservedDataEnhanced represents enhanced preserved data
type PreservedDataEnhanced struct {
	// Core preserved data
	PreservedData
	
	// Enhanced fields
	UnknownFields      map[string]*UnknownField       `json:"unknownFields,omitempty"`
	ComplexFields      map[string]*ComplexField       `json:"complexFields,omitempty"`
	PreservationPolicy string                         `json:"preservationPolicy,omitempty"`
	Strategies         map[string]preservation.StrategyType `json:"strategies,omitempty"`
	ValidationResults  []ValidationResult             `json:"validationResults,omitempty"`
	MetadataSnapshot   *MetadataSnapshot              `json:"metadataSnapshot,omitempty"`
}

// UnknownField represents a field not recognized in the target schema
type UnknownField struct {
	Path         string                 `json:"path"`
	Value        interface{}            `json:"value"`
	Type         string                 `json:"type"`
	SourceSchema schema.GroupVersionKind `json:"sourceSchema"`
	DetectedAt   metav1.Time            `json:"detectedAt"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ComplexField represents a complex field requiring special handling
type ComplexField struct {
	Path             string                       `json:"path"`
	Type             ComplexFieldType             `json:"type"`
	OriginalValue    interface{}                  `json:"originalValue"`
	PreservedValue   interface{}                  `json:"preservedValue"`
	Strategy         preservation.StrategyType    `json:"strategy"`
	TransformApplied bool                         `json:"transformApplied"`
	Metadata         map[string]interface{}       `json:"metadata,omitempty"`
}

// ComplexFieldType defines types of complex fields
type ComplexFieldType string

const (
	ComplexFieldTypeNested     ComplexFieldType = "nested"
	ComplexFieldTypeArray      ComplexFieldType = "array"
	ComplexFieldTypeMap        ComplexFieldType = "map"
	ComplexFieldTypeReference  ComplexFieldType = "reference"
	ComplexFieldTypePolymorphic ComplexFieldType = "polymorphic"
	ComplexFieldTypeCustom     ComplexFieldType = "custom"
)

// ValidationResult represents the result of a validation
type ValidationResult struct {
	Field   string `json:"field"`
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
	Level   string `json:"level,omitempty"` // info, warning, error
}

// MetadataSnapshot captures metadata state at preservation time
type MetadataSnapshot struct {
	CreationTimestamp  metav1.Time             `json:"creationTimestamp"`
	Generation         int64                   `json:"generation"`
	ResourceVersion    string                  `json:"resourceVersion"`
	UID                types.UID               `json:"uid"`
	OwnerReferences    []metav1.OwnerReference `json:"ownerReferences,omitempty"`
	Finalizers         []string                `json:"finalizers,omitempty"`
	DeletionTimestamp  *metav1.Time            `json:"deletionTimestamp,omitempty"`
	DeletionGracePeriod *int64                  `json:"deletionGracePeriodSeconds,omitempty"`
	ClusterName        string                  `json:"clusterName,omitempty"`
	ManagedFields      []metav1.ManagedFieldsEntry `json:"managedFields,omitempty"`
}

// UnknownFieldHandler handles unknown fields during preservation
type UnknownFieldHandler interface {
	HandleUnknownField(ctx context.Context, field reflect.StructField, value reflect.Value, targetSchema schema.GroupVersionKind) (*UnknownField, error)
	RestoreUnknownField(ctx context.Context, obj runtime.Object, unknown *UnknownField) error
}

// DefaultUnknownFieldHandler is the default implementation
type DefaultUnknownFieldHandler struct {
	logger logr.Logger
}

// NewDefaultUnknownFieldHandler creates a new default handler
func NewDefaultUnknownFieldHandler(logger logr.Logger) *DefaultUnknownFieldHandler {
	return &DefaultUnknownFieldHandler{
		logger: logger.WithName("unknown-field-handler"),
	}
}

// HandleUnknownField implements UnknownFieldHandler
func (h *DefaultUnknownFieldHandler) HandleUnknownField(ctx context.Context, field reflect.StructField, value reflect.Value, targetSchema schema.GroupVersionKind) (*UnknownField, error) {
	h.logger.V(2).Info("Handling unknown field", "field", field.Name, "type", field.Type)

	unknown := &UnknownField{
		Path:         field.Name,
		Type:         field.Type.String(),
		SourceSchema: targetSchema,
		DetectedAt:   metav1.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Extract value if possible
	if value.CanInterface() {
		unknown.Value = value.Interface()
	}

	// Extract field tags
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		unknown.Metadata["jsonTag"] = jsonTag
	}
	if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
		unknown.Metadata["yamlTag"] = yamlTag
	}

	return unknown, nil
}

// RestoreUnknownField implements UnknownFieldHandler
func (h *DefaultUnknownFieldHandler) RestoreUnknownField(ctx context.Context, obj runtime.Object, unknown *UnknownField) error {
	h.logger.V(2).Info("Restoring unknown field", "field", unknown.Path)
	
	// Store in annotation for future reference
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}
	
	meta := accessor.GetObjectMeta()
	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	
	// Serialize unknown field
	data, err := json.Marshal(unknown)
	if err != nil {
		return fmt.Errorf("failed to serialize unknown field: %w", err)
	}
	
	key := fmt.Sprintf("%s.%s", UnknownFieldsAnnotation, strings.ToLower(unknown.Path))
	annotations[key] = string(data)
	meta.SetAnnotations(annotations)
	
	return nil
}

// PreserveDataEnhanced performs enhanced data preservation
func (dp *DataPreserverEnhanced) PreserveDataEnhanced(ctx context.Context, obj runtime.Object, targetVersion string) (*PreservedDataEnhanced, error) {
	dp.logger.V(1).Info("Enhanced data preservation starting",
		"objectType", reflect.TypeOf(obj).String(),
		"targetVersion", targetVersion)
	
	startTime := metav1.Now()
	dp.metrics.RecordPreservationStart()
	
	// Check cache first
	cacheKey := dp.generateCacheKey(obj, targetVersion)
	if cached, ok := dp.cache.Get(cacheKey); ok {
		dp.logger.V(2).Info("Using cached preservation data", "key", cacheKey)
		dp.metrics.RecordCacheHit()
		return cached, nil
	}
	
	// Create enhanced preserved data
	enhanced := &PreservedDataEnhanced{
		PreservedData: PreservedData{
			Status:             make(map[string]interface{}),
			Annotations:        make(map[string]string),
			Labels:             make(map[string]string),
			CustomFields:       make(map[string]interface{}),
			ResourceReferences: []ResourceReference{},
			FieldMappings:      make(map[string]string),
		},
		UnknownFields:     make(map[string]*UnknownField),
		ComplexFields:     make(map[string]*ComplexField),
		Strategies:        make(map[string]preservation.StrategyType),
		ValidationResults: []ValidationResult{},
	}
	
	// Extract comprehensive metadata
	if err := dp.extractEnhancedMetadata(obj, enhanced); err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to extract enhanced metadata: %w", err)
	}
	
	// Detect and preserve unknown fields
	if err := dp.detectAndPreserveUnknownFields(ctx, obj, targetVersion, enhanced); err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to detect unknown fields: %w", err)
	}
	
	// Handle complex fields with strategies
	if err := dp.handleComplexFields(ctx, obj, targetVersion, enhanced); err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to handle complex fields: %w", err)
	}
	
	// Apply preservation policies
	if err := dp.applyPreservationPolicies(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to apply preservation policies: %w", err)
	}
	
	// Preserve status with enhanced handling
	if err := dp.preserveStatusEnhanced(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to preserve status: %w", err)
	}
	
	// Preserve annotations and labels with rules
	dp.preserveAnnotationsAndLabelsWithRules(obj, enhanced)
	
	// Validate preserved data
	dp.validatePreservedData(enhanced)
	
	// Calculate data integrity
	hash, err := dp.calculateEnhancedDataHash(enhanced)
	if err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to calculate data hash: %w", err)
	}
	
	// Store preservation data
	if err := dp.storeEnhancedPreservationData(obj, enhanced, hash); err != nil {
		dp.metrics.RecordPreservationError()
		return nil, fmt.Errorf("failed to store preservation data: %w", err)
	}
	
	// Cache the result
	dp.cache.Put(cacheKey, enhanced)
	
	// Record metrics
	duration := metav1.Now().Sub(startTime.Time)
	dp.metrics.RecordPreservationComplete(duration)
	
	dp.logger.Info("Enhanced data preservation complete",
		"preservedFields", len(enhanced.CustomFields),
		"unknownFields", len(enhanced.UnknownFields),
		"complexFields", len(enhanced.ComplexFields),
		"duration", duration)
	
	return enhanced, nil
}

// RestoreDataEnhanced performs enhanced data restoration
func (dp *DataPreserverEnhanced) RestoreDataEnhanced(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	dp.logger.V(1).Info("Enhanced data restoration starting",
		"objectType", reflect.TypeOf(obj).String())
	
	startTime := metav1.Now()
	dp.metrics.RecordRestorationStart()
	
	// Validate data before restoration
	if err := dp.validateDataForRestoration(enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Restore metadata
	if err := dp.restoreEnhancedMetadata(obj, enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("failed to restore metadata: %w", err)
	}
	
	// Restore complex fields
	if err := dp.restoreComplexFields(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("failed to restore complex fields: %w", err)
	}
	
	// Restore unknown fields
	if err := dp.restoreUnknownFields(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("failed to restore unknown fields: %w", err)
	}
	
	// Restore status
	if err := dp.restoreStatusEnhanced(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("failed to restore status: %w", err)
	}
	
	// Restore annotations and labels
	dp.restoreAnnotationsAndLabelsEnhanced(obj, enhanced)
	
	// Update resource references
	if err := dp.updateResourceReferencesEnhanced(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("failed to update resource references: %w", err)
	}
	
	// Apply field mappings
	if err := dp.applyFieldMappingsEnhanced(ctx, obj, enhanced); err != nil {
		dp.metrics.RecordRestorationError()
		return fmt.Errorf("failed to apply field mappings: %w", err)
	}
	
	// Verify integrity
	if err := dp.verifyDataIntegrityEnhanced(obj, enhanced); err != nil {
		dp.logger.Error(err, "Data integrity verification failed")
		// Continue but record the error
		dp.metrics.RecordIntegrityCheckFailure()
	}
	
	// Update conversion history
	dp.updateConversionHistoryEnhanced(obj, enhanced)
	
	// Record metrics
	duration := metav1.Now().Sub(startTime.Time)
	dp.metrics.RecordRestorationComplete(duration)
	
	dp.logger.Info("Enhanced data restoration complete",
		"duration", duration)
	
	return nil
}

// extractEnhancedMetadata extracts comprehensive metadata
func (dp *DataPreserverEnhanced) extractEnhancedMetadata(obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}
	
	meta := accessor.GetObjectMeta()
	
	// Create metadata snapshot
	enhanced.MetadataSnapshot = &MetadataSnapshot{
		CreationTimestamp:   meta.GetCreationTimestamp(),
		Generation:          meta.GetGeneration(),
		ResourceVersion:     meta.GetResourceVersion(),
		UID:                 meta.GetUID(),
		OwnerReferences:     meta.GetOwnerReferences(),
		Finalizers:          meta.GetFinalizers(),
		DeletionTimestamp:   meta.GetDeletionTimestamp(),
		DeletionGracePeriod: meta.GetDeletionGracePeriodSeconds(),
		ClusterName:         meta.GetClusterName(),
		ManagedFields:       meta.GetManagedFields(),
	}
	
	// Extract to core preserved data as well
	enhanced.Metadata = PreservedMetadata{
		CreationTimestamp: meta.GetCreationTimestamp(),
		Generation:        meta.GetGeneration(),
		ResourceVersion:   meta.GetResourceVersion(),
		UID:               meta.GetUID(),
		OwnerReferences:   meta.GetOwnerReferences(),
		Finalizers:        meta.GetFinalizers(),
	}
	
	return nil
}

// detectAndPreserveUnknownFields detects fields not in target schema
func (dp *DataPreserverEnhanced) detectAndPreserveUnknownFields(ctx context.Context, obj runtime.Object, targetVersion string, enhanced *PreservedDataEnhanced) error {
	dp.logger.V(2).Info("Detecting unknown fields", "targetVersion", targetVersion)
	
	objValue := reflect.ValueOf(obj).Elem()
	objType := objValue.Type()
	
	targetSchema := dp.getTargetSchemaWithVersion(targetVersion)
	
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i)
		
		// Skip system fields
		if dp.isSystemField(field.Name) {
			continue
		}
		
		// Check if field exists in target schema
		if !dp.fieldExistsInTargetSchema(field.Name, targetSchema) {
			// This is an unknown field
			unknown, err := dp.unknownFieldHandler.HandleUnknownField(ctx, field, fieldValue, obj.GetObjectKind().GroupVersionKind())
			if err != nil {
				dp.logger.Error(err, "Failed to handle unknown field", "field", field.Name)
				continue
			}
			
			enhanced.UnknownFields[field.Name] = unknown
			dp.logger.V(1).Info("Detected unknown field", "field", field.Name, "type", field.Type)
		}
	}
	
	return nil
}

// handleComplexFields handles complex fields requiring special preservation
func (dp *DataPreserverEnhanced) handleComplexFields(ctx context.Context, obj runtime.Object, targetVersion string, enhanced *PreservedDataEnhanced) error {
	dp.logger.V(2).Info("Handling complex fields")
	
	objValue := reflect.ValueOf(obj).Elem()
	
	// Walk through the object structure
	if err := dp.walkStructForComplexFields(ctx, objValue, "", obj, enhanced); err != nil {
		return fmt.Errorf("failed to walk struct: %w", err)
	}
	
	return nil
}

// walkStructForComplexFields recursively walks struct to find complex fields
func (dp *DataPreserverEnhanced) walkStructForComplexFields(ctx context.Context, v reflect.Value, path string, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	switch v.Kind() {
	case reflect.Struct:
		vType := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := vType.Field(i)
			fieldValue := v.Field(i)
			fieldPath := dp.buildFieldPath(path, field.Name)
			
			// Check if this is a complex field
			if dp.isComplexField(field, fieldValue) {
				complex, err := dp.handleComplexField(ctx, field, fieldValue, fieldPath, obj, enhanced)
				if err != nil {
					dp.logger.Error(err, "Failed to handle complex field", "field", fieldPath)
					continue
				}
				
				enhanced.ComplexFields[fieldPath] = complex
			} else {
				// Continue walking
				if err := dp.walkStructForComplexFields(ctx, fieldValue, fieldPath, obj, enhanced); err != nil {
					return err
				}
			}
		}
		
	case reflect.Slice, reflect.Array:
		// Arrays and slices are complex fields
		complex := &ComplexField{
			Path:          path,
			Type:          ComplexFieldTypeArray,
			OriginalValue: dp.extractValue(v),
		}
		
		// Apply appropriate strategy
		strategy := dp.selectStrategyForComplexField(complex)
		complex.Strategy = strategy.Type()
		
		enhanced.ComplexFields[path] = complex
		
	case reflect.Map:
		// Maps are complex fields
		complex := &ComplexField{
			Path:          path,
			Type:          ComplexFieldTypeMap,
			OriginalValue: dp.extractValue(v),
		}
		
		// Apply appropriate strategy
		strategy := dp.selectStrategyForComplexField(complex)
		complex.Strategy = strategy.Type()
		
		enhanced.ComplexFields[path] = complex
		
	case reflect.Ptr:
		if !v.IsNil() {
			return dp.walkStructForComplexFields(ctx, v.Elem(), path, obj, enhanced)
		}
	}
	
	return nil
}

// isComplexField determines if a field is complex
func (dp *DataPreserverEnhanced) isComplexField(field reflect.StructField, value reflect.Value) bool {
	// Check for nested structs (excluding system types)
	if value.Kind() == reflect.Struct && !dp.isSystemType(field.Type) {
		return true
	}
	
	// Check for polymorphic fields
	if field.Type.Kind() == reflect.Interface {
		return true
	}
	
	// Check for custom types with special handling
	if dp.requiresCustomHandling(field) {
		return true
	}
	
	return false
}

// handleComplexField handles a single complex field
func (dp *DataPreserverEnhanced) handleComplexField(ctx context.Context, field reflect.StructField, value reflect.Value, path string, obj runtime.Object, enhanced *PreservedDataEnhanced) (*ComplexField, error) {
	complex := &ComplexField{
		Path:     path,
		Metadata: make(map[string]interface{}),
	}
	
	// Determine complex field type
	switch {
	case dp.isNestedStruct(field, value):
		complex.Type = ComplexFieldTypeNested
	case dp.isResourceReference(field.Type):
		complex.Type = ComplexFieldTypeReference
	case field.Type.Kind() == reflect.Interface:
		complex.Type = ComplexFieldTypePolymorphic
	default:
		complex.Type = ComplexFieldTypeCustom
	}
	
	// Extract original value
	if value.CanInterface() {
		complex.OriginalValue = value.Interface()
	}
	
	// Apply preservation strategy
	fieldDesc := preservation.FieldDescriptor{
		Path:  path,
		Value: complex.OriginalValue,
		Type:  field.Type,
		Tags:  dp.extractFieldTags(field),
	}
	
	preserved, err := dp.policyEngine.PreserveField(obj, fieldDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to preserve complex field: %w", err)
	}
	
	if preserved != nil {
		complex.PreservedValue = preserved.OriginalValue
		complex.Strategy = preserved.Strategy
		complex.TransformApplied = preserved.Strategy == preservation.StrategyTypeTransform
		enhanced.Strategies[path] = preserved.Strategy
	}
	
	return complex, nil
}

// applyPreservationPolicies applies configured preservation policies
func (dp *DataPreserverEnhanced) applyPreservationPolicies(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	dp.logger.V(2).Info("Applying preservation policies")
	
	// Get applicable policy from annotations or use default
	policyName := dp.getPreservationPolicy(obj)
	enhanced.PreservationPolicy = policyName
	
	// Apply policies to all fields
	objValue := reflect.ValueOf(obj).Elem()
	if err := dp.applyPoliciesRecursively(ctx, objValue, "", obj, enhanced); err != nil {
		return fmt.Errorf("failed to apply policies: %w", err)
	}
	
	return nil
}

// applyPoliciesRecursively applies policies to all fields recursively
func (dp *DataPreserverEnhanced) applyPoliciesRecursively(ctx context.Context, v reflect.Value, path string, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	if !v.IsValid() {
		return nil
	}
	
	switch v.Kind() {
	case reflect.Struct:
		vType := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := vType.Field(i)
			fieldValue := v.Field(i)
			fieldPath := dp.buildFieldPath(path, field.Name)
			
			// Skip if already handled
			if _, exists := enhanced.ComplexFields[fieldPath]; exists {
				continue
			}
			
			// Create field descriptor
			fieldDesc := preservation.FieldDescriptor{
				Path:  fieldPath,
				Value: dp.extractValue(fieldValue),
				Type:  field.Type,
				Tags:  dp.extractFieldTags(field),
			}
			
			// Apply policy
			preserved, err := dp.policyEngine.PreserveField(obj, fieldDesc)
			if err != nil {
				dp.logger.Error(err, "Failed to apply policy", "field", fieldPath)
				continue
			}
			
			if preserved != nil {
				enhanced.Strategies[fieldPath] = preserved.Strategy
			}
			
			// Continue recursively
			if err := dp.applyPoliciesRecursively(ctx, fieldValue, fieldPath, obj, enhanced); err != nil {
				return err
			}
		}
		
	case reflect.Ptr:
		if !v.IsNil() {
			return dp.applyPoliciesRecursively(ctx, v.Elem(), path, obj, enhanced)
		}
	}
	
	return nil
}

// preserveStatusEnhanced preserves status with enhanced handling
func (dp *DataPreserverEnhanced) preserveStatusEnhanced(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	objValue := reflect.ValueOf(obj).Elem()
	statusField := objValue.FieldByName("Status")
	
	if !statusField.IsValid() || statusField.IsZero() {
		return nil
	}
	
	// Apply status-specific preservation strategy
	fieldDesc := preservation.FieldDescriptor{
		Path:  "Status",
		Value: statusField.Interface(),
		Type:  statusField.Type(),
	}
	
	preserved, err := dp.policyEngine.PreserveField(obj, fieldDesc)
	if err != nil {
		return fmt.Errorf("failed to preserve status: %w", err)
	}
	
	if preserved != nil {
		// Convert to map for storage
		statusData, err := dp.structToMap(preserved.OriginalValue)
		if err != nil {
			return fmt.Errorf("failed to convert status to map: %w", err)
		}
		
		enhanced.Status = statusData
		enhanced.Strategies["Status"] = preserved.Strategy
	}
	
	return nil
}

// preserveAnnotationsAndLabelsWithRules preserves with custom rules
func (dp *DataPreserverEnhanced) preserveAnnotationsAndLabelsWithRules(obj runtime.Object, enhanced *PreservedDataEnhanced) {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		dp.logger.Error(err, "Failed to access object metadata")
		return
	}
	
	meta := accessor.GetObjectMeta()
	
	// Preserve annotations with rules
	for key, value := range meta.GetAnnotations() {
		if dp.shouldPreserveAnnotation(key, value) {
			enhanced.Annotations[key] = value
		}
	}
	
	// Preserve labels with rules
	for key, value := range meta.GetLabels() {
		if dp.shouldPreserveLabel(key, value) {
			enhanced.Labels[key] = value
		}
	}
}

// shouldPreserveAnnotation determines if an annotation should be preserved
func (dp *DataPreserverEnhanced) shouldPreserveAnnotation(key, value string) bool {
	// System annotations to exclude
	systemPrefixes := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"deployment.kubernetes.io/revision",
		"kubernetes.io/change-cause",
	}
	
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}
	
	// Preserve our own conversion annotations
	if strings.HasPrefix(key, "observability.io/") {
		return strings.Contains(key, "user-") || strings.Contains(key, "custom-")
	}
	
	// Preserve all user annotations
	return true
}

// shouldPreserveLabel determines if a label should be preserved
func (dp *DataPreserverEnhanced) shouldPreserveLabel(key, value string) bool {
	// System labels to exclude
	systemPrefixes := []string{
		"pod-template-hash",
		"controller-revision-hash",
		"statefulset.kubernetes.io/pod-name",
	}
	
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(key, prefix) {
			return false
		}
	}
	
	// Preserve all user labels
	return true
}

// validatePreservedData validates the preserved data
func (dp *DataPreserverEnhanced) validatePreservedData(enhanced *PreservedDataEnhanced) {
	// Validate unknown fields
	for path, unknown := range enhanced.UnknownFields {
		result := ValidationResult{
			Field: path,
			Valid: unknown.Value != nil,
			Level: "warning",
		}
		
		if !result.Valid {
			result.Message = "Unknown field has nil value"
		} else {
			result.Message = fmt.Sprintf("Unknown field detected: %s", unknown.Type)
		}
		
		enhanced.ValidationResults = append(enhanced.ValidationResults, result)
	}
	
	// Validate complex fields
	for path, complex := range enhanced.ComplexFields {
		result := ValidationResult{
			Field: path,
			Valid: true,
			Level: "info",
		}
		
		if complex.PreservedValue == nil && complex.OriginalValue != nil {
			result.Valid = false
			result.Level = "error"
			result.Message = "Complex field preservation failed"
		} else if complex.TransformApplied {
			result.Message = "Transform applied to complex field"
		}
		
		enhanced.ValidationResults = append(enhanced.ValidationResults, result)
	}
	
	// Validate critical metadata
	if enhanced.MetadataSnapshot != nil {
		if enhanced.MetadataSnapshot.UID == "" {
			enhanced.ValidationResults = append(enhanced.ValidationResults, ValidationResult{
				Field:   "metadata.uid",
				Valid:   false,
				Level:   "error",
				Message: "UID is missing",
			})
		}
	}
}

// Helper methods

func (dp *DataPreserverEnhanced) generateCacheKey(obj runtime.Object, targetVersion string) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	accessor, _ := metav1.ObjectMetaAccessor(obj)
	meta := accessor.GetObjectMeta()
	
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		gvk.Group,
		gvk.Version,
		gvk.Kind,
		meta.GetNamespace(),
		meta.GetName())
}

func (dp *DataPreserverEnhanced) getTargetSchemaWithVersion(version string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "observability.io",
		Version: version,
		Kind:    "ObservabilityPlatform",
	}
}

func (dp *DataPreserverEnhanced) fieldExistsInTargetSchema(fieldName string, targetSchema schema.GroupVersionKind) bool {
	// This would check against actual OpenAPI schema
	// For now, use version-specific field lists
	switch targetSchema.Version {
	case "v1beta1":
		return dp.isV1Beta1Field(fieldName)
	case "v1alpha1":
		return dp.isV1Alpha1Field(fieldName)
	case "v1":
		return dp.isV1Field(fieldName)
	default:
		return false
	}
}

func (dp *DataPreserverEnhanced) isV1Field(fieldName string) bool {
	v1Fields := []string{
		"Spec", "Status", "Components", "Global", "HighAvailability",
		"Backup", "Alerting", "NodeSelector", "Tolerations", "Affinity",
		"SecurityContext", "ServiceAccount", "ImagePullSecrets",
		// Add all v1 fields
	}
	
	for _, field := range v1Fields {
		if field == fieldName {
			return true
		}
	}
	return false
}

func (dp *DataPreserverEnhanced) isSystemType(t reflect.Type) bool {
	systemTypes := []string{
		"Time", "Duration", "IntOrString",
		"Quantity", "ObjectMeta", "TypeMeta",
	}
	
	typeName := t.Name()
	for _, st := range systemTypes {
		if typeName == st {
			return true
		}
	}
	
	return false
}

func (dp *DataPreserverEnhanced) requiresCustomHandling(field reflect.StructField) bool {
	// Check for custom tags
	if custom := field.Tag.Get("preserve"); custom == "custom" {
		return true
	}
	
	// Check for known custom types
	customTypes := []string{
		"CustomConfig", "AdvancedSettings", "ExperimentalFeatures",
	}
	
	for _, ct := range customTypes {
		if strings.Contains(field.Name, ct) {
			return true
		}
	}
	
	return false
}

func (dp *DataPreserverEnhanced) isNestedStruct(field reflect.StructField, value reflect.Value) bool {
	return value.Kind() == reflect.Struct && 
		!dp.isSystemType(field.Type) && 
		field.Type.NumField() > 0
}

func (dp *DataPreserverEnhanced) extractFieldTags(field reflect.StructField) map[string]string {
	tags := make(map[string]string)
	
	tagNames := []string{"json", "yaml", "protobuf", "preserve", "validate"}
	for _, name := range tagNames {
		if tag := field.Tag.Get(name); tag != "" {
			tags[name] = tag
		}
	}
	
	return tags
}

func (dp *DataPreserverEnhanced) extractValue(v reflect.Value) interface{} {
	if !v.IsValid() || !v.CanInterface() {
		return nil
	}
	return v.Interface()
}

func (dp *DataPreserverEnhanced) selectStrategyForComplexField(complex *ComplexField) preservation.PreservationStrategy {
	// Select strategy based on field type
	// This would use the strategy registry
	// For now, return a default
	strategy, _ := dp.strategyRegistry.Get("deep-copy")
	return strategy
}

func (dp *DataPreserverEnhanced) getPreservationPolicy(obj runtime.Object) string {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return "default"
	}
	
	meta := accessor.GetObjectMeta()
	if policy, ok := meta.GetAnnotations()[PreservationPolicyAnnotation]; ok {
		return policy
	}
	
	return "default"
}

func (dp *DataPreserverEnhanced) buildFieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

func (dp *DataPreserverEnhanced) isSystemField(fieldName string) bool {
	systemFields := []string{
		"TypeMeta", "ObjectMeta", "ListMeta",
	}
	
	for _, field := range systemFields {
		if fieldName == field {
			return true
		}
	}
	return false
}

func (dp *DataPreserverEnhanced) isResourceReference(t reflect.Type) bool {
	if t.Kind() == reflect.Struct {
		// Check for reference fields
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Name == "Kind" || field.Name == "APIVersion" || field.Name == "Name" {
				return true
			}
		}
	}
	return false
}

func (dp *DataPreserverEnhanced) structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}

func (dp *DataPreserverEnhanced) calculateEnhancedDataHash(enhanced *PreservedDataEnhanced) (string, error) {
	// Create a deterministic representation
	hashData := struct {
		Status        map[string]interface{}
		CustomFields  map[string]interface{}
		UnknownFields map[string]*UnknownField
		ComplexFields map[string]*ComplexField
	}{
		Status:        enhanced.Status,
		CustomFields:  enhanced.CustomFields,
		UnknownFields: enhanced.UnknownFields,
		ComplexFields: enhanced.ComplexFields,
	}
	
	data, err := json.Marshal(hashData)
	if err != nil {
		return "", err
	}
	
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

func (dp *DataPreserverEnhanced) storeEnhancedPreservationData(obj runtime.Object, enhanced *PreservedDataEnhanced, hash string) error {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}
	
	meta := accessor.GetObjectMeta()
	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	
	// Store core preservation data
	preservedJSON, err := json.Marshal(enhanced)
	if err != nil {
		return fmt.Errorf("failed to marshal enhanced preserved data: %w", err)
	}
	
	annotations[ConversionDataAnnotation] = string(preservedJSON)
	annotations[DataIntegrityHashAnnotation] = hash
	annotations[LastConversionVersionAnnotation] = obj.GetObjectKind().GroupVersionKind().Version
	
	// Store preservation policy used
	if enhanced.PreservationPolicy != "" {
		annotations[PreservationPolicyAnnotation] = enhanced.PreservationPolicy
	}
	
	// Store unknown fields summary
	if len(enhanced.UnknownFields) > 0 {
		unknownFieldNames := make([]string, 0, len(enhanced.UnknownFields))
		for name := range enhanced.UnknownFields {
			unknownFieldNames = append(unknownFieldNames, name)
		}
		annotations[UnknownFieldsAnnotation] = strings.Join(unknownFieldNames, ",")
	}
	
	// Store preserved fields list
	preservedFields := make([]string, 0)
	for field := range enhanced.CustomFields {
		preservedFields = append(preservedFields, field)
	}
	for field := range enhanced.ComplexFields {
		preservedFields = append(preservedFields, field)
	}
	if len(preservedFields) > 0 {
		annotations[PreservedFieldsAnnotation] = strings.Join(preservedFields, ",")
	}
	
	meta.SetAnnotations(annotations)
	return nil
}

// Restoration methods

func (dp *DataPreserverEnhanced) validateDataForRestoration(enhanced *PreservedDataEnhanced) error {
	// Check data integrity
	if enhanced.MetadataSnapshot == nil {
		return fmt.Errorf("metadata snapshot is missing")
	}
	
	// Validate preservation policy
	if enhanced.PreservationPolicy == "" {
		dp.logger.V(1).Info("No preservation policy specified, using default")
		enhanced.PreservationPolicy = "default"
	}
	
	// Check for critical validation failures
	for _, result := range enhanced.ValidationResults {
		if !result.Valid && result.Level == "error" {
			return fmt.Errorf("critical validation failure for field %s: %s", result.Field, result.Message)
		}
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) restoreEnhancedMetadata(obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	if enhanced.MetadataSnapshot == nil {
		return nil
	}
	
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}
	
	meta := accessor.GetObjectMeta()
	
	// Restore critical metadata
	if len(enhanced.MetadataSnapshot.OwnerReferences) > 0 {
		meta.SetOwnerReferences(enhanced.MetadataSnapshot.OwnerReferences)
	}
	
	if len(enhanced.MetadataSnapshot.Finalizers) > 0 {
		meta.SetFinalizers(enhanced.MetadataSnapshot.Finalizers)
	}
	
	// Note: Some metadata like UID and ResourceVersion should not be restored
	
	return nil
}

func (dp *DataPreserverEnhanced) restoreComplexFields(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	for path, complex := range enhanced.ComplexFields {
		if complex.PreservedValue == nil {
			continue
		}
		
		// Get the strategy used
		strategy, err := dp.strategyRegistry.Get(string(complex.Strategy))
		if err != nil {
			dp.logger.Error(err, "Strategy not found", "strategy", complex.Strategy)
			continue
		}
		
		// Create preserved field for restoration
		preserved := &preservation.PreservedField{
			Descriptor: preservation.FieldDescriptor{
				Path: path,
			},
			OriginalValue: complex.PreservedValue,
			Strategy:      complex.Strategy,
		}
		
		// Restore using strategy
		if err := strategy.Restore(ctx, obj, preserved); err != nil {
			dp.logger.Error(err, "Failed to restore complex field", "field", path)
			// Continue with other fields
		}
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) restoreUnknownFields(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	for _, unknown := range enhanced.UnknownFields {
		if err := dp.unknownFieldHandler.RestoreUnknownField(ctx, obj, unknown); err != nil {
			dp.logger.Error(err, "Failed to restore unknown field", "field", unknown.Path)
			// Continue with other fields
		}
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) restoreStatusEnhanced(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	if len(enhanced.Status) == 0 {
		return nil
	}
	
	objValue := reflect.ValueOf(obj).Elem()
	statusField := objValue.FieldByName("Status")
	
	if !statusField.IsValid() || !statusField.CanSet() {
		return fmt.Errorf("status field not found or not settable")
	}
	
	// Apply preserved status fields
	for key, value := range enhanced.Status {
		if err := dp.setFieldValue(statusField, key, value); err != nil {
			dp.logger.Error(err, "Failed to restore status field",
				"field", key,
				"value", value)
			// Continue with other fields
		}
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) restoreAnnotationsAndLabelsEnhanced(obj runtime.Object, enhanced *PreservedDataEnhanced) {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		dp.logger.Error(err, "Failed to access object metadata")
		return
	}
	
	meta := accessor.GetObjectMeta()
	
	// Restore annotations
	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	
	for key, value := range enhanced.Annotations {
		annotations[key] = value
	}
	
	// Add restoration tracking
	annotations[DataPreservationLabel] = "true"
	
	meta.SetAnnotations(annotations)
	
	// Restore labels
	labels := meta.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	
	for key, value := range enhanced.Labels {
		labels[key] = value
	}
	
	meta.SetLabels(labels)
}

func (dp *DataPreserverEnhanced) updateResourceReferencesEnhanced(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	// Update resource references with version mapping
	for _, ref := range enhanced.ResourceReferences {
		updatedRef := dp.updateReferenceVersion(ref)
		
		if err := dp.applyReference(obj, updatedRef); err != nil {
			dp.logger.Error(err, "Failed to update resource reference",
				"reference", ref,
				"fieldPath", ref.FieldPath)
			// Continue with other references
		}
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) applyFieldMappingsEnhanced(ctx context.Context, obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	objValue := reflect.ValueOf(obj).Elem()
	
	// Apply field mappings
	for sourceField, targetField := range enhanced.FieldMappings {
		// Get value from custom fields or complex fields
		var value interface{}
		
		if customValue, ok := enhanced.CustomFields[sourceField]; ok {
			value = customValue
		} else if complexField, ok := enhanced.ComplexFields[sourceField]; ok {
			value = complexField.PreservedValue
			if value == nil {
				value = complexField.OriginalValue
			}
		}
		
		if value != nil {
			if err := dp.setFieldByPath(objValue, targetField, value); err != nil {
				dp.logger.Error(err, "Failed to apply field mapping",
					"sourceField", sourceField,
					"targetField", targetField,
					"value", value)
				// Continue with other mappings
			}
		}
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) verifyDataIntegrityEnhanced(obj runtime.Object, enhanced *PreservedDataEnhanced) error {
	// Recalculate hash
	currentHash, err := dp.calculateEnhancedDataHash(enhanced)
	if err != nil {
		return fmt.Errorf("failed to calculate current hash: %w", err)
	}
	
	// Get stored hash
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}
	
	meta := accessor.GetObjectMeta()
	storedHash := meta.GetAnnotations()[DataIntegrityHashAnnotation]
	
	if storedHash != currentHash {
		return fmt.Errorf("data integrity check failed: hash mismatch (stored: %s, current: %s)", storedHash, currentHash)
	}
	
	// Additional integrity checks
	if enhanced.MetadataSnapshot != nil && enhanced.Metadata.UID != enhanced.MetadataSnapshot.UID {
		return fmt.Errorf("UID mismatch in metadata")
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) updateConversionHistoryEnhanced(obj runtime.Object, enhanced *PreservedDataEnhanced) {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		dp.logger.Error(err, "Failed to access object metadata")
		return
	}
	
	meta := accessor.GetObjectMeta()
	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	
	// Get existing history
	var history []ConversionHistory
	if historyJSON, ok := annotations[ConversionHistoryAnnotation]; ok {
		_ = json.Unmarshal([]byte(historyJSON), &history)
	}
	
	// Add new entry with enhanced information
	newEntry := ConversionHistory{
		Timestamp:      metav1.Now(),
		SourceVersion:  annotations[LastConversionVersionAnnotation],
		TargetVersion:  obj.GetObjectKind().GroupVersionKind().Version,
		Success:        true,
		DataPreserved:  true,
		FieldsModified: dp.getModifiedFields(enhanced),
	}
	history = append(history, newEntry)
	
	// Keep only last 10 entries
	if len(history) > 10 {
		history = history[len(history)-10:]
	}
	
	// Update annotation
	if historyJSON, err := json.Marshal(history); err == nil {
		annotations[ConversionHistoryAnnotation] = string(historyJSON)
		meta.SetAnnotations(annotations)
	}
}

func (dp *DataPreserverEnhanced) getModifiedFields(enhanced *PreservedDataEnhanced) []string {
	modified := make([]string, 0)
	
	// Add complex fields that were transformed
	for path, complex := range enhanced.ComplexFields {
		if complex.TransformApplied {
			modified = append(modified, path)
		}
	}
	
	// Add fields with specific strategies
	for path, strategy := range enhanced.Strategies {
		if strategy == preservation.StrategyTypeTransform {
			modified = append(modified, path)
		}
	}
	
	return modified
}

// Helper methods for enhanced functionality

func (dp *DataPreserverEnhanced) setFieldValue(structValue reflect.Value, fieldName string, value interface{}) error {
	field := structValue.FieldByName(fieldName)
	if !field.IsValid() {
		// Try to find field by JSON tag
		structType := structValue.Type()
		for i := 0; i < structType.NumField(); i++ {
			f := structType.Field(i)
			jsonTag := f.Tag.Get("json")
			if jsonTag == fieldName || strings.Split(jsonTag, ",")[0] == fieldName {
				field = structValue.Field(i)
				break
			}
		}
	}
	
	if !field.IsValid() || !field.CanSet() {
		return fmt.Errorf("field %s not found or not settable", fieldName)
	}
	
	// Convert value to appropriate type
	targetValue := reflect.ValueOf(value)
	if targetValue.Type().ConvertibleTo(field.Type()) {
		field.Set(targetValue.Convert(field.Type()))
	} else {
		return fmt.Errorf("cannot convert %T to %v", value, field.Type())
	}
	
	return nil
}

func (dp *DataPreserverEnhanced) setFieldByPath(v reflect.Value, path string, value interface{}) error {
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

func (dp *DataPreserverEnhanced) updateReferenceVersion(ref ResourceReference) ResourceReference {
	updated := ref
	
	// Map old API versions to new ones
	versionMappings := map[string]string{
		"observability.io/v1alpha1": "observability.io/v1beta1",
		"observability.io/v1beta1":  "observability.io/v1",
		"v1alpha1":                  "v1beta1",
		"v1beta1":                   "v1",
	}
	
	if newVersion, ok := versionMappings[ref.APIVersion]; ok {
		updated.APIVersion = newVersion
	}
	
	return updated
}

func (dp *DataPreserverEnhanced) applyReference(obj runtime.Object, ref ResourceReference) error {
	objValue := reflect.ValueOf(obj).Elem()
	return dp.setFieldByPath(objValue, ref.FieldPath, ref)
}

// preservationCache provides caching for preservation results
type preservationCache struct {
	cache map[string]*PreservedDataEnhanced
	mu    sync.RWMutex
	ttl   time.Duration
}

func newPreservationCache() *preservationCache {
	return &preservationCache{
		cache: make(map[string]*PreservedDataEnhanced),
		ttl:   5 * time.Minute,
	}
}

func (c *preservationCache) Get(key string) (*PreservedDataEnhanced, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	data, exists := c.cache[key]
	return data, exists
}

func (c *preservationCache) Put(key string, data *PreservedDataEnhanced) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[key] = data
	
	// Simple TTL implementation - in production, use a proper cache
	go func() {
		time.Sleep(c.ttl)
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
	}()
}

// PreservationMetrics tracks preservation metrics
type PreservationMetrics struct {
	preservationAttempts   int64
	preservationSuccesses  int64
	preservationErrors     int64
	restorationAttempts    int64
	restorationSuccesses   int64
	restorationErrors      int64
	cacheHits              int64
	cacheMisses            int64
	integrityCheckFailures int64
	mu                     sync.RWMutex
}

func NewPreservationMetrics() *PreservationMetrics {
	return &PreservationMetrics{}
}

func (m *PreservationMetrics) RecordPreservationStart() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preservationAttempts++
}

func (m *PreservationMetrics) RecordPreservationComplete(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preservationSuccesses++
}

func (m *PreservationMetrics) RecordPreservationError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preservationErrors++
}

func (m *PreservationMetrics) RecordRestorationStart() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restorationAttempts++
}

func (m *PreservationMetrics) RecordRestorationComplete(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restorationSuccesses++
}

func (m *PreservationMetrics) RecordRestorationError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restorationErrors++
}

func (m *PreservationMetrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheHits++
}

func (m *PreservationMetrics) RecordIntegrityCheckFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.integrityCheckFailures++
}

// GetMetrics returns a copy of the metrics
func (m *PreservationMetrics) GetMetrics() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]int64{
		"preservation_attempts":      m.preservationAttempts,
		"preservation_successes":     m.preservationSuccesses,
		"preservation_errors":        m.preservationErrors,
		"restoration_attempts":       m.restorationAttempts,
		"restoration_successes":      m.restorationSuccesses,
		"restoration_errors":         m.restorationErrors,
		"cache_hits":                 m.cacheHits,
		"cache_misses":               m.cacheMisses,
		"integrity_check_failures":   m.integrityCheckFailures,
	}
}

// GetMetrics returns preservation metrics from the DataPreserverEnhanced
func (dp *DataPreserverEnhanced) GetMetrics() map[string]int64 {
	if dp.metrics != nil {
		return dp.metrics.GetMetrics()
	}
	return make(map[string]int64)
}
