/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/webhooks"
)

// TestConversionWebhookIntegration tests the conversion webhook in a more realistic environment
func TestConversionWebhookIntegration(t *testing.T) {
	// Set up logging
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	// Create scheme
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	require.NoError(t, apiextensionsv1.AddToScheme(scheme))

	// Create test environment
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{"../../config/crd/bases"},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{"../../config/webhook"},
		},
	}

	// Start test environment
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, testEnv.Stop())
	}()

	// Create manager
	mgr, err := manager.New(cfg, manager.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		Port:               9443,
		CertDir:            testEnv.WebhookInstallOptions.LocalServingCertDir,
	})
	require.NoError(t, err)

	// Create and register webhook
	conversionWebhook, err := webhooks.NewConversionWebhook(mgr, log.Log)
	require.NoError(t, err)

	// Register webhook with manager
	mgr.GetWebhookServer().Register("/convert", conversionWebhook)

	// Start manager in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for webhook server to be ready
	time.Sleep(2 * time.Second)

	// Create client
	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)

	// Test cases
	t.Run("create v1alpha1 and read as v1beta1", func(t *testing.T) {
		// Create v1alpha1 object
		v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform-1",
				Namespace: "default",
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled:  true,
						Version:  "v2.48.0",
						Replicas: 3,
					},
				},
			},
		}

		// Create the object
		err := k8sClient.Create(context.Background(), v1alpha1Platform)
		require.NoError(t, err)

		// Read it back as v1beta1
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{
			Name:      "test-platform-1",
			Namespace: "default",
		}, v1beta1Platform)
		require.NoError(t, err)

		// Verify conversion
		assert.Equal(t, "test-platform-1", v1beta1Platform.Name)
		assert.NotNil(t, v1beta1Platform.Spec.Components.Prometheus)
		assert.True(t, v1beta1Platform.Spec.Components.Prometheus.Enabled)
		assert.Equal(t, "v2.48.0", v1beta1Platform.Spec.Components.Prometheus.Version)
		assert.Equal(t, int32(3), v1beta1Platform.Spec.Components.Prometheus.Replicas)
	})

	t.Run("create v1beta1 and read as v1alpha1", func(t *testing.T) {
		// Create v1beta1 object
		v1beta1Platform := &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform-2",
				Namespace: "default",
			},
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: v1beta1.Components{
					Grafana: &v1beta1.GrafanaSpec{
						Enabled: true,
						Version: "10.2.0",
						Plugins: []string{"piechart-panel"}, // This will be lost in conversion
					},
				},
			},
		}

		// Create the object
		err := k8sClient.Create(context.Background(), v1beta1Platform)
		require.NoError(t, err)

		// Read it back as v1alpha1
		v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{
			Name:      "test-platform-2",
			Namespace: "default",
		}, v1alpha1Platform)
		require.NoError(t, err)

		// Verify conversion
		assert.Equal(t, "test-platform-2", v1alpha1Platform.Name)
		assert.NotNil(t, v1alpha1Platform.Spec.Components.Grafana)
		assert.True(t, v1alpha1Platform.Spec.Components.Grafana.Enabled)
		assert.Equal(t, "10.2.0", v1alpha1Platform.Spec.Components.Grafana.Version)
		// Verify annotation about lost fields
		assert.Contains(t, v1alpha1Platform.Annotations, "observability.io/conversion-lost-fields")
	})

	t.Run("update v1alpha1 object through v1beta1", func(t *testing.T) {
		// Create v1alpha1 object
		v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform-3",
				Namespace: "default",
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled:  true,
						Version:  "v2.47.0",
						Replicas: 1,
					},
				},
			},
		}

		// Create the object
		err := k8sClient.Create(context.Background(), v1alpha1Platform)
		require.NoError(t, err)

		// Read it as v1beta1
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{
			Name:      "test-platform-3",
			Namespace: "default",
		}, v1beta1Platform)
		require.NoError(t, err)

		// Update through v1beta1
		v1beta1Platform.Spec.Components.Prometheus.Version = "v2.48.0"
		v1beta1Platform.Spec.Components.Prometheus.Replicas = 3
		err = k8sClient.Update(context.Background(), v1beta1Platform)
		require.NoError(t, err)

		// Read back as v1alpha1
		updatedV1alpha1 := &v1alpha1.ObservabilityPlatform{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{
			Name:      "test-platform-3",
			Namespace: "default",
		}, updatedV1alpha1)
		require.NoError(t, err)

		// Verify update was applied
		assert.Equal(t, "v2.48.0", updatedV1alpha1.Spec.Components.Prometheus.Version)
		assert.Equal(t, int32(3), updatedV1alpha1.Spec.Components.Prometheus.Replicas)
	})
}

