/*
Copyright 2025.

Licensed under the MIT License.
*/

package grafana

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/managers"
)

const (
	// Component name
	componentName = "grafana"

	// Default values
	defaultPort         = 3000
	defaultImage        = "grafana/grafana"
	defaultDataPath     = "/var/lib/grafana"
	defaultLogsPath     = "/var/log/grafana"
	defaultPluginsPath  = "/var/lib/grafana/plugins"
	defaultConfigPath   = "/etc/grafana"

	// Labels
	labelComponent = "grafana"

	// Annotations
	annotationPasswordGenerated = "observability.io/grafana-password-generated"
)

// GrafanaManager manages Grafana deployments
type GrafanaManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// NewGrafanaManager creates a new Grafana manager
func NewGrafanaManager(client client.Client, scheme *runtime.Scheme) managers.GrafanaManager {
	return &GrafanaManager{
		Client: client,
		Scheme: scheme,
	}
}

// Reconcile reconciles the Grafana component
func (m *GrafanaManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Grafana component with provided configuration
func (m *GrafanaManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	log := log.FromContext(ctx).WithValues("component", componentName)

	// Check if Grafana is enabled
	if platform.Spec.Components.Grafana == nil || !platform.Spec.Components.Grafana.Enabled {
		log.V(1).Info("Grafana is disabled, skipping reconciliation")
		return nil
	}

	grafanaSpec := platform.Spec.Components.Grafana
	log.Info("Reconciling Grafana", "version", grafanaSpec.Version)

	// 1. Create/Update admin password secret
	if err := m.reconcileAdminSecret(ctx, platform, grafanaSpec); err != nil {
		return fmt.Errorf("failed to reconcile admin secret: %w", err)
	}

	// 2. Create ConfigMaps for datasources
	if err := m.reconcileDataSourceConfigMaps(ctx, platform, grafanaSpec); err != nil {
		return fmt.Errorf("failed to reconcile datasource ConfigMaps: %w", err)
	}

	// 3. Create ConfigMaps for dashboards
	if err := m.reconcileDashboardConfigMaps(ctx, platform, grafanaSpec); err != nil {
		return fmt.Errorf("failed to reconcile dashboard ConfigMaps: %w", err)
	}

	// 4. Create main configuration ConfigMap
	if err := m.reconcileConfigMap(ctx, platform, grafanaSpec); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMap: %w", err)
	}

	// 5. Create Service
	if err := m.reconcileService(ctx, platform); err != nil {
		return fmt.Errorf("failed to reconcile Service: %w", err)
	}

	// 6. Create Deployment
	if err := m.reconcileDeployment(ctx, platform, grafanaSpec); err != nil {
		return fmt.Errorf("failed to reconcile Deployment: %w", err)
	}

	// 7. Create Ingress if configured
	if grafanaSpec.Ingress != nil && grafanaSpec.Ingress.Enabled {
		if err := m.reconcileIngress(ctx, platform, grafanaSpec); err != nil {
			return fmt.Errorf("failed to reconcile Ingress: %w", err)
		}
	}

	log.Info("Grafana reconciliation completed successfully")
	return nil
}

// Delete removes the Grafana component resources
func (m *GrafanaManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	log.Info("Deleting Grafana resources")

	// Delete in reverse order of creation
	resources := []client.Object{
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getIngressName(platform),
				Namespace: platform.Namespace,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getDeploymentName(platform),
				Namespace: platform.Namespace,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getServiceName(platform),
				Namespace: platform.Namespace,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getConfigMapName(platform),
				Namespace: platform.Namespace,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getDataSourceConfigMapName(platform),
				Namespace: platform.Namespace,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getAdminSecretName(platform),
				Namespace: platform.Namespace,
			},
		},
	}

	for _, resource := range resources {
		if err := m.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete resource", "resource", resource.GetName())
		}
	}

	return nil
}

