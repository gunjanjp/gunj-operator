/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package security

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestSecurityValidator_ValidateSecurity(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)

	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErrs int
		errTexts []string
	}{
		{
			name: "restricted security level - valid configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(true),
									RunAsUser:    ptr.To(int64(1000)),
									RunAsGroup:   ptr.To(int64(1000)),
									FSGroup:      ptr.To(int64(1000)),
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
									ReadOnlyRootFilesystem:   ptr.To(true),
									RunAsNonRoot:            ptr.To(true),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
							},
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
							Ingress: []observabilityv1beta1.NetworkPolicyRule{
								{
									From: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{"name": "gunj-system"},
											},
										},
									},
								},
							},
							Egress: []observabilityv1beta1.NetworkPolicyRule{
								{
									To: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "restricted security level - missing security context",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							// Missing SecurityContext
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
							Ingress: []observabilityv1beta1.NetworkPolicyRule{
								{
									From: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{"name": "gunj-system"},
											},
										},
									},
								},
							},
							Egress: []observabilityv1beta1.NetworkPolicyRule{
								{
									To: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErrs: 1,
			errTexts: []string{"Prometheus must have security context defined"},
		},
		{
			name: "restricted security level - root user",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(false), // Invalid
									RunAsUser:    ptr.To(int64(0)), // Invalid root user
									RunAsGroup:   ptr.To(int64(0)), // Invalid root group
									FSGroup:      ptr.To(int64(0)), // Invalid FSGroup
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(true), // Invalid
									ReadOnlyRootFilesystem:   ptr.To(false), // Invalid
									RunAsNonRoot:            ptr.To(false), // Invalid
								},
							},
						},
					},
				},
			},
			wantErrs: 11, // Multiple validation errors
			errTexts: []string{
				"must be true for restricted security level",
				"must be >= 1000 for restricted security level",
			},
		},
		{
			name: "baseline security level - privileged container",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "baseline",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								ContainerSecurityContext: &corev1.SecurityContext{
									Privileged: ptr.To(true), // Not allowed in baseline
								},
							},
						},
					},
				},
			},
			wantErrs: 1,
			errTexts: []string{"privileged containers not allowed in baseline security level"},
		},
		{
			name: "baseline security level - dangerous capabilities",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "baseline",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								ContainerSecurityContext: &corev1.SecurityContext{
									Capabilities: &corev1.Capabilities{
										Add: []corev1.Capability{"SYS_ADMIN", "NET_ADMIN"}, // Dangerous
									},
								},
							},
						},
					},
				},
			},
			wantErrs: 2,
			errTexts: []string{"not allowed in baseline security level"},
		},
		{
			name: "missing network policies",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(true),
									RunAsUser:    ptr.To(int64(1000)),
									RunAsGroup:   ptr.To(int64(1000)),
									FSGroup:      ptr.To(int64(1000)),
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
									ReadOnlyRootFilesystem:   ptr.To(true),
									RunAsNonRoot:            ptr.To(true),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
							},
						},
					},
					// Missing Security spec with NetworkPolicy
				},
			},
			wantErrs: 1,
			errTexts: []string{"network policy configuration required"},
		},
		{
			name: "missing security annotations",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					// Missing required annotations
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(true),
									RunAsUser:    ptr.To(int64(1000)),
									RunAsGroup:   ptr.To(int64(1000)),
									FSGroup:      ptr.To(int64(1000)),
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
									ReadOnlyRootFilesystem:   ptr.To(true),
									RunAsNonRoot:            ptr.To(true),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
							},
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
							Ingress: []observabilityv1beta1.NetworkPolicyRule{
								{
									From: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{"name": "gunj-system"},
											},
										},
									},
								},
							},
							Egress: []observabilityv1beta1.NetworkPolicyRule{
								{
									To: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErrs: 2,
			errTexts: []string{"security annotation", "is required"},
		},
		{
			name: "sensitive environment variables",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							ExtraEnvVars: []corev1.EnvVar{
								{
									Name:  "ADMIN_PASSWORD",
									Value: "plaintext-password", // Should use secret
								},
								{
									Name:  "API_KEY",
									Value: "my-api-key", // Should use secret
								},
							},
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(true),
									RunAsUser:    ptr.To(int64(1000)),
									RunAsGroup:   ptr.To(int64(1000)),
									FSGroup:      ptr.To(int64(1000)),
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
									ReadOnlyRootFilesystem:   ptr.To(true),
									RunAsNonRoot:            ptr.To(true),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
							},
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
							Ingress: []observabilityv1beta1.NetworkPolicyRule{
								{
									From: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{"name": "gunj-system"},
											},
										},
									},
								},
							},
							Egress: []observabilityv1beta1.NetworkPolicyRule{
								{
									To: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErrs: 2,
			errTexts: []string{"should use valueFrom.secretKeyRef instead of plaintext value"},
		},
		{
			name: "invalid security annotation values",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "invalid-level",
						"security.gunj-operator.io/compliance-profile": "invalid-profile",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(true),
									RunAsUser:    ptr.To(int64(1000)),
									RunAsGroup:   ptr.To(int64(1000)),
									FSGroup:      ptr.To(int64(1000)),
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
									ReadOnlyRootFilesystem:   ptr.To(true),
									RunAsNonRoot:            ptr.To(true),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
							},
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
							Ingress: []observabilityv1beta1.NetworkPolicyRule{
								{
									From: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{"name": "gunj-system"},
											},
										},
									},
								},
							},
							Egress: []observabilityv1beta1.NetworkPolicyRule{
								{
									To: []observabilityv1beta1.NetworkPolicyPeer{
										{
											NamespaceSelector: &metav1.LabelSelector{},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErrs: 2,
			errTexts: []string{
				"must be one of: privileged, baseline, restricted",
				"must be one of: cis, nist, pci-dss, hipaa, soc2, custom",
			},
		},
		{
			name: "network policy without rules",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
						"security.gunj-operator.io/compliance-profile": "cis",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								PodSecurityContext: &corev1.PodSecurityContext{
									RunAsNonRoot: ptr.To(true),
									RunAsUser:    ptr.To(int64(1000)),
									RunAsGroup:   ptr.To(int64(1000)),
									FSGroup:      ptr.To(int64(1000)),
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
								ContainerSecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
									ReadOnlyRootFilesystem:   ptr.To(true),
									RunAsNonRoot:            ptr.To(true),
									Capabilities: &corev1.Capabilities{
										Drop: []corev1.Capability{"ALL"},
									},
									SeccompProfile: &corev1.SeccompProfile{
										Type: corev1.SeccompProfileTypeRuntimeDefault,
									},
								},
							},
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
							// Missing ingress and egress rules
						},
					},
				},
			},
			wantErrs: 2,
			errTexts: []string{
				"at least one ingress rule must be defined",
				"at least one egress rule must be defined",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.platform).
				Build()

			validator := NewSecurityValidator(client)
			errs := validator.ValidateSecurity(context.Background(), tt.platform)

			assert.Equal(t, tt.wantErrs, len(errs), "unexpected number of errors")

			// Check for specific error messages
			for _, expectedText := range tt.errTexts {
				found := false
				for _, err := range errs {
					if containsString(err.ErrorBody(), expectedText) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error containing '%s' not found", expectedText)
			}
		})
	}
}

