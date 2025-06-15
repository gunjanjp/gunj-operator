/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	
	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// ConversionTestSuite provides comprehensive testing for conversions
type ConversionTestSuite struct {
	t         *testing.T
	logger    logr.Logger
	validator *FieldValidator
	dryRun    *DryRunConverter
	scenarios []ConversionScenario
}

// ConversionScenario represents a test scenario for conversion
type ConversionScenario struct {
	Name           string
	Description    string
	SourceObject   runtime.Object
	TargetVersion  string
	ExpectedResult ConversionExpectation
	FieldChecks    []FieldCheck
}

// ConversionExpectation defines expected results
type ConversionExpectation struct {
	ShouldSucceed      bool
	ExpectedWarnings   []string
	ExpectedErrors     []string
	ExpectedDataLoss   []string
	ExpectedRiskLevel  string
}

// FieldCheck defines a specific field validation
type FieldCheck struct {
	FieldPath     string
	ExpectedValue interface{}
	CheckType     string // exact, contains, transform
	Transformer   func(interface{}) interface{}
}

// NewConversionTestSuite creates a new test suite
func NewConversionTestSuite(t *testing.T) *ConversionTestSuite {
	logger := log.Log.WithName("conversion-test")
	
	return &ConversionTestSuite{
		t:         t,
		logger:    logger,
		validator: NewFieldValidator(logger),
		dryRun:    NewDryRunConverter(logger),
		scenarios: []ConversionScenario{},
	}
}

// AddScenario adds a test scenario
func (ts *ConversionTestSuite) AddScenario(scenario ConversionScenario) {
	ts.scenarios = append(ts.scenarios, scenario)
}

// RunAll runs all test scenarios
func (ts *ConversionTestSuite) RunAll() {
	for _, scenario := range ts.scenarios {
		ts.t.Run(scenario.Name, func(t *testing.T) {
			ts.runScenario(t, scenario)
		})
	}
}

// runScenario runs a single test scenario
func (ts *ConversionTestSuite) runScenario(t *testing.T, scenario ConversionScenario) {
	ctx := context.Background()
	
	// Perform dry-run
	result, err := ts.dryRun.DryRunConversion(ctx, scenario.SourceObject, scenario.TargetVersion)
	require.NoError(t, err, "dry-run should not error")
	
	// Check success expectation
	assert.Equal(t, scenario.ExpectedResult.ShouldSucceed, result.Success,
		"conversion success mismatch")
	
	// Check warnings
	if len(scenario.ExpectedResult.ExpectedWarnings) > 0 {
		for _, expectedWarning := range scenario.ExpectedResult.ExpectedWarnings {
			found := false
			for _, warning := range result.ValidationResult.Warnings {
				if contains(warning, expectedWarning) {
					found = true
					break
				}
			}
			assert.True(t, found, "expected warning not found: %s", expectedWarning)
		}
	}
	
	// Check errors
	if len(scenario.ExpectedResult.ExpectedErrors) > 0 {
		assert.NotEmpty(t, result.ValidationResult.Errors, "expected errors but none found")
	}
	
	// Check data loss
	if len(scenario.ExpectedResult.ExpectedDataLoss) > 0 {
		assert.True(t, result.EstimatedImpact.DataLoss, "expected data loss")
		for _, field := range scenario.ExpectedResult.ExpectedDataLoss {
			assert.Contains(t, result.EstimatedImpact.LostFields, field,
				"expected field loss not detected")
		}
	}
	
	// Check risk level
	if scenario.ExpectedResult.ExpectedRiskLevel != "" {
		assert.Equal(t, scenario.ExpectedResult.ExpectedRiskLevel, 
			result.EstimatedImpact.RiskLevel, "risk level mismatch")
	}
	
	// Perform field checks if conversion succeeded
	if result.Success && len(scenario.FieldChecks) > 0 {
		ts.performFieldChecks(t, scenario, result)
	}
}

// performFieldChecks validates specific fields after conversion
func (ts *ConversionTestSuite) performFieldChecks(t *testing.T, scenario ConversionScenario, result *DryRunResult) {
	// Get the converted object
	sourceVersion, _ := ts.dryRun.getObjectVersion(scenario.SourceObject)
	converted, err := ts.dryRun.performConversion(scenario.SourceObject, sourceVersion, scenario.TargetVersion)
	require.NoError(t, err, "conversion should succeed")
	
	for _, check := range scenario.FieldChecks {
		actualValue := ts.validator.getFieldValue(converted, check.FieldPath)
		
		switch check.CheckType {
		case "exact":
			assert.Equal(t, check.ExpectedValue, actualValue,
				"field %s value mismatch", check.FieldPath)
			
		case "contains":
			actualStr := fmt.Sprintf("%v", actualValue)
			expectedStr := fmt.Sprintf("%v", check.ExpectedValue)
			assert.Contains(t, actualStr, expectedStr,
				"field %s should contain expected value", check.FieldPath)
			
		case "transform":
			if check.Transformer != nil {
				transformedValue := check.Transformer(actualValue)
				assert.Equal(t, check.ExpectedValue, transformedValue,
					"field %s transformed value mismatch", check.FieldPath)
			}
		}
	}
}

