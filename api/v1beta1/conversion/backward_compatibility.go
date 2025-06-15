/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/pkg/apis/observability/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("backward-compatibility")

// BackwardCompatibilityManager handles backward compatibility between API versions
type BackwardCompatibilityManager struct {
	client                client.Client
	scheme                *runtime.Scheme
	supportedVersions     []string
	deprecatedFields      map[string]map[string]DeprecationInfo
	defaultValues         map[string]map[string]interface{}
	featureGates          map[string]FeatureGate
	clientCapabilities    map[string]ClientCapability
	serializationHandlers map[string]SerializationHandler
}

// DeprecationInfo contains information about deprecated fields
type DeprecationInfo struct {
	DeprecatedIn   string
	RemovedIn      string
	Replacement    string
	AutoMigrate    bool
	MigrationFunc  func(interface{}) (interface{}, error)
}

// FeatureGate represents a feature that can be enabled/disabled based on client version
type FeatureGate struct {
	Name            string
	MinVersion      string
	DefaultEnabled  bool
	DegradationFunc func(interface{}) (interface{}, error)
}

// ClientCapability represents capabilities of a client version
type ClientCapability struct {
	Version            string
	SupportedResources []schema.GroupVersionResource
	Features           []string
	MaxPayloadSize     int64
}

// SerializationHandler handles custom serialization for backward compatibility
type SerializationHandler interface {
	Serialize(obj runtime.Object, clientVersion string) ([]byte, error)
	Deserialize(data []byte, clientVersion string) (runtime.Object, error)
}

// NewBackwardCompatibilityManager creates a new backward compatibility manager
func NewBackwardCompatibilityManager(client client.Client, scheme *runtime.Scheme) *BackwardCompatibilityManager {
	bcm := &BackwardCompatibilityManager{
		client:                client,
		scheme:                scheme,
		supportedVersions:     []string{"v1alpha1", "v1beta1", "v1"},
		deprecatedFields:      make(map[string]map[string]DeprecationInfo),
		defaultValues:         make(map[string]map[string]interface{}),
		featureGates:          make(map[string]FeatureGate),
		clientCapabilities:    make(map[string]ClientCapability),
		serializationHandlers: make(map[string]SerializationHandler),
	}

	// Initialize deprecated fields
	bcm.initializeDeprecatedFields()
	
	// Initialize default values
	bcm.initializeDefaultValues()
	
	// Initialize feature gates
	bcm.initializeFeatureGates()
	
	// Initialize client capabilities
	bcm.initializeClientCapabilities()
	
	return bcm
}

// NegotiateVersion negotiates the best API version based on client capabilities
func (bcm *BackwardCompatibilityManager) NegotiateVersion(clientVersion string, acceptHeaders []string) (string, error) {
	log.V(1).Info("Negotiating API version", "clientVersion", clientVersion, "acceptHeaders", acceptHeaders)
	
	// Parse client version
	cv, err := version.ParseSemantic(clientVersion)
	if err != nil {
		// Try to extract version from user agent or other headers
		cv = bcm.extractVersionFromHeaders(acceptHeaders)
		if cv == nil {
			// Default to oldest supported version for maximum compatibility
			return "v1alpha1", nil
		}
	}
	
	// Determine best version based on client capabilities
	bestVersion := bcm.selectBestVersion(cv)
	
	log.V(1).Info("Version negotiation complete", "selectedVersion", bestVersion)
	return bestVersion, nil
}

// HandleUnknownFields gracefully handles unknown fields in the request
func (bcm *BackwardCompatibilityManager) HandleUnknownFields(obj runtime.Object, clientVersion string) (runtime.Object, map[string]interface{}, error) {
	log.V(2).Info("Handling unknown fields", "type", reflect.TypeOf(obj), "clientVersion", clientVersion)
	
	// Convert to unstructured to access all fields
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, nil, fmt.Errorf("converting to unstructured: %w", err)
	}
	
	// Extract unknown fields
	unknownFields := make(map[string]interface{})
	knownFields := bcm.getKnownFieldsForVersion(obj, clientVersion)
	
	for key, value := range unstructuredObj {
		if !bcm.isKnownField(key, knownFields) && !bcm.isSystemField(key) {
			unknownFields[key] = value
			// Store unknown fields in annotations for preservation
			bcm.storeUnknownFieldsInAnnotations(obj, key, value)
		}
	}
	
	if len(unknownFields) > 0 {
		log.Info("Unknown fields detected", "count", len(unknownFields), "fields", unknownFields)
	}
	
	return obj, unknownFields, nil
}