// TestWebhookHTTPHandling tests the HTTP handling of the conversion webhook
func TestWebhookHTTPHandling(t *testing.T) {
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
		method             string
		contentType        string
		body               interface{}
		expectedStatusCode int
		expectedError      bool
	}{
		{
			name:        "valid POST request",
			method:      http.MethodPost,
			contentType: "application/json",
			body: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               "test-uid",
					DesiredAPIVersion: "observability.io/v1beta1",
					Objects: []runtime.RawExtension{
						{
							Raw: mustMarshal(t, &v1alpha1.ObservabilityPlatform{
								TypeMeta: metav1.TypeMeta{
									APIVersion: "observability.io/v1alpha1",
									Kind:       "ObservabilityPlatform",
								},
								ObjectMeta: metav1.ObjectMeta{
									Name: "test",
								},
							}),
						},
					},
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedError:      false,
		},
		{
			name:               "GET request not allowed",
			method:             http.MethodGet,
			contentType:        "application/json",
			body:               nil,
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedError:      true,
		},
		{
			name:        "missing content-type",
			method:      http.MethodPost,
			contentType: "",
			body: &apiextensionsv1.ConversionReview{
				Request: &apiextensionsv1.ConversionRequest{
					UID: "test",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      true,
		},
		{
			name:               "invalid JSON body",
			method:             http.MethodPost,
			contentType:        "application/json",
			body:               "invalid json",
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      true,
		},
		{
			name:        "missing request in ConversionReview",
			method:      http.MethodPost,
			contentType: "application/json",
			body: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: nil,
			},
			expectedStatusCode: http.StatusOK, // Still returns 200 but with failure in response
			expectedError:      false,
		},
		{
			name:        "empty objects array",
			method:      http.MethodPost,
			contentType: "application/json",
			body: &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               "test-uid",
					DesiredAPIVersion: "observability.io/v1beta1",
					Objects:           []runtime.RawExtension{},
				},
			},
			expectedStatusCode: http.StatusOK,
			expectedError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var reqBody []byte
			var err error
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					reqBody = []byte(str)
				} else {
					reqBody, err = json.Marshal(tt.body)
					require.NoError(t, err)
				}
			}

			// Create HTTP request
			req := httptest.NewRequest(tt.method, "/convert", bytes.NewReader(reqBody))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Handle the request
			webhook.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			// For successful requests, check the response
			if !tt.expectedError && tt.expectedStatusCode == http.StatusOK {
				var response apiextensionsv1.ConversionReview
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotNil(t, response.Response)
			}
		})
	}
}

