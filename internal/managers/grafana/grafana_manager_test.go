/*
Copyright 2025.

Licensed under the MIT License.
*/

package grafana

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestNewGrafanaManager(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	
	manager := NewGrafanaManager(client, scheme)
	assert.NotNil(t, manager)
}

func TestGrafanaManager_Reconcile(t *testing.T) {
	tests := []struct {
		name      string
		platform  *observabilityv1beta1.ObservabilityPlatform
		wantErr   bool
		wantSkip  bool
	}{
		{
			name: "successful reconciliation",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
							Replicas: 2,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
							AdminPassword: "test-password",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "grafana disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: false,
							Version: "10.2.0",
						},
					},
				},
			},
			wantErr:  false,
			wantSkip: true,
		},
		{
			name: "grafana with ingress",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
							Ingress: &observabilityv1beta1.IngressConfig{
								Enabled:   true,
								ClassName: "nginx",
								Host:      "grafana.example.com",
								TLS: &observabilityv1beta1.IngressTLS{
									Enabled:    true,
									SecretName: "grafana-tls",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			_ = networkingv1.AddToScheme(scheme)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.platform).
				Build()

			m := &GrafanaManager{
				Client: client,
				Scheme: scheme,
			}

			err := m.Reconcile(context.Background(), tt.platform)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if !tt.wantSkip {
				// Check if resources were created
				// Check Secret
				secret := &corev1.Secret{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      m.getAdminSecretName(tt.platform),
					Namespace: tt.platform.Namespace,
				}, secret)
				assert.NoError(t, err)
				assert.NotEmpty(t, secret.Data["admin-password"])

				// Check ConfigMap
				cm := &corev1.ConfigMap{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      m.getConfigMapName(tt.platform),
					Namespace: tt.platform.Namespace,
				}, cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data, "grafana.ini")

				// Check Service
				svc := &corev1.Service{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      m.getServiceName(tt.platform),
					Namespace: tt.platform.Namespace,
				}, svc)
				assert.NoError(t, err)

				// Check Deployment
				deployment := &appsv1.Deployment{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      m.getDeploymentName(tt.platform),
					Namespace: tt.platform.Namespace,
				}, deployment)
				assert.NoError(t, err)
				assert.Equal(t, tt.platform.Spec.Components.Grafana.Replicas, *deployment.Spec.Replicas)

				// Check Ingress if enabled
				if tt.platform.Spec.Components.Grafana.Ingress != nil && tt.platform.Spec.Components.Grafana.Ingress.Enabled {
					ingress := &networkingv1.Ingress{}
					err = client.Get(context.Background(), types.NamespacedName{
						Name:      m.getIngressName(tt.platform),
						Namespace: tt.platform.Namespace,
					}, ingress)
					assert.NoError(t, err)
					assert.Equal(t, tt.platform.Spec.Components.Grafana.Ingress.Host, ingress.Spec.Rules[0].Host)
				}
			}
		})
	}
}

func TestGrafanaManager_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		platform       *observabilityv1beta1.ObservabilityPlatform
		deployment     *appsv1.Deployment
		expectedPhase  string
		expectedReady  int32
	}{
		{
			name: "all replicas ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:  true,
							Version:  "10.2.0",
							Replicas: 2,
						},
					},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "grafana-test-platform",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &[]int32{2}[0],
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      2,
					ReadyReplicas: 2,
				},
			},
			expectedPhase: "Ready",
			expectedReady: 2,
		},
		{
			name: "partial replicas ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:  true,
							Version:  "10.2.0",
							Replicas: 3,
						},
					},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "grafana-test-platform",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &[]int32{3}[0],
				},
				Status: appsv1.DeploymentStatus{
					Replicas:      3,
					ReadyReplicas: 1,
				},
			},
			expectedPhase: "Degraded",
			expectedReady: 1,
		},
		{
			name: "grafana disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: false,
						},
					},
				},
			},
			expectedPhase: "Disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)

			builder := fake.NewClientBuilder().WithScheme(scheme)
			if tt.deployment != nil {
				builder = builder.WithObjects(tt.deployment)
			}
			client := builder.Build()

			m := &GrafanaManager{
				Client: client,
				Scheme: scheme,
			}

			status, err := m.GetStatus(context.Background(), tt.platform)
			assert.NoError(t, err)
			assert.NotNil(t, status)
			assert.Equal(t, tt.expectedPhase, status.Phase)
			assert.Equal(t, tt.expectedReady, status.Ready)
		})
	}
}

