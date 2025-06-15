/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package webhooks_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/webhooks"
)

func TestConversionWebhook(t *testing.T) {
	// Create scheme
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	// Create fake client
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create fake manager
	mgr := &fakeManager{
		client: client,
		scheme: scheme,
		logger: log.Log,
	}

	// Create conversion webhook
	webhook, err := webhooks.NewConversionWebhook(mgr, log.Log)
	require.NoError(t, err)

	tests := []struct {
		name               string
		request            *apiextensionsv1.ConversionReview
		expectedSuccess    bool
		expectedObjectsLen int
		validateResult     func(t *testing.T, response *apiextensionsv1.ConversionResponse)
	}{
		{
			name: "convert v1alpha1 to v1beta1",
			request: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               "test-uid-1",
					DesiredAPIVersion: "observability.io/v1beta1",
					Objects: []runtime.RawExtension{
						{
							Raw: mustMarshal(t, &v1alpha1.ObservabilityPlatform{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "observability.io/v1alpha1",
									Kind:       "ObservabilityPlatform",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-platform",
									Namespace: "default",
								},
								Spec: v1alpha1.ObservabilityPlatformSpec{
									Components: v1alpha1.Components{
										Prometheus: &v1alpha1.PrometheusSpec{
											Enabled:  true,
											Version:  "v2.48.0",
											Replicas: 3,
											Storage: &v1alpha1.StorageConfig{
												Size:             "100Gi",
												StorageClassName: "fast-ssd",
											},
										},
									},
								},
							}),
						},
					},
				},
			},
			expectedSuccess:    true,
			expectedObjectsLen: 1,
			validateResult: func(t *testing.T, response *apiextensionsv1.ConversionResponse) {
				require.Len(t, response.ConvertedObjects, 1)
				
				// Unmarshal the converted object
				var converted v1beta1.ObservabilityPlatform
				err := json.Unmarshal(response.ConvertedObjects[0].Raw, &converted)
				require.NoError(t, err)
				
				// Validate conversion
				assert.Equal(t, "observability.io/v1beta1", converted.APIVersion)
				assert.Equal(t, "ObservabilityPlatform", converted.Kind)
				assert.Equal(t, "test-platform", converted.Name)
				assert.Equal(t, "default", converted.Namespace)
				
				// Check Prometheus conversion
				require.NotNil(t, converted.Spec.Components)
				require.NotNil(t, converted.Spec.Components.Prometheus)
				assert.True(t, converted.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, "v2.48.0", converted.Spec.Components.Prometheus.Version)
				assert.Equal(t, int32(3), converted.Spec.Components.Prometheus.Replicas)
				
				// Check annotations
				assert.Equal(t, "v1alpha1", converted.Annotations["observability.io/converted-from"])
			},
		},
		{
			name: "convert v1beta1 to v1alpha1",
			request: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               "test-uid-2",
					DesiredAPIVersion: "observability.io/v1alpha1",
					Objects: []runtime.RawExtension{
						{
							Raw: mustMarshal(t, &v1beta1.ObservabilityPlatform{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "observability.io/v1beta1",
									Kind:       "ObservabilityPlatform",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-platform",
									Namespace: "default",
								},
								Spec: v1beta1.ObservabilityPlatformSpec{
									Components: &v1beta1.Components{
										Prometheus: &v1beta1.PrometheusSpec{
											Enabled:  true,
											Version:  "v2.48.0",
											Replicas: 3,
											// These fields will be lost in conversion
											ExternalLabels: map[string]string{
												"cluster": "production",
											},
											AdditionalScrapeConfigs: "- job_name: custom",
										},
										Grafana: &v1beta1.GrafanaSpec{
											Enabled: true,
											Version: "10.2.0",
											// This will be lost
											Plugins: []string{"piechart-panel"},
										},
									},
									// Security will be lost in conversion
									Security: &v1beta1.SecurityConfig{
										TLS: v1beta1.TLSConfig{
											Enabled: true,
										},
									},
								},
							}),
						},
					},
				},
			},
			expectedSuccess:    true,
			expectedObjectsLen: 1,
			validateResult: func(t *testing.T, response *apiextensionsv1.ConversionResponse) {
				require.Len(t, response.ConvertedObjects, 1)
				
				// Unmarshal the converted object
				var converted v1alpha1.ObservabilityPlatform
				err := json.Unmarshal(response.ConvertedObjects[0].Raw, &converted)
				require.NoError(t, err)
				
				// Validate conversion
				assert.Equal(t, "observability.io/v1alpha1", converted.APIVersion)
				assert.Equal(t, "ObservabilityPlatform", converted.Kind)
				assert.Equal(t, "test-platform", converted.Name)
				assert.Equal(t, "default", converted.Namespace)
				
				// Check basic fields are preserved
				require.NotNil(t, converted.Spec.Components.Prometheus)
				assert.True(t, converted.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, "v2.48.0", converted.Spec.Components.Prometheus.Version)
				assert.Equal(t, int32(3), converted.Spec.Components.Prometheus.Replicas)
				
				require.NotNil(t, converted.Spec.Components.Grafana)
				assert.True(t, converted.Spec.Components.Grafana.Enabled)
				assert.Equal(t, "10.2.0", converted.Spec.Components.Grafana.Version)
				
				// Check annotations
				assert.Equal(t, "v1beta1", converted.Annotations["observability.io/converted-from"])
			},
		},
		{
			name: "same version returns as-is",
			request: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               "test-uid-3",
					DesiredAPIVersion: "observability.io/v1beta1",
					Objects: []runtime.RawExtension{
						{
							Raw: mustMarshal(t, &v1beta1.ObservabilityPlatform{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "observability.io/v1beta1",
									Kind:       "ObservabilityPlatform",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name:      "test-platform",
									Namespace: "default",
								},
							}),
						},
					},
				},
			},
			expectedSuccess:    true,
			expectedObjectsLen: 1,
		},
		{
			name: "invalid target version",
			request: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               "test-uid-4",
					DesiredAPIVersion: "observability.io/v2",
					Objects: []runtime.RawExtension{
						{
							Raw: mustMarshal(t, &v1alpha1.ObservabilityPlatform{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "observability.io/v1alpha1",
									Kind:       "ObservabilityPlatform",
								},
							}),
						},
					},
				},
			},
			expectedSuccess: false,
		},
		{
			name: "nil request",
			request: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: nil,
			},
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal request
			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Handle the request
			webhook.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, http.StatusOK, rr.Code)

			// Unmarshal response
			var response apiextensionsv1.ConversionReview
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			// Validate response
			require.NotNil(t, response.Response)
			assert.Equal(t, tt.request.Request.UID, response.Response.UID)

			if tt.expectedSuccess {
				assert.Equal(t, metav1.StatusSuccess, response.Response.Result.Status)
				assert.Len(t, response.Response.ConvertedObjects, tt.expectedObjectsLen)
				
				if tt.validateResult != nil {
					tt.validateResult(t, response.Response)
				}
			} else {
				assert.Equal(t, metav1.StatusFailure, response.Response.Result.Status)
				assert.NotEmpty(t, response.Response.Result.Message)
			}
		})
	}
}

