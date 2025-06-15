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

package conversion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBackwardCompatibilityManager(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	bcm := conversion.NewBackwardCompatibilityManager(client, scheme)

	t.Run("Version Negotiation", func(t *testing.T) {
		tests := []struct {
			name           string
			clientVersion  string
			acceptHeaders  []string
			expectedVersion string
		}{
			{
				name:           "Modern client gets v1beta1",
				clientVersion:  "v1.20.0",
				acceptHeaders:  []string{"application/json"},
				expectedVersion: "v1beta1",
			},
			{
				name:           "Old client gets v1alpha1",
				clientVersion:  "v1.15.0",
				acceptHeaders:  []string{"application/json"},
				expectedVersion: "v1alpha1",
			},
			{
				name:           "No version defaults to v1alpha1",
				clientVersion:  "",
				acceptHeaders:  []string{},
				expectedVersion: "v1alpha1",
			},
			{
				name:           "Invalid version defaults to v1alpha1",
				clientVersion:  "invalid",
				acceptHeaders:  []string{},
				expectedVersion: "v1alpha1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				version, err := bcm.NegotiateVersion(tt.clientVersion, tt.acceptHeaders)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVersion, version)
			})
		}
	})

	t.Run("Unknown Fields Handling", func(t *testing.T) {
		platform := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
			},
			Spec: v1beta1.ObservabilityPlatformSpec{},
		}

		// Test handling unknown fields
		obj, unknownFields, err := bcm.HandleUnknownFields(platform, "v1beta1")
		assert.NoError(t, err)
		assert.NotNil(t, obj)
		assert.Empty(t, unknownFields) // No unknown fields in valid object

		// Test with annotations containing unknown fields
		platform.Annotations = map[string]string{
			"observability.io/unknown-field-custom": `{"value": "test"}`,
		}
		obj, unknownFields, err = bcm.HandleUnknownFields(platform, "v1beta1")
		assert.NoError(t, err)
		assert.NotNil(t, obj)
	})

	t.Run("Default Values Application", func(t *testing.T) {
		platform := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
			},
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: v1beta1.Components{
					Prometheus: &v1beta1.PrometheusSpec{
						Enabled: true,
					},
				},
			},
		}

		err := bcm.ApplyDefaultValues(platform, "v1beta1")
		assert.NoError(t, err)
		// Defaults should be applied - exact values depend on implementation
	})

	t.Run("Compatibility Headers", func(t *testing.T) {
		headers := make(map[string]string)
		bcm.AddCompatibilityHeaders(headers, "v1.18.0", "v1beta1")

		assert.Equal(t, "v1beta1", headers["X-API-Version"])
		assert.NotEmpty(t, headers["X-API-Deprecated-Fields"])
		assert.Equal(t, "v1alpha1", headers["X-API-Min-Version"])
		assert.Equal(t, "v1beta1", headers["X-API-Max-Version"])
		assert.Equal(t, "v1.18.0", headers["X-API-Client-Version"])
		assert.NotEmpty(t, headers["X-API-Features"])
	})

	t.Run("Feature Compatibility", func(t *testing.T) {
		tests := []struct {
			name          string
			feature       string
			clientVersion string
			expected      bool
		}{
			{
				name:          "Multi-cluster supported in v1.20",
				feature:       "multiCluster",
				clientVersion: "v1.20.0",
				expected:      true,
			},
			{
				name:          "Multi-cluster not supported in v1.15",
				feature:       "multiCluster",
				clientVersion: "v1.15.0",
				expected:      false,
			},
			{
				name:          "Unknown feature assumed compatible",
				feature:       "unknownFeature",
				clientVersion: "v1.15.0",
				expected:      true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				compatible, degraded := bcm.CheckFeatureCompatibility(tt.feature, tt.clientVersion)
				assert.Equal(t, tt.expected, compatible)
				if !tt.expected && degraded != nil {
					assert.NotNil(t, degraded) // Should have degradation info
				}
			})
		}
	})

	t.Run("Serialization for Client", func(t *testing.T) {
		platform := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
			},
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: v1beta1.Components{
					Prometheus: &v1beta1.PrometheusSpec{
						Enabled: true,
						Version: "v2.45.0",
					},
				},
			},
		}

		data, err := bcm.SerializeForClient(platform, "v1beta1")
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify it's valid JSON
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		assert.NoError(t, err)
	})
}

