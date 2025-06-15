/*
Copyright 2025.

Licensed under the MIT License.
*/

package prometheus

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	componentName = "prometheus"
	
	// Default values
	defaultPort        = 9090
	defaultMetricsPath = "/metrics"
	defaultDataPath    = "/prometheus"
	defaultImage       = "prom/prometheus"
	
	// Labels
	labelComponent = "prometheus"
)

// PrometheusManager manages Prometheus deployments
type PrometheusManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// NewPrometheusManager creates a new Prometheus manager
func NewPrometheusManager(client client.Client, scheme *runtime.Scheme) managers.PrometheusManager {
	return &PrometheusManager{
		Client: client,
		Scheme: scheme,
	}
}

// Reconcile reconciles the Prometheus component
func (m *PrometheusManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Prometheus component with provided configuration
func (m *PrometheusManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Check if Prometheus is enabled
	if platform.Spec.Components.Prometheus == nil || !platform.Spec.Components.Prometheus.Enabled {
		log.V(1).Info("Prometheus is disabled, skipping reconciliation")
		return nil
	}
	
	prometheusSpec := platform.Spec.Components.Prometheus
	log.Info("Reconciling Prometheus", "version", prometheusSpec.Version)
	
	// 1. Create ConfigMap
	if err := m.reconcileConfigMap(ctx, platform, prometheusSpec); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMap: %w", err)
	}
	
	// 2. Create Service
	if err := m.reconcileService(ctx, platform); err != nil {
		return fmt.Errorf("failed to reconcile Service: %w", err)
	}
	
	// 3. Create StatefulSet
	if err := m.reconcileStatefulSet(ctx, platform, prometheusSpec); err != nil {
		return fmt.Errorf("failed to reconcile StatefulSet: %w", err)
	}
	
	// 4. Create ServiceMonitor (if configured)
	if err := m.reconcileServiceMonitor(ctx, platform); err != nil {
		return fmt.Errorf("failed to reconcile ServiceMonitor: %w", err)
	}
	
	// 5. Create PodDisruptionBudget for HA
	if prometheusSpec.Replicas > 1 {
		if err := m.reconcilePodDisruptionBudget(ctx, platform, prometheusSpec); err != nil {
			return fmt.Errorf("failed to reconcile PodDisruptionBudget: %w", err)
		}
	}
	
	log.Info("Prometheus reconciliation completed successfully")
	return nil
}

// Delete removes the Prometheus component resources
func (m *PrometheusManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	log.Info("Deleting Prometheus resources")
	
	// Delete in reverse order of creation
	resources := []client.Object{
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getPDBName(platform),
				Namespace: platform.Namespace,
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getStatefulSetName(platform),
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
	}
	
	for _, resource := range resources {
		if err := m.Client.Delete(ctx, resource); err != nil {
			log.Error(err, "Failed to delete resource", "resource", resource.GetName())
		}
	}
	
	return nil
}

// GetStatus returns the current status of the Prometheus component
func (m *PrometheusManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	status := &observabilityv1beta1.ComponentStatus{
		Phase: "Unknown",
	}
	
	// Check if Prometheus is enabled
	if platform.Spec.Components.Prometheus == nil || !platform.Spec.Components.Prometheus.Enabled {
		status.Phase = "Disabled"
		status.Message = "Prometheus is disabled"
		return status, nil
	}
	
	// Check StatefulSet status
	sts := &appsv1.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      m.getStatefulSetName(platform),
		Namespace: platform.Namespace,
	}, sts)
	
	if err != nil {
		status.Phase = "Failed"
		status.Message = fmt.Sprintf("Failed to get StatefulSet: %v", err)
		return status, nil
	}
	
	// Update status based on StatefulSet
	status.Version = platform.Spec.Components.Prometheus.Version
	status.Replicas = sts.Status.Replicas
	status.Ready = sts.Status.ReadyReplicas
	
	if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
		status.Phase = "Ready"
		status.Message = fmt.Sprintf("All %d replicas are ready", status.Ready)
	} else if sts.Status.ReadyReplicas > 0 {
		status.Phase = "Degraded"
		status.Message = fmt.Sprintf("%d/%d replicas are ready", status.Ready, status.Replicas)
	} else {
		status.Phase = "NotReady"
		status.Message = "No replicas are ready"
	}
	
	log.V(1).Info("Got Prometheus status", "phase", status.Phase)
	return status, nil
}