// GetStatus returns the current status of the Grafana component
func (m *GrafanaManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	log := log.FromContext(ctx).WithValues("component", componentName)

	status := &observabilityv1beta1.ComponentStatus{
		Phase: "Unknown",
	}

	// Check if Grafana is enabled
	if platform.Spec.Components.Grafana == nil || !platform.Spec.Components.Grafana.Enabled {
		status.Phase = "Disabled"
		status.Message = "Grafana is disabled"
		return status, nil
	}

	// Check Deployment status
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      m.getDeploymentName(platform),
		Namespace: platform.Namespace,
	}, deployment)

	if err != nil {
		status.Phase = "Failed"
		status.Message = fmt.Sprintf("Failed to get Deployment: %v", err)
		return status, nil
	}

	// Update status based on Deployment
	status.Version = platform.Spec.Components.Grafana.Version
	status.Replicas = deployment.Status.Replicas
	status.Ready = deployment.Status.ReadyReplicas

	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		status.Phase = "Ready"
		status.Message = fmt.Sprintf("All %d replicas are ready", status.Ready)
	} else if deployment.Status.ReadyReplicas > 0 {
		status.Phase = "Degraded"
		status.Message = fmt.Sprintf("%d/%d replicas are ready", status.Ready, status.Replicas)
	} else {
		status.Phase = "NotReady"
		status.Message = "No replicas are ready"
	}

	log.V(1).Info("Got Grafana status", "phase", status.Phase)
	return status, nil
}

// Validate validates the Grafana configuration
func (m *GrafanaManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Grafana == nil {
		return nil
	}

	grafana := platform.Spec.Components.Grafana

	// Validate version format (no 'v' prefix for Grafana)
	if strings.HasPrefix(grafana.Version, "v") {
		return fmt.Errorf("grafana version should not start with 'v', got: %s", grafana.Version)
	}

	// Validate resources
	if grafana.Resources.Requests != nil {
		if _, err := resource.ParseQuantity(grafana.Resources.Requests.Memory().String()); err != nil {
			return fmt.Errorf("invalid memory request: %w", err)
		}
		if _, err := resource.ParseQuantity(grafana.Resources.Requests.Cpu().String()); err != nil {
			return fmt.Errorf("invalid CPU request: %w", err)
		}
	}

	// Validate ingress configuration
	if grafana.Ingress != nil && grafana.Ingress.Enabled {
		if grafana.Ingress.Host == "" {
			return fmt.Errorf("ingress host is required when ingress is enabled")
		}
	}

	// Validate SMTP configuration
	if grafana.SMTP != nil {
		if grafana.SMTP.Host == "" {
			return fmt.Errorf("SMTP host is required when SMTP is configured")
		}
		if grafana.SMTP.Port == 0 {
			return fmt.Errorf("SMTP port is required when SMTP is configured")
		}
		if grafana.SMTP.From == "" {
			return fmt.Errorf("SMTP from address is required when SMTP is configured")
		}
	}

	return nil
}

// GetServiceURL returns the service URL for Grafana
func (m *GrafanaManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		m.getServiceName(platform),
		platform.Namespace,
		defaultPort)
}

// ImportDashboards imports dashboards into Grafana
func (m *GrafanaManager) ImportDashboards(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)

	// Dashboards are imported via provisioning ConfigMaps
	// This method could be used for dynamic dashboard imports via Grafana API
	log.Info("Dashboard import requested - dashboards are provisioned via ConfigMaps")

	return nil
}

// ConfigureDataSources configures data sources in Grafana
func (m *GrafanaManager) ConfigureDataSources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)

	// Update datasource ConfigMaps
	if err := m.reconcileDataSourceConfigMaps(ctx, platform, platform.Spec.Components.Grafana); err != nil {
		return fmt.Errorf("failed to update datasource ConfigMaps: %w", err)
	}

	// Restart Grafana pods to pick up changes
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      m.getDeploymentName(platform),
		Namespace: platform.Namespace,
	}, deployment)

	if err != nil {
		return fmt.Errorf("failed to get Deployment: %w", err)
	}

	// Add annotation to force pod restart
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["observability.io/restart-time"] = metav1.Now().Format(metav1.RFC3339)

	if err := m.Client.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update Deployment: %w", err)
	}

	log.Info("DataSources configuration updated")
	return nil
}

