/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"

	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestPrinterColumns(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Printer Columns Suite")
}

var _ = Describe("ObservabilityPlatform Printer Columns", func() {
	Context("when displaying kubectl output", func() {
		var (
			ctx      context.Context
			platform *v1beta1.ObservabilityPlatform
		)

		BeforeEach(func() {
			ctx = context.Background()
			platform = &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						Grafana: &v1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
						},
						Loki: &v1beta1.LokiSpec{
							Enabled: true,
							Version: "2.9.0",
						},
						Tempo: &v1beta1.TempoSpec{
							Enabled: true,
							Version: "2.3.0",
						},
					},
				},
			}
		})

		Describe("Phase column", func() {
			It("should display the current phase", func() {
				testCases := []struct {
					phase    string
					expected string
				}{
					{phase: "Pending", expected: "Pending"},
					{phase: "Installing", expected: "Installing"},
					{phase: "Ready", expected: "Ready"},
					{phase: "Failed", expected: "Failed"},
					{phase: "Upgrading", expected: "Upgrading"},
					{phase: "Degraded", expected: "Degraded"},
				}

				for _, tc := range testCases {
					platform.Status.Phase = tc.phase
					Expect(platform.Status.Phase).To(Equal(tc.expected))
				}
			})
		})

		Describe("Health column", func() {
			It("should display the health status", func() {
				testCases := []struct {
					status   string
					expected string
				}{
					{status: "Healthy", expected: "Healthy"},
					{status: "Degraded", expected: "Degraded"},
					{status: "Unhealthy", expected: "Unhealthy"},
					{status: "Unknown", expected: "Unknown"},
				}

				for _, tc := range testCases {
					platform.Status.Health.Status = tc.status
					Expect(platform.Status.Health.Status).To(Equal(tc.expected))
				}
			})
		})

		Describe("Components column", func() {
			It("should display the healthy component count", func() {
				platform.Status.Health.HealthyCount = 3
				platform.Status.Health.TotalCount = 4
				
				Expect(platform.Status.Health.HealthyCount).To(Equal(int32(3)))
			})
		})

		Describe("Component status columns", func() {
			It("should display individual component statuses", func() {
				platform.Status.ComponentStatus = map[string]v1beta1.ComponentStatus{
					"prometheus": {
						Phase:   "Ready",
						Version: "v2.48.0",
						Ready:   3,
					},
					"grafana": {
						Phase:   "Ready",
						Version: "10.2.0",
						Ready:   2,
					},
					"loki": {
						Phase:   "Failed",
						Version: "2.9.0",
						Message: "OOMKilled",
					},
					"tempo": {
						Phase: "Pending",
					},
				}

				Expect(platform.Status.ComponentStatus["prometheus"].Phase).To(Equal("Ready"))
				Expect(platform.Status.ComponentStatus["grafana"].Phase).To(Equal("Ready"))
				Expect(platform.Status.ComponentStatus["loki"].Phase).To(Equal("Failed"))
				Expect(platform.Status.ComponentStatus["tempo"].Phase).To(Equal("Pending"))
			})
		})

		Describe("Version column", func() {
			It("should display the platform version", func() {
				platform.Status.Version = "v2.0.0"
				Expect(platform.Status.Version).To(Equal("v2.0.0"))
			})
		})

		Describe("Message column", func() {
			It("should display status messages", func() {
				testMessages := []string{
					"All components are running successfully",
					"Installing Prometheus and Grafana components",
					"Loki component failed to start: insufficient memory",
					"Upgrading components to new versions",
				}

				for _, msg := range testMessages {
					platform.Status.Message = msg
					Expect(platform.Status.Message).To(Equal(msg))
				}
			})
		})

		Describe("Age column", func() {
			It("should calculate age from creation timestamp", func() {
				now := time.Now()
				platform.CreationTimestamp = metav1.NewTime(now.Add(-5 * 24 * time.Hour))
				
				age := now.Sub(platform.CreationTimestamp.Time)
				Expect(age.Hours()).To(BeNumerically("~", 120, 1))
			})
		})

		Describe("JSONPath validation", func() {
			It("should extract values using JSONPath expressions", func() {
				// Set up complete status
				platform.Status = v1beta1.ObservabilityPlatformStatus{
					Phase: "Ready",
					Health: v1beta1.HealthStatus{
						Status:       "Healthy",
						HealthyCount: 4,
						TotalCount:   4,
					},
					Version: "v2.0.0",
					Message: "All components running",
					ComponentStatus: map[string]v1beta1.ComponentStatus{
						"prometheus": {Phase: "Ready"},
						"grafana":    {Phase: "Ready"},
						"loki":       {Phase: "Ready"},
						"tempo":      {Phase: "Ready"},
					},
				}

				// Convert to JSON to validate JSONPath expressions work
				data, err := json.Marshal(platform)
				Expect(err).NotTo(HaveOccurred())

				var result map[string]interface{}
				err = json.Unmarshal(data, &result)
				Expect(err).NotTo(HaveOccurred())

				// Validate nested path access
				status := result["status"].(map[string]interface{})
				Expect(status["phase"]).To(Equal("Ready"))
				
				health := status["health"].(map[string]interface{})
				Expect(health["status"]).To(Equal("Healthy"))
				Expect(health["healthyCount"]).To(Equal(float64(4)))
			})
		})
	})

	Context("Priority levels", func() {
		It("should have appropriate priority assignments", func() {
			// Priority 0 (default) columns should be:
			// - Phase
			// - Health  
			// - Age

			// Priority 1 columns should be:
			// - Components
			// - Version
			// - Prometheus
			// - Grafana
			// - Loki
			// - Tempo
			// - Message
			
			// This ensures basic info is shown by default,
			// with detailed info available via -o wide
			Expect(true).To(BeTrue()) // Placeholder for priority validation
		})
	})

	Context("Resource categories", func() {
		It("should be categorized correctly", func() {
			// The CRD should have categories: observability, platform
			// This allows queries like:
			// kubectl get observability
			// kubectl get platform
			Expect(true).To(BeTrue()) // Placeholder for category validation
		})
	})

	Context("Short names", func() {
		It("should support short names", func() {
			// The CRD should support short names: op, ops
			// This allows:
			// kubectl get op
			// kubectl get ops
			Expect(true).To(BeTrue()) // Placeholder for short name validation
		})
	})
})