// Validate validates the Prometheus configuration
func (m *PrometheusManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Prometheus == nil {
		return nil
	}
	
	prometheus := platform.Spec.Components.Prometheus
	
	// Validate version format
	if !strings.HasPrefix(prometheus.Version, "v") {
		return fmt.Errorf("prometheus version must start with 'v', got: %s", prometheus.Version)
	}
	
	// Validate resources
	if prometheus.Resources.Requests != nil {
		if _, err := resource.ParseQuantity(prometheus.Resources.Requests.Memory().String()); err != nil {
			return fmt.Errorf("invalid memory request: %w", err)
		}
		if _, err := resource.ParseQuantity(prometheus.Resources.Requests.Cpu().String()); err != nil {
			return fmt.Errorf("invalid CPU request: %w", err)
		}
	}
	
	// Validate retention
	if prometheus.Retention != "" {
		// Simple validation - should be a duration string like "30d"
		if !strings.HasSuffix(prometheus.Retention, "d") &&
			!strings.HasSuffix(prometheus.Retention, "h") &&
			!strings.HasSuffix(prometheus.Retention, "m") {
			return fmt.Errorf("invalid retention format: %s (expected format like '30d', '24h')", prometheus.Retention)
		}
	}
	
	return nil
}

// GetServiceURL returns the service URL for Prometheus
func (m *PrometheusManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", 
		m.getServiceName(platform), 
		platform.Namespace, 
		defaultPort)
}

// UpdateConfiguration updates Prometheus configuration without restart
func (m *PrometheusManager) UpdateConfiguration(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Update ConfigMap
	if err := m.reconcileConfigMap(ctx, platform, platform.Spec.Components.Prometheus); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}
	
	// TODO: Implement configuration reload via Prometheus API
	// For now, we'll rely on Prometheus's automatic config reload
	log.Info("Configuration updated, Prometheus will reload automatically")
	
	return nil
}

// ScaleReplicas scales Prometheus replicas
func (m *PrometheusManager) ScaleReplicas(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, replicas int32) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Get StatefulSet
	sts := &appsv1.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      m.getStatefulSetName(platform),
		Namespace: platform.Namespace,
	}, sts)
	
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	
	// Update replicas
	sts.Spec.Replicas = &replicas
	if err := m.Client.Update(ctx, sts); err != nil {
		return fmt.Errorf("failed to update StatefulSet replicas: %w", err)
	}
	
	log.Info("Scaled Prometheus replicas", "replicas", replicas)
	return nil
}

// reconcileConfigMap creates or updates the Prometheus ConfigMap
func (m *PrometheusManager) reconcileConfigMap(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, prometheusSpec *observabilityv1beta1.PrometheusSpec) error {
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
		
		// Generate prometheus.yml
		prometheusYAML := m.generatePrometheusConfig(platform, prometheusSpec)
		
		configMap.Data = map[string]string{
			"prometheus.yml": prometheusYAML,
		}
		
		// Add additional scrape configs if provided
		if prometheusSpec.AdditionalScrapeConfigs != "" {
			configMap.Data["additional-scrape-configs.yml"] = prometheusSpec.AdditionalScrapeConfigs
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update ConfigMap: %w", err)
	}
	
	log.V(1).Info("ConfigMap reconciled", "name", configMap.Name)
	return nil
}