// reconcileAdminSecret creates or updates the admin password secret
func (m *GrafanaManager) reconcileAdminSecret(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) error {
	log := log.FromContext(ctx)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getAdminSecretName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, secret, func() error {
		// Set labels
		secret.Labels = m.getLabels(platform)

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, secret, m.Scheme); err != nil {
			return err
		}

		// Only generate password if not already set
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}

		// Use provided password or generate one
		if _, exists := secret.Data["admin-password"]; !exists {
			var password string
			if grafanaSpec.AdminPassword != "" {
				password = grafanaSpec.AdminPassword
			} else {
				password = m.generatePassword()
				// Mark that we generated the password
				if secret.Annotations == nil {
					secret.Annotations = make(map[string]string)
				}
				secret.Annotations[annotationPasswordGenerated] = "true"
			}
			secret.Data["admin-password"] = []byte(password)
			secret.Data["admin-user"] = []byte("admin")
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update admin secret: %w", err)
	}

	log.V(1).Info("Admin secret reconciled", "name", secret.Name)
	return nil
}

// reconcileDataSourceConfigMaps creates ConfigMaps for datasources
func (m *GrafanaManager) reconcileDataSourceConfigMaps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) error {
	log := log.FromContext(ctx)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getDataSourceConfigMapName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, configMap, func() error {
		// Set labels
		configMap.Labels = m.getLabels(platform)

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, configMap, m.Scheme); err != nil {
			return err
		}

		// Generate datasources YAML
		datasourcesYAML := m.generateDataSourcesConfig(platform, grafanaSpec)

		configMap.Data = map[string]string{
			"datasources.yaml": datasourcesYAML,
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update datasource ConfigMap: %w", err)
	}

	log.V(1).Info("DataSource ConfigMap reconciled", "name", configMap.Name)
	return nil
}

// reconcileDashboardConfigMaps creates ConfigMaps for dashboard provisioning
func (m *GrafanaManager) reconcileDashboardConfigMaps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) error {
	log := log.FromContext(ctx)

	// Create dashboard provider ConfigMap
	providerCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getDashboardProviderConfigMapName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, providerCM, func() error {
		// Set labels
		providerCM.Labels = m.getLabels(platform)

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, providerCM, m.Scheme); err != nil {
			return err
		}

		// Generate dashboard provider config
		providerYAML := m.generateDashboardProviderConfig(platform)

		providerCM.Data = map[string]string{
			"dashboards.yaml": providerYAML,
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update dashboard provider ConfigMap: %w", err)
	}

	// Create default dashboards ConfigMap
	dashboardsCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getDefaultDashboardsConfigMapName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, m.Client, dashboardsCM, func() error {
		// Set labels
		dashboardsCM.Labels = m.getLabels(platform)
		dashboardsCM.Labels["grafana_dashboard"] = "1"

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, dashboardsCM, m.Scheme); err != nil {
			return err
		}

		// Add default dashboards
		dashboardsCM.Data = map[string]string{
			"platform-overview.json": m.generatePlatformOverviewDashboard(),
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update default dashboards ConfigMap: %w", err)
	}

	log.V(1).Info("Dashboard ConfigMaps reconciled")
	return nil
}

// reconcileConfigMap creates or updates the main Grafana ConfigMap
func (m *GrafanaManager) reconcileConfigMap(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) error {
	log := log.FromContext(ctx)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getConfigMapName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, configMap, func() error {
		// Set labels
		configMap.Labels = m.getLabels(platform)

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, configMap, m.Scheme); err != nil {
			return err
		}

		// Generate grafana.ini
		grafanaINI := m.generateGrafanaConfig(platform, grafanaSpec)

		configMap.Data = map[string]string{
			"grafana.ini": grafanaINI,
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update ConfigMap: %w", err)
	}

	log.V(1).Info("ConfigMap reconciled", "name", configMap.Name)
	return nil
}

// reconcileService creates or updates the Grafana Service
func (m *GrafanaManager) reconcileService(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getServiceName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, service, func() error {
		// Set labels
		service.Labels = m.getLabels(platform)

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, service, m.Scheme); err != nil {
			return err
		}

		// Set spec
		service.Spec = corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: m.getSelectorLabels(platform),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       defaultPort,
					TargetPort: intstr.FromInt(defaultPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update Service: %w", err)
	}

	log.V(1).Info("Service reconciled", "name", service.Name)
	return nil
}

// reconcileDeployment creates or updates the Grafana Deployment
func (m *GrafanaManager) reconcileDeployment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) error {
	log := log.FromContext(ctx)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getDeploymentName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, deployment, func() error {
		// Set labels
		deployment.Labels = m.getLabels(platform)

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, deployment, m.Scheme); err != nil {
			return err
		}

		// Build Deployment spec
		deployment.Spec = m.buildDeploymentSpec(platform, grafanaSpec)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update Deployment: %w", err)
	}

	log.V(1).Info("Deployment reconciled", "name", deployment.Name)
	return nil
}

