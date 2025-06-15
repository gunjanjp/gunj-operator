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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

var legacyLog = ctrl.Log.WithName("legacy-client-support")

// LegacyClientSupport provides backward compatibility for older clients
type LegacyClientSupport struct {
	clientDetector     ClientDetector
	responseAdapter    ResponseAdapter
	requestTransformer RequestTransformer
	featureMasker      FeatureMasker
	errorTranslator    ErrorTranslator
	metrics            *LegacyMetrics
}

// ClientDetector detects legacy client versions and capabilities
type ClientDetector struct {
	patterns      map[string]*regexp.Regexp
	versionMap    map[string]ClientVersion
	capabilities  map[string][]string
	quirksHandler QuirksHandler
}

// ClientVersion represents a detected client version
type ClientVersion struct {
	Major        int
	Minor        int
	Patch        int
	PreRelease   string
	ClientType   string
	Capabilities []string
	Limitations  []string
	Quirks       []string
}

// QuirksHandler handles known client quirks and bugs
type QuirksHandler interface {
	HandleQuirk(quirk string, request interface{}, response interface{}) error
	ApplyWorkaround(quirk string, data interface{}) interface{}
}

// ResponseAdapter adapts responses for legacy clients
type ResponseAdapter struct {
	transformers  map[string]ResponseTransformer
	fieldMappers  map[string]FieldMapper
	typeConverters map[string]TypeConverter
}

// ResponseTransformer transforms responses for specific client versions
type ResponseTransformer interface {
	Transform(response interface{}, clientVersion ClientVersion) (interface{}, error)
	CanTransform(response interface{}) bool
}

// FieldMapper maps fields between API versions
type FieldMapper interface {
	MapField(source, target interface{}, fieldPath string) error
	GetMapping(sourceVersion, targetVersion string) map[string]string
}

// TypeConverter converts types for legacy clients
type TypeConverter interface {
	Convert(value interface{}, fromType, toType string) (interface{}, error)
	IsConvertible(fromType, toType string) bool
}

// RequestTransformer transforms legacy requests to current format
type RequestTransformer struct {
	validators    map[string]RequestValidator
	normalizers   map[string]RequestNormalizer
	enrichers     map[string]RequestEnricher
}

// RequestValidator validates legacy requests
type RequestValidator interface {
	Validate(request interface{}, clientVersion ClientVersion) error
	GetValidationRules(clientVersion ClientVersion) []ValidationRule
}

// RequestNormalizer normalizes legacy requests
type RequestNormalizer interface {
	Normalize(request interface{}) (interface{}, error)
	NeedsNormalization(request interface{}) bool
}

// RequestEnricher enriches legacy requests with required fields
type RequestEnricher interface {
	Enrich(request interface{}, clientVersion ClientVersion) (interface{}, error)
	GetRequiredFields(clientVersion ClientVersion) []string
}

// FeatureMasker hides unsupported features from legacy clients
type FeatureMasker struct {
	featureGates   map[string]FeatureGateConfig
	versionMatrix  map[string][]string
	degradationMap map[string]DegradationStrategy
}

// FeatureGateConfig configures feature visibility
type FeatureGateConfig struct {
	Name             string
	MinVersion       ClientVersion
	Enabled          bool
	DegradationMode  string
	AlternativeValue interface{}
}

// DegradationStrategy defines how to degrade features
type DegradationStrategy interface {
	Degrade(feature string, value interface{}) interface{}
	CanDegrade(feature string) bool
}

// ErrorTranslator translates modern errors to legacy format
type ErrorTranslator struct {
	errorMappings   map[string]ErrorMapping
	codeTranslator  CodeTranslator
	messageAdapter  MessageAdapter
}

// ErrorMapping maps modern errors to legacy formats
type ErrorMapping struct {
	ModernCode   string
	LegacyCode   string
	Transform    func(error) error
	StatusCode   int
}

// CodeTranslator translates error codes
type CodeTranslator interface {
	TranslateCode(modernCode string, clientVersion ClientVersion) string
	GetLegacyCode(modernCode string) string
}

// MessageAdapter adapts error messages
type MessageAdapter interface {
	AdaptMessage(message string, clientVersion ClientVersion) string
	SimplifyMessage(message string) string
}

// LegacyMetrics tracks legacy client usage
type LegacyMetrics struct {
	clientVersions    map[string]int
	featureUsage      map[string]map[string]int
	transformations   map[string]int
	errors            map[string]int
	deprecationHits   map[string]int
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Field      string
	Type       string
	Required   bool
	Validator  func(interface{}) error
}