// reconcileService creates or updates the Prometheus Service
func (m *PrometheusManager) reconcileService(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
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

// reconcileStatefulSet creates or updates the Prometheus StatefulSet
func (m *PrometheusManager) reconcileStatefulSet(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, prometheusSpec *observabilityv1beta1.PrometheusSpec) error {
	log := log.FromContext(ctx)
	
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getStatefulSetName(platform),
			Namespace: platform.Namespace,
		},
	}
	
	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, sts, func() error {
		// Set labels
		sts.Labels = m.getLabels(platform)
		
		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, sts, m.Scheme); err != nil {
			return err
		}
		
		// Build StatefulSet spec
		sts.Spec = m.buildStatefulSetSpec(platform, prometheusSpec)
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update StatefulSet: %w", err)
	}
	
	log.V(1).Info("StatefulSet reconciled", "name", sts.Name)
	return nil
}

// reconcileServiceMonitor creates or updates the ServiceMonitor for Prometheus self-monitoring
func (m *PrometheusManager) reconcileServiceMonitor(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	// TODO: Implement ServiceMonitor creation when prometheus-operator CRDs are available
	// For now, we'll skip this as it requires the prometheus-operator
	return nil
}

// reconcilePodDisruptionBudget creates or updates the PodDisruptionBudget
func (m *PrometheusManager) reconcilePodDisruptionBudget(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, prometheusSpec *observabilityv1beta1.PrometheusSpec) error {
	log := log.FromContext(ctx)
	
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getPDBName(platform),
			Namespace: platform.Namespace,
		},
	}
	
	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, pdb, func() error {
		// Set labels
		pdb.Labels = m.getLabels(platform)
		
		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, pdb, m.Scheme); err != nil {
			return err
		}
		
		// Calculate minAvailable (at least 1 or half of replicas)
		minAvailable := int32(1)
		if prometheusSpec.Replicas > 2 {
			minAvailable = prometheusSpec.Replicas / 2
		}
		
		// Set spec
		pdb.Spec = policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: minAvailable,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: m.getSelectorLabels(platform),
			},
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update PodDisruptionBudget: %w", err)
	}
	
	log.V(1).Info("PodDisruptionBudget reconciled", "name", pdb.Name)
	return nil
}

// buildStatefulSetSpec builds the StatefulSet specification
func (m *PrometheusManager) buildStatefulSetSpec(platform *observabilityv1beta1.ObservabilityPlatform, prometheusSpec *observabilityv1beta1.PrometheusSpec) appsv1.StatefulSetSpec {
	replicas := prometheusSpec.Replicas
	labels := m.getSelectorLabels(platform)
	
	// Build container
	container := corev1.Container{
		Name:  componentName,
		Image: fmt.Sprintf("%s:%s", defaultImage, prometheusSpec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: defaultPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Args: []string{
			"--config.file=/etc/prometheus/prometheus.yml",
			fmt.Sprintf("--storage.tsdb.path=%s", defaultDataPath),
			fmt.Sprintf("--storage.tsdb.retention.time=%s", prometheusSpec.Retention),
			"--web.console.libraries=/usr/share/prometheus/console_libraries",
			"--web.console.templates=/usr/share/prometheus/consoles",
			"--web.enable-lifecycle",
		},
		Resources: prometheusSpec.Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config",
				MountPath: "/etc/prometheus",
			},
			{
				Name:      "data",
				MountPath: defaultDataPath,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/healthy",
					Port: intstr.FromInt(defaultPort),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/-/ready",
					Port: intstr.FromInt(defaultPort),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             &[]bool{true}[0],
			RunAsUser:                &[]int64{65534}[0], // nobody user
			AllowPrivilegeEscalation: &[]bool{false}[0],
			ReadOnlyRootFilesystem:   &[]bool{true}[0],
		},
	}
	
	// Add remote write configuration if specified
	if len(prometheusSpec.RemoteWrite) > 0 {
		for i, rw := range prometheusSpec.RemoteWrite {
			container.Args = append(container.Args, 
				fmt.Sprintf("--remote-write.url=%s", rw.URL))
			if rw.RemoteTimeout != "" {
				container.Args = append(container.Args,
					fmt.Sprintf("--remote-write.remote-timeout=%s", rw.RemoteTimeout))
			}
			// Headers would need to be configured in the prometheus.yml
			_ = i // Use index if needed
		}
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
	}
	
	// Build volume claim templates
	var volumeClaimTemplates []corev1.PersistentVolumeClaim
	if prometheusSpec.Storage.Size.String() != "" {
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: prometheusSpec.Storage.Size,
					},
				},
			},
		}
		
		if prometheusSpec.Storage.StorageClassName != "" {
			pvc.Spec.StorageClassName = &prometheusSpec.Storage.StorageClassName
		}
		
		volumeClaimTemplates = append(volumeClaimTemplates, pvc)
	}
	
	// Build pod spec
	podSpec := corev1.PodSpec{
		ServiceAccountName: fmt.Sprintf("%s-observability", platform.Name),
		Containers:         []corev1.Container{container},
		Volumes:            volumes,
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:      &[]int64{65534}[0],
			RunAsNonRoot: &[]bool{true}[0],
			RunAsUser:    &[]int64{65534}[0],
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
	
	return appsv1.StatefulSetSpec{
		ServiceName: m.getServiceName(platform),
		Replicas:    &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: podSpec,
		},
		VolumeClaimTemplates: volumeClaimTemplates,
	}
}