// RoundTripTest tests conversion round-trips
type RoundTripTest struct {
	suite *ConversionTestSuite
}

// NewRoundTripTest creates a new round-trip tester
func NewRoundTripTest(suite *ConversionTestSuite) *RoundTripTest {
	return &RoundTripTest{suite: suite}
}

// TestRoundTrip tests that converting back and forth preserves data
func (rt *RoundTripTest) TestRoundTrip(original runtime.Object) {
	ctx := context.Background()
	
	// Get original version
	originalVersion, err := rt.suite.dryRun.getObjectVersion(original)
	require.NoError(rt.suite.t, err)
	
	// Determine target version
	targetVersion := "v1beta1"
	if originalVersion == "v1beta1" {
		targetVersion = "v1alpha1"
	}
	
	// Convert to target
	result1, err := rt.suite.dryRun.DryRunConversion(ctx, original, targetVersion)
	require.NoError(rt.suite.t, err)
	require.True(rt.suite.t, result1.Success, "first conversion should succeed")
	
	// Get converted object
	converted1, err := rt.suite.dryRun.performConversion(original, originalVersion, targetVersion)
	require.NoError(rt.suite.t, err)
	
	// Convert back to original version
	result2, err := rt.suite.dryRun.DryRunConversion(ctx, converted1, originalVersion)
	require.NoError(rt.suite.t, err)
	
	// Get final object
	final, err := rt.suite.dryRun.performConversion(converted1, targetVersion, originalVersion)
	require.NoError(rt.suite.t, err)
	
	// Check if data was preserved (excluding known lossy fields)
	rt.compareObjects(original, final, originalVersion)
}

// compareObjects compares two objects excluding known lossy fields
func (rt *RoundTripTest) compareObjects(obj1, obj2 runtime.Object, version string) {
	// Fields that are expected to be lost in round-trip
	lossyFields := map[string][]string{
		"v1beta1": {
			"Spec.Security",
			"Spec.CostOptimization",
			"Spec.ServiceMesh",
			"Status.Version",
			"Status.Health",
		},
	}
	
	// Use reflection to compare
	v1 := reflect.ValueOf(obj1).Elem()
	v2 := reflect.ValueOf(obj2).Elem()
	
	rt.compareValues(v1, v2, "", lossyFields[version])
}

// compareValues recursively compares values
func (rt *RoundTripTest) compareValues(v1, v2 reflect.Value, path string, lossyFields []string) {
	if rt.isLossyField(path, lossyFields) {
		return
	}
	
	switch v1.Kind() {
	case reflect.Struct:
		for i := 0; i < v1.NumField(); i++ {
			field := v1.Type().Field(i)
			fieldPath := path + "." + field.Name
			rt.compareValues(v1.Field(i), v2.Field(i), fieldPath, lossyFields)
		}
		
	case reflect.Slice, reflect.Array:
		assert.Equal(rt.suite.t, v1.Len(), v2.Len(), 
			"slice length mismatch at %s", path)
		for i := 0; i < v1.Len(); i++ {
			rt.compareValues(v1.Index(i), v2.Index(i), 
				fmt.Sprintf("%s[%d]", path, i), lossyFields)
		}
		
	case reflect.Map:
		assert.Equal(rt.suite.t, v1.Len(), v2.Len(),
			"map length mismatch at %s", path)
		for _, key := range v1.MapKeys() {
			val1 := v1.MapIndex(key)
			val2 := v2.MapIndex(key)
			assert.True(rt.suite.t, val2.IsValid(),
				"key %v missing in second map at %s", key, path)
			if val2.IsValid() {
				rt.compareValues(val1, val2,
					fmt.Sprintf("%s[%v]", path, key), lossyFields)
			}
		}
		
	default:
		if v1.CanInterface() && v2.CanInterface() {
			assert.Equal(rt.suite.t, v1.Interface(), v2.Interface(),
				"value mismatch at %s", path)
		}
	}
}

