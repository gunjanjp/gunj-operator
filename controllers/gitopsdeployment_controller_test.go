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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("GitOpsDeployment Controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Second * 1
	)

	Context("When creating a GitOpsDeployment", func() {
		It("Should handle ArgoCD deployment", func() {
			ctx := context.Background()
			
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gitops-argocd",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create Git credentials secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "git-credentials",
					Namespace: ns.Name,
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"username": "test-user",
					"password": "test-token",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			// Create GitOpsDeployment
			deployment := &observabilityv1beta1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-argocd",
					Namespace: ns.Name,
				},
				Spec: observabilityv1beta1.GitOpsDeploymentSpec{
					GitProvider: "github",
					Repository: observabilityv1beta1.GitRepository{
						URL:    "https://github.com/test/repo.git",
						Branch: "main",
						Path:   "/test",
						SecretRef: &observabilityv1beta1.SecretReference{
							Name: secret.Name,
						},
					},
					GitOpsEngine: "argocd",
					ArgoCD: &observabilityv1beta1.ArgoCDConfig{
						ApplicationName: "test-app",
						Project:         "default",
					},
					Environments: []observabilityv1beta1.Environment{
						{
							Name:      "test",
							Namespace: "test-env",
						},
					},
					AutoSync: true,
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// Check that the deployment was created
			deploymentKey := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
			createdDeployment := &observabilityv1beta1.GitOpsDeployment{}
			
			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify initial status
			Expect(createdDeployment.Status.Phase).Should(BeEmpty())

			// Trigger reconciliation
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: deploymentKey,
			})
			Expect(err).NotTo(HaveOccurred())

			// Check status after reconciliation
			Eventually(func() string {
				err := k8sClient.Get(ctx, deploymentKey, createdDeployment)
				if err != nil {
					return ""
				}
				return createdDeployment.Status.Phase
			}, timeout, interval).Should(Equal("Pending"))

			// Verify finalizer was added
			Expect(createdDeployment.ObjectMeta.Finalizers).Should(ContainElement("gitopsdeployment.observability.io/finalizer"))
		})

		It("Should handle Flux deployment", func() {
			ctx := context.Background()
			
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gitops-flux",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create GitOpsDeployment with Flux
			deployment := &observabilityv1beta1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-flux",
					Namespace: ns.Name,
				},
				Spec: observabilityv1beta1.GitOpsDeploymentSpec{
					GitProvider: "github",
					Repository: observabilityv1beta1.GitRepository{
						URL:    "https://github.com/test/repo.git",
						Branch: "main",
						Path:   "/test",
					},
					GitOpsEngine: "flux",
					Flux: &observabilityv1beta1.FluxConfig{
						KustomizationName: "test-kustomization",
						Interval:          "5m",
					},
					Environments: []observabilityv1beta1.Environment{
						{
							Name:      "test",
							Namespace: "test-env",
						},
					},
					AutoSync: true,
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// Verify creation
			deploymentKey := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
			createdDeployment := &observabilityv1beta1.GitOpsDeployment{}
			
			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle drift detection configuration", func() {
			ctx := context.Background()
			
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gitops-drift",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create GitOpsDeployment with drift detection
			deployment := &observabilityv1beta1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-drift",
					Namespace: ns.Name,
				},
				Spec: observabilityv1beta1.GitOpsDeploymentSpec{
					GitProvider: "github",
					Repository: observabilityv1beta1.GitRepository{
						URL:    "https://github.com/test/repo.git",
						Branch: "main",
						Path:   "/test",
					},
					GitOpsEngine: "argocd",
					ArgoCD: &observabilityv1beta1.ArgoCDConfig{
						ApplicationName: "test-app",
					},
					Environments: []observabilityv1beta1.Environment{
						{
							Name:      "test",
							Namespace: "test-env",
						},
					},
					DriftDetection: &observabilityv1beta1.DriftDetectionConfig{
						Enabled:       true,
						CheckInterval: "1m",
						AutoRemediate: true,
						IgnoreFields: []string{
							"metadata.annotations",
							"status",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// Verify drift detection config
			deploymentKey := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
			createdDeployment := &observabilityv1beta1.GitOpsDeployment{}
			
			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdDeployment.Spec.DriftDetection).NotTo(BeNil())
			Expect(createdDeployment.Spec.DriftDetection.Enabled).To(BeTrue())
			Expect(createdDeployment.Spec.DriftDetection.AutoRemediate).To(BeTrue())
		})

		It("Should handle promotion workflow", func() {
			ctx := context.Background()
			
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gitops-promotion",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create GitOpsDeployment with promotion workflow
			deployment := &observabilityv1beta1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-promotion",
					Namespace: ns.Name,
				},
				Spec: observabilityv1beta1.GitOpsDeploymentSpec{
					GitProvider: "github",
					Repository: observabilityv1beta1.GitRepository{
						URL:    "https://github.com/test/repo.git",
						Branch: "main",
						Path:   "/test",
					},
					GitOpsEngine: "argocd",
					ArgoCD: &observabilityv1beta1.ArgoCDConfig{
						ApplicationName: "test-app",
					},
					Environments: []observabilityv1beta1.Environment{
						{
							Name:      "dev",
							Namespace: "test-dev",
						},
						{
							Name:      "staging",
							Namespace: "test-staging",
							PromotionPolicy: &observabilityv1beta1.PromotionPolicy{
								AutoPromotion:    true,
								FromEnvironment:  "dev",
								ApprovalRequired: false,
								Conditions: []observabilityv1beta1.PromotionCondition{
									{
										Type:   "HealthCheck",
										Status: "Healthy",
									},
								},
							},
						},
						{
							Name:      "prod",
							Namespace: "test-prod",
							PromotionPolicy: &observabilityv1beta1.PromotionPolicy{
								AutoPromotion:    true,
								FromEnvironment:  "staging",
								ApprovalRequired: true,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// Verify environments and promotion policies
			deploymentKey := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
			createdDeployment := &observabilityv1beta1.GitOpsDeployment{}
			
			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentKey, createdDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdDeployment.Spec.Environments).To(HaveLen(3))
			Expect(createdDeployment.Spec.Environments[1].PromotionPolicy).NotTo(BeNil())
			Expect(createdDeployment.Spec.Environments[1].PromotionPolicy.FromEnvironment).To(Equal("dev"))
		})
	})

	Context("When deleting a GitOpsDeployment", func() {
		It("Should clean up resources", func() {
			ctx := context.Background()
			
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gitops-delete",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create GitOpsDeployment
			deployment := &observabilityv1beta1.GitOpsDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-delete",
					Namespace: ns.Name,
				},
				Spec: observabilityv1beta1.GitOpsDeploymentSpec{
					GitProvider: "github",
					Repository: observabilityv1beta1.GitRepository{
						URL:    "https://github.com/test/repo.git",
						Branch: "main",
					},
					GitOpsEngine: "argocd",
					ArgoCD: &observabilityv1beta1.ArgoCDConfig{
						ApplicationName: "test-app",
					},
					Environments: []observabilityv1beta1.Environment{
						{
							Name:      "test",
							Namespace: "test-env",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// Delete the deployment
			deploymentKey := types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}
			Eventually(func() error {
				d := &observabilityv1beta1.GitOpsDeployment{}
				if err := k8sClient.Get(ctx, deploymentKey, d); err != nil {
					return err
				}
				return k8sClient.Delete(ctx, d)
			}, timeout, interval).Should(Succeed())

			// Verify deletion
			Eventually(func() bool {
				d := &observabilityv1beta1.GitOpsDeployment{}
				err := k8sClient.Get(ctx, deploymentKey, d)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