// NewLegacyClientSupport creates a new legacy client support instance
func NewLegacyClientSupport() *LegacyClientSupport {
	lcs := &LegacyClientSupport{
		metrics: &LegacyMetrics{
			clientVersions:  make(map[string]int),
			featureUsage:    make(map[string]map[string]int),
			transformations: make(map[string]int),
			errors:          make(map[string]int),
			deprecationHits: make(map[string]int),
		},
	}

	// Initialize components
	lcs.initializeClientDetector()
	lcs.initializeResponseAdapter()
	lcs.initializeRequestTransformer()
	lcs.initializeFeatureMasker()
	lcs.initializeErrorTranslator()

	return lcs
}

// DetectClient detects the client version from the request
func (lcs *LegacyClientSupport) DetectClient(r *http.Request) ClientVersion {
	userAgent := r.Header.Get("User-Agent")
	xClientVersion := r.Header.Get("X-Client-Version")
	acceptHeader := r.Header.Get("Accept")

	// Try direct version header first
	if xClientVersion != "" {
		if version := lcs.parseVersionString(xClientVersion); version != (ClientVersion{}) {
			return version
		}
	}

	// Detect from User-Agent
	return lcs.clientDetector.DetectFromUserAgent(userAgent, acceptHeader)
}

// HandleLegacyRequest processes a request from a legacy client
func (lcs *LegacyClientSupport) HandleLegacyRequest(ctx context.Context, r *http.Request, clientVersion ClientVersion) (*http.Request, error) {
	legacyLog.V(1).Info("Handling legacy request", "clientVersion", clientVersion)

	// Track client version
	lcs.metrics.clientVersions[clientVersion.String()]++

	// Transform the request body if needed
	if r.Body != nil && r.ContentLength > 0 {
		body, err := lcs.transformRequestBody(r, clientVersion)
		if err != nil {
			return nil, fmt.Errorf("transforming request body: %w", err)
		}
		r.Body = body
	}

	// Add compatibility headers
	lcs.addCompatibilityHeaders(r, clientVersion)

	// Handle known quirks
	if err := lcs.handleClientQuirks(r, clientVersion); err != nil {
		return nil, fmt.Errorf("handling client quirks: %w", err)
	}

	return r, nil
}

// AdaptResponse adapts a response for a legacy client
func (lcs *LegacyClientSupport) AdaptResponse(response interface{}, clientVersion ClientVersion) (interface{}, error) {
	legacyLog.V(2).Info("Adapting response for legacy client", "clientVersion", clientVersion)

	// Track transformation
	lcs.metrics.transformations["response"]++

	// Apply response transformations
	adapted := response
	for _, transformer := range lcs.responseAdapter.transformers {
		if transformer.CanTransform(adapted) {
			var err error
			adapted, err = transformer.Transform(adapted, clientVersion)
			if err != nil {
				return nil, fmt.Errorf("transforming response: %w", err)
			}
		}
	}

	// Mask unsupported features
	adapted = lcs.featureMasker.MaskFeatures(adapted, clientVersion)

	// Apply field mappings for legacy format
	if err := lcs.applyFieldMappings(adapted, clientVersion); err != nil {
		return nil, fmt.Errorf("applying field mappings: %w", err)
	}

	return adapted, nil
}

// TranslateError translates an error for a legacy client
func (lcs *LegacyClientSupport) TranslateError(err error, clientVersion ClientVersion) error {
	if err == nil {
		return nil
	}

	legacyLog.V(2).Info("Translating error for legacy client", "error", err, "clientVersion", clientVersion)
	
	// Track error
	lcs.metrics.errors[err.Error()]++

	return lcs.errorTranslator.Translate(err, clientVersion)
}

// Private helper methods

func (lcs *LegacyClientSupport) initializeClientDetector() {
	lcs.clientDetector = ClientDetector{
		patterns:     make(map[string]*regexp.Regexp),
		versionMap:   make(map[string]ClientVersion),
		capabilities: make(map[string][]string),
		quirksHandler: &DefaultQuirksHandler{},
	}

	// Initialize patterns for known clients
	lcs.clientDetector.patterns["kubectl"] = regexp.MustCompile(`kubectl/v?(\d+)\.(\d+)\.(\d+)`)
	lcs.clientDetector.patterns["client-go"] = regexp.MustCompile(`client-go/v?(\d+)\.(\d+)\.(\d+)`)
	lcs.clientDetector.patterns["openapi-gen"] = regexp.MustCompile(`openapi-gen/v?(\d+)\.(\d+)\.(\d+)`)
	
	// Define known client capabilities
	lcs.clientDetector.capabilities["kubectl/1.15"] = []string{"basic", "watch"}
	lcs.clientDetector.capabilities["kubectl/1.18"] = []string{"basic", "watch", "patch", "dry-run"}
	lcs.clientDetector.capabilities["kubectl/1.20"] = []string{"basic", "watch", "patch", "dry-run", "server-side-apply"}
}

