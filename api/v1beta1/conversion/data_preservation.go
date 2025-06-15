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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Annotation keys for data preservation
	ConversionDataAnnotation      = "observability.io/conversion-data"
	LastConversionVersionAnnotation = "observability.io/last-conversion-version"
	PreservedFieldsAnnotation     = "observability.io/preserved-fields"
	DataIntegrityHashAnnotation   = "observability.io/data-integrity-hash"
	ConversionHistoryAnnotation   = "observability.io/conversion-history"
	
	// Label keys for conversion tracking
	ConversionSourceVersionLabel = "observability.io/source-version"
	ConversionTargetVersionLabel = "observability.io/target-version"
	ConversionTimestampLabel     = "observability.io/conversion-timestamp"
)

// DataPreserver handles data preservation during API conversions
type DataPreserver struct {
	logger logr.Logger
	client client.Client
}

// NewDataPreserver creates a new data preserver instance
func NewDataPreserver(logger logr.Logger, client client.Client) *DataPreserver {
	return &DataPreserver{
		logger: logger.WithName("data-preserver"),
		client: client,
	}
}

// PreservedData represents data that needs to be preserved during conversion
type PreservedData struct {
	// Status fields that should be maintained
	Status map[string]interface{} `json:"status,omitempty"`
	
	// User annotations that should be preserved
	Annotations map[string]string `json:"annotations,omitempty"`
	
	// User labels that should be preserved
	Labels map[string]string `json:"labels,omitempty"`
	
	// Custom fields not present in target version
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	
	// Resource references that need updating
	ResourceReferences []ResourceReference `json:"resourceReferences,omitempty"`
	
	// Metadata to preserve
	Metadata PreservedMetadata `json:"metadata,omitempty"`
	
	// Field mappings for data transformation
	FieldMappings map[string]string `json:"fieldMappings,omitempty"`
}

// ResourceReference represents a reference to another resource
type ResourceReference struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	FieldPath  string `json:"fieldPath"`
}

// PreservedMetadata contains metadata that should be preserved
type PreservedMetadata struct {
	CreationTimestamp metav1.Time       `json:"creationTimestamp"`
	Generation        int64             `json:"generation"`
	ResourceVersion   string            `json:"resourceVersion"`
	UID               types.UID         `json:"uid"`
	OwnerReferences   []metav1.OwnerReference `json:"ownerReferences,omitempty"`
	Finalizers        []string          `json:"finalizers,omitempty"`
}

// ConversionHistory tracks conversion events
type ConversionHistory struct {
	Timestamp      metav1.Time `json:"timestamp"`
	SourceVersion  string      `json:"sourceVersion"`
	TargetVersion  string      `json:"targetVersion"`
	Success        bool        `json:"success"`
	DataPreserved  bool        `json:"dataPreserved"`
	FieldsModified []string    `json:"fieldsModified,omitempty"`
}

// PreservationRule defines how to preserve specific fields
type PreservationRule struct {
	FieldPath    string                 `json:"fieldPath"`
	PreserveType PreservationType       `json:"preserveType"`
	Transform    func(interface{}) interface{} `json:"-"`
	Validator    func(interface{}) error       `json:"-"`
}

// PreservationType defines how a field should be preserved
type PreservationType string

const (
	// PreserveAsIs preserves the field without modification
	PreserveAsIs PreservationType = "as-is"
	
	// PreserveWithTransform applies a transformation before preserving
	PreserveWithTransform PreservationType = "transform"
	
	// PreserveInAnnotation stores the field in an annotation
	PreserveInAnnotation PreservationType = "annotation"
	
	// PreserveWithValidation validates before preserving
	PreserveWithValidation PreservationType = "validate"
)