// ApplyDefaultValues applies default values for missing required fields
func (bcm *BackwardCompatibilityManager) ApplyDefaultValues(obj runtime.Object, clientVersion string) error {
	log.V(2).Info("Applying default values", "type", reflect.TypeOf(obj), "clientVersion", clientVersion)
	
	objType := reflect.TypeOf(obj).String()
	defaults, ok := bcm.defaultValues[objType]
	if !ok {
		return nil
	}
	
	// Use reflection to set default values
	objValue := reflect.ValueOf(obj).Elem()
	for fieldPath, defaultValue := range defaults {
		if err := bcm.setFieldValue(objValue, fieldPath, defaultValue, clientVersion); err != nil {
			log.Error(err, "Failed to set default value", "field", fieldPath)
			// Continue with other fields
		}
	}
	
	return nil
}

// AddCompatibilityHeaders adds compatibility headers to the response
func (bcm *BackwardCompatibilityManager) AddCompatibilityHeaders(headers map[string]string, clientVersion, negotiatedVersion string) {
	headers["X-API-Version"] = negotiatedVersion
	headers["X-API-Deprecated-Fields"] = bcm.getDeprecatedFieldsHeader(negotiatedVersion)
	headers["X-API-Min-Version"] = bcm.getMinSupportedVersion()
	headers["X-API-Max-Version"] = bcm.getMaxSupportedVersion()
	headers["X-API-Client-Version"] = clientVersion
	headers["X-API-Features"] = bcm.getAvailableFeatures(clientVersion)
}

// CheckFeatureCompatibility checks if a feature is compatible with the client version
func (bcm *BackwardCompatibilityManager) CheckFeatureCompatibility(feature string, clientVersion string) (bool, interface{}) {
	gate, exists := bcm.featureGates[feature]
	if !exists {
		return true, nil // Unknown features are assumed compatible
	}
	
	cv, err := version.ParseSemantic(clientVersion)
	if err != nil {
		return gate.DefaultEnabled, nil
	}
	
	minVer, err := version.ParseSemantic(gate.MinVersion)
	if err != nil {
		return gate.DefaultEnabled, nil
	}
	
	if cv.LessThan(minVer) {
		log.V(1).Info("Feature not compatible with client version", "feature", feature, "clientVersion", clientVersion)
		if gate.DegradationFunc != nil {
			// Return degraded version of the feature
			degraded, err := gate.DegradationFunc(nil)
			if err == nil {
				return false, degraded
			}
		}
		return false, nil
	}
	
	return true, nil
}

// SerializeForClient serializes an object for a specific client version
func (bcm *BackwardCompatibilityManager) SerializeForClient(obj runtime.Object, clientVersion string) ([]byte, error) {
	log.V(2).Info("Serializing for client", "type", reflect.TypeOf(obj), "clientVersion", clientVersion)
	
	// Apply version-specific transformations
	transformedObj, err := bcm.transformForVersion(obj, clientVersion)
	if err != nil {
		return nil, fmt.Errorf("transforming object: %w", err)
	}
	
	// Check for custom serialization handler
	if handler, exists := bcm.serializationHandlers[clientVersion]; exists {
		return handler.Serialize(transformedObj, clientVersion)
	}
	
	// Default JSON serialization
	return json.Marshal(transformedObj)
}

// Helper methods

func (bcm *BackwardCompatibilityManager) initializeDeprecatedFields() {
	// v1alpha1 deprecated fields
	bcm.deprecatedFields["*v1alpha1.ObservabilityPlatform"] = map[string]DeprecationInfo{
		"spec.monitoring": {
			DeprecatedIn: "v1beta1",
			RemovedIn:    "v1",
			Replacement:  "spec.components",
			AutoMigrate:  true,
			MigrationFunc: func(value interface{}) (interface{}, error) {
				// Convert old monitoring field to new components structure
				return bcm.migrateMonitoringToComponents(value)
			},
		},
		"spec.alerting.enabled": {
			DeprecatedIn: "v1beta1",
			RemovedIn:    "v1",
			Replacement:  "spec.components.alertmanager.enabled",
			AutoMigrate:  true,
		},
	}
}