func (lcs *LegacyClientSupport) initializeResponseAdapter() {
	lcs.responseAdapter = ResponseAdapter{
		transformers:   make(map[string]ResponseTransformer),
		fieldMappers:   make(map[string]FieldMapper),
		typeConverters: make(map[string]TypeConverter),
	}

	// Add default transformers
	lcs.responseAdapter.transformers["v1alpha1"] = &V1Alpha1Transformer{}
	lcs.responseAdapter.transformers["v1beta1"] = &V1Beta1Transformer{}
	
	// Add field mappers
	lcs.responseAdapter.fieldMappers["default"] = &DefaultFieldMapper{
		mappings: map[string]map[string]string{
			"v1beta1->v1alpha1": {
				"spec.components":          "spec.monitoring",
				"spec.components.prometheus": "spec.monitoring.prometheus",
				"spec.components.grafana":    "spec.monitoring.grafana",
			},
		},
	}
	
	// Add type converters
	lcs.responseAdapter.typeConverters["default"] = &DefaultTypeConverter{}
}

func (lcs *LegacyClientSupport) initializeRequestTransformer() {
	lcs.requestTransformer = RequestTransformer{
		validators:  make(map[string]RequestValidator),
		normalizers: make(map[string]RequestNormalizer),
		enrichers:   make(map[string]RequestEnricher),
	}

	// Add validators
	lcs.requestTransformer.validators["default"] = &DefaultRequestValidator{}
	
	// Add normalizers
	lcs.requestTransformer.normalizers["default"] = &DefaultRequestNormalizer{}
	
	// Add enrichers
	lcs.requestTransformer.enrichers["default"] = &DefaultRequestEnricher{
		requiredFields: map[string][]string{
			"v1alpha1": {"apiVersion", "kind", "metadata.name"},
			"v1beta1":  {"apiVersion", "kind", "metadata.name", "metadata.namespace"},
		},
	}
}

func (lcs *LegacyClientSupport) initializeFeatureMasker() {
	lcs.featureMasker = FeatureMasker{
		featureGates:   make(map[string]FeatureGateConfig),
		versionMatrix:  make(map[string][]string),
		degradationMap: make(map[string]DegradationStrategy),
	}

	// Define feature gates
	lcs.featureMasker.featureGates["multiCluster"] = FeatureGateConfig{
		Name:       "multiCluster",
		MinVersion: ClientVersion{Major: 1, Minor: 20},
		Enabled:    true,
		DegradationMode: "hide",
	}
	
	lcs.featureMasker.featureGates["costOptimization"] = FeatureGateConfig{
		Name:       "costOptimization",
		MinVersion: ClientVersion{Major: 1, Minor: 18},
		Enabled:    true,
		DegradationMode: "disable",
		AlternativeValue: map[string]interface{}{
			"enabled": false,
			"message": "Cost optimization requires newer client version",
		},
	}
	
	// Version feature matrix
	lcs.featureMasker.versionMatrix["v1.15"] = []string{"basic-monitoring", "alerts"}
	lcs.featureMasker.versionMatrix["v1.18"] = []string{"basic-monitoring", "alerts", "cost-optimization"}
	lcs.featureMasker.versionMatrix["v1.20"] = []string{"basic-monitoring", "alerts", "cost-optimization", "multi-cluster"}
}

func (lcs *LegacyClientSupport) initializeErrorTranslator() {
	lcs.errorTranslator = ErrorTranslator{
		errorMappings:  make(map[string]ErrorMapping),
		codeTranslator: &DefaultCodeTranslator{},
		messageAdapter: &DefaultMessageAdapter{},
	}

	// Define error mappings
	lcs.errorTranslator.errorMappings["FieldNotSupported"] = ErrorMapping{
		ModernCode: "FieldNotSupported",
		LegacyCode: "InvalidField",
		StatusCode: http.StatusBadRequest,
		Transform: func(err error) error {
			return fmt.Errorf("invalid field in request")
		},
	}
	
	lcs.errorTranslator.errorMappings["VersionNotSupported"] = ErrorMapping{
		ModernCode: "VersionNotSupported",
		LegacyCode: "BadRequest",
		StatusCode: http.StatusBadRequest,
		Transform: func(err error) error {
			return fmt.Errorf("invalid API version")
		},
	}
}