// PreserveData extracts data that needs to be preserved during conversion
func (dp *DataPreserver) PreserveData(ctx context.Context, obj runtime.Object, targetVersion string) (*PreservedData, error) {
	dp.logger.V(1).Info("Preserving data for conversion",
		"objectType", reflect.TypeOf(obj).String(),
		"targetVersion", targetVersion)
	
	preserved := &PreservedData{
		Status:             make(map[string]interface{}),
		Annotations:        make(map[string]string),
		Labels:             make(map[string]string),
		CustomFields:       make(map[string]interface{}),
		ResourceReferences: []ResourceReference{},
		FieldMappings:      make(map[string]string),
	}
	
	// Extract metadata
	meta, err := dp.extractMetadata(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}
	preserved.Metadata = meta
	
	// Preserve status
	if err := dp.preserveStatus(obj, preserved); err != nil {
		return nil, fmt.Errorf("failed to preserve status: %w", err)
	}
	
	// Preserve annotations and labels
	dp.preserveAnnotationsAndLabels(obj, preserved)
	
	// Extract custom fields
	if err := dp.extractCustomFields(obj, targetVersion, preserved); err != nil {
		return nil, fmt.Errorf("failed to extract custom fields: %w", err)
	}
	
	// Extract resource references
	if err := dp.extractResourceReferences(obj, preserved); err != nil {
		return nil, fmt.Errorf("failed to extract resource references: %w", err)
	}
	
	// Generate field mappings
	dp.generateFieldMappings(obj, targetVersion, preserved)
	
	// Calculate data integrity hash
	hash, err := dp.calculateDataHash(preserved)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate data hash: %w", err)
	}
	
	// Store preservation data in annotations
	if err := dp.storePreservationData(obj, preserved, hash); err != nil {
		return nil, fmt.Errorf("failed to store preservation data: %w", err)
	}
	
	dp.logger.Info("Data preservation complete",
		"preservedFields", len(preserved.CustomFields),
		"annotations", len(preserved.Annotations),
		"labels", len(preserved.Labels),
		"references", len(preserved.ResourceReferences))
	
	return preserved, nil
}

// RestoreData restores preserved data after conversion
func (dp *DataPreserver) RestoreData(ctx context.Context, obj runtime.Object, preserved *PreservedData) error {
	dp.logger.V(1).Info("Restoring preserved data",
		"objectType", reflect.TypeOf(obj).String())
	
	// Restore status fields
	if err := dp.restoreStatus(obj, preserved); err != nil {
		return fmt.Errorf("failed to restore status: %w", err)
	}
	
	// Restore annotations and labels
	dp.restoreAnnotationsAndLabels(obj, preserved)
	
	// Update resource references
	if err := dp.updateResourceReferences(obj, preserved); err != nil {
		return fmt.Errorf("failed to update resource references: %w", err)
	}
	
	// Apply field mappings
	if err := dp.applyFieldMappings(obj, preserved); err != nil {
		return fmt.Errorf("failed to apply field mappings: %w", err)
	}
	
	// Verify data integrity
	if err := dp.verifyDataIntegrity(obj, preserved); err != nil {
		dp.logger.Error(err, "Data integrity verification failed")
		// Continue with restoration but log the error
	}
	
	// Update conversion history
	dp.updateConversionHistory(obj)
	
	dp.logger.Info("Data restoration complete")
	return nil
}

// extractMetadata extracts metadata from the object
func (dp *DataPreserver) extractMetadata(obj runtime.Object) (PreservedMetadata, error) {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return PreservedMetadata{}, err
	}
	
	meta := accessor.GetObjectMeta()
	return PreservedMetadata{
		CreationTimestamp: meta.GetCreationTimestamp(),
		Generation:        meta.GetGeneration(),
		ResourceVersion:   meta.GetResourceVersion(),
		UID:               meta.GetUID(),
		OwnerReferences:   meta.GetOwnerReferences(),
		Finalizers:        meta.GetFinalizers(),
	}, nil
}

// preserveStatus preserves status fields
func (dp *DataPreserver) preserveStatus(obj runtime.Object, preserved *PreservedData) error {
	objValue := reflect.ValueOf(obj).Elem()
	statusField := objValue.FieldByName("Status")
	
	if !statusField.IsValid() || statusField.IsZero() {
		return nil
	}
	
	// Convert status to map for preservation
	statusData, err := dp.structToMap(statusField.Interface())
	if err != nil {
		return fmt.Errorf("failed to convert status to map: %w", err)
	}
	
	preserved.Status = statusData
	return nil
}

// preserveAnnotationsAndLabels preserves user-defined annotations and labels
func (dp *DataPreserver) preserveAnnotationsAndLabels(obj runtime.Object, preserved *PreservedData) {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		dp.logger.Error(err, "Failed to access object metadata")
		return
	}
	
	meta := accessor.GetObjectMeta()
	
	// Preserve user annotations (exclude system annotations)
	for key, value := range meta.GetAnnotations() {
		if !dp.isSystemAnnotation(key) {
			preserved.Annotations[key] = value
		}
	}
	
	// Preserve user labels (exclude system labels)
	for key, value := range meta.GetLabels() {
		if !dp.isSystemLabel(key) {
			preserved.Labels[key] = value
		}
	}
}