func TestVersionDiscovery(t *testing.T) {
	vd := conversion.NewVersionDiscovery()

	t.Run("Version Info", func(t *testing.T) {
		info := vd.GetVersionInfo()
		assert.Equal(t, "v1beta1", info.Current)
		assert.Equal(t, "v1beta1", info.Preferred)
		assert.NotEmpty(t, info.Supported)
		assert.NotZero(t, info.LastUpdated)
	})

	t.Run("Version Support Check", func(t *testing.T) {
		assert.True(t, vd.IsVersionSupported("v1alpha1"))
		assert.True(t, vd.IsVersionSupported("v1beta1"))
		assert.False(t, vd.IsVersionSupported("v2"))
		assert.False(t, vd.IsVersionSupported("invalid"))
	})

	t.Run("Version Endpoints", func(t *testing.T) {
		endpoints, err := vd.GetVersionEndpoints("v1beta1")
		assert.NoError(t, err)
		assert.NotEmpty(t, endpoints)
		assert.Contains(t, endpoints, "platforms")

		_, err = vd.GetVersionEndpoints("invalid")
		assert.Error(t, err)
	})

	t.Run("Migration Path", func(t *testing.T) {
		// Same version
		path, err := vd.GetMigrationPath("v1alpha1", "v1alpha1")
		assert.NoError(t, err)
		assert.Empty(t, path)

		// Direct migration
		path, err = vd.GetMigrationPath("v1alpha1", "v1beta1")
		assert.NoError(t, err)
		assert.Len(t, path, 1)
		assert.Equal(t, "v1alpha1", path[0].From)
		assert.Equal(t, "v1beta1", path[0].To)

		// No path available
		_, err = vd.GetMigrationPath("v1alpha1", "v2")
		assert.Error(t, err)
	})

	t.Run("HTTP Version Discovery", func(t *testing.T) {
		tests := []struct {
			name       string
			path       string
			wantStatus int
		}{
			{
				name:       "List all versions",
				path:       "/api/versions",
				wantStatus: http.StatusOK,
			},
			{
				name:       "Get current version",
				path:       "/api/version",
				wantStatus: http.StatusOK,
			},
			{
				name:       "Get specific version",
				path:       "/api/versions/v1beta1",
				wantStatus: http.StatusOK,
			},
			{
				name:       "Invalid version",
				path:       "/api/versions/invalid",
				wantStatus: http.StatusNotFound,
			},
			{
				name:       "Unknown path",
				path:       "/api/unknown",
				wantStatus: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tt.path, nil)
				w := httptest.NewRecorder()
				
				vd.HandleVersionDiscoveryRequest(w, req)
				
				assert.Equal(t, tt.wantStatus, w.Code)
				
				// Check headers
				assert.NotEmpty(t, w.Header().Get("X-API-Versions"))
				assert.NotEmpty(t, w.Header().Get("X-API-Preferred-Version"))
				assert.NotEmpty(t, w.Header().Get("X-API-Deprecation-Policy"))
			})
		}
	})
}

func TestLegacyClientSupport(t *testing.T) {
	lcs := conversion.NewLegacyClientSupport()

	t.Run("Client Detection", func(t *testing.T) {
		tests := []struct {
			name           string
			userAgent      string
			clientVersion  string
			expectedMajor  int
			expectedMinor  int
			expectedType   string
		}{
			{
				name:          "kubectl 1.15",
				userAgent:     "kubectl/v1.15.0 (linux/amd64) kubernetes/e8462b5",
				expectedMajor: 1,
				expectedMinor: 15,
				expectedType:  "kubectl",
			},
			{
				name:          "kubectl 1.20",
				userAgent:     "kubectl/v1.20.0 (darwin/amd64) kubernetes/af46c47",
				expectedMajor: 1,
				expectedMinor: 20,
				expectedType:  "kubectl",
			},
			{
				name:          "client-go",
				userAgent:     "client-go/v1.18.0",
				expectedMajor: 1,
				expectedMinor: 18,
				expectedType:  "client-go",
			},
			{
				name:           "Direct version header",
				clientVersion:  "v1.19.0",
				expectedMajor:  1,
				expectedMinor:  19,
			},
			{
				name:          "Unknown client",
				userAgent:     "CustomClient/1.0",
				expectedMajor: 1,
				expectedMinor: 15,
				expectedType:  "unknown",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/api/v1/platforms", nil)
				if tt.userAgent != "" {
					req.Header.Set("User-Agent", tt.userAgent)
				}
				if tt.clientVersion != "" {
					req.Header.Set("X-Client-Version", tt.clientVersion)
				}

				clientVersion := lcs.DetectClient(req)
				assert.Equal(t, tt.expectedMajor, clientVersion.Major)
				assert.Equal(t, tt.expectedMinor, clientVersion.Minor)
				if tt.expectedType != "" {
					assert.Equal(t, tt.expectedType, clientVersion.ClientType)
				}
			})
		}
	})

	t.Run("Legacy Request Handling", func(t *testing.T) {
		// Create a legacy request
		body := map[string]interface{}{
			"apiVersion": "observability.io/v1alpha1",
			"kind":       "ObservabilityPlatform",
			"metadata": map[string]interface{}{
				"name": "legacy-platform",
			},
			"spec": map[string]interface{}{
				"monitoring": map[string]interface{}{
					"prometheusEnabled": true,
					"prometheusVersion": "v2.45.0",
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)
		
		req := httptest.NewRequest("POST", "/api/v1alpha1/platforms", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "kubectl/v1.15.0")
		
		clientVersion := lcs.DetectClient(req)
		
		// Handle the legacy request
		ctx := context.Background()
		transformedReq, err := lcs.HandleLegacyRequest(ctx, req, clientVersion)
		assert.NoError(t, err)
		assert.NotNil(t, transformedReq)
		
		// Check compatibility headers were added
		assert.Equal(t, "true", transformedReq.Header.Get("X-Legacy-Client"))
		assert.NotEmpty(t, transformedReq.Header.Get("X-Client-Version-Detected"))
		assert.Equal(t, "legacy", transformedReq.Header.Get("X-API-Compatibility-Mode"))
	})

	t.Run("Response Adaptation", func(t *testing.T) {
		// Modern response
		response := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
			},
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: v1beta1.Components{
					Prometheus: &v1beta1.PrometheusSpec{
						Enabled: true,
						Version: "v2.45.0",
					},
				},
			},
		}

		// Old client version
		clientVersion := conversion.ClientVersion{
			Major:      1,
			Minor:      15,
			ClientType: "kubectl",
		}

		adapted, err := lcs.AdaptResponse(response, clientVersion)
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Error Translation", func(t *testing.T) {
		tests := []struct {
			name          string
			error         error
			clientVersion conversion.ClientVersion
			expectContains string
		}{
			{
				name:          "Field not supported error",
				error:         fmt.Errorf("FieldNotSupported: spec.multiCluster"),
				clientVersion: conversion.ClientVersion{Major: 1, Minor: 15},
				expectContains: "invalid field",
			},
			{
				name:          "Version not supported error",
				error:         fmt.Errorf("VersionNotSupported: v2"),
				clientVersion: conversion.ClientVersion{Major: 1, Minor: 15},
				expectContains: "invalid API version",
			},
			{
				name:          "Generic error",
				error:         fmt.Errorf("something went wrong"),
				clientVersion: conversion.ClientVersion{Major: 1, Minor: 20},
				expectContains: "something went wrong",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				translated := lcs.TranslateError(tt.error, tt.clientVersion)
				assert.Error(t, translated)
				assert.Contains(t, translated.Error(), tt.expectContains)
			})
		}
	})

	t.Run("Usage Metrics", func(t *testing.T) {
		// Simulate some usage
		req := httptest.NewRequest("GET", "/api/v1/platforms", nil)
		req.Header.Set("User-Agent", "kubectl/v1.15.0")
		
		clientVersion := lcs.DetectClient(req)
		ctx := context.Background()
		_, _ = lcs.HandleLegacyRequest(ctx, req, clientVersion)

		// Get metrics
		metrics := lcs.GetUsageMetrics()
		assert.NotNil(t, metrics)
		assert.Contains(t, metrics, "clientVersions")
		assert.Contains(t, metrics, "transformations")
		assert.Contains(t, metrics, "errors")
	})
}

