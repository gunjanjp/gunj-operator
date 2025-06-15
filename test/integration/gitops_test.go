package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("GitOps Integration Tests", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Second * 2
	)

	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = fmt.Sprintf("test-gitops-%d", time.Now().Unix())

		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterEach(func() {
		// Clean up namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	Context("Basic GitOps Deployment", func() {
		It("should create a GitOps deployment with Git sync", func() {
			gitops := &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gitops",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:    "https://github.com/example/observability-config",
					Branch:        "main",
					Path:          "environments/dev",
					SyncInterval:  "1m",
					SyncProvider:  observabilityv1.Git,
					AutoSync:      true,
				},
			}

			// Create GitOps deployment
			Expect(k8sClient.Create(ctx, gitops)).To(Succeed())

			// Wait for GitOps to be ready
			Eventually(func() observabilityv1.GitOpsPhase {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(Equal(observabilityv1.GitOpsPhaseReady))

			// Verify Git repository created
			gitRepo := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-git-repo", gitops.Name),
					Namespace: gitops.Namespace,
				}, gitRepo)
			}, timeout, interval).Should(Succeed())

			// Verify sync CronJob created
			cronJob := &corev1.ConfigMap{} // Using ConfigMap as placeholder for CronJob
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-sync", gitops.Name),
					Namespace: gitops.Namespace,
				}, cronJob)
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("ArgoCD Integration", func() {
		It("should create ArgoCD Application for GitOps deployment", func() {
			gitops := &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-argocd",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:   "https://github.com/example/observability-config",
					Branch:       "main",
					Path:         "environments/staging",
					SyncProvider: observabilityv1.ArgoCD,
					ArgoCDConfig: &observabilityv1.ArgoCDConfig{
						ApplicationName: "observability-staging",
						Project:         "default",
						SyncPolicy: &observabilityv1.ArgoCDSyncPolicy{
							Automated: &observabilityv1.ArgoCDAutomatedSync{
								Prune:    true,
								SelfHeal: true,
							},
						},
					},
				},
			}

			// Create GitOps deployment
			Expect(k8sClient.Create(ctx, gitops)).To(Succeed())

			// Wait for ArgoCD Application to be created
			Eventually(func() bool {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return false
				}
				return updated.Status.ArgoCDAppName != ""
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Flux Integration", func() {
		It("should create Flux Kustomization for GitOps deployment", func() {
			gitops := &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flux",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:   "https://github.com/example/observability-config",
					Branch:       "main",
					Path:         "environments/prod",
					SyncProvider: observabilityv1.Flux,
					FluxConfig: &observabilityv1.FluxConfig{
						Interval: "5m",
						Prune:    &[]bool{true}[0],
						HealthChecks: []observabilityv1.FluxHealthCheck{
							{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       "prometheus",
								Namespace:  namespace,
							},
						},
					},
				},
			}

			// Create GitOps deployment
			Expect(k8sClient.Create(ctx, gitops)).To(Succeed())

			// Wait for status update
			Eventually(func() observabilityv1.GitOpsPhase {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(Equal(observabilityv1.GitOpsPhaseReady))
		})
	})

	Context("Health Checks and Rollback", func() {
		It("should perform health checks and rollback on failure", func() {
			gitops := &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rollback",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:   "https://github.com/example/observability-config",
					Branch:       "main",
					Path:         "environments/test",
					SyncProvider: observabilityv1.Git,
					HealthChecks: &observabilityv1.HealthCheckConfig{
						Enabled:  true,
						Interval: "30s",
						Timeout:  "2m",
						Resources: []observabilityv1.ResourceCheck{
							{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       "test-app",
								Namespace:  namespace,
							},
						},
					},
					RollbackConfig: &observabilityv1.RollbackConfig{
						AutoRollback:     true,
						FailureThreshold: 3,
					},
				},
			}

			// Create GitOps deployment
			Expect(k8sClient.Create(ctx, gitops)).To(Succeed())

			// Create a failing deployment to trigger rollback
			failingDeployment := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: namespace,
					Labels: map[string]string{
						"test": "failing",
					},
				},
				Data: map[string]string{
					"status": "unhealthy",
				},
			}
			Expect(k8sClient.Create(ctx, failingDeployment)).To(Succeed())

			// Wait for rollback to be triggered
			Eventually(func() bool {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return false
				}
				return updated.Status.RollbackStatus != nil && updated.Status.RollbackStatus.InProgress
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Multi-Environment Promotion", func() {
		var (
			devGitOps  *observabilityv1.GitOpsDeployment
			prodGitOps *observabilityv1.GitOpsDeployment
		)

		BeforeEach(func() {
			// Create dev environment
			devGitOps = &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-environment",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:   "https://github.com/example/observability-config",
					Branch:       "main",
					Path:         "environments/dev",
					SyncProvider: observabilityv1.Git,
				},
			}
			Expect(k8sClient.Create(ctx, devGitOps)).To(Succeed())

			// Create prod environment
			prodGitOps = &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-environment",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:   "https://github.com/example/observability-config",
					Branch:       "main",
					Path:         "environments/prod",
					SyncProvider: observabilityv1.Git,
				},
			}
			Expect(k8sClient.Create(ctx, prodGitOps)).To(Succeed())
		})

		It("should promote from dev to prod with approval", func() {
			promotion := &observabilityv1.GitOpsPromotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "promote-to-prod",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsPromotionSpec{
					SourceRef: corev1.ObjectReference{
						Name:      devGitOps.Name,
						Namespace: devGitOps.Namespace,
					},
					TargetRef: &corev1.ObjectReference{
						Name:      prodGitOps.Name,
						Namespace: prodGitOps.Namespace,
					},
					FromEnvironment: "dev",
					ToEnvironment:   "prod",
					Strategy:        observabilityv1.PromotionStrategyDirect,
					ApprovalPolicy: &observabilityv1.ApprovalPolicy{
						Required:     true,
						MinApprovals: 1,
						Approvers:    []string{"admin@example.com"},
						Timeout:      "1h",
					},
				},
			}

			// Create promotion
			Expect(k8sClient.Create(ctx, promotion)).To(Succeed())

			// Wait for approval request
			Eventually(func() observabilityv1.PromotionPhase {
				updated := &observabilityv1.GitOpsPromotion{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      promotion.Name,
					Namespace: promotion.Namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(Equal(observabilityv1.PromotionPhasePendingApproval))

			// Simulate approval
			approval := &observabilityv1.ApprovalRequest{}
			Eventually(func() error {
				approvalList := &observabilityv1.ApprovalRequestList{}
				err := k8sClient.List(ctx, approvalList, 
					client.InNamespace(namespace),
					client.MatchingLabels{
						"observability.io/promotion": promotion.Name,
					})
				if err != nil {
					return err
				}
				if len(approvalList.Items) == 0 {
					return fmt.Errorf("no approval request found")
				}
				approval = &approvalList.Items[0]
				return nil
			}, timeout, interval).Should(Succeed())

			// Approve the promotion
			approval.Status.Approved = true
			approval.Status.Approvers = []string{"admin@example.com"}
			approval.Status.ApprovedAt = &metav1.Time{Time: time.Now()}
			Expect(k8sClient.Status().Update(ctx, approval)).To(Succeed())

			// Wait for promotion to complete
			Eventually(func() observabilityv1.PromotionPhase {
				updated := &observabilityv1.GitOpsPromotion{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      promotion.Name,
					Namespace: promotion.Namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.Phase
			}, timeout, interval).Should(Equal(observabilityv1.PromotionPhaseCompleted))
		})
	})

	Context("Webhook Integration", func() {
		It("should handle GitHub webhook events", func() {
			gitops := &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "webhook-test",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:     "https://github.com/example/observability-config",
					Branch:         "main",
					Path:           "environments/webhook-test",
					SyncProvider:   observabilityv1.Git,
					WebhookSecret:  "test-secret",
					AutoSync:       true,
				},
			}

			// Create GitOps deployment
			Expect(k8sClient.Create(ctx, gitops)).To(Succeed())

			// Simulate webhook event by updating status
			Eventually(func() error {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return err
				}

				updated.Status.WebhookEvent = &observabilityv1.WebhookEventStatus{
					Type:       "push",
					CommitSHA:  "abc123",
					Branch:     "main",
					ReceivedAt: fmt.Sprintf("%d", time.Now().Unix()),
				}
				return k8sClient.Status().Update(ctx, updated)
			}, timeout, interval).Should(Succeed())

			// Verify sync triggered
			Eventually(func() string {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return ""
				}
				return updated.Status.LastSyncedCommit
			}, timeout, interval).Should(Equal("abc123"))
		})
	})

	Context("Drift Detection", func() {
		It("should detect and correct configuration drift", func() {
			gitops := &observabilityv1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "drift-test",
					Namespace: namespace,
				},
				Spec: observabilityv1.GitOpsDeploymentSpec{
					Repository:   "https://github.com/example/observability-config",
					Branch:       "main",
					Path:         "environments/drift-test",
					SyncProvider: observabilityv1.Git,
					DriftDetection: &observabilityv1.DriftDetectionConfig{
						Enabled:      true,
						Interval:     "30s",
						AutoCorrect:  true,
						MaxDriftSize: 5,
					},
				},
			}

			// Create GitOps deployment
			Expect(k8sClient.Create(ctx, gitops)).To(Succeed())

			// Create a resource that will drift
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "drift-test-config",
					Namespace: namespace,
					Labels: map[string]string{
						"observability.io/gitops": gitops.Name,
					},
				},
				Data: map[string]string{
					"key": "original-value",
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())

			// Modify the resource to create drift
			Eventually(func() error {
				updated := &corev1.ConfigMap{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
				}, updated)
				if err != nil {
					return err
				}
				updated.Data["key"] = "drifted-value"
				return k8sClient.Update(ctx, updated)
			}, timeout, interval).Should(Succeed())

			// Wait for drift to be detected and corrected
			Eventually(func() string {
				cm := &corev1.ConfigMap{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      configMap.Name,
					Namespace: configMap.Namespace,
				}, cm)
				if err != nil {
					return ""
				}
				return cm.Data["key"]
			}, timeout, interval).Should(Equal("original-value"))

			// Verify drift event recorded
			Eventually(func() bool {
				updated := &observabilityv1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      gitops.Name,
					Namespace: gitops.Namespace,
				}, updated)
				if err != nil {
					return false
				}
				return updated.Status.DriftStatus != nil && updated.Status.DriftStatus.DriftDetected
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Performance and Scale", func() {
		It("should handle multiple GitOps deployments concurrently", func() {
			const numDeployments = 10
			deployments := make([]*observabilityv1.GitOpsDeployment, numDeployments)

			// Create multiple GitOps deployments
			for i := 0; i < numDeployments; i++ {
				deployments[i] = &observabilityv1.GitOpsDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("scale-test-%d", i),
						Namespace: namespace,
					},
					Spec: observabilityv1.GitOpsDeploymentSpec{
						Repository:   "https://github.com/example/observability-config",
						Branch:       "main",
						Path:         fmt.Sprintf("environments/scale-test-%d", i),
						SyncProvider: observabilityv1.Git,
						AutoSync:     true,
					},
				}
				Expect(k8sClient.Create(ctx, deployments[i])).To(Succeed())
			}

			// Wait for all deployments to be ready
			for i := 0; i < numDeployments; i++ {
				Eventually(func() observabilityv1.GitOpsPhase {
					updated := &observabilityv1.GitOpsDeployment{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      deployments[i].Name,
						Namespace: deployments[i].Namespace,
					}, updated)
					if err != nil {
						return ""
					}
					return updated.Status.Phase
				}, timeout, interval).Should(Equal(observabilityv1.GitOpsPhaseReady))
			}
		})
	})
})

// Test helper functions
func createTestSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	return k8sClient.Create(ctx, secret)
}

func waitForGitOpsPhase(ctx context.Context, name, namespace string, phase observabilityv1.GitOpsPhase, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		gitops := &observabilityv1.GitOpsDeployment{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, gitops)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return gitops.Status.Phase == phase, nil
	})
}