func (lcs *LegacyClientSupport) parseVersionString(version string) ClientVersion {
	// Parse version string like "v1.20.0" or "1.20.0-beta.1"
	re := regexp.MustCompile(`v?(\d+)\.(\d+)(?:\.(\d+))?(?:-(.+))?`)
	matches := re.FindStringSubmatch(version)
	
	if len(matches) < 3 {
		return ClientVersion{}
	}
	
	cv := ClientVersion{}
	cv.Major, _ = strconv.Atoi(matches[1])
	cv.Minor, _ = strconv.Atoi(matches[2])
	if len(matches) > 3 && matches[3] != "" {
		cv.Patch, _ = strconv.Atoi(matches[3])
	}
	if len(matches) > 4 {
		cv.PreRelease = matches[4]
	}
	
	return cv
}

func (lcs *LegacyClientSupport) transformRequestBody(r *http.Request, clientVersion ClientVersion) (io.ReadCloser, error) {
	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body.Close()
	
	// Parse JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Not JSON, return as-is
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	
	// Transform the data
	transformed := data
	
	// Validate
	if validator, ok := lcs.requestTransformer.validators["default"]; ok {
		if err := validator.Validate(transformed, clientVersion); err != nil {
			return nil, err
		}
	}
	
	// Normalize
	if normalizer, ok := lcs.requestTransformer.normalizers["default"]; ok {
		if normalizer.NeedsNormalization(transformed) {
			transformed, err = normalizer.Normalize(transformed)
			if err != nil {
				return nil, err
			}
		}
	}
	
	// Enrich
	if enricher, ok := lcs.requestTransformer.enrichers["default"]; ok {
		transformed, err = enricher.Enrich(transformed, clientVersion)
		if err != nil {
			return nil, err
		}
	}
	
	// Re-encode
	newBody, err := json.Marshal(transformed)
	if err != nil {
		return nil, err
	}
	
	// Update content length
	r.ContentLength = int64(len(newBody))
	r.Header.Set("Content-Length", strconv.FormatInt(r.ContentLength, 10))
	
	return io.NopCloser(bytes.NewReader(newBody)), nil
}

func (lcs *LegacyClientSupport) addCompatibilityHeaders(r *http.Request, clientVersion ClientVersion) {
	r.Header.Set("X-Legacy-Client", "true")
	r.Header.Set("X-Client-Version-Detected", clientVersion.String())
	r.Header.Set("X-API-Compatibility-Mode", "legacy")
}

func (lcs *LegacyClientSupport) handleClientQuirks(r *http.Request, clientVersion ClientVersion) error {
	// Handle known quirks for specific client versions
	for _, quirk := range clientVersion.Quirks {
		if err := lcs.clientDetector.quirksHandler.HandleQuirk(quirk, r, nil); err != nil {
			return err
		}
	}
	return nil
}

func (lcs *LegacyClientSupport) applyFieldMappings(data interface{}, clientVersion ClientVersion) error {
	// Apply field mappings based on client version
	mapper, ok := lcs.responseAdapter.fieldMappers["default"]
	if !ok {
		return nil
	}
	
	// Determine mapping direction
	mappingKey := fmt.Sprintf("v1beta1->%s", clientVersion.APIVersion())
	mappings := mapper.GetMapping("v1beta1", clientVersion.APIVersion())
	
	// Apply mappings
	for source, target := range mappings {
		if err := mapper.MapField(data, data, source); err != nil {
			legacyLog.V(2).Info("Failed to map field", "source", source, "target", target, "error", err)
		}
	}
	
	return nil
}

// Helper types and methods

// String returns string representation of ClientVersion
func (cv ClientVersion) String() string {
	version := fmt.Sprintf("%d.%d.%d", cv.Major, cv.Minor, cv.Patch)
	if cv.PreRelease != "" {
		version += "-" + cv.PreRelease
	}
	if cv.ClientType != "" {
		version = cv.ClientType + "/" + version
	}
	return version
}

// APIVersion returns the API version string for this client
func (cv ClientVersion) APIVersion() string {
	if cv.Major == 1 && cv.Minor < 18 {
		return "v1alpha1"
	}
	return "v1beta1"
}