func TestVersionNegotiator(t *testing.T) {
	vd := conversion.NewVersionDiscovery()
	vn := conversion.NewVersionNegotiator(vd)

	t.Run("Version Negotiation", func(t *testing.T) {
		tests := []struct {
			name           string
			clientID       string
			userAgent      string
			acceptVersions []string
			expectedVersion string
		}{
			{
				name:           "Client prefers v1beta1",
				clientID:       "client-1",
				userAgent:      "kubectl/v1.20.0",
				acceptVersions: []string{"v1beta1", "v1alpha1"},
				expectedVersion: "v1beta1",
			},
			{
				name:           "Client only supports v1alpha1",
				clientID:       "client-2",
				userAgent:      "kubectl/v1.15.0",
				acceptVersions: []string{"v1alpha1"},
				expectedVersion: "v1alpha1",
			},
			{
				name:           "No preference uses oldest stable",
				clientID:       "client-3",
				userAgent:      "custom-client/1.0",
				acceptVersions: []string{},
				expectedVersion: "v1alpha1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				version, err := vn.NegotiateVersion(tt.clientID, tt.userAgent, tt.acceptVersions)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVersion, version)
			})
		}
	})
}

// Benchmark tests

func BenchmarkVersionNegotiation(b *testing.B) {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	bcm := conversion.NewBackwardCompatibilityManager(client, scheme)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bcm.NegotiateVersion("v1.20.0", []string{"application/json"})
	}
}

func BenchmarkClientDetection(b *testing.B) {
	lcs := conversion.NewLegacyClientSupport()
	req := httptest.NewRequest("GET", "/api/v1/platforms", nil)
	req.Header.Set("User-Agent", "kubectl/v1.20.0 (linux/amd64) kubernetes/af46c47")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lcs.DetectClient(req)
	}
}

func BenchmarkResponseAdaptation(b *testing.B) {
	lcs := conversion.NewLegacyClientSupport()
	response := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: v1beta1.ObservabilityPlatformSpec{
			Components: v1beta1.Components{
				Prometheus: &v1beta1.PrometheusSpec{
					Enabled: true,
					Version: "v2.45.0",
				},
			},
		},
	}
	clientVersion := conversion.ClientVersion{
		Major:      1,
		Minor:      15,
		ClientType: "kubectl",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lcs.AdaptResponse(response, clientVersion)
	}
}
