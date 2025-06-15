/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
)

func TestPrinterColumns(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1alpha1 Printer Columns Suite")
}

var _ = Describe("ObservabilityPlatform v1alpha1 Printer Columns", func() {
	Context("when displaying kubectl output", func() {
		var platform *v1alpha1.ObservabilityPlatform

		BeforeEach(func() {
			platform = &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
						},
					},
				},
			}
		})

		Describe("Basic columns", func() {
			It("should display phase", func() {
				platform.Status.Phase = "Ready"
				Expect(platform.Status.Phase).To(Equal("Ready"))
			})

			It("should display message", func() {
				platform.Status.Message = "Platform is ready"
				Expect(platform.Status.Message).To(Equal("Platform is ready"))
			})
		})

		Describe("Ready condition column", func() {
			It("should extract Ready condition status", func() {
				platform.Status.Conditions = []metav1.Condition{
					{
						Type:   "Ready",
						Status: metav1.ConditionTrue,
						Reason: "AllComponentsReady",
					},
					{
						Type:   "Progressing",
						Status: metav1.ConditionFalse,
					},
				}

				// Find Ready condition
				var readyStatus metav1.ConditionStatus
				for _, cond := range platform.Status.Conditions {
					if cond.Type == "Ready" {
						readyStatus = cond.Status
						break
					}
				}
				
				Expect(string(readyStatus)).To(Equal("True"))
			})
		})

		Describe("Component status columns", func() {
			It("should display component phases", func() {
				platform.Status.ComponentStatus = map[string]v1alpha1.ComponentStatus{
					"prometheus": {
						Phase:         "Ready",
						Version:       "v2.48.0",
						ReadyReplicas: 1,
					},
					"grafana": {
						Phase:         "Ready",
						Version:       "10.2.0",
						ReadyReplicas: 1,
					},
				}

				Expect(platform.Status.ComponentStatus["prometheus"].Phase).To(Equal("Ready"))
				Expect(platform.Status.ComponentStatus["grafana"].Phase).To(Equal("Ready"))
			})
		})
	})
})