func TestConversionWebhook_ComplexObjects(t *testing.T) {
	// Create scheme
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	// Create fake client
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create fake manager
	mgr := &fakeManager{
		client: client,
		scheme: scheme,
		logger: log.Log,
	}

	// Create conversion webhook
	webhook, err := webhooks.NewConversionWebhook(mgr, log.Log)
	require.NoError(t, err)

	// Test complex object with all fields
	v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "observability.io/v1alpha1",
			Kind:       "ObservabilityPlatform",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex-platform",
			Namespace: "monitoring",
			Labels: map[string]string{
				"environment": "production",
				"team":        "platform",
			},
			Annotations: map[string]string{
				"description": "Production observability platform",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Paused: true,
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							Memory: "4Gi",
							CPU:    "2",
						},
						Limits: v1alpha1.ResourceList{
							Memory: "8Gi",
							CPU:    "4",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "100Gi",
						StorageClassName: "fast-ssd",
					},
					Retention: "30d",
					RemoteWrite: []v1alpha1.RemoteWriteSpec{
						{
							URL:           "https://remote.example.com/write",
							RemoteTimeout: "30s",
							Headers: map[string]string{
								"X-Auth-Token": "secret",
							},
						},
					},
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Enabled:       true,
					Version:       "10.2.0",
					Replicas:      2,
					AdminPassword: "admin123",
					Ingress: &v1alpha1.IngressConfig{
						Enabled:   true,
						ClassName: "nginx",
						Host:      "grafana.example.com",
						Path:      "/",
						TLS: &v1alpha1.IngressTLS{
							Enabled:    true,
							SecretName: "grafana-tls",
						},
						Annotations: map[string]string{
							"nginx.ingress.kubernetes.io/ssl-redirect": "true",
						},
					},
					DataSources: []v1alpha1.DataSourceConfig{
						{
							Name:      "Prometheus",
							Type:      "prometheus",
							URL:       "http://prometheus:9090",
							IsDefault: true,
						},
					},
				},
				Loki: &v1alpha1.LokiSpec{
					Enabled:   true,
					Version:   "2.9.0",
					Retention: "7d",
					S3: &v1alpha1.S3Config{
						Enabled:    true,
						BucketName: "loki-logs",
						Region:     "us-east-1",
						Endpoint:   "s3.amazonaws.com",
					},
				},
				Tempo: &v1alpha1.TempoSpec{
					Enabled:   true,
					Version:   "2.3.0",
					Retention: "24h",
				},
			},
			Global: v1alpha1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster": "production",
					"region":  "us-east-1",
				},
				LogLevel: "info",
				NodeSelector: map[string]string{
					"node-role": "observability",
				},
			},
			HighAvailability: &v1alpha1.HighAvailabilityConfig{
				Enabled:     true,
				MinReplicas: 3,
			},
			Backup: &v1alpha1.BackupConfig{
				Enabled:       true,
				Schedule:      "0 2 * * *",
				RetentionDays: 7,
				Destination: v1alpha1.BackupDestination{
					Type: "s3",
					S3: &v1alpha1.S3Config{
						BucketName: "backups",
						Region:     "us-east-1",
					},
				},
			},
			Alerting: &v1alpha1.AlertingConfig{
				AlertManager: &v1alpha1.AlertManagerSpec{
					Enabled:  true,
					Replicas: 3,
					Config:   "global:\n  resolve_timeout: 5m",
				},
			},
		},
		Status: v1alpha1.ObservabilityPlatformStatus{
			Phase:              "Ready",
			ObservedGeneration: 5,
			Message:            "All components are running",
			ComponentStatus: map[string]v1alpha1.ComponentStatus{
				"prometheus": {
					Phase:          "Ready",
					Version:        "v2.48.0",
					ReadyReplicas:  3,
					LastUpdateTime: &metav1.Time{},
				},
			},
		},
	}

	// Convert to v1beta1
	reqBody, err := json.Marshal(&apiextensionsv1.ConversionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "ConversionReview",
		},
		Request: &apiextensionsv1.ConversionRequest{
			UID:               "test-complex",
			DesiredAPIVersion: "observability.io/v1beta1",
			Objects: []runtime.RawExtension{
				{Raw: mustMarshal(t, v1alpha1Platform)},
			},
		},
	})
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle the request
	webhook.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Unmarshal response
	var response apiextensionsv1.ConversionReview
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Validate response
	require.NotNil(t, response.Response)
	assert.Equal(t, metav1.StatusSuccess, response.Response.Result.Status)
	require.Len(t, response.Response.ConvertedObjects, 1)

	// Unmarshal the converted object
	var converted v1beta1.ObservabilityPlatform
	err = json.Unmarshal(response.Response.ConvertedObjects[0].Raw, &converted)
	require.NoError(t, err)

	// Validate all fields are properly converted
	assert.Equal(t, "complex-platform", converted.Name)
	assert.Equal(t, "monitoring", converted.Namespace)
	assert.Equal(t, v1alpha1Platform.Labels, converted.Labels)
	assert.True(t, converted.Spec.Paused)

	// Check Prometheus conversion
	require.NotNil(t, converted.Spec.Components.Prometheus)
	assert.True(t, converted.Spec.Components.Prometheus.Enabled)
	assert.Equal(t, "v2.48.0", converted.Spec.Components.Prometheus.Version)
	assert.Equal(t, int32(3), converted.Spec.Components.Prometheus.Replicas)
	assert.Equal(t, "30d", converted.Spec.Components.Prometheus.Retention)

	// Check resource conversion
	assert.Equal(t, "4Gi", converted.Spec.Components.Prometheus.Resources.Requests.Memory().String())
	assert.Equal(t, "2", converted.Spec.Components.Prometheus.Resources.Requests.Cpu().String())

	// Check storage conversion
	assert.Equal(t, "100Gi", converted.Spec.Components.Prometheus.Storage.Size.String())
	assert.Equal(t, "fast-ssd", converted.Spec.Components.Prometheus.Storage.StorageClassName)

	// Check remote write conversion
	require.Len(t, converted.Spec.Components.Prometheus.RemoteWrite, 1)
	assert.Equal(t, "https://remote.example.com/write", converted.Spec.Components.Prometheus.RemoteWrite[0].URL)

	// Check Grafana conversion
	require.NotNil(t, converted.Spec.Components.Grafana)
	assert.True(t, converted.Spec.Components.Grafana.Enabled)
	assert.Equal(t, "10.2.0", converted.Spec.Components.Grafana.Version)
	assert.Equal(t, int32(2), converted.Spec.Components.Grafana.Replicas)
	assert.Equal(t, "admin123", converted.Spec.Components.Grafana.AdminPassword)

	// Check ingress conversion
	require.NotNil(t, converted.Spec.Components.Grafana.Ingress)
	assert.True(t, converted.Spec.Components.Grafana.Ingress.Enabled)
	assert.Equal(t, "nginx", converted.Spec.Components.Grafana.Ingress.ClassName)
	assert.Equal(t, "grafana.example.com", converted.Spec.Components.Grafana.Ingress.Host)

	// Check datasource conversion
	require.Len(t, converted.Spec.Components.Grafana.DataSources, 1)
	assert.Equal(t, "Prometheus", converted.Spec.Components.Grafana.DataSources[0].Name)
	assert.True(t, converted.Spec.Components.Grafana.DataSources[0].IsDefault)

	// Check global config conversion
	assert.Equal(t, v1alpha1Platform.Spec.Global.ExternalLabels, converted.Spec.Global.ExternalLabels)
	assert.Equal(t, "info", converted.Spec.Global.LogLevel)
	assert.Equal(t, v1alpha1Platform.Spec.Global.NodeSelector, converted.Spec.Global.NodeSelector)

	// Check status conversion
	assert.Equal(t, "Ready", converted.Status.Phase)
	assert.Equal(t, int64(5), converted.Status.ObservedGeneration)
	assert.Equal(t, "All components are running", converted.Status.Message)

	// Now test round-trip conversion back to v1alpha1
	reqBody2, err := json.Marshal(&apiextensionsv1.ConversionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "ConversionReview",
		},
		Request: &apiextensionsv1.ConversionRequest{
			UID:               "test-roundtrip",
			DesiredAPIVersion: "observability.io/v1alpha1",
			Objects: []runtime.RawExtension{
				{Raw: response.Response.ConvertedObjects[0].Raw},
			},
		},
	})
	require.NoError(t, err)

	// Create HTTP request for reverse conversion
	req2 := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(reqBody2))
	req2.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr2 := httptest.NewRecorder()

	// Handle the request
	webhook.ServeHTTP(rr2, req2)

	// Check status code
	assert.Equal(t, http.StatusOK, rr2.Code)

	// Unmarshal response
	var response2 apiextensionsv1.ConversionReview
	err = json.Unmarshal(rr2.Body.Bytes(), &response2)
	require.NoError(t, err)

	// Validate response
	require.NotNil(t, response2.Response)
	assert.Equal(t, metav1.StatusSuccess, response2.Response.Result.Status)
	require.Len(t, response2.Response.ConvertedObjects, 1)

	// Unmarshal the round-trip converted object
	var roundtrip v1alpha1.ObservabilityPlatform
	err = json.Unmarshal(response2.Response.ConvertedObjects[0].Raw, &roundtrip)
	require.NoError(t, err)

	// Verify key fields survived round-trip
	assert.Equal(t, v1alpha1Platform.Name, roundtrip.Name)
	assert.Equal(t, v1alpha1Platform.Namespace, roundtrip.Namespace)
	assert.Equal(t, v1alpha1Platform.Spec.Paused, roundtrip.Spec.Paused)
	assert.Equal(t, v1alpha1Platform.Spec.Components.Prometheus.Enabled, roundtrip.Spec.Components.Prometheus.Enabled)
	assert.Equal(t, v1alpha1Platform.Spec.Components.Prometheus.Version, roundtrip.Spec.Components.Prometheus.Version)
	assert.Equal(t, v1alpha1Platform.Spec.Components.Prometheus.Replicas, roundtrip.Spec.Components.Prometheus.Replicas)
	assert.Equal(t, v1alpha1Platform.Spec.Global.LogLevel, roundtrip.Spec.Global.LogLevel)
}

// Helper functions

func mustMarshal(t *testing.T, obj interface{}) []byte {
	data, err := json.Marshal(obj)
	require.NoError(t, err)
	return data
}

// fakeManager implements the manager.Manager interface for testing
type fakeManager struct {
	manager.Manager
	client runtime.Client
	scheme *runtime.Scheme
	logger log.Logger
}

func (m *fakeManager) GetClient() runtime.Client {
	return m.client
}

func (m *fakeManager) GetScheme() *runtime.Scheme {
	return m.scheme
}

func (m *fakeManager) GetLogger() log.Logger {
	return m.logger
}