// TestConversionWebhookConcurrency tests concurrent conversion requests
func TestConversionWebhookConcurrency(t *testing.T) {
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

	// Number of concurrent requests
	concurrency := 100

	// Channel to collect results
	results := make(chan bool, concurrency)

	// Launch concurrent requests
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			// Create unique platform for each goroutine
			platform := &v1alpha1.ObservabilityPlatform{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "observability.io/v1alpha1",
					Kind:       "ObservabilityPlatform",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("platform-%d", id),
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: int32(id % 5 + 1),
						},
					},
				},
			}

			// Create conversion request
			review := &apiextensionsv1.ConversionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "ConversionReview",
				},
				Request: &apiextensionsv1.ConversionRequest{
					UID:               types.UID(fmt.Sprintf("uid-%d", id)),
					DesiredAPIVersion: "observability.io/v1beta1",
					Objects: []runtime.RawExtension{
						{Raw: mustMarshal(t, platform)},
					},
				},
			}

			// Marshal request
			reqBody, err := json.Marshal(review)
			if err != nil {
				results <- false
				return
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Handle the request
			webhook.ServeHTTP(rr, req)

			// Check response
			if rr.Code != http.StatusOK {
				results <- false
				return
			}

			// Unmarshal response
			var response apiextensionsv1.ConversionReview
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				results <- false
				return
			}

			// Validate response
			if response.Response == nil ||
				response.Response.Result.Status != metav1.StatusSuccess ||
				len(response.Response.ConvertedObjects) != 1 {
				results <- false
				return
			}

			// Verify converted object
			var converted v1beta1.ObservabilityPlatform
			if err := json.Unmarshal(response.Response.ConvertedObjects[0].Raw, &converted); err != nil {
				results <- false
				return
			}

			// Check that conversion was correct
			results <- converted.Name == fmt.Sprintf("platform-%d", id) &&
				converted.Spec.Components.Prometheus.Replicas == int32(id%5+1)
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < concurrency; i++ {
		if <-results {
			successCount++
		}
	}

	// All requests should succeed
	assert.Equal(t, concurrency, successCount, "Not all concurrent requests succeeded")
}

// TestWebhookWithLargeObjects tests conversion of large objects
func TestWebhookWithLargeObjects(t *testing.T) {
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

	// Create a large platform object
	largePlatform := &v1alpha1.ObservabilityPlatform{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "observability.io/v1alpha1",
			Kind:       "ObservabilityPlatform",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "large-platform",
			Namespace: "default",
			Labels:    make(map[string]string),
			Annotations: make(map[string]string),
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:     true,
					Version:     "v2.48.0",
					RemoteWrite: make([]v1alpha1.RemoteWriteSpec, 0),
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Enabled:     true,
					Version:     "10.2.0",
					DataSources: make([]v1alpha1.DataSourceConfig, 0),
				},
			},
			Global: v1alpha1.GlobalConfig{
				ExternalLabels: make(map[string]string),
			},
		},
	}

	// Add lots of labels and annotations
	for i := 0; i < 100; i++ {
		largePlatform.Labels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
		largePlatform.Annotations[fmt.Sprintf("annotation-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	// Add lots of remote write configs
	for i := 0; i < 50; i++ {
		largePlatform.Spec.Components.Prometheus.RemoteWrite = append(
			largePlatform.Spec.Components.Prometheus.RemoteWrite,
			v1alpha1.RemoteWriteSpec{
				URL:           fmt.Sprintf("https://remote-%d.example.com/write", i),
				RemoteTimeout: "30s",
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer token-%d", i),
				},
			},
		)
	}

	// Add lots of datasources
	for i := 0; i < 50; i++ {
		largePlatform.Spec.Components.Grafana.DataSources = append(
			largePlatform.Spec.Components.Grafana.DataSources,
			v1alpha1.DataSourceConfig{
				Name:      fmt.Sprintf("DataSource-%d", i),
				Type:      "prometheus",
				URL:       fmt.Sprintf("http://prometheus-%d:9090", i),
				IsDefault: i == 0,
			},
		)
	}

	// Add lots of external labels
	for i := 0; i < 100; i++ {
		largePlatform.Spec.Global.ExternalLabels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	// Create conversion request
	review := &apiextensionsv1.ConversionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "ConversionReview",
		},
		Request: &apiextensionsv1.ConversionRequest{
			UID:               "large-object-test",
			DesiredAPIVersion: "observability.io/v1beta1",
			Objects: []runtime.RawExtension{
				{Raw: mustMarshal(t, largePlatform)},
			},
		},
	}

	// Marshal request
	reqBody, err := json.Marshal(review)
	require.NoError(t, err)

	// Log the size of the request
	t.Logf("Request body size: %d bytes", len(reqBody))

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle the request
	webhook.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Unmarshal response
	var response apiextensionsv1.ConversionReview
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Validate response
	require.NotNil(t, response.Response)
	assert.Equal(t, metav1.StatusSuccess, response.Response.Result.Status)
	require.Len(t, response.Response.ConvertedObjects, 1)

	// Verify converted object
	var converted v1beta1.ObservabilityPlatform
	err = json.Unmarshal(response.Response.ConvertedObjects[0].Raw, &converted)
	require.NoError(t, err)

	// Verify all data was preserved
	assert.Len(t, converted.Labels, 100)
	assert.Len(t, converted.Annotations, 101) // 100 original + conversion annotation
	assert.Len(t, converted.Spec.Components.Prometheus.RemoteWrite, 50)
	assert.Len(t, converted.Spec.Components.Grafana.DataSources, 50)
	assert.Len(t, converted.Spec.Global.ExternalLabels, 100)
}