// extractCustomFields extracts fields not present in target version
func (dp *DataPreserver) extractCustomFields(obj runtime.Object, targetVersion string, preserved *PreservedData) error {
	// Get source and target schemas
	sourceSchema := dp.getObjectSchema(obj)
	targetSchema := dp.getTargetSchema(targetVersion)
	
	// Find fields present in source but not in target
	objValue := reflect.ValueOf(obj).Elem()
	objType := objValue.Type()
	
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldName := field.Name
		
		// Skip system fields
		if dp.isSystemField(fieldName) {
			continue
		}
		
		// Check if field exists in target schema
		if !dp.fieldExistsInSchema(fieldName, targetSchema) {
			fieldValue := objValue.Field(i).Interface()
			if !reflect.ValueOf(fieldValue).IsZero() {
				preserved.CustomFields[fieldName] = fieldValue
				dp.logger.V(2).Info("Preserved custom field",
					"field", fieldName,
					"value", fieldValue)
			}
		}
	}
	
	return nil
}

// extractResourceReferences extracts references to other resources
func (dp *DataPreserver) extractResourceReferences(obj runtime.Object, preserved *PreservedData) error {
	// Walk through object structure to find resource references
	objValue := reflect.ValueOf(obj).Elem()
	dp.walkStructForReferences(objValue, "", preserved)
	return nil
}

// walkStructForReferences recursively walks struct to find resource references
func (dp *DataPreserver) walkStructForReferences(v reflect.Value, path string, preserved *PreservedData) {
	switch v.Kind() {
	case reflect.Struct:
		vType := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := vType.Field(i)
			fieldValue := v.Field(i)
			fieldPath := dp.buildFieldPath(path, field.Name)
			
			// Check if this is a resource reference
			if dp.isResourceReference(field.Type) {
				ref := dp.extractReference(fieldValue, fieldPath)
				if ref != nil {
					preserved.ResourceReferences = append(preserved.ResourceReferences, *ref)
				}
			} else {
				dp.walkStructForReferences(fieldValue, fieldPath, preserved)
			}
		}
		
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			dp.walkStructForReferences(v.Index(i), elemPath, preserved)
		}
		
	case reflect.Map:
		for _, key := range v.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			elemPath := dp.buildFieldPath(path, keyStr)
			dp.walkStructForReferences(v.MapIndex(key), elemPath, preserved)
		}
		
	case reflect.Ptr:
		if !v.IsNil() {
			dp.walkStructForReferences(v.Elem(), path, preserved)
		}
	}
}

// generateFieldMappings generates mappings between source and target fields
func (dp *DataPreserver) generateFieldMappings(obj runtime.Object, targetVersion string, preserved *PreservedData) {
	// Define known field mappings between versions
	mappings := dp.getFieldMappings(obj.GetObjectKind().GroupVersionKind().Version, targetVersion)
	
	for sourceField, targetField := range mappings {
		preserved.FieldMappings[sourceField] = targetField
		dp.logger.V(2).Info("Generated field mapping",
			"source", sourceField,
			"target", targetField)
	}
}

// storePreservationData stores preservation data in object annotations
func (dp *DataPreserver) storePreservationData(obj runtime.Object, preserved *PreservedData, hash string) error {
	accessor, err := metav1.ObjectMetaAccessor(obj)
	if err != nil {
		return err
	}
	
	meta := accessor.GetObjectMeta()
	annotations := meta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	
	// Serialize preserved data
	preservedJSON, err := json.Marshal(preserved)
	if err != nil {
		return fmt.Errorf("failed to marshal preserved data: %w", err)
	}
	
	// Store in annotations
	annotations[ConversionDataAnnotation] = string(preservedJSON)
	annotations[DataIntegrityHashAnnotation] = hash
	annotations[LastConversionVersionAnnotation] = obj.GetObjectKind().GroupVersionKind().Version
	
	// Update preserved fields list
	var preservedFields []string
	for field := range preserved.CustomFields {
		preservedFields = append(preservedFields, field)
	}
	if len(preservedFields) > 0 {
		annotations[PreservedFieldsAnnotation] = strings.Join(preservedFields, ",")
	}
	
	meta.SetAnnotations(annotations)
	return nil
}