func TestSecurityValidator_GenerateSecurityRecommendations(t *testing.T) {
	tests := []struct {
		name             string
		platform         *observabilityv1beta1.ObservabilityPlatform
		wantRecommends   []string
		notWantRecommends []string
	}{
		{
			name: "baseline level platform",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-platform",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "baseline",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			wantRecommends: []string{
				"Consider upgrading from 'baseline' to 'restricted' security level",
				"Add security context for Prometheus component",
				"Enable network policies",
				"Enable TLS",
				"Enable audit logging",
			},
		},
		{
			name: "restricted level with missing features",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-platform",
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "restricted",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							SecurityContext: &observabilityv1beta1.SecurityContext{
								ContainerSecurityContext: &corev1.SecurityContext{
									ReadOnlyRootFilesystem:   ptr.To(false),
									AllowPrivilegeEscalation: ptr.To(true),
								},
							},
						},
					},
					Security: &observabilityv1beta1.SecuritySpec{
						TLS: observabilityv1beta1.TLSSpec{
							Enabled: true,
						},
						NetworkPolicy: &observabilityv1beta1.NetworkPolicySpec{
							Enabled: true,
						},
					},
				},
			},
			wantRecommends: []string{
				"Enable read-only root filesystem for Prometheus",
				"Disable privilege escalation for Prometheus",
				"Enable audit logging",
			},
			notWantRecommends: []string{
				"Enable TLS",
				"Enable network policies",
				"Consider upgrading",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			validator := NewSecurityValidator(client)

			recommendations := validator.GenerateSecurityRecommendations(tt.platform)

			// Check wanted recommendations
			for _, want := range tt.wantRecommends {
				found := false
				for _, rec := range recommendations {
					if containsString(rec, want) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected recommendation containing '%s' not found", want)
			}

			// Check unwanted recommendations
			for _, notWant := range tt.notWantRecommends {
				for _, rec := range recommendations {
					assert.NotContains(t, rec, notWant, "unexpected recommendation found")
				}
			}
		})
	}
}

func TestSecurityValidator_GetSecurityLevel(t *testing.T) {
	tests := []struct {
		name         string
		platform     *observabilityv1beta1.ObservabilityPlatform
		defaultLevel PodSecurityLevel
		wantLevel    PodSecurityLevel
	}{
		{
			name: "annotation override",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "privileged",
					},
				},
			},
			defaultLevel: PodSecurityLevelRestricted,
			wantLevel:    PodSecurityLevelPrivileged,
		},
		{
			name: "spec override",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Security: &observabilityv1beta1.SecuritySpec{
						PodSecurityPolicy: "baseline",
					},
				},
			},
			defaultLevel: PodSecurityLevelRestricted,
			wantLevel:    PodSecurityLevelBaseline,
		},
		{
			name:         "default level",
			platform:     &observabilityv1beta1.ObservabilityPlatform{},
			defaultLevel: PodSecurityLevelRestricted,
			wantLevel:    PodSecurityLevelRestricted,
		},
		{
			name: "annotation takes precedence over spec",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"security.gunj-operator.io/pod-security-level": "baseline",
					},
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Security: &observabilityv1beta1.SecuritySpec{
						PodSecurityPolicy: "restricted",
					},
				},
			},
			defaultLevel: PodSecurityLevelPrivileged,
			wantLevel:    PodSecurityLevelBaseline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().Build()
			validator := NewSecurityValidator(client)
			validator.DefaultSecurityLevel = tt.defaultLevel

			level := validator.getSecurityLevel(tt.platform)
			assert.Equal(t, tt.wantLevel, level)
		})
	}
}

// Helper function to check if a string contains another string
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (contains(s[1:], substr) || contains(s[:len(s)-1], substr) || (len(substr) <= len(s) && s[:len(substr)] == substr) || (len(substr) <= len(s) && s[len(s)-len(substr):] == substr)))
}