// reconcileIngress creates or updates the Grafana Ingress
func (m *GrafanaManager) reconcileIngress(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) error {
	log := log.FromContext(ctx)

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getIngressName(platform),
			Namespace: platform.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, ingress, func() error {
		// Set labels
		ingress.Labels = m.getLabels(platform)

		// Set annotations
		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
		}
		// Add user-provided annotations
		for k, v := range grafanaSpec.Ingress.Annotations {
			ingress.Annotations[k] = v
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, ingress, m.Scheme); err != nil {
			return err
		}

		// Build Ingress spec
		pathType := networkingv1.PathTypePrefix
		path := grafanaSpec.Ingress.Path
		if path == "" {
			path = "/"
		}

		ingress.Spec = networkingv1.IngressSpec{
			IngressClassName: &grafanaSpec.Ingress.ClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: grafanaSpec.Ingress.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: m.getServiceName(platform),
											Port: networkingv1.ServiceBackendPort{
												Number: defaultPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Add TLS if configured
		if grafanaSpec.Ingress.TLS != nil && grafanaSpec.Ingress.TLS.Enabled {
			ingress.Spec.TLS = []networkingv1.IngressTLS{
				{
					Hosts:      []string{grafanaSpec.Ingress.Host},
					SecretName: grafanaSpec.Ingress.TLS.SecretName,
				},
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update Ingress: %w", err)
	}

	log.V(1).Info("Ingress reconciled", "name", ingress.Name)
	return nil
}

// buildDeploymentSpec builds the Deployment specification
func (m *GrafanaManager) buildDeploymentSpec(platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) appsv1.DeploymentSpec {
	replicas := grafanaSpec.Replicas
	labels := m.getSelectorLabels(platform)

	// Build container
	container := corev1.Container{
		Name:  componentName,
		Image: fmt.Sprintf("%s:%s", defaultImage, grafanaSpec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: defaultPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: "GF_SECURITY_ADMIN_USER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.getAdminSecretName(platform),
						},
						Key: "admin-user",
					},
				},
			},
			{
				Name: "GF_SECURITY_ADMIN_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.getAdminSecretName(platform),
						},
						Key: "admin-password",
					},
				},
			},
			{
				Name:  "GF_PATHS_DATA",
				Value: defaultDataPath,
			},
			{
				Name:  "GF_PATHS_LOGS",
				Value: defaultLogsPath,
			},
			{
				Name:  "GF_PATHS_PLUGINS",
				Value: defaultPluginsPath,
			},
			{
				Name:  "GF_PATHS_PROVISIONING",
				Value: "/etc/grafana/provisioning",
			},
		},
		Resources: grafanaSpec.Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config",
				MountPath: defaultConfigPath,
				SubPath:   "grafana.ini",
			},
			{
				Name:      "datasources",
				MountPath: "/etc/grafana/provisioning/datasources",
			},
			{
				Name:      "dashboard-provider",
				MountPath: "/etc/grafana/provisioning/dashboards",
			},
			{
				Name:      "dashboards",
				MountPath: "/var/lib/grafana/dashboards",
			},
			{
				Name:      "data",
				MountPath: defaultDataPath,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/api/health",
					Port: intstr.FromInt(defaultPort),
				},
			},
			InitialDelaySeconds: 60,
			PeriodSeconds:       10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/api/health",
					Port: intstr.FromInt(defaultPort),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             &[]bool{true}[0],
			RunAsUser:                &[]int64{472}[0], // Grafana user
			AllowPrivilegeEscalation: &[]bool{false}[0],
			ReadOnlyRootFilesystem:   &[]bool{false}[0], // Grafana needs to write to disk
		},
	}

	// Add plugin installation if configured
	var initContainers []corev1.Container
	if len(grafanaSpec.Plugins) > 0 {
		pluginInstallCmd := fmt.Sprintf("grafana-cli plugins install %s", strings.Join(grafanaSpec.Plugins, " && grafana-cli plugins install "))
		initContainers = append(initContainers, corev1.Container{
			Name:    "install-plugins",
			Image:   fmt.Sprintf("%s:%s", defaultImage, grafanaSpec.Version),
			Command: []string{"sh", "-c", pluginInstallCmd},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugins",
					MountPath: defaultPluginsPath,
				},
			},
		})
		
		// Add plugins volume mount to main container
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "plugins",
			MountPath: defaultPluginsPath,
		})
	}

	// Build volumes
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.getConfigMapName(platform),
					},
				},
			},
		},
		{
			Name: "datasources",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.getDataSourceConfigMapName(platform),
					},
				},
			},
		},
		{
			Name: "dashboard-provider",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.getDashboardProviderConfigMapName(platform),
					},
				},
			},
		},
		{
			Name: "dashboards",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: m.getDefaultDashboardsConfigMapName(platform),
					},
				},
			},
		},
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	// Add plugins volume if needed
	if len(grafanaSpec.Plugins) > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "plugins",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// Build pod spec
	podSpec := corev1.PodSpec{
		ServiceAccountName: fmt.Sprintf("%s-observability", platform.Name),
		InitContainers:     initContainers,
		Containers:         []corev1.Container{container},
		Volumes:            volumes,
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:      &[]int64{472}[0], // Grafana group
			RunAsNonRoot: &[]bool{true}[0],
			RunAsUser:    &[]int64{472}[0],
		},
	}

	// Add node selector if specified
	if len(platform.Spec.Global.NodeSelector) > 0 {
		podSpec.NodeSelector = platform.Spec.Global.NodeSelector
	}

	// Add tolerations if specified
	if len(platform.Spec.Global.Tolerations) > 0 {
		podSpec.Tolerations = platform.Spec.Global.Tolerations
	}

	// Add affinity if specified
	if platform.Spec.Global.Affinity != nil {
		podSpec.Affinity = platform.Spec.Global.Affinity
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: podSpec,
		},
	}
}