// TestBatchConversion tests converting multiple objects in a single request
func TestBatchConversion(t *testing.T) {
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

	// Create multiple platforms
	platforms := make([]runtime.RawExtension, 10)
	for i := 0; i < 10; i++ {
		platform := &v1alpha1.ObservabilityPlatform{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "observability.io/v1alpha1",
				Kind:       "ObservabilityPlatform",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("platform-%d", i),
				Namespace: fmt.Sprintf("namespace-%d", i%3),
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled:  true,
						Version:  fmt.Sprintf("v2.%d.0", 40+i),
						Replicas: int32(i + 1),
					},
				},
			},
		}
		platforms[i] = runtime.RawExtension{Raw: mustMarshal(t, platform)}
	}

	// Create conversion request
	review := &apiextensionsv1.ConversionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "ConversionReview",
		},
		Request: &apiextensionsv1.ConversionRequest{
			UID:               "batch-test",
			DesiredAPIVersion: "observability.io/v1beta1",
			Objects:           platforms,
		},
	}

	// Marshal request
	reqBody, err := json.Marshal(review)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Handle the request
	webhook.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)

	// Unmarshal response
	var response apiextensionsv1.ConversionReview
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Validate response
	require.NotNil(t, response.Response)
	assert.Equal(t, metav1.StatusSuccess, response.Response.Result.Status)
	require.Len(t, response.Response.ConvertedObjects, 10)

	// Verify each converted object
	for i, obj := range response.Response.ConvertedObjects {
		var converted v1beta1.ObservabilityPlatform
		err = json.Unmarshal(obj.Raw, &converted)
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("platform-%d", i), converted.Name)
		assert.Equal(t, fmt.Sprintf("namespace-%d", i%3), converted.Namespace)
		assert.Equal(t, fmt.Sprintf("v2.%d.0", 40+i), converted.Spec.Components.Prometheus.Version)
		assert.Equal(t, int32(i+1), converted.Spec.Components.Prometheus.Replicas)
	}
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
	client client.Client
	scheme *runtime.Scheme
	logger log.Logger
}

func (m *fakeManager) GetClient() client.Client {
	return m.client
}

func (m *fakeManager) GetScheme() *runtime.Scheme {
	return m.scheme
}

func (m *fakeManager) GetConfig() *rest.Config {
	return &rest.Config{}
}

func (m *fakeManager) GetWebhookServer() *webhook.Server {
	return &webhook.Server{}
}

func (m *fakeManager) GetLogger() log.Logger {
	return m.logger
}
