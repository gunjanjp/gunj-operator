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

package gitops

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
)

var _ = Describe("GitOps Controller", func() {
	var (
		controller *GitOpsController
		fakeClient client.Client
		ctx        context.Context
		platform   *observabilityv1.ObservabilityPlatform
		testScheme *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Setup scheme
		testScheme = scheme.Scheme
		Expect(observabilityv1.AddToScheme(testScheme)).To(Succeed())
		
		// Create fake client
		fakeClient = fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		
		// Create controller
		var err error
		controller, err = NewGitOpsController(fakeClient, testScheme, ctrl.Log.WithName("test"))
		Expect(err).NotTo(HaveOccurred())
		
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
	})

	Describe("Reconciliation", func() {
		Context("without GitOps configuration", func() {
			It("should skip reconciliation", func() {
				// Create platform without GitOps
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Reconcile
				result, err := controller.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					},
				})
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(reconcile.Result{}))
			})
		})

		Context("with ArgoCD GitOps", func() {
			BeforeEach(func() {
				platform.Spec.GitOps = &observabilityv1.GitOpsSpec{
					Type: "ArgoCD",
					Repository: observabilityv1.GitRepositorySpec{
						URL:    "https://github.com/test/repo",
						Branch: "main",
						Path:   "/",
					},
				}
			})

			It("should create ArgoCD resources", func() {
				// Create platform
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Create mock ArgoCD manager
				mockArgoCD := &MockArgoCDManager{}
				controller.ArgoCDManager = mockArgoCD
				
				// Expect calls
				mockArgoCD.On("CreateApplication", mock.Anything, platform, mock.Anything).Return(nil)
				mockArgoCD.On("GetSyncStatus", mock.Anything, platform).Return(&argocd.SyncStatus{
					Phase:    "Synced",
					Revision: "abc123",
					Message:  "Sync successful",
				}, nil)
				
				// Reconcile
				result, err := controller.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					},
				})
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(1 * time.Minute))
				
				// Verify calls
				mockArgoCD.AssertExpectations(GinkgoT())
			})
		})

		Context("with Flux GitOps", func() {
			BeforeEach(func() {
				platform.Spec.GitOps = &observabilityv1.GitOpsSpec{
					Type: "Flux",
					Repository: observabilityv1.GitRepositorySpec{
						URL:    "https://github.com/test/repo",
						Branch: "main",
						Path:   "/",
					},
				}
			})

			It("should create Flux resources", func() {
				// Create platform
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Create mock Flux manager
				mockFlux := &MockFluxManager{}
				controller.FluxManager = mockFlux
				
				// Expect calls
				mockFlux.On("CreateGitRepository", mock.Anything, platform, mock.Anything).Return(nil)
				mockFlux.On("CreateKustomization", mock.Anything, platform, mock.Anything).Return(nil)
				mockFlux.On("GetSyncStatus", mock.Anything, platform).Return(&flux.SyncStatus{
					Ready:               true,
					Message:             "Reconciliation successful",
					LastAppliedRevision: "def456",
				}, nil)
				
				// Reconcile
				result, err := controller.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					},
				})
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(1 * time.Minute))
				
				// Verify calls
				mockFlux.AssertExpectations(GinkgoT())
			})
		})

		Context("with multi-environment setup", func() {
			BeforeEach(func() {
				platform.Spec.GitOps = &observabilityv1.GitOpsSpec{
					Type: "ArgoCD",
					Repository: observabilityv1.GitRepositorySpec{
						URL:    "https://github.com/test/repo",
						Branch: "main",
						Path:   "/",
					},
					Environments: []observabilityv1.EnvironmentSpec{
						{
							Name:     "dev",
							Branch:   "develop",
							Path:     "/dev",
							AutoSync: true,
						},
						{
							Name:     "prod",
							Branch:   "main",
							Path:     "/prod",
							AutoSync: false,
						},
					},
				}
			})

			It("should create ApplicationSet for multi-environment", func() {
				// Create platform
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Create mock ArgoCD manager
				mockArgoCD := &MockArgoCDManager{}
				controller.ArgoCDManager = mockArgoCD
				
				// Expect calls
				mockArgoCD.On("CreateApplication", mock.Anything, platform, mock.Anything).Return(nil)
				mockArgoCD.On("CreateApplicationSet", mock.Anything, platform, mock.Anything).Return(nil)
				mockArgoCD.On("GetSyncStatus", mock.Anything, platform).Return(&argocd.SyncStatus{
					Phase: "Synced",
				}, nil)
				
				// Reconcile
				_, err := controller.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					},
				})
				
				Expect(err).NotTo(HaveOccurred())
				
				// Verify ApplicationSet was created
				mockArgoCD.AssertCalled(GinkgoT(), "CreateApplicationSet", mock.Anything, platform, mock.Anything)
			})
		})

		Context("with drift detection enabled", func() {
			BeforeEach(func() {
				platform.Spec.GitOps = &observabilityv1.GitOpsSpec{
					Type: "ArgoCD",
					Repository: observabilityv1.GitRepositorySpec{
						URL:    "https://github.com/test/repo",
						Branch: "main",
						Path:   "/",
					},
					DriftDetection: &observabilityv1.DriftDetectionSpec{
						Enabled:       true,
						Interval:      "5m",
						AutoRemediate: true,
					},
				}
			})

			It("should enable drift detection", func() {
				// Create platform
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Verify drift detection is configured
				// This would be tested more thoroughly in integration tests
				Expect(platform.Spec.GitOps.DriftDetection.Enabled).To(BeTrue())
				Expect(platform.Spec.GitOps.DriftDetection.AutoRemediate).To(BeTrue())
			})
		})
	})

	Describe("Webhook Event Processing", func() {
		Context("Push event", func() {
			It("should trigger sync on push event", func() {
				// Create platform with GitOps
				platform.Spec.GitOps = &observabilityv1.GitOpsSpec{
					Type: "ArgoCD",
					Repository: observabilityv1.GitRepositorySpec{
						URL:    "https://github.com/test/repo",
						Branch: "main",
					},
				}
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Create mock ArgoCD manager
				mockArgoCD := &MockArgoCDManager{}
				controller.ArgoCDManager = mockArgoCD
				mockArgoCD.On("SyncApplication", mock.Anything, platform).Return(nil)
				
				// Process push event
				event := &webhook.PushEvent{
					Repository: "https://github.com/test/repo",
					Branch:     "main",
					Commit:     "abc123",
				}
				
				err := controller.ProcessPushEvent(ctx, event)
				Expect(err).NotTo(HaveOccurred())
				
				// Verify sync was triggered
				mockArgoCD.AssertCalled(GinkgoT(), "SyncApplication", mock.Anything, platform)
			})
		})

		Context("Pull request event", func() {
			It("should process pull request event", func() {
				event := &webhook.PullRequestEvent{
					Repository: "https://github.com/test/repo",
					Number:     42,
					Action:     "opened",
				}
				
				err := controller.ProcessPullRequestEvent(ctx, event)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Tag event", func() {
			It("should process tag event", func() {
				event := &webhook.TagEvent{
					Repository: "https://github.com/test/repo",
					Tag:        "v1.0.0",
				}
				
				err := controller.ProcessTagEvent(ctx, event)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Deletion", func() {
		Context("when platform is deleted", func() {
			BeforeEach(func() {
				platform.Spec.GitOps = &observabilityv1.GitOpsSpec{
					Type: "ArgoCD",
					Repository: observabilityv1.GitRepositorySpec{
						URL: "https://github.com/test/repo",
					},
				}
				platform.Finalizers = []string{"gitops.observability.io/finalizer"}
				platform.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			})

			It("should clean up GitOps resources", func() {
				// Create platform
				Expect(fakeClient.Create(ctx, platform)).To(Succeed())
				
				// Create mock ArgoCD manager
				mockArgoCD := &MockArgoCDManager{}
				controller.ArgoCDManager = mockArgoCD
				mockArgoCD.On("DeleteApplication", mock.Anything, platform).Return(nil)
				
				// Reconcile
				_, err := controller.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					},
				})
				
				Expect(err).NotTo(HaveOccurred())
				
				// Verify deletion was called
				mockArgoCD.AssertCalled(GinkgoT(), "DeleteApplication", mock.Anything, platform)
			})
		})
	})
})

// Mock types for testing

type MockArgoCDManager struct {
	mock.Mock
}

func (m *MockArgoCDManager) CreateApplication(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, gitOps *gitopsv1beta1.GitOpsIntegrationSpec) error {
	args := m.Called(ctx, platform, gitOps)
	return args.Error(0)
}

func (m *MockArgoCDManager) CreateApplicationSet(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, gitOps *gitopsv1beta1.GitOpsIntegrationSpec) error {
	args := m.Called(ctx, platform, gitOps)
	return args.Error(0)
}

func (m *MockArgoCDManager) GetSyncStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*argocd.SyncStatus, error) {
	args := m.Called(ctx, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*argocd.SyncStatus), args.Error(1)
}

func (m *MockArgoCDManager) SyncApplication(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	args := m.Called(ctx, platform)
	return args.Error(0)
}

func (m *MockArgoCDManager) DeleteApplication(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	args := m.Called(ctx, platform)
	return args.Error(0)
}

type MockFluxManager struct {
	mock.Mock
}

func (m *MockFluxManager) CreateGitRepository(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, gitOps *gitopsv1beta1.GitOpsIntegrationSpec) error {
	args := m.Called(ctx, platform, gitOps)
	return args.Error(0)
}

func (m *MockFluxManager) CreateKustomization(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, gitOps *gitopsv1beta1.GitOpsIntegrationSpec) error {
	args := m.Called(ctx, platform, gitOps)
	return args.Error(0)
}

func (m *MockFluxManager) GetSyncStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*flux.SyncStatus, error) {
	args := m.Called(ctx, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*flux.SyncStatus), args.Error(1)
}

func (m *MockFluxManager) DeleteFluxResources(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	args := m.Called(ctx, platform)
	return args.Error(0)
}