// restoreStatus restores preserved status fields
func (dp *DataPreserver) restoreStatus(obj runtime.Object, preserved *PreservedData) error {
	if len(preserved.Status) == 0 {
		return nil
	}
	
	objValue := reflect.ValueOf(obj).Elem()
	statusField := objValue.FieldByName("Status")
	
	if !statusField.IsValid() || !statusField.CanSet() {
		return fmt.Errorf("status field not found or not settable")
	}
	
	// Apply preserved status fields
	for key, value := range preserved.Status {
		if err := dp.setFieldValue(statusField, key, value); err != nil {
			dp.logger.Error(err, "Failed to restore status field",
				"field", key,
				"value", value)
			// Continue with other fields
		}
	}
	
	return nil
}

// restoreAnnotationsAndLabels restores preserved annotations and labels
func (dp *DataPreserver) restoreAnnotationsAndLabels(obj runtime.Object, preserved *PreservedData) {
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
	for key, value := range preserved.Annotations {
		annotations[key] = value
	}
	meta.SetAnnotations(annotations)
	
	// Restore labels
	labels := meta.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for key, value := range preserved.Labels {
		labels[key] = value
	}
	meta.SetLabels(labels)
}

// updateResourceReferences updates resource references to new API versions
func (dp *DataPreserver) updateResourceReferences(obj runtime.Object, preserved *PreservedData) error {
	for _, ref := range preserved.ResourceReferences {
		// Update reference to use new API version if needed
		updatedRef := dp.updateReferenceVersion(ref)
		
		// Apply updated reference to object
		if err := dp.applyReference(obj, updatedRef); err != nil {
			dp.logger.Error(err, "Failed to update resource reference",
				"reference", ref,
				"fieldPath", ref.FieldPath)
			// Continue with other references
		}
	}
	return nil
}

// applyFieldMappings applies field mappings to converted object
func (dp *DataPreserver) applyFieldMappings(obj runtime.Object, preserved *PreservedData) error {
	objValue := reflect.ValueOf(obj).Elem()
	
	// Apply custom fields that can be mapped
	for fieldName, fieldValue := range preserved.CustomFields {
		if targetField, ok := preserved.FieldMappings[fieldName]; ok {
			if err := dp.setFieldByPath(objValue, targetField, fieldValue); err != nil {
				dp.logger.Error(err, "Failed to apply field mapping",
					"sourceField", fieldName,
					"targetField", targetField,
					"value", fieldValue)
				// Continue with other mappings
			}
		}
	}
	
	return nil
}

// verifyDataIntegrity verifies that data was preserved correctly
func (dp *DataPreserver) verifyDataIntegrity(obj runtime.Object, preserved *PreservedData) error {
	// Calculate current hash
	currentHash, err := dp.calculateDataHash(preserved)
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
		return fmt.Errorf("data integrity check failed: hash mismatch")
	}
	
	return nil
}

