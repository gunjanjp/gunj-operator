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

package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("HealthCheckManager", func() {
	var (
		healthManager *HealthCheckManager
		ctx          context.Context
		platform     *observabilityv1beta1.ObservabilityPlatform
		scheme       *runtime.Scheme
		fakeClient   client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		_ = observabilityv1beta1.AddToScheme(scheme)
		_ = appsv1.AddToScheme(scheme)
		_ = corev1.AddToScheme(scheme)

		platform = &observabilityv1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "test-namespace",
			},
			Spec: observabilityv1beta1.ObservabilityPlatformSpec{
				Components: observabilityv1beta1.Components{
					Prometheus: &observabilityv1beta1.PrometheusSpec{
						Enabled: true,
						Version: "v2.48.0",
					},
					Grafana: &observabilityv1beta1.GrafanaSpec{
						Enabled: true,
						Version: "10.2.0",
					},
					Loki: &observabilityv1beta1.LokiSpec{
						Enabled: true,
						Version: "2.9.0",
					},
					Tempo: &observabilityv1beta1.TempoSpec{
						Enabled: true,
						Version: "2.3.0",
					},
				},
			},
		}
	})

	Context("when checking component health", func() {
		It("should report healthy when all components are ready", func() {
			// Create fake deployments and services
			prometheusDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-prometheus",
					Namespace: "test-namespace",
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      3,
					ReadyReplicas: 3,
				},
			}

			grafanaDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-grafana",
					Namespace: "test-namespace",
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      2,
					ReadyReplicas: 2,
				},
			}

			lokiStatefulSet := &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-loki",
					Namespace: "test-namespace",
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			}

			tempoDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-tempo",
					Namespace: "test-namespace",
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			}

			// Create services
			prometheusService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-prometheus",
					Namespace: "test-namespace",
				},
			}

			// Create fake client with objects
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					prometheusDeployment,
					grafanaDeployment,
					lokiStatefulSet,
					tempoDeployment,
					prometheusService,
				).
				Build()

			// Mock HTTP client for health endpoints
			healthManager = NewHealthCheckManager(fakeClient)
			healthManager.httpClient = &http.Client{
				Transport: &mockTransport{
					responses: map[string]*http.Response{
						"http://test-platform-prometheus.test-namespace.svc.cluster.local:9090/-/healthy": {
							StatusCode: http.StatusOK,
							Body:       http.NoBody,
						},
						"http://test-platform-grafana.test-namespace.svc.cluster.local:3000/api/health": {
							StatusCode: http.StatusOK,
							Body:       http.NoBody,
						},
						"http://test-platform-loki.test-namespace.svc.cluster.local:3100/ready": {
							StatusCode: http.StatusOK,
							Body:       http.NoBody,
						},
						"http://test-platform-tempo.test-namespace.svc.cluster.local:3200/ready": {
							StatusCode: http.StatusOK,
							Body:       http.NoBody,
						},
					},
				},
			}

			health, err := healthManager.CheckComponentHealth(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			Expect(health).NotTo(BeNil())
			Expect(health.Healthy).To(BeTrue())
			Expect(health.Message).To(Equal("All components are healthy"))
			Expect(health.Components).To(HaveLen(4))
			
			// Check individual components
			Expect(health.Components["prometheus"].Healthy).To(BeTrue())
			Expect(health.Components["prometheus"].AvailableReplicas).To(Equal(int32(3)))
			Expect(health.Components["prometheus"].DesiredReplicas).To(Equal(int32(3)))
			
			Expect(health.Components["grafana"].Healthy).To(BeTrue())
			Expect(health.Components["loki"].Healthy).To(BeTrue())
			Expect(health.Components["tempo"].Healthy).To(BeTrue())
		})

		It("should report unhealthy when components are not ready", func() {
			// Create deployment with no ready replicas
			prometheusDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-prometheus",
					Namespace: "test-namespace",
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      3,
					ReadyReplicas: 0,
				},
			}

			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(prometheusDeployment).
				Build()

			healthManager = NewHealthCheckManager(fakeClient)

			// Only check Prometheus
			platform.Spec.Components.Grafana.Enabled = false
			platform.Spec.Components.Loki.Enabled = false
			platform.Spec.Components.Tempo.Enabled = false

			health, err := healthManager.CheckComponentHealth(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			Expect(health.Healthy).To(BeFalse())
			Expect(health.Components["prometheus"].Healthy).To(BeFalse())
			Expect(health.Components["prometheus"].AvailableReplicas).To(Equal(int32(0)))
			Expect(health.Components["prometheus"].DesiredReplicas).To(Equal(int32(3)))
		})

		It("should handle missing deployments", func() {
			fakeClient = fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			healthManager = NewHealthCheckManager(fakeClient)

			health, err := healthManager.CheckComponentHealth(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			Expect(health.Healthy).To(BeFalse())
			Expect(health.Components["prometheus"].Healthy).To(BeFalse())
			Expect(health.Components["prometheus"].Message).To(Equal("Deployment not found"))
		})
	})

	Context("when checking HTTP endpoints", func() {
		It("should return healthy for successful HTTP responses", func() {
			healthManager = NewHealthCheckManager(nil)
			
			// Mock server that returns 200 OK
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			healthy, message := healthManager.checkHTTPEndpoint(ctx, server.URL)
			Expect(healthy).To(BeTrue())
			Expect(message).To(Equal("Healthy"))
		})

		It("should return unhealthy for failed HTTP responses", func() {
			healthManager = NewHealthCheckManager(nil)
			
			// Mock server that returns 503
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer server.Close()

			healthy, message := healthManager.checkHTTPEndpoint(ctx, server.URL)
			Expect(healthy).To(BeFalse())
			Expect(message).To(ContainSubstring("HTTP status 503"))
		})

		It("should handle connection errors", func() {
			healthManager = NewHealthCheckManager(nil)
			
			// Invalid endpoint
			healthy, message := healthManager.checkHTTPEndpoint(ctx, "http://invalid-endpoint:9999")
			Expect(healthy).To(BeFalse())
			Expect(message).To(ContainSubstring("HTTP check failed"))
		})
	})
})