func TestGrafanaManager_Validate(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Version: "10.2.0",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid version format",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Version: "v10.2.0", // Should not have 'v' prefix
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "should not start with 'v'",
		},
		{
			name: "ingress without host",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Version: "10.2.0",
							Ingress: &observabilityv1beta1.IngressConfig{
								Enabled: true,
								Host:    "",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "ingress host is required",
		},
		{
			name: "smtp without required fields",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Version: "10.2.0",
							SMTP: &observabilityv1beta1.SMTPConfig{
								Host: "",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "SMTP host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GrafanaManager{}
			err := m.Validate(tt.platform)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGrafanaManager_GetServiceURL(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "monitoring",
		},
	}

	m := &GrafanaManager{}
	url := m.GetServiceURL(platform)
	assert.Equal(t, "http://grafana-test-platform.monitoring.svc.cluster.local:3000", url)
}

func TestGrafanaManager_Delete(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Grafana: &observabilityv1beta1.GrafanaSpec{
					Enabled: true,
					Version: "10.2.0",
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)

	// Create resources first
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-test-platform-admin",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"admin-password": []byte("test"),
		},
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-test-platform-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"grafana.ini": "test",
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-test-platform",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 3000},
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-test-platform",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "grafana"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "grafana"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "grafana", Image: "grafana/grafana:10.2.0"},
					},
				},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, secret, cm, svc, deployment).
		Build()

	m := &GrafanaManager{
		Client: client,
		Scheme: scheme,
	}

	err := m.Delete(context.Background(), platform)
	assert.NoError(t, err)

	// Verify resources were deleted
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      "grafana-test-platform",
		Namespace: "default",
	}, deployment)
	assert.Error(t, err)
}

func TestGrafanaManager_generatePassword(t *testing.T) {
	m := &GrafanaManager{}
	
	// Generate multiple passwords and ensure they're different
	passwords := make(map[string]bool)
	for i := 0; i < 10; i++ {
		pwd := m.generatePassword()
		assert.Len(t, pwd, 16)
		assert.NotContains(t, passwords, pwd, "Generated duplicate password")
		passwords[pwd] = true
	}
}

func TestGrafanaManager_ConfigureDataSources(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Prometheus: &observabilityv1beta1.PrometheusSpec{
					Enabled: true,
				},
				Loki: &observabilityv1beta1.LokiSpec{
					Enabled: true,
				},
				Tempo: &observabilityv1beta1.TempoSpec{
					Enabled: true,
				},
				Grafana: &observabilityv1beta1.GrafanaSpec{
					Enabled: true,
					Version: "10.2.0",
					DataSources: []observabilityv1beta1.DataSourceConfig{
						{
							Name:      "External-Prometheus",
							Type:      "prometheus",
							URL:       "http://prometheus.external.com:9090",
							IsDefault: false,
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-test-platform",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "grafana"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "grafana"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "grafana", Image: "grafana/grafana:10.2.0"},
					},
				},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, deployment).
		Build()

	m := &GrafanaManager{
		Client: client,
		Scheme: scheme,
	}

	err := m.ConfigureDataSources(context.Background(), platform)
	assert.NoError(t, err)

	// Check that datasource ConfigMap was created
	cm := &corev1.ConfigMap{}
	err = client.Get(context.Background(), types.NamespacedName{
		Name:      m.getDataSourceConfigMapName(platform),
		Namespace: platform.Namespace,
	}, cm)
	assert.NoError(t, err)
	assert.Contains(t, cm.Data, "datasources.yaml")
	
	// Verify datasources content
	dsConfig := cm.Data["datasources.yaml"]
	assert.Contains(t, dsConfig, "Prometheus")
	assert.Contains(t, dsConfig, "Loki")
	assert.Contains(t, dsConfig, "Tempo")
	assert.Contains(t, dsConfig, "External-Prometheus")
}

func TestGrafanaManager_buildDeploymentSpec(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Global: observabilityv1beta1.GlobalConfig{
				NodeSelector: map[string]string{
					"node-type": "monitoring",
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "monitoring",
						Operator: corev1.TolerationOpEqual,
						Value:    "true",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
			},
		},
	}

	grafanaSpec := &observabilityv1beta1.GrafanaSpec{
		Replicas: 2,
		Version:  "10.2.0",
		Plugins:  []string{"piechart-panel", "cloudwatch-datasource"},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	m := &GrafanaManager{}
	spec := m.buildDeploymentSpec(platform, grafanaSpec)

	assert.Equal(t, int32(2), *spec.Replicas)
	assert.Len(t, spec.Template.Spec.InitContainers, 1)
	assert.Equal(t, "install-plugins", spec.Template.Spec.InitContainers[0].Name)
	assert.Contains(t, spec.Template.Spec.InitContainers[0].Command[2], "piechart-panel")
	assert.Equal(t, platform.Spec.Global.NodeSelector, spec.Template.Spec.NodeSelector)
	assert.Equal(t, platform.Spec.Global.Tolerations, spec.Template.Spec.Tolerations)
}

func TestGrafanaManager_generateGrafanaConfig(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-platform",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Global: observabilityv1beta1.GlobalConfig{
				LogLevel: "debug",
			},
		},
	}

	grafanaSpec := &observabilityv1beta1.GrafanaSpec{
		SMTP: &observabilityv1beta1.SMTPConfig{
			Host:     "smtp.example.com",
			Port:     587,
			User:     "user@example.com",
			Password: "password",
			From:     "grafana@example.com",
			TLS:      true,
		},
	}

	m := &GrafanaManager{}
	config := m.generateGrafanaConfig(platform, grafanaSpec)

	assert.Contains(t, config, "level = debug")
	assert.Contains(t, config, "[smtp]")
	assert.Contains(t, config, "enabled = true")
	assert.Contains(t, config, "host = smtp.example.com:587")
	assert.Contains(t, config, "from_address = grafana@example.com")
	assert.Contains(t, config, "skip_verify = false")
}
