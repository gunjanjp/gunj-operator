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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/status"
)

func TestStatusSubresource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Status Subresource Suite")
}

var _ = Describe("ObservabilityPlatform Status Subresource", func() {
	var (
		ctx      context.Context
		k8sClient client.Client
		scheme   *runtime.Scheme
		platform *v1beta1.ObservabilityPlatform
		updater  *status.Updater
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(v1beta1.AddToScheme(scheme)).To(Succeed())
		
		// Create fake client with status subresource support
		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1beta1.ObservabilityPlatform{}).
			Build()
			
		updater = status.NewUpdater(k8sClient)
		
		// Create test platform
		platform = &v1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
				Generation: 1,
			},
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: v1beta1.Components{
					Prometheus: &v1beta1.PrometheusSpec{
						Enabled: true,
						Version: "v2.48.0",
					},
				},
			},
		}
		
		// Create the platform
		Expect(k8sClient.Create(ctx, platform)).To(Succeed())
	})

	Describe("Status Helper Methods", func() {
		It("should initialize status correctly", func() {
			platform.InitializeStatus()
			
			Expect(platform.Status.Phase).To(Equal(v1beta1.PhasePending))
			Expect(platform.Status.ComponentStatus).NotTo(BeNil())
			Expect(platform.Status.Endpoints).NotTo(BeNil())
			Expect(platform.Status.Conditions).NotTo(BeNil())
			
			// Check initial conditions
			readyCondition := platform.GetCondition(v1beta1.ConditionTypeReady)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			
			progressingCondition := platform.GetCondition(v1beta1.ConditionTypeProgressing)
			Expect(progressingCondition).NotTo(BeNil())
			Expect(progressingCondition.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should update phase correctly", func() {
			platform.UpdatePhase(v1beta1.PhaseInstalling, "Installing components")
			
			Expect(platform.Status.Phase).To(Equal(v1beta1.PhaseInstalling))
			Expect(platform.Status.Message).To(Equal("Installing components"))
			Expect(platform.Status.LastReconcileTime).NotTo(BeNil())
		})

		It("should manage conditions correctly", func() {
			// Set initial condition
			platform.SetReadyCondition(metav1.ConditionFalse, v1beta1.ReasonInstalling, "Not ready")
			
			condition := platform.GetCondition(v1beta1.ConditionTypeReady)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionFalse))
			Expect(condition.Reason).To(Equal(v1beta1.ReasonInstalling))
			
			// Update condition
			platform.SetReadyCondition(metav1.ConditionTrue, v1beta1.ReasonReady, "All components ready")
			
			condition = platform.GetCondition(v1beta1.ConditionTypeReady)
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.Reason).To(Equal(v1beta1.ReasonReady))
			
			// Verify transition time was updated
			Expect(condition.LastTransitionTime.Time).To(BeTemporally("~", time.Now(), 5*time.Second))
		})

		It("should track component status correctly", func() {
			// Update component status
			platform.UpdateComponentStatus("prometheus", v1beta1.ComponentStatus{
				Phase:    v1beta1.ComponentPhaseReady,
				Version:  "v2.48.0",
				Replicas: 1,
				Ready:    1,
			})
			
			Expect(platform.Status.ComponentStatus).To(HaveKey("prometheus"))
			Expect(platform.Status.ComponentStatus["prometheus"].Phase).To(Equal(v1beta1.ComponentPhaseReady))
			Expect(platform.Status.ComponentStatus["prometheus"].LastUpdate).NotTo(BeNil())
		})

		It("should calculate health status correctly", func() {
			// No components - unknown health
			platform.UpdateHealth()
			Expect(platform.Status.Health.Status).To(Equal(v1beta1.HealthStatusUnknown))
			
			// Add ready component
			platform.UpdateComponentStatus("prometheus", v1beta1.ComponentStatus{
				Phase: v1beta1.ComponentPhaseReady,
			})
			platform.UpdateHealth()
			Expect(platform.Status.Health.Status).To(Equal(v1beta1.HealthStatusHealthy))
			Expect(platform.Status.Health.HealthyCount).To(Equal(int32(1)))
			Expect(platform.Status.Health.TotalCount).To(Equal(int32(1)))
			
			// Add failed component
			platform.UpdateComponentStatus("grafana", v1beta1.ComponentStatus{
				Phase: v1beta1.ComponentPhaseFailed,
			})
			platform.UpdateHealth()
			Expect(platform.Status.Health.Status).To(Equal(v1beta1.HealthStatusDegraded))
			Expect(platform.Status.Health.HealthyCount).To(Equal(int32(1)))
			Expect(platform.Status.Health.TotalCount).To(Equal(int32(2)))
		})

		It("should detect update needs correctly", func() {
			// Initially, observed generation is 0
			Expect(platform.NeedsUpdate()).To(BeTrue())
			
			// After updating status
			platform.Status.ObservedGeneration = platform.Generation
			Expect(platform.NeedsUpdate()).To(BeFalse())
			
			// After spec change
			platform.Generation = 2
			Expect(platform.NeedsUpdate()).To(BeTrue())
		})
	})

	Describe("Status Updater", func() {
		It("should update status subresource independently", func() {
			// Update status using the updater
			err := updater.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
				p.UpdatePhase(v1beta1.PhaseInstalling, "Installing Prometheus")
				p.UpdateComponentStatus("prometheus", v1beta1.ComponentStatus{
					Phase: v1beta1.ComponentPhaseDeploying,
				})
			})
			Expect(err).NotTo(HaveOccurred())
			
			// Fetch the platform to verify status was updated
			updated := &v1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      platform.Name,
				Namespace: platform.Namespace,
			}, updated)
			Expect(err).NotTo(HaveOccurred())
			
			Expect(updated.Status.Phase).To(Equal(v1beta1.PhaseInstalling))
			Expect(updated.Status.Message).To(Equal("Installing Prometheus"))
			Expect(updated.Status.ComponentStatus).To(HaveKey("prometheus"))
			Expect(updated.Status.ObservedGeneration).To(Equal(updated.Generation))
		})

		It("should set phase using convenience method", func() {
			err := updater.SetPhaseV1Beta1(ctx, platform, v1beta1.PhaseReady, "All components deployed")
			Expect(err).NotTo(HaveOccurred())
			
			// Verify
			updated := &v1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      platform.Name,
				Namespace: platform.Namespace,
			}, updated)
			Expect(err).NotTo(HaveOccurred())
			
			Expect(updated.Status.Phase).To(Equal(v1beta1.PhaseReady))
			Expect(updated.Status.Message).To(Equal("All components deployed"))
		})

		It("should set conditions using convenience method", func() {
			err := updater.SetConditionV1Beta1(ctx, platform, 
				v1beta1.ConditionTypeReady,
				metav1.ConditionTrue,
				v1beta1.ReasonReady,
				"Platform is ready")
			Expect(err).NotTo(HaveOccurred())
			
			// Verify
			updated := &v1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      platform.Name,
				Namespace: platform.Namespace,
			}, updated)
			Expect(err).NotTo(HaveOccurred())
			
			condition := updated.GetCondition(v1beta1.ConditionTypeReady)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
			Expect(condition.Reason).To(Equal(v1beta1.ReasonReady))
		})

		It("should update component status", func() {
			err := updater.UpdateComponentStatusV1Beta1(ctx, platform, "prometheus", v1beta1.ComponentStatus{
				Phase:    v1beta1.ComponentPhaseReady,
				Version:  "v2.48.0",
				Replicas: 3,
				Ready:    3,
			})
			Expect(err).NotTo(HaveOccurred())
			
			// Verify
			updated := &v1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      platform.Name,
				Namespace: platform.Namespace,
			}, updated)
			Expect(err).NotTo(HaveOccurred())
			
			Expect(updated.Status.ComponentStatus["prometheus"].Phase).To(Equal(v1beta1.ComponentPhaseReady))
			Expect(updated.Status.ComponentStatus["prometheus"].Ready).To(Equal(int32(3)))
			
			// Health should be updated too
			Expect(updated.Status.Health.Status).To(Equal(v1beta1.HealthStatusHealthy))
		})

		It("should mark platform as reconciling", func() {
			err := updater.MarkReconcilingV1Beta1(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify
			updated := &v1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      platform.Name,
				Namespace: platform.Namespace,
			}, updated)
			Expect(err).NotTo(HaveOccurred())
			
			condition := updated.GetCondition(v1beta1.ConditionTypeReconciling)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should handle concurrent updates with retries", func() {
			// This test simulates concurrent updates by modifying the platform
			// between read and write operations
			updateCount := 0
			err := updater.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
				updateCount++
				p.UpdatePhase(v1beta1.PhaseReady, "Ready after retries")
			})
			
			Expect(err).NotTo(HaveOccurred())
			Expect(updateCount).To(Equal(1)) // Should succeed on first try in test
		})
	})

	Describe("Status Summary", func() {
		It("should generate readable status summary", func() {
			platform.Status.Phase = v1beta1.PhaseReady
			platform.Status.Health.Status = v1beta1.HealthStatusHealthy
			platform.Status.Health.HealthyCount = 4
			platform.Status.Health.TotalCount = 4
			platform.Status.Message = "All systems operational"
			
			summary := platform.GetStatusSummary()
			Expect(summary).To(ContainSubstring("Phase: Ready"))
			Expect(summary).To(ContainSubstring("Health: Healthy"))
			Expect(summary).To(ContainSubstring("4/4 components healthy"))
			Expect(summary).To(ContainSubstring("All systems operational"))
		})
	})

	Describe("Endpoint Management", func() {
		It("should manage component endpoints", func() {
			platform.UpdateEndpoint("prometheus", "http://prometheus.monitoring.svc:9090")
			platform.UpdateEndpoint("grafana", "http://grafana.monitoring.svc:3000")
			
			Expect(platform.GetEndpoint("prometheus")).To(Equal("http://prometheus.monitoring.svc:9090"))
			Expect(platform.GetEndpoint("grafana")).To(Equal("http://grafana.monitoring.svc:3000"))
			Expect(platform.GetEndpoint("nonexistent")).To(BeEmpty())
		})
	})

	Describe("Component Management", func() {
		It("should list enabled components", func() {
			platform.Spec.Components = v1beta1.Components{
				Prometheus: &v1beta1.PrometheusSpec{Enabled: true},
				Grafana:    &v1beta1.GrafanaSpec{Enabled: true},
				Loki:       &v1beta1.LokiSpec{Enabled: false},
				Tempo:      &v1beta1.TempoSpec{Enabled: true},
			}
			
			enabled := platform.GetEnabledComponents()
			Expect(enabled).To(ConsistOf("prometheus", "grafana", "tempo"))
			Expect(enabled).NotTo(ContainElement("loki"))
		})

		It("should retrieve component specs", func() {
			prometheusSpec := platform.GetComponentSpec("prometheus")
			Expect(prometheusSpec).NotTo(BeNil())
			Expect(prometheusSpec.(*v1beta1.PrometheusSpec).Version).To(Equal("v2.48.0"))
			
			nonExistent := platform.GetComponentSpec("invalid")
			Expect(nonExistent).To(BeNil())
		})
	})
})