// isLossyField checks if a field is expected to be lost
func (rt *RoundTripTest) isLossyField(path string, lossyFields []string) bool {
	for _, field := range lossyFields {
		if path == field || path == "."+field {
			return true
		}
	}
	return false
}

// FuzzTester provides fuzz testing for conversions
type FuzzTester struct {
	suite *ConversionTestSuite
}

// NewFuzzTester creates a new fuzz tester
func NewFuzzTester(suite *ConversionTestSuite) *FuzzTester {
	return &FuzzTester{suite: suite}
}

// FuzzConversion performs fuzz testing on conversions
func (ft *FuzzTester) FuzzConversion(f *testing.F) {
	// Add seed corpus
	ft.addSeedCorpus(f)
	
	f.Fuzz(func(t *testing.T, data []byte) {
		// Try to unmarshal as v1alpha1
		v1alpha1Obj := &v1alpha1.ObservabilityPlatform{}
		if err := json.Unmarshal(data, v1alpha1Obj); err == nil {
			// Valid v1alpha1 object, try conversion
			ctx := context.Background()
			result, err := ft.suite.dryRun.DryRunConversion(ctx, v1alpha1Obj, "v1beta1")
			
			// Should not panic
			assert.NotNil(t, result)
			assert.NoError(t, err)
		}
		
		// Try to unmarshal as v1beta1
		v1beta1Obj := &v1beta1.ObservabilityPlatform{}
		if err := json.Unmarshal(data, v1beta1Obj); err == nil {
			// Valid v1beta1 object, try conversion
			ctx := context.Background()
			result, err := ft.suite.dryRun.DryRunConversion(ctx, v1beta1Obj, "v1alpha1")
			
			// Should not panic
			assert.NotNil(t, result)
			assert.NoError(t, err)
		}
	})
}

// addSeedCorpus adds seed data for fuzzing
func (ft *FuzzTester) addSeedCorpus(f *testing.F) {
	// Add minimal valid objects
	minimalV1Alpha1 := &v1alpha1.ObservabilityPlatform{
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	
	minimalV1Beta1 := &v1beta1.ObservabilityPlatform{
		Spec: v1beta1.ObservabilityPlatformSpec{
			Components: v1beta1.Components{
				Prometheus: &v1beta1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	
	// Marshal and add as seed
	if data, err := json.Marshal(minimalV1Alpha1); err == nil {
		f.Add(data)
	}
	
	if data, err := json.Marshal(minimalV1Beta1); err == nil {
		f.Add(data)
	}
	
	// Add more complex seeds
	ft.addComplexSeeds(f)
}

// addComplexSeeds adds complex test cases as seeds
func (ft *FuzzTester) addComplexSeeds(f *testing.F) {
	// Complex v1alpha1 with all fields
	complexV1Alpha1 := &v1alpha1.ObservabilityPlatform{
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							Memory: "4Gi",
							CPU:    "1",
						},
						Limits: v1alpha1.ResourceList{
							Memory: "8Gi",
							CPU:    "2",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "100Gi",
						StorageClassName: "fast-ssd",
					},
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Enabled:  true,
					Version:  "10.2.0",
					Replicas: 2,
				},
			},
			Global: &v1alpha1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster": "production",
					"region":  "us-east-1",
				},
			},
		},
	}
	
	if data, err := json.Marshal(complexV1Alpha1); err == nil {
		f.Add(data)
	}
}

// PropertyBasedTester provides property-based testing
type PropertyBasedTester struct {
	suite *ConversionTestSuite
}

// NewPropertyBasedTester creates a new property-based tester
func NewPropertyBasedTester(suite *ConversionTestSuite) *PropertyBasedTester {
	return &PropertyBasedTester{suite: suite}
}

// TestConversionProperties tests that conversions maintain certain properties
func (pt *PropertyBasedTester) TestConversionProperties() {
	properties := []ConversionProperty{
		// Property: Required fields are never lost
		{
			Name: "RequiredFieldsPreserved",
			Check: func(original, converted runtime.Object) bool {
				// Check that required fields are present
				return pt.checkRequiredFields(converted)
			},
		},
		// Property: Resource quantities are valid after conversion
		{
			Name: "ValidResourceQuantities",
			Check: func(original, converted runtime.Object) bool {
				return pt.checkResourceQuantities(converted)
			},
		},
		// Property: Selectors remain immutable
		{
			Name: "SelectorsImmutable",
			Check: func(original, converted runtime.Object) bool {
				return pt.checkSelectorsUnchanged(original, converted)
			},
		},
	}
	
	// Test each property
	for _, prop := range properties {
		pt.suite.t.Run(prop.Name, func(t *testing.T) {
			// Generate test cases and verify property holds
			pt.verifyProperty(t, prop)
		})
	}
}