// generateDataSourcesConfig generates the datasources YAML configuration
func (m *GrafanaManager) generateDataSourcesConfig(platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) string {
	config := `apiVersion: 1

datasources:`

	// Add Prometheus datasource if enabled
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		prometheusURL := fmt.Sprintf("http://prometheus-%s.%s.svc.cluster.local:9090", platform.Name, platform.Namespace)
		config += fmt.Sprintf(`
  - name: Prometheus
    type: prometheus
    access: proxy
    url: %s
    isDefault: true
    jsonData:
      timeInterval: 15s`, prometheusURL)
	}

	// Add Loki datasource if enabled
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		lokiURL := fmt.Sprintf("http://loki-%s.%s.svc.cluster.local:3100", platform.Name, platform.Namespace)
		config += fmt.Sprintf(`
  - name: Loki
    type: loki
    access: proxy
    url: %s`, lokiURL)
	}

	// Add Tempo datasource if enabled
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		tempoURL := fmt.Sprintf("http://tempo-%s.%s.svc.cluster.local:3200", platform.Name, platform.Namespace)
		config += fmt.Sprintf(`
  - name: Tempo
    type: tempo
    access: proxy
    url: %s
    jsonData:
      tracesToLogs:
        datasourceUid: loki
        mapTagNamesEnabled: true
        mappedTags:
          - key: service.name
            value: service
      serviceMap:
        datasourceUid: prometheus
      search:
        hide: false
      nodeGraph:
        enabled: true`, tempoURL)
	}

	// Add custom datasources if provided
	for _, ds := range grafanaSpec.DataSources {
		config += fmt.Sprintf(`
  - name: %s
    type: %s
    access: %s
    url: %s
    isDefault: %t`, ds.Name, ds.Type, ds.Access, ds.URL, ds.IsDefault)
		
		if len(ds.JSONData) > 0 {
			config += "\n    jsonData:"
			for k, v := range ds.JSONData {
				config += fmt.Sprintf("\n      %s: %s", k, v)
			}
		}
	}

	return config
}

// generateDashboardProviderConfig generates the dashboard provider configuration
func (m *GrafanaManager) generateDashboardProviderConfig(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return `apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: false
    options:
      path: /var/lib/grafana/dashboards`
}