// updateConversionHistory updates the conversion history in annotations
func (dp *DataPreserver) updateConversionHistory(obj runtime.Object) {
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
	
	// Add new entry
	newEntry := ConversionHistory{
		Timestamp:     metav1.Now(),
		SourceVersion: annotations[LastConversionVersionAnnotation],
		TargetVersion: obj.GetObjectKind().GroupVersionKind().Version,
		Success:       true,
		DataPreserved: true,
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

// Helper methods

func (dp *DataPreserver) isSystemAnnotation(key string) bool {
	systemPrefixes := []string{
		"kubectl.kubernetes.io/",
		"kubernetes.io/",
		"k8s.io/",
		"observability.io/conversion-",
		"observability.io/last-conversion-",
		"observability.io/preserved-",
		"observability.io/data-integrity-",
	}
	
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (dp *DataPreserver) isSystemLabel(key string) bool {
	systemPrefixes := []string{
		"app.kubernetes.io/",
		"kubernetes.io/",
		"k8s.io/",
		"observability.io/conversion-",
	}
	
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (dp *DataPreserver) isSystemField(fieldName string) bool {
	systemFields := []string{
		"TypeMeta",
		"ObjectMeta",
		"ListMeta",
	}
	
	for _, field := range systemFields {
		if fieldName == field {
			return true
		}
	}
	return false
}

func (dp *DataPreserver) buildFieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

func (dp *DataPreserver) isResourceReference(t reflect.Type) bool {
	// Check if type represents a resource reference
	if t.Kind() == reflect.Struct {
		// Look for fields that indicate a resource reference
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Name == "Kind" || field.Name == "APIVersion" || field.Name == "Name" {
				return true
			}
		}
	}
	return false
}

func (dp *DataPreserver) extractReference(v reflect.Value, fieldPath string) *ResourceReference {
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	ref := &ResourceReference{
		FieldPath: fieldPath,
	}
	
	// Extract reference fields
	if field := v.FieldByName("Kind"); field.IsValid() && field.Kind() == reflect.String {
		ref.Kind = field.String()
	}
	if field := v.FieldByName("APIVersion"); field.IsValid() && field.Kind() == reflect.String {
		ref.APIVersion = field.String()
	}
	if field := v.FieldByName("Name"); field.IsValid() && field.Kind() == reflect.String {
		ref.Name = field.String()
	}
	if field := v.FieldByName("Namespace"); field.IsValid() && field.Kind() == reflect.String {
		ref.Namespace = field.String()
	}
	
	// Only return if we have meaningful data
	if ref.Kind != "" || ref.Name != "" {
		return ref
	}
	return nil
}

func (dp *DataPreserver) updateReferenceVersion(ref ResourceReference) ResourceReference {
	// Update API version if needed
	updated := ref
	
	// Map old API versions to new ones
	versionMappings := map[string]string{
		"observability.io/v1alpha1": "observability.io/v1beta1",
		"v1alpha1":                  "v1beta1",
	}
	
	if newVersion, ok := versionMappings[ref.APIVersion]; ok {
		updated.APIVersion = newVersion
	}
	
	return updated
}

func (dp *DataPreserver) applyReference(obj runtime.Object, ref ResourceReference) error {
	objValue := reflect.ValueOf(obj).Elem()
	return dp.setFieldByPath(objValue, ref.FieldPath, ref)
}

func (dp *DataPreserver) setFieldByPath(v reflect.Value, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	current := v
	
	for i, part := range parts {
		// Handle array/slice indices
		if strings.Contains(part, "[") {
			// Extract field name and index
			fieldName := part[:strings.Index(part, "[")]
			indexStr := part[strings.Index(part, "[")+1 : strings.Index(part, "]")]
			
			// Get field
			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return fmt.Errorf("field %s not found", fieldName)
			}
			
			// Parse index
			var index int
			fmt.Sscanf(indexStr, "%d", &index)
			
			// Access element
			if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
				if index >= field.Len() {
					return fmt.Errorf("index %d out of bounds for field %s", index, fieldName)
				}
				current = field.Index(index)
			} else {
				return fmt.Errorf("field %s is not a slice or array", fieldName)
			}
		} else {
			// Regular field access
			field := current.FieldByName(part)
			if !field.IsValid() {
				return fmt.Errorf("field %s not found", part)
			}
			
			// If this is the last part, set the value
			if i == len(parts)-1 {
				if !field.CanSet() {
					return fmt.Errorf("field %s cannot be set", part)
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
			
			current = field
		}
	}
	
	return nil
}

func (dp *DataPreserver) setFieldValue(structValue reflect.Value, fieldName string, value interface{}) error {
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

func (dp *DataPreserver) structToMap(v interface{}) (map[string]interface{}, error) {
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

func (dp *DataPreserver) calculateDataHash(preserved *PreservedData) (string, error) {
	// Serialize preserved data for hashing
	data, err := json.Marshal(preserved)
	if err != nil {
		return "", err
	}
	
	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

func (dp *DataPreserver) getObjectSchema(obj runtime.Object) schema.GroupVersionKind {
	return obj.GetObjectKind().GroupVersionKind()
}

func (dp *DataPreserver) getTargetSchema(version string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "observability.io",
		Version: version,
		Kind:    "ObservabilityPlatform",
	}
}

func (dp *DataPreserver) fieldExistsInSchema(fieldName string, schema schema.GroupVersionKind) bool {
	// This would typically check against OpenAPI schema
	// For now, we'll use a predefined list based on version
	switch schema.Version {
	case "v1beta1":
		return dp.isV1Beta1Field(fieldName)
	case "v1alpha1":
		return dp.isV1Alpha1Field(fieldName)
	default:
		return false
	}
}

func (dp *DataPreserver) isV1Beta1Field(fieldName string) bool {
	v1beta1Fields := []string{
		"Spec", "Status", "Components", "Global", "HighAvailability",
		"Backup", "Alerting", "NodeSelector", "Paused",
		// Add all v1beta1 fields here
	}
	
	for _, field := range v1beta1Fields {
		if field == fieldName {
			return true
		}
	}
	return false
}

func (dp *DataPreserver) isV1Alpha1Field(fieldName string) bool {
	v1alpha1Fields := []string{
		"Spec", "Status", "Components", "Global", "HighAvailability",
		"Backup", "Alerting", "Paused",
		// Add all v1alpha1 fields here
	}
	
	for _, field := range v1alpha1Fields {
		if field == fieldName {
			return true
		}
	}
	return false
}

func (dp *DataPreserver) getFieldMappings(sourceVersion, targetVersion string) map[string]string {
	// Define field mappings between versions
	mappings := make(map[string]string)
	
	if sourceVersion == "v1alpha1" && targetVersion == "v1beta1" {
		// v1alpha1 to v1beta1 mappings
		mappings["Spec.Global.ExternalLabels"] = "Spec.Global.ExternalLabels"
		mappings["Spec.Components.Prometheus.CustomConfig"] = "Spec.Components.Prometheus.AdditionalScrapeConfigs"
		mappings["Spec.Backup.RetentionDays"] = "Spec.Backup.Retention"
		mappings["Spec.Alerting.AlertManager"] = "Spec.Alerting.Alertmanager"
		// Add more mappings as needed
	} else if sourceVersion == "v1beta1" && targetVersion == "v1alpha1" {
		// v1beta1 to v1alpha1 mappings (reverse)
		mappings["Spec.Components.Prometheus.AdditionalScrapeConfigs"] = "Spec.Components.Prometheus.CustomConfig"
		mappings["Spec.Backup.Retention"] = "Spec.Backup.RetentionDays"
		mappings["Spec.Alerting.Alertmanager"] = "Spec.Alerting.AlertManager"
		// Add more mappings as needed
	}
	
	return mappings
}

// GetPreservationRules returns preservation rules for specific fields
func (dp *DataPreserver) GetPreservationRules(sourceVersion, targetVersion string) []PreservationRule {
	rules := []PreservationRule{
		// Status preservation rules
		{
			FieldPath:    "Status",
			PreserveType: PreserveAsIs,
		},
		// Metadata preservation rules
		{
			FieldPath:    "metadata.finalizers",
			PreserveType: PreserveAsIs,
		},
		{
			FieldPath:    "metadata.ownerReferences",
			PreserveType: PreserveAsIs,
		},
		// Custom field preservation rules
		{
			FieldPath:    "spec.customConfig",
			PreserveType: PreserveInAnnotation,
		},
		// Transform rules for specific fields
		{
			FieldPath:    "spec.backup.retentionDays",
			PreserveType: PreserveWithTransform,
			Transform: func(v interface{}) interface{} {
				// Convert days to duration string
				if days, ok := v.(int32); ok {
					return fmt.Sprintf("%dd", days)
				}
				return v
			},
		},
		// Validation rules
		{
			FieldPath:    "spec.components.prometheus.version",
			PreserveType: PreserveWithValidation,
			Validator: func(v interface{}) error {
				if version, ok := v.(string); ok {
					// Validate version format
					if !strings.HasPrefix(version, "v") {
						return fmt.Errorf("version must start with 'v'")
					}
				}
				return nil
			},
		},
	}
	
	return rules
}

// ValidatePreservedData validates that preserved data can be safely restored
func (dp *DataPreserver) ValidatePreservedData(preserved *PreservedData) error {
	// Validate status fields
	if preserved.Status != nil {
		for field, value := range preserved.Status {
			if value == nil {
				return fmt.Errorf("null value for status field %s", field)
			}
		}
	}
	
	// Validate resource references
	for _, ref := range preserved.ResourceReferences {
		if ref.Kind == "" && ref.Name == "" {
			return fmt.Errorf("invalid resource reference at %s", ref.FieldPath)
		}
	}
	
	// Validate field mappings
	for source, target := range preserved.FieldMappings {
		if source == "" || target == "" {
			return fmt.Errorf("invalid field mapping: %s -> %s", source, target)
		}
	}
	
	return nil
}