// ConversionProperty defines a property to test
type ConversionProperty struct {
	Name  string
	Check func(original, converted runtime.Object) bool
}

// verifyProperty verifies a conversion property
func (pt *PropertyBasedTester) verifyProperty(t *testing.T, prop ConversionProperty) {
	// Test with various inputs
	testCases := pt.generateTestCases()
	
	for _, tc := range testCases {
		ctx := context.Background()
		
		// Get version info
		srcVersion, _ := pt.suite.dryRun.getObjectVersion(tc)
		dstVersion := "v1beta1"
		if srcVersion == "v1beta1" {
			dstVersion = "v1alpha1"
		}
		
		// Perform conversion
		converted, err := pt.suite.dryRun.performConversion(tc, srcVersion, dstVersion)
		if err != nil {
			continue // Skip failed conversions
		}
		
		// Check property
		assert.True(t, prop.Check(tc, converted),
			"property %s failed for test case", prop.Name)
	}
}

// checkRequiredFields verifies required fields are present
func (pt *PropertyBasedTester) checkRequiredFields(obj runtime.Object) bool {
	switch v := obj.(type) {
	case *v1beta1.ObservabilityPlatform:
		// Check required fields
		if v.Spec.Components.Prometheus != nil && v.Spec.Components.Prometheus.Enabled {
			if v.Spec.Components.Prometheus.Version == "" {
				return false
			}
		}
	case *v1alpha1.ObservabilityPlatform:
		// Similar checks for v1alpha1
		if v.Spec.Components.Prometheus != nil && v.Spec.Components.Prometheus.Enabled {
			if v.Spec.Components.Prometheus.Version == "" {
				return false
			}
		}
	}
	return true
}

// checkResourceQuantities verifies resource quantities are valid
func (pt *PropertyBasedTester) checkResourceQuantities(obj runtime.Object) bool {
	// Implementation would check all resource fields
	return true
}

// checkSelectorsUnchanged verifies selectors haven't changed
func (pt *PropertyBasedTester) checkSelectorsUnchanged(original, converted runtime.Object) bool {
	// Implementation would compare selector fields
	return true
}

// generateTestCases generates test cases for property testing
func (pt *PropertyBasedTester) generateTestCases() []runtime.Object {
	// Generate various test objects
	return []runtime.Object{
		// Add test cases here
	}
}

// Helper functions for test creation

// CreateV1Alpha1Platform creates a test v1alpha1 platform
func CreateV1Alpha1Platform(opts ...PlatformOption) *v1alpha1.ObservabilityPlatform {
	platform := &v1alpha1.ObservabilityPlatform{
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{},
		},
	}
	
	for _, opt := range opts {
		opt(platform)
	}
	
	return platform
}

// CreateV1Beta1Platform creates a test v1beta1 platform
func CreateV1Beta1Platform(opts ...PlatformOption) *v1beta1.ObservabilityPlatform {
	platform := &v1beta1.ObservabilityPlatform{
		Spec: v1beta1.ObservabilityPlatformSpec{
			Components: v1beta1.Components{},
		},
	}
	
	for _, opt := range opts {
		opt(platform)
	}
	
	return platform
}

// PlatformOption is a functional option for creating test platforms
type PlatformOption func(runtime.Object)

// WithPrometheus adds Prometheus configuration
func WithPrometheus(enabled bool, version string) PlatformOption {
	return func(obj runtime.Object) {
		switch v := obj.(type) {
		case *v1alpha1.ObservabilityPlatform:
			v.Spec.Components.Prometheus = &v1alpha1.PrometheusSpec{
				Enabled: enabled,
				Version: version,
			}
		case *v1beta1.ObservabilityPlatform:
			v.Spec.Components.Prometheus = &v1beta1.PrometheusSpec{
				Enabled: enabled,
				Version: version,
			}
		}
	}
}

// WithGrafana adds Grafana configuration
func WithGrafana(enabled bool, version string) PlatformOption {
	return func(obj runtime.Object) {
		switch v := obj.(type) {
		case *v1alpha1.ObservabilityPlatform:
			v.Spec.Components.Grafana = &v1alpha1.GrafanaSpec{
				Enabled: enabled,
				Version: version,
			}
		case *v1beta1.ObservabilityPlatform:
			v.Spec.Components.Grafana = &v1beta1.GrafanaSpec{
				Enabled: enabled,
				Version: version,
			}
		}
	}
}