// SupportsFeature checks if the client supports a feature
func (cv ClientVersion) SupportsFeature(feature string) bool {
	for _, capability := range cv.Capabilities {
		if capability == feature {
			return true
		}
	}
	return false
}

// Default implementations

// DefaultQuirksHandler handles known client quirks
type DefaultQuirksHandler struct{}

func (dqh *DefaultQuirksHandler) HandleQuirk(quirk string, request interface{}, response interface{}) error {
	switch quirk {
	case "kubectl-1.15-patch-bug":
		// Handle known kubectl 1.15 patch bug
		return nil
	case "missing-content-type":
		// Add missing content type
		if req, ok := request.(*http.Request); ok {
			if req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}
		}
		return nil
	default:
		return nil
	}
}

func (dqh *DefaultQuirksHandler) ApplyWorkaround(quirk string, data interface{}) interface{} {
	// Apply workarounds for known quirks
	return data
}

// V1Alpha1Transformer transforms responses for v1alpha1 clients
type V1Alpha1Transformer struct{}

func (t *V1Alpha1Transformer) Transform(response interface{}, clientVersion ClientVersion) (interface{}, error) {
	// Transform v1beta1 response to v1alpha1 format
	// This is a simplified example
	return response, nil
}

func (t *V1Alpha1Transformer) CanTransform(response interface{}) bool {
	// Check if response can be transformed
	return true
}

// V1Beta1Transformer transforms responses for v1beta1 clients
type V1Beta1Transformer struct{}

func (t *V1Beta1Transformer) Transform(response interface{}, clientVersion ClientVersion) (interface{}, error) {
	// No transformation needed for v1beta1
	return response, nil
}

func (t *V1Beta1Transformer) CanTransform(response interface{}) bool {
	return true
}

// DefaultFieldMapper maps fields between versions
type DefaultFieldMapper struct {
	mappings map[string]map[string]string
}

func (dfm *DefaultFieldMapper) MapField(source, target interface{}, fieldPath string) error {
	// Implement field mapping logic
	return nil
}

func (dfm *DefaultFieldMapper) GetMapping(sourceVersion, targetVersion string) map[string]string {
	key := fmt.Sprintf("%s->%s", sourceVersion, targetVersion)
	if mapping, ok := dfm.mappings[key]; ok {
		return mapping
	}
	return map[string]string{}
}

// DefaultTypeConverter converts types between versions
type DefaultTypeConverter struct{}

func (dtc *DefaultTypeConverter) Convert(value interface{}, fromType, toType string) (interface{}, error) {
	// Implement type conversion logic
	return value, nil
}

func (dtc *DefaultTypeConverter) IsConvertible(fromType, toType string) bool {
	// Check if types are convertible
	return true
}

// DefaultRequestValidator validates requests
type DefaultRequestValidator struct{}

func (drv *DefaultRequestValidator) Validate(request interface{}, clientVersion ClientVersion) error {
	// Implement validation logic
	return nil
}

func (drv *DefaultRequestValidator) GetValidationRules(clientVersion ClientVersion) []ValidationRule {
	// Return validation rules for the client version
	return []ValidationRule{}
}

// DefaultRequestNormalizer normalizes requests
type DefaultRequestNormalizer struct{}

func (drn *DefaultRequestNormalizer) Normalize(request interface{}) (interface{}, error) {
	// Implement normalization logic
	return request, nil
}

func (drn *DefaultRequestNormalizer) NeedsNormalization(request interface{}) bool {
	// Check if request needs normalization
	return false
}

// DefaultRequestEnricher enriches requests
type DefaultRequestEnricher struct {
	requiredFields map[string][]string
}

func (dre *DefaultRequestEnricher) Enrich(request interface{}, clientVersion ClientVersion) (interface{}, error) {
	// Add required fields if missing
	return request, nil
}

func (dre *DefaultRequestEnricher) GetRequiredFields(clientVersion ClientVersion) []string {
	apiVersion := clientVersion.APIVersion()
	if fields, ok := dre.requiredFields[apiVersion]; ok {
		return fields
	}
	return []string{}
}

// MaskFeatures masks unsupported features
func (fm *FeatureMasker) MaskFeatures(data interface{}, clientVersion ClientVersion) interface{} {
	// Mask features based on client capabilities
	for feature, config := range fm.featureGates {
		if !fm.isFeatureSupported(feature, clientVersion) {
			// Apply degradation strategy
			if strategy, ok := fm.degradationMap[config.DegradationMode]; ok && strategy.CanDegrade(feature) {
				data = strategy.Degrade(feature, data)
			}
		}
	}
	return data
}