// generatePrometheusConfig generates the prometheus.yml configuration
func (m *PrometheusManager) generatePrometheusConfig(platform *observabilityv1beta1.ObservabilityPlatform, prometheusSpec *observabilityv1beta1.PrometheusSpec) string {
	config := `global:
  scrape_interval: 15s
  evaluation_interval: 15s`
	
	// Add external labels
	if len(prometheusSpec.ExternalLabels) > 0 || len(platform.Spec.Global.ExternalLabels) > 0 {
		config += "\n  external_labels:"
		
		// Global external labels first
		for k, v := range platform.Spec.Global.ExternalLabels {
			config += fmt.Sprintf("\n    %s: %s", k, v)
		}
		
		// Component-specific external labels (override global)
		for k, v := range prometheusSpec.ExternalLabels {
			config += fmt.Sprintf("\n    %s: %s", k, v)
		}
	}
	
	// Add alerting configuration
	config += `

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          # - alertmanager:9093`
	
	// Add rule files
	config += `

rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"`
	
	// Add scrape configs
	config += `

scrape_configs:
  # Prometheus self-monitoring
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Kubernetes service discovery
  - job_name: 'kubernetes-apiservers'
    kubernetes_sd_configs:
      - role: endpoints
    scheme: https
    tls_config:
      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
    relabel_configs:
      - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
        action: keep
        regex: default;kubernetes;https

  # Kubernetes nodes
  - job_name: 'kubernetes-nodes'
    kubernetes_sd_configs:
      - role: node
    scheme: https
    tls_config:
      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
    relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)

  # Kubernetes pods
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: kubernetes_pod_name`
	
	// Add remote write configuration if specified
	if len(prometheusSpec.RemoteWrite) > 0 {
		config += "\n\nremote_write:"
		for _, rw := range prometheusSpec.RemoteWrite {
			config += fmt.Sprintf("\n  - url: %s", rw.URL)
			if rw.RemoteTimeout != "" {
				config += fmt.Sprintf("\n    remote_timeout: %s", rw.RemoteTimeout)
			}
			if len(rw.Headers) > 0 {
				config += "\n    headers:"
				for k, v := range rw.Headers {
					config += fmt.Sprintf("\n      %s: %s", k, v)
				}
			}
		}
	}
	
	return config
}

// Helper methods for resource naming
func (m *PrometheusManager) getConfigMapName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("prometheus-%s-config", platform.Name)
}

func (m *PrometheusManager) getServiceName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("prometheus-%s", platform.Name)
}

func (m *PrometheusManager) getStatefulSetName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("prometheus-%s", platform.Name)
}

func (m *PrometheusManager) getPDBName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("prometheus-%s", platform.Name)
}

// Helper methods for labels
func (m *PrometheusManager) getLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
		"app.kubernetes.io/part-of":    "observability-platform",
		"app.kubernetes.io/component":  labelComponent,
		"observability.io/platform":    platform.Name,
	}
}

func (m *PrometheusManager) getSelectorLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      componentName,
		"app.kubernetes.io/instance":  platform.Name,
		"app.kubernetes.io/component": labelComponent,
	}
}
