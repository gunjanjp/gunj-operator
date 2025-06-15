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

package drift

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
)

var _ = Describe("Drift Detection Manager", func() {
	var (
		manager    *DetectionManager
		ctx        context.Context
		platform   *observabilityv1.ObservabilityPlatform
		gitOps     *gitopsv1beta1.GitOpsIntegrationSpec
		testScheme *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Setup scheme
		testScheme = scheme.Scheme
		Expect(observabilityv1.AddToScheme(testScheme)).To(Succeed())
		
		// Create fake client
		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		
		// Create manager
		manager = NewDetectionManager(fakeClient, testScheme, ctrl.Log.WithName("test"))
		
		// Create test platform
		platform = &observabilityv1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
			},
			Spec: observabilityv1.ObservabilityPlatformSpec{
				Components: observabilityv1.Components{
					Prometheus: &observabilityv1.PrometheusSpec{
						Enabled: true,
						Version: "v2.48.0",
					},
				},
			},
		}
		
		// Create GitOps spec
		gitOps = &gitopsv1beta1.GitOpsIntegrationSpec{
			DriftDetection: &gitopsv1beta1.DriftDetectionSpec{
				Enabled:       true,
				Interval:      "5m",
				AutoRemediate: true,
			},
		}
	})

	Describe("Drift Detection", func() {
		Context("when resources match desired state", func() {
			It("should report no drift", func() {
				// Create desired state
				desiredState := map[string]*unstructured.Unstructured{
					"apps/v1/Deployment/default/prometheus-server": createDeployment("prometheus-server", "default"),
				}
				
				// Create matching current state
				currentDeployment := createDeployment("prometheus-server", "default")
				Expect(manager.Client.Create(ctx, currentDeployment)).To(Succeed())
				
				// Check drift
				report, err := manager.CheckDrift(ctx, platform, gitOps, desiredState)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(report.DriftDetected).To(BeFalse())
				Expect(report.DriftedResources).To(BeEmpty())
			})
		})

		Context("when resource is modified", func() {
			It("should detect modification drift", func() {
				// Create desired state
				desired := createDeployment("prometheus-server", "default")
				replicas := int32(3)
				desired.Object["spec"].(map[string]interface{})["replicas"] = replicas
				
				desiredState := map[string]*unstructured.Unstructured{
					"apps/v1/Deployment/default/prometheus-server": desired,
				}
				
				// Create current state with different replicas
				current := createDeployment("prometheus-server", "default")
				currentReplicas := int32(1)
				current.Object["spec"].(map[string]interface{})["replicas"] = currentReplicas
				Expect(manager.Client.Create(ctx, current)).To(Succeed())
				
				// Check drift
				report, err := manager.CheckDrift(ctx, platform, gitOps, desiredState)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(report.DriftDetected).To(BeTrue())
				Expect(report.DriftedResources).To(HaveLen(1))
				Expect(report.DriftedResources[0].DriftType).To(Equal("Modified"))
				Expect(report.DriftedResources[0].Name).To(Equal("prometheus-server"))
			})
		})

		Context("when resource is deleted", func() {
			It("should detect deletion drift", func() {
				// Create desired state
				desiredState := map[string]*unstructured.Unstructured{
					"apps/v1/Deployment/default/prometheus-server": createDeployment("prometheus-server", "default"),
				}
				
				// Don't create the resource in cluster (simulating deletion)
				
				// Check drift
				report, err := manager.CheckDrift(ctx, platform, gitOps, desiredState)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(report.DriftDetected).To(BeTrue())
				Expect(report.DriftedResources).To(HaveLen(1))
				Expect(report.DriftedResources[0].DriftType).To(Equal("Deleted"))
				Expect(report.DriftedResources[0].Name).To(Equal("prometheus-server"))
			})
		})

		Context("when resource is added", func() {
			It("should detect addition drift", func() {
				// Create empty desired state
				desiredState := map[string]*unstructured.Unstructured{}
				
				// Create unexpected resource in cluster
				unexpected := createDeployment("rogue-deployment", "default")
				unexpected.SetLabels(map[string]string{
					"observability.io/platform": platform.Name,
				})
				Expect(manager.Client.Create(ctx, unexpected)).To(Succeed())
				
				// Check drift
				report, err := manager.CheckDrift(ctx, platform, gitOps, desiredState)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(report.DriftDetected).To(BeTrue())
				Expect(report.DriftedResources).To(HaveLen(1))
				Expect(report.DriftedResources[0].DriftType).To(Equal("Added"))
				Expect(report.DriftedResources[0].Name).To(Equal("rogue-deployment"))
			})
		})

		Context("with auto-remediation enabled", func() {
			It("should remediate drift automatically", func() {
				// Create mock metrics recorder
				mockMetrics := &MockMetricsRecorder{}
				manager.MetricsRecorder = mockMetrics
				
				// Expect metrics calls
				mockMetrics.On("RecordDriftDetected", platform.Name, mock.Anything, "Deleted").Once()
				mockMetrics.On("RecordDriftRemediated", platform.Name, mock.Anything, true).Once()
				mockMetrics.On("RecordDriftCheckDuration", platform.Name, mock.Anything).Once()
				
				// Create desired state
				desiredState := map[string]*unstructured.Unstructured{
					"apps/v1/Deployment/default/prometheus-server": createDeployment("prometheus-server", "default"),
				}
				
				// Check drift (resource doesn't exist, so it's deleted)
				report, err := manager.CheckDrift(ctx, platform, gitOps, desiredState)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(report.DriftDetected).To(BeTrue())
				Expect(report.Remediation).NotTo(BeNil())
				Expect(report.Remediation.Attempted).To(BeTrue())
				Expect(report.Remediation.Success).To(BeTrue())
				
				// Verify metrics were recorded
				mockMetrics.AssertExpectations(GinkgoT())
			})
		})

		Context("with notifications enabled", func() {
			It("should send drift notifications", func() {
				// Add notification policy
				gitOps.DriftDetection.NotificationPolicy = &gitopsv1beta1.NotificationPolicy{
					Channels: []gitopsv1beta1.NotificationChannel{
						{
							Type: "Slack",
							Config: map[string]string{
								"webhook": "https://hooks.slack.com/test",
							},
						},
					},
					Severity: "Warning",
				}
				
				// Create mock notifier
				mockNotifier := &MockNotifier{}
				manager.Notifier = mockNotifier
				
				// Expect notification
				mockNotifier.On("NotifyDrift", mock.Anything, mock.Anything).Return(nil).Once()
				
				// Create desired state
				desiredState := map[string]*unstructured.Unstructured{
					"apps/v1/Deployment/default/prometheus-server": createDeployment("prometheus-server", "default"),
				}
				
				// Check drift
				report, err := manager.CheckDrift(ctx, platform, gitOps, desiredState)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(report.DriftDetected).To(BeTrue())
				
				// Verify notification was sent
				mockNotifier.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Resource Normalization", func() {
		It("should normalize resources for comparison", func() {
			// Create resource with metadata that should be ignored
			resource := createDeployment("test", "default")
			resource.SetResourceVersion("12345")
			resource.SetUID("abc-123")
			resource.SetGeneration(5)
			resource.SetAnnotations(map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
				"custom-annotation": "keep-this",
			})
			
			// Normalize
			normalized := manager.normalizeForComparison(resource)
			
			// Check that certain fields are removed
			Expect(normalized.GetResourceVersion()).To(BeEmpty())
			Expect(normalized.GetUID()).To(BeEmpty())
			Expect(normalized.GetGeneration()).To(BeZero())
			
			// Check that kubectl annotation is removed but custom annotation is kept
			annotations := normalized.GetAnnotations()
			Expect(annotations).NotTo(HaveKey("kubectl.kubernetes.io/last-applied-configuration"))
			Expect(annotations).To(HaveKeyWithValue("custom-annotation", "keep-this"))
		})
	})

	Describe("Checksum Calculation", func() {
		It("should calculate consistent checksums", func() {
			// Create two identical resources
			resource1 := createDeployment("test", "default")
			resource2 := createDeployment("test", "default")
			
			// Add different metadata that should be ignored
			resource1.SetResourceVersion("123")
			resource2.SetResourceVersion("456")
			
			// Calculate checksums
			checksum1 := manager.CalculateChecksum(resource1)
			checksum2 := manager.CalculateChecksum(resource2)
			
			// Checksums should be identical
			Expect(checksum1).To(Equal(checksum2))
			Expect(checksum1).To(HaveLen(64)) // SHA256 in hex
		})

		It("should calculate different checksums for different resources", func() {
			// Create two different resources
			resource1 := createDeployment("test1", "default")
			resource2 := createDeployment("test2", "default")
			
			// Calculate checksums
			checksum1 := manager.CalculateChecksum(resource1)
			checksum2 := manager.CalculateChecksum(resource2)
			
			// Checksums should be different
			Expect(checksum1).NotTo(Equal(checksum2))
		})
	})

	Describe("Drift History", func() {
		It("should store drift history", func() {
			// Create drift report
			report := &DriftReport{
				Platform:      platform.Name,
				Namespace:     platform.Namespace,
				CheckTime:     time.Now(),
				DriftDetected: true,
				DriftedResources: []gitopsv1beta1.DriftedResource{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "prometheus-server",
						DriftType:  "Modified",
					},
				},
			}
			
			// Store history
			err := manager.StoreDriftHistory(ctx, platform, report)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify ConfigMap was created
			historyMap := &corev1.ConfigMap{}
			err = manager.Client.Get(ctx, client.ObjectKey{
				Name:      platform.Name + "-drift-history",
				Namespace: platform.Namespace,
			}, historyMap)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(historyMap.Data).To(HaveLen(1))
		})

		It("should limit drift history entries", func() {
			// Create ConfigMap with many entries
			historyMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      platform.Name + "-drift-history",
					Namespace: platform.Namespace,
				},
				Data: make(map[string]string),
			}
			
			// Add 105 entries
			for i := 0; i < 105; i++ {
				timestamp := time.Now().Add(time.Duration(-i) * time.Hour).Format(time.RFC3339)
				historyMap.Data[timestamp] = "{}"
			}
			
			Expect(manager.Client.Create(ctx, historyMap)).To(Succeed())
			
			// Store new history
			report := &DriftReport{
				Platform:  platform.Name,
				Namespace: platform.Namespace,
				CheckTime: time.Now(),
			}
			
			err := manager.StoreDriftHistory(ctx, platform, report)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify only 100 entries remain
			err = manager.Client.Get(ctx, client.ObjectKey{
				Name:      historyMap.Name,
				Namespace: historyMap.Namespace,
			}, historyMap)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(historyMap.Data).To(HaveLen(100))
		})
	})
})

// Helper functions

func createDeployment(name, namespace string) *unstructured.Unstructured {
	deployment := &unstructured.Unstructured{}
	deployment.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	deployment.SetName(name)
	deployment.SetNamespace(namespace)
	deployment.Object["spec"] = map[string]interface{}{
		"replicas": int32(1),
		"selector": map[string]interface{}{
			"matchLabels": map[string]interface{}{
				"app": name,
			},
		},
		"template": map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"app": name,
				},
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  name,
						"image": "nginx:latest",
					},
				},
			},
		},
	}
	return deployment
}

// Mock types

type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) NotifyDrift(ctx context.Context, drift *DriftReport) error {
	args := m.Called(ctx, drift)
	return args.Error(0)
}

type MockMetricsRecorder struct {
	mock.Mock
}

func (m *MockMetricsRecorder) RecordDriftDetected(platform, resource string, driftType string) {
	m.Called(platform, resource, driftType)
}

func (m *MockMetricsRecorder) RecordDriftRemediated(platform, resource string, success bool) {
	m.Called(platform, resource, success)
}

func (m *MockMetricsRecorder) RecordDriftCheckDuration(platform string, duration time.Duration) {
	m.Called(platform, duration)
}