func (bcm *BackwardCompatibilityManager) initializeDefaultValues() {
	// Default values for v1beta1
	bcm.defaultValues["*v1beta1.ObservabilityPlatform"] = map[string]interface{}{
		"spec.components.prometheus.retention":     "30d",
		"spec.components.prometheus.replicas":      int32(1),
		"spec.components.grafana.adminPassword":    "admin",
		"spec.components.loki.retentionDays":       int32(7),
		"spec.global.externalLabels.cluster":       "default",
		"spec.resources.requests.memory":           "128Mi",
		"spec.resources.requests.cpu":              "100m",
	}
	
	// Default values for v1alpha1 (legacy)
	bcm.defaultValues["*v1alpha1.ObservabilityPlatform"] = map[string]interface{}{
		"spec.monitoring.retention": "15d",
		"spec.monitoring.replicas":  int32(1),
	}
}

func (bcm *BackwardCompatibilityManager) initializeFeatureGates() {
	bcm.featureGates["multiCluster"] = FeatureGate{
		Name:           "multiCluster",
		MinVersion:     "v1.20.0",
		DefaultEnabled: false,
		DegradationFunc: func(value interface{}) (interface{}, error) {
			// Return single cluster configuration
			return map[string]interface{}{
				"enabled": false,
				"message": "Multi-cluster support requires client v1.20.0 or higher",
			}, nil
		},
	}
	
	bcm.featureGates["costOptimization"] = FeatureGate{
		Name:           "costOptimization",
		MinVersion:     "v1.18.0",
		DefaultEnabled: false,
		DegradationFunc: func(value interface{}) (interface{}, error) {
			return nil, nil // Simply omit the feature
		},
	}
}

func (bcm *BackwardCompatibilityManager) initializeClientCapabilities() {
	bcm.clientCapabilities["v1.15"] = ClientCapability{
		Version: "v1.15",
		SupportedResources: []schema.GroupVersionResource{
			{Group: "observability.io", Version: "v1alpha1", Resource: "observabilityplatforms"},
		},
		Features:       []string{"basic-monitoring", "alerts"},
		MaxPayloadSize: 1048576, // 1MB
	}
	
	bcm.clientCapabilities["v1.20"] = ClientCapability{
		Version: "v1.20",
		SupportedResources: []schema.GroupVersionResource{
			{Group: "observability.io", Version: "v1alpha1", Resource: "observabilityplatforms"},
			{Group: "observability.io", Version: "v1beta1", Resource: "observabilityplatforms"},
		},
		Features:       []string{"basic-monitoring", "alerts", "multi-cluster", "cost-optimization"},
		MaxPayloadSize: 5242880, // 5MB
	}
}

func (bcm *BackwardCompatibilityManager) extractVersionFromHeaders(headers []string) *version.Version {
	for _, header := range headers {
		if strings.HasPrefix(header, "User-Agent:") {
			// Extract version from user agent
			parts := strings.Split(header, "/")
			if len(parts) >= 2 {
				v, err := version.ParseSemantic(parts[1])
				if err == nil {
					return v
				}
			}
		}
	}
	return nil
}

func (bcm *BackwardCompatibilityManager) selectBestVersion(clientVersion *version.Version) string {
	// Logic to select the best API version based on client capabilities
	if clientVersion == nil {
		return "v1alpha1" // Most compatible
	}
	
	v120, _ := version.ParseSemantic("v1.20.0")
	if clientVersion.AtLeast(v120) {
		return "v1beta1"
	}
	
	return "v1alpha1"
}

func (bcm *BackwardCompatibilityManager) getKnownFieldsForVersion(obj runtime.Object, version string) map[string]bool {
	// Return known fields for the specific version
	knownFields := make(map[string]bool)
	
	// This would be populated based on the OpenAPI schema for each version
	// For now, returning a basic set
	switch version {
	case "v1alpha1":
		knownFields = map[string]bool{
			"apiVersion": true,
			"kind":       true,
			"metadata":   true,
			"spec":       true,
			"status":     true,
		}
	case "v1beta1":
		knownFields = map[string]bool{
			"apiVersion": true,
			"kind":       true,
			"metadata":   true,
			"spec":       true,
			"status":     true,
		}
	}
	
	return knownFields
}

func (bcm *BackwardCompatibilityManager) isKnownField(field string, knownFields map[string]bool) bool {
	// Check if field is known, supporting nested fields
	parts := strings.Split(field, ".")
	return knownFields[parts[0]]
}

func (bcm *BackwardCompatibilityManager) isSystemField(field string) bool {
	systemFields := []string{"apiVersion", "kind", "metadata", "status"}
	for _, sf := range systemFields {
		if field == sf || strings.HasPrefix(field, sf+".") {
			return true
		}
	}
	return false
}