func (fm *FeatureMasker) isFeatureSupported(feature string, clientVersion ClientVersion) bool {
	config, ok := fm.featureGates[feature]
	if !ok {
		return true // Unknown features are assumed supported
	}
	
	// Check version requirement
	if clientVersion.Major < config.MinVersion.Major ||
		(clientVersion.Major == config.MinVersion.Major && clientVersion.Minor < config.MinVersion.Minor) {
		return false
	}
	
	return config.Enabled
}

// DefaultCodeTranslator translates error codes
type DefaultCodeTranslator struct{}

func (dct *DefaultCodeTranslator) TranslateCode(modernCode string, clientVersion ClientVersion) string {
	// Translate modern error codes to legacy format
	return dct.GetLegacyCode(modernCode)
}

func (dct *DefaultCodeTranslator) GetLegacyCode(modernCode string) string {
	// Map modern codes to legacy codes
	switch modernCode {
	case "FieldNotSupported":
		return "InvalidField"
	case "VersionNotSupported":
		return "BadRequest"
	default:
		return modernCode
	}
}

// DefaultMessageAdapter adapts error messages
type DefaultMessageAdapter struct{}

func (dma *DefaultMessageAdapter) AdaptMessage(message string, clientVersion ClientVersion) string {
	// Simplify message for older clients
	if clientVersion.Major == 1 && clientVersion.Minor < 18 {
		return dma.SimplifyMessage(message)
	}
	return message
}

func (dma *DefaultMessageAdapter) SimplifyMessage(message string) string {
	// Remove technical details for older clients
	if idx := strings.Index(message, ":"); idx > 0 {
		return message[:idx]
	}
	return message
}

// Translate translates an error
func (et *ErrorTranslator) Translate(err error, clientVersion ClientVersion) error {
	errorStr := err.Error()
	
	// Find matching error mapping
	for _, mapping := range et.errorMappings {
		if strings.Contains(errorStr, mapping.ModernCode) {
			if mapping.Transform != nil {
				return mapping.Transform(err)
			}
			break
		}
	}
	
	// Adapt the message
	if et.messageAdapter != nil {
		errorStr = et.messageAdapter.AdaptMessage(errorStr, clientVersion)
	}
	
	return fmt.Errorf(errorStr)
}

// DetectFromUserAgent detects client version from user agent
func (cd *ClientDetector) DetectFromUserAgent(userAgent, acceptHeader string) ClientVersion {
	// Try each pattern
	for clientType, pattern := range cd.patterns {
		matches := pattern.FindStringSubmatch(userAgent)
		if len(matches) >= 4 {
			cv := ClientVersion{
				ClientType: clientType,
			}
			cv.Major, _ = strconv.Atoi(matches[1])
			cv.Minor, _ = strconv.Atoi(matches[2])
			cv.Patch, _ = strconv.Atoi(matches[3])
			
			// Add capabilities based on version
			versionKey := fmt.Sprintf("%s/%d.%d", clientType, cv.Major, cv.Minor)
			if caps, ok := cd.capabilities[versionKey]; ok {
				cv.Capabilities = caps
			}
			
			// Detect known quirks
			cv.Quirks = cd.detectQuirks(cv)
			
			return cv
		}
	}
	
	// Default to unknown old client
	return ClientVersion{
		Major:        1,
		Minor:        15,
		ClientType:   "unknown",
		Capabilities: []string{"basic"},
	}
}

func (cd *ClientDetector) detectQuirks(cv ClientVersion) []string {
	quirks := []string{}
	
	// Known quirks for specific versions
	if cv.ClientType == "kubectl" && cv.Major == 1 && cv.Minor == 15 {
		quirks = append(quirks, "kubectl-1.15-patch-bug")
	}
	
	// Missing content type in older clients
	if cv.Major == 1 && cv.Minor < 16 {
		quirks = append(quirks, "missing-content-type")
	}
	
	return quirks
}

// GetUsageMetrics returns legacy client usage metrics
func (lcs *LegacyClientSupport) GetUsageMetrics() map[string]interface{} {
	return map[string]interface{}{
		"clientVersions":  lcs.metrics.clientVersions,
		"featureUsage":    lcs.metrics.featureUsage,
		"transformations": lcs.metrics.transformations,
		"errors":          lcs.metrics.errors,
		"deprecationHits": lcs.metrics.deprecationHits,
	}
}