// generateGrafanaConfig generates the grafana.ini configuration
func (m *GrafanaManager) generateGrafanaConfig(platform *observabilityv1beta1.ObservabilityPlatform, grafanaSpec *observabilityv1beta1.GrafanaSpec) string {
	config := `[server]
http_port = 3000

[database]
type = sqlite3

[auth]
disable_login_form = false

[auth.anonymous]
enabled = false

[security]
admin_user = admin
allow_embedding = true

[users]
auto_assign_org_role = Editor

[dashboards]
default_home_dashboard_path = /var/lib/grafana/dashboards/platform-overview.json

[log]
mode = console
level = ` + platform.Spec.Global.LogLevel

	// Add SMTP configuration if provided
	if grafanaSpec.SMTP != nil {
		config += fmt.Sprintf(`

[smtp]
enabled = true
host = %s:%d
user = %s
password = %s
from_address = %s
from_name = Grafana
skip_verify = %t`, 
			grafanaSpec.SMTP.Host,
			grafanaSpec.SMTP.Port,
			grafanaSpec.SMTP.User,
			grafanaSpec.SMTP.Password,
			grafanaSpec.SMTP.From,
			!grafanaSpec.SMTP.TLS)
	}

	return config
}

// generatePlatformOverviewDashboard generates a default platform overview dashboard
func (m *GrafanaManager) generatePlatformOverviewDashboard() string {
	return `{
  "dashboard": {
    "id": null,
    "uid": "platform-overview",
    "title": "Platform Overview",
    "tags": ["observability", "platform"],
    "timezone": "browser",
    "schemaVersion": 27,
    "version": 1,
    "refresh": "30s",
    "panels": [
      {
        "datasource": {
          "type": "prometheus",
          "uid": "${datasource}"
        },
        "fieldConfig": {
          "defaults": {
            "color": {
              "mode": "palette-classic"
            },
            "custom": {
              "axisLabel": "",
              "axisPlacement": "auto",
              "barAlignment": 0,
              "drawStyle": "line",
              "fillOpacity": 10,
              "gradientMode": "none",
              "hideFrom": {
                "tooltip": false,
                "viz": false,
                "legend": false
              },
              "lineInterpolation": "linear",
              "lineWidth": 1,
              "pointSize": 5,
              "scaleDistribution": {
                "type": "linear"
              },
              "showPoints": "never",
              "spanNulls": true,
              "stacking": {
                "group": "A",
                "mode": "none"
              },
              "thresholdsStyle": {
                "mode": "off"
              }
            },
            "mappings": [],
            "thresholds": {
              "mode": "absolute",
              "steps": [
                {
                  "color": "green",
                  "value": null
                },
                {
                  "color": "red",
                  "value": 80
                }
              ]
            },
            "unit": "short"
          },
          "overrides": []
        },
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 0
        },
        "id": 1,
        "options": {
          "legend": {
            "calcs": [],
            "displayMode": "list",
            "placement": "bottom"
          },
          "tooltip": {
            "mode": "single"
          }
        },
        "pluginVersion": "8.0.0",
        "targets": [
          {
            "expr": "up{job=~\"prometheus|grafana|loki|tempo\"}",
            "refId": "A"
          }
        ],
        "title": "Component Health",
        "type": "timeseries"
      }
    ]
  },
  "overwrite": true
}`
}

// generatePassword generates a secure random password
func (m *GrafanaManager) generatePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a default password if random generation fails
		return "admin-" + base64.URLEncoding.EncodeToString([]byte("gunj-operator"))[:16]
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// Helper methods for resource naming
func (m *GrafanaManager) getAdminSecretName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s-admin", platform.Name)
}

func (m *GrafanaManager) getConfigMapName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s-config", platform.Name)
}

func (m *GrafanaManager) getDataSourceConfigMapName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s-datasources", platform.Name)
}

func (m *GrafanaManager) getDashboardProviderConfigMapName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s-dashboard-provider", platform.Name)
}

func (m *GrafanaManager) getDefaultDashboardsConfigMapName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s-dashboards", platform.Name)
}

func (m *GrafanaManager) getServiceName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s", platform.Name)
}

func (m *GrafanaManager) getDeploymentName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s", platform.Name)
}

func (m *GrafanaManager) getIngressName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("grafana-%s", platform.Name)
}

// Helper methods for labels
func (m *GrafanaManager) getLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
		"app.kubernetes.io/part-of":    "observability-platform",
		"app.kubernetes.io/component":  labelComponent,
		"observability.io/platform":    platform.Name,
	}
}

func (m *GrafanaManager) getSelectorLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      componentName,
		"app.kubernetes.io/instance":  platform.Name,
		"app.kubernetes.io/component": labelComponent,
	}
}