func (bcm *BackwardCompatibilityManager) storeUnknownFieldsInAnnotations(obj runtime.Object, key string, value interface{}) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return
	}
	
	annotations := metaObj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	
	// Store unknown fields as JSON in annotations
	unknownData, _ := json.Marshal(value)
	annotations[fmt.Sprintf("observability.io/unknown-field-%s", key)] = string(unknownData)
	
	metaObj.SetAnnotations(annotations)
}

func (bcm *BackwardCompatibilityManager) setFieldValue(objValue reflect.Value, fieldPath string, value interface{}, clientVersion string) error {
	parts := strings.Split(fieldPath, ".")
	current := objValue
	
	for i, part := range parts {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				current.Set(reflect.New(current.Type().Elem()))
			}
			current = current.Elem()
		}
		
		field := current.FieldByName(part)
		if !field.IsValid() {
			return fmt.Errorf("field %s not found", part)
		}
		
		if i == len(parts)-1 {
			// Set the value
			if field.CanSet() {
				val := reflect.ValueOf(value)
				if field.Type() != val.Type() {
					// Try type conversion
					if val.Type().ConvertibleTo(field.Type()) {
						val = val.Convert(field.Type())
					} else {
						return fmt.Errorf("cannot convert %v to %v", val.Type(), field.Type())
					}
				}
				field.Set(val)
			} else {
				return fmt.Errorf("cannot set field %s", part)
			}
		} else {
			current = field
		}
	}
	
	return nil
}

func (bcm *BackwardCompatibilityManager) getDeprecatedFieldsHeader(version string) string {
	deprecated := []string{}
	for objType, fields := range bcm.deprecatedFields {
		for field, info := range fields {
			if info.DeprecatedIn == version || bcm.isVersionAfter(version, info.DeprecatedIn) {
				deprecated = append(deprecated, fmt.Sprintf("%s.%s", objType, field))
			}
		}
	}
	return strings.Join(deprecated, ",")
}

func (bcm *BackwardCompatibilityManager) getMinSupportedVersion() string {
	return "v1alpha1"
}

func (bcm *BackwardCompatibilityManager) getMaxSupportedVersion() string {
	return "v1beta1"
}

func (bcm *BackwardCompatibilityManager) getAvailableFeatures(clientVersion string) string {
	features := []string{}
	for name, gate := range bcm.featureGates {
		compatible, _ := bcm.CheckFeatureCompatibility(name, clientVersion)
		if compatible {
			features = append(features, name)
		}
	}
	return strings.Join(features, ",")
}

func (bcm *BackwardCompatibilityManager) isVersionAfter(v1, v2 string) bool {
	// Simple version comparison
	versions := []string{"v1alpha1", "v1beta1", "v1"}
	v1Idx, v2Idx := -1, -1
	
	for i, v := range versions {
		if v == v1 {
			v1Idx = i
		}
		if v == v2 {
			v2Idx = i
		}
	}
	
	return v1Idx > v2Idx
}

func (bcm *BackwardCompatibilityManager) transformForVersion(obj runtime.Object, clientVersion string) (runtime.Object, error) {
	// Apply version-specific transformations
	switch platform := obj.(type) {
	case *v1beta1.ObservabilityPlatform:
		if clientVersion == "v1alpha1" {
			// Transform v1beta1 to v1alpha1 format
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
			if err := bcm.convertV1Beta1ToV1Alpha1(platform, v1alpha1Platform); err != nil {
				return nil, err
			}
			return v1alpha1Platform, nil
		}
	}
	
	return obj, nil
}

func (bcm *BackwardCompatibilityManager) migrateMonitoringToComponents(value interface{}) (interface{}, error) {
	// Convert old monitoring structure to new components structure
	monitoring, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid monitoring value type")
	}
	
	components := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"enabled": monitoring["prometheusEnabled"],
			"version": monitoring["prometheusVersion"],
		},
		"grafana": map[string]interface{}{
			"enabled": monitoring["grafanaEnabled"],
			"version": monitoring["grafanaVersion"],
		},
	}
	
	return components, nil
}

func (bcm *BackwardCompatibilityManager) convertV1Beta1ToV1Alpha1(src *v1beta1.ObservabilityPlatform, dst *v1alpha1.ObservabilityPlatform) error {
	// Implement conversion logic
	dst.ObjectMeta = src.ObjectMeta
	dst.TypeMeta = metav1.TypeMeta{
		APIVersion: "observability.io/v1alpha1",
		Kind:       "ObservabilityPlatform",
	}
	
	// Convert spec - this would be more complex in reality
	// Here we're showing the pattern
	if src.Spec.Components.Prometheus != nil {
		// Map new structure to old structure
	}
	
	return nil
}