var _ = Describe("HealthServer", func() {
	var (
		healthServer      *HealthServer
		healthManager     *HealthCheckManager
		ctx              context.Context
		cancel           context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		healthManager = NewHealthCheckManager(nil)
		healthServer = NewHealthServer("8082", healthManager)
	})

	AfterEach(func() {
		cancel()
		time.Sleep(100 * time.Millisecond) // Give server time to shut down
	})

	Context("when handling health endpoints", func() {
		It("should respond to liveness checks", func() {
			go healthServer.Start(ctx)
			time.Sleep(100 * time.Millisecond) // Give server time to start

			healthServer.UpdateLastHealthCheck()

			resp, err := http.Get("http://localhost:8082/healthz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should respond to readiness checks", func() {
			go healthServer.Start(ctx)
			time.Sleep(100 * time.Millisecond)

			// Initially not ready
			resp, err := http.Get("http://localhost:8082/readyz")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
			resp.Body.Close()

			// Mark as ready
			healthServer.SetReady(true)

			resp, err = http.Get("http://localhost:8082/readyz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should fail liveness if no recent health checks", func() {
			go healthServer.Start(ctx)
			time.Sleep(100 * time.Millisecond)

			// Set last health check to old time
			healthServer.healthMu.Lock()
			healthServer.lastHealthCheck = time.Now().Add(-10 * time.Minute)
			healthServer.healthMu.Unlock()

			resp, err := http.Get("http://localhost:8082/healthz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
		})
	})
})

// mockTransport is a mock HTTP transport for testing
type mockTransport struct {
	responses map[string]*http.Response
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if resp, ok := t.responses[req.URL.String()]; ok {
		return resp, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       http.NoBody,
	}, nil
}

// Unit tests using standard testing package
func TestHealthCheckManager_GetComponentHealth(t *testing.T) {
	manager := NewHealthCheckManager(nil)
	
	// Add some test data
	manager.componentHealth["test-prometheus"] = &ComponentHealth{
		Name:              "prometheus",
		Healthy:           true,
		LastChecked:       time.Now(),
		Message:           "Healthy",
		AvailableReplicas: 3,
		DesiredReplicas:   3,
	}

	health := manager.GetComponentHealth()
	if len(health) != 1 {
		t.Errorf("Expected 1 component, got %d", len(health))
	}

	if !health["test-prometheus"].Healthy {
		t.Error("Expected component to be healthy")
	}
}

func TestHealthServer_SetReady(t *testing.T) {
	server := NewHealthServer("8083", nil)
	
	if server.IsReady() {
		t.Error("Expected server to not be ready initially")
	}

	server.SetReady(true)
	
	if !server.IsReady() {
		t.Error("Expected server to be ready after SetReady(true)")
	}
}
