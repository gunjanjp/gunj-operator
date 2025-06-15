/*
Copyright 2025.

Licensed under the MIT License.
*/

package tempo

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
	componentName = "tempo"
	
	// Default values
	defaultHTTPPort       = 3200
	defaultGRPCPort       = 9095
	defaultOTLPGRPCPort   = 4317
	defaultOTLPHTTPPort   = 4318
	defaultJaegerThriftCompactPort = 6831
	defaultJaegerThriftBinaryPort  = 6832
	defaultJaegerThriftHTTPPort    = 14268
	defaultJaegerGRPCPort          = 14250
	defaultZipkinPort              = 9411
	defaultDataPath       = "/var/tempo"
	defaultImage          = "grafana/tempo"
	
	// Labels
	labelComponent = "tempo"
)

// TempoManager manages Tempo deployments
type TempoManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// NewTempoManager creates a new Tempo manager
func NewTempoManager(client client.Client, scheme *runtime.Scheme) managers.TempoManager {
	return &TempoManager{
		Client: client,
		Scheme: scheme,
	}
}

// Reconcile reconciles the Tempo component
func (m *TempoManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Tempo component with provided configuration
func (m *TempoManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Check if Tempo is enabled
	if platform.Spec.Components.Tempo == nil || !platform.Spec.Components.Tempo.Enabled {
		log.V(1).Info("Tempo is disabled, skipping reconciliation")
		return nil
	}
	
	tempoSpec := platform.Spec.Components.Tempo
	log.Info("Reconciling Tempo", "version", tempoSpec.Version)
	
	// 1. Create ConfigMap
	if err := m.reconcileConfigMap(ctx, platform, tempoSpec); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMap: %w", err)
	}
	
	// 2. Create Services
	if err := m.reconcileServices(ctx, platform); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}
	
	// 3. Create StatefulSet
	if err := m.reconcileStatefulSet(ctx, platform, tempoSpec); err != nil {
		return fmt.Errorf("failed to reconcile StatefulSet: %w", err)
	}
	
	// 4. Create PodDisruptionBudget for HA
	if tempoSpec.Replicas > 1 {
		if err := m.reconcilePodDisruptionBudget(ctx, platform, tempoSpec); err != nil {
			return fmt.Errorf("failed to reconcile PodDisruptionBudget: %w", err)
		}
	}
	
	log.Info("Successfully reconciled Tempo")
	return nil
}

// Delete removes the Tempo component resources
func (m *TempoManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Delete in reverse order of creation
	// 1. Delete PodDisruptionBudget
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-pdb", platform.Name, componentName),
			Namespace: platform.Namespace,
		},
	}
	if err := m.Delete(ctx, pdb); err != nil {
		log.Error(err, "Failed to delete PodDisruptionBudget")
	}
	
	// 2. Delete StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", platform.Name, componentName),
			Namespace: platform.Namespace,
		},
	}
	if err := m.Delete(ctx, sts); err != nil {
		log.Error(err, "Failed to delete StatefulSet")
	}
	
	// 3. Delete Services
	services := []string{
		fmt.Sprintf("%s-%s", platform.Name, componentName),
		fmt.Sprintf("%s-%s-query", platform.Name, componentName),
		fmt.Sprintf("%s-%s-ingester", platform.Name, componentName),
		fmt.Sprintf("%s-%s-distributor", platform.Name, componentName),
	}
	
	for _, svcName := range services {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: platform.Namespace,
			},
		}
		if err := m.Delete(ctx, svc); err != nil {
			log.Error(err, "Failed to delete Service", "service", svcName)
		}
	}
	
	// 4. Delete ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-config", platform.Name, componentName),
			Namespace: platform.Namespace,
		},
	}
	if err := m.Delete(ctx, cm); err != nil {
		log.Error(err, "Failed to delete ConfigMap")
	}
	
	log.Info("Successfully deleted Tempo resources")
	return nil
}

// GetStatus returns the current status of the Tempo component
func (m *TempoManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	status := &observabilityv1beta1.ComponentStatus{
		Name: componentName,
	}
	
	// Check if Tempo is enabled
	if platform.Spec.Components.Tempo == nil || !platform.Spec.Components.Tempo.Enabled {
		status.Status = observabilityv1beta1.ComponentStatusDisabled
		status.Message = "Tempo is disabled"
		return status, nil
	}
	
	// Check StatefulSet status
	sts := &appsv1.StatefulSet{}
	if err := m.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, componentName),
		Namespace: platform.Namespace,
	}, sts); err != nil {
		status.Status = observabilityv1beta1.ComponentStatusFailed
		status.Message = fmt.Sprintf("Failed to get StatefulSet: %v", err)
		return status, nil
	}
	
	// Check if all replicas are ready
	if sts.Status.ReadyReplicas == sts.Status.Replicas && sts.Status.Replicas == *sts.Spec.Replicas {
		status.Status = observabilityv1beta1.ComponentStatusReady
		status.Message = fmt.Sprintf("All %d replicas are ready", sts.Status.Replicas)
		status.Ready = true
	} else {
		status.Status = observabilityv1beta1.ComponentStatusPending
		status.Message = fmt.Sprintf("Waiting for replicas: %d/%d ready", sts.Status.ReadyReplicas, *sts.Spec.Replicas)
		status.Ready = false
	}
	
	log.V(1).Info("Retrieved Tempo status", "status", status.Status)
	return status, nil
}

// Validate validates the Tempo configuration
func (m *TempoManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Tempo == nil || !platform.Spec.Components.Tempo.Enabled {
		return nil
	}
	
	tempoSpec := platform.Spec.Components.Tempo
	
	// Validate version format
	if tempoSpec.Version == "" {
		return fmt.Errorf("tempo version is required")
	}
	
	// Validate replicas
	if tempoSpec.Replicas < 1 {
		return fmt.Errorf("tempo replicas must be at least 1")
	}
	
	// Validate storage
	if tempoSpec.Storage.Size == "" {
		return fmt.Errorf("tempo storage size is required")
	}
	
	// Validate S3 configuration if provided
	if tempoSpec.S3 != nil {
		if tempoSpec.S3.BucketName == "" {
			return fmt.Errorf("tempo S3 bucket name is required when S3 is configured")
		}
		if tempoSpec.S3.Region == "" {
			return fmt.Errorf("tempo S3 region is required when S3 is configured")
		}
	}
	
	return nil
}

// GetServiceURL returns the service URL for Tempo
func (m *TempoManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("%s-%s.%s.svc.cluster.local:%d", platform.Name, componentName, platform.Namespace, defaultHTTPPort)
}

// ConfigureReceivers configures trace receivers
func (m *TempoManager) ConfigureReceivers(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Update the ConfigMap with new receiver configuration
	if err := m.reconcileConfigMap(ctx, platform, platform.Spec.Components.Tempo); err != nil {
		return fmt.Errorf("failed to update receiver configuration: %w", err)
	}
	
	// Trigger a rolling update of the StatefulSet
	sts := &appsv1.StatefulSet{}
	if err := m.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, componentName),
		Namespace: platform.Namespace,
	}, sts); err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	
	// Update annotation to trigger rolling update
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["tempo.observability.io/config-hash"] = fmt.Sprintf("%d", metav1.Now().Unix())
	
	if err := m.Update(ctx, sts); err != nil {
		return fmt.Errorf("failed to update StatefulSet: %w", err)
	}
	
	log.Info("Successfully configured receivers")
	return nil
}

// UpdateSampling updates sampling configuration
func (m *TempoManager) UpdateSampling(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Update the ConfigMap with new sampling configuration
	if err := m.reconcileConfigMap(ctx, platform, platform.Spec.Components.Tempo); err != nil {
		return fmt.Errorf("failed to update sampling configuration: %w", err)
	}
	
	// Trigger a rolling update similar to ConfigureReceivers
	sts := &appsv1.StatefulSet{}
	if err := m.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, componentName),
		Namespace: platform.Namespace,
	}, sts); err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["tempo.observability.io/sampling-hash"] = fmt.Sprintf("%d", metav1.Now().Unix())
	
	if err := m.Update(ctx, sts); err != nil {
		return fmt.Errorf("failed to update StatefulSet: %w", err)
	}
	
	log.Info("Successfully updated sampling configuration")
	return nil
}

// reconcileConfigMap creates or updates the Tempo ConfigMap
func (m *TempoManager) reconcileConfigMap(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, tempoSpec *observabilityv1beta1.TempoSpec) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-config", platform.Name, componentName),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform),
		},
		Data: map[string]string{
			"tempo.yaml": m.generateTempoConfig(platform, tempoSpec),
		},
	}
	
	// Set controller reference
	if err := controllerutil.SetControllerReference(platform, cm, m.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	
	// Create or update the ConfigMap
	if err := m.createOrUpdate(ctx, cm); err != nil {
		return fmt.Errorf("failed to create/update ConfigMap: %w", err)
	}
	
	log.V(1).Info("Successfully reconciled ConfigMap")
	return nil
}

// reconcileServices creates or updates Tempo services
func (m *TempoManager) reconcileServices(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Main service for all protocols
	mainSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", platform.Name, componentName),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform),
		},
		Spec: corev1.ServiceSpec{
			Selector: m.getLabels(platform),
			Ports: []corev1.ServicePort{
				{
					Name:       "tempo-http",
					Port:       defaultHTTPPort,
					TargetPort: intstr.FromInt(defaultHTTPPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "tempo-grpc",
					Port:       defaultGRPCPort,
					TargetPort: intstr.FromInt(defaultGRPCPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "otlp-grpc",
					Port:       defaultOTLPGRPCPort,
					TargetPort: intstr.FromInt(defaultOTLPGRPCPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "otlp-http",
					Port:       defaultOTLPHTTPPort,
					TargetPort: intstr.FromInt(defaultOTLPHTTPPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "jaeger-thrift-compact",
					Port:       defaultJaegerThriftCompactPort,
					TargetPort: intstr.FromInt(defaultJaegerThriftCompactPort),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       "jaeger-thrift-binary",
					Port:       defaultJaegerThriftBinaryPort,
					TargetPort: intstr.FromInt(defaultJaegerThriftBinaryPort),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       "jaeger-thrift-http",
					Port:       defaultJaegerThriftHTTPPort,
					TargetPort: intstr.FromInt(defaultJaegerThriftHTTPPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "jaeger-grpc",
					Port:       defaultJaegerGRPCPort,
					TargetPort: intstr.FromInt(defaultJaegerGRPCPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "zipkin",
					Port:       defaultZipkinPort,
					TargetPort: intstr.FromInt(defaultZipkinPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	
	// Set controller reference
	if err := controllerutil.SetControllerReference(platform, mainSvc, m.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	
	// Create or update the service
	if err := m.createOrUpdate(ctx, mainSvc); err != nil {
		return fmt.Errorf("failed to create/update main service: %w", err)
	}
	
	// Create a headless service for StatefulSet
	headlessSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-headless", platform.Name, componentName),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform),
		},
		Spec: corev1.ServiceSpec{
			Selector:  m.getLabels(platform),
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "tempo-grpc",
					Port:       defaultGRPCPort,
					TargetPort: intstr.FromInt(defaultGRPCPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	
	// Set controller reference
	if err := controllerutil.SetControllerReference(platform, headlessSvc, m.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	
	// Create or update the headless service
	if err := m.createOrUpdate(ctx, headlessSvc); err != nil {
		return fmt.Errorf("failed to create/update headless service: %w", err)
	}
	
	log.V(1).Info("Successfully reconciled services")
	return nil
}

// reconcileStatefulSet creates or updates the Tempo StatefulSet
func (m *TempoManager) reconcileStatefulSet(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, tempoSpec *observabilityv1beta1.TempoSpec) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Prepare volume mounts
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/etc/tempo",
		},
		{
			Name:      "storage",
			MountPath: defaultDataPath,
		},
	}
	
	// Prepare volumes
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-%s-config", platform.Name, componentName),
					},
				},
			},
		},
	}
	
	// Prepare container
	container := corev1.Container{
		Name:  componentName,
		Image: m.getImage(tempoSpec),
		Args: []string{
			"-config.file=/etc/tempo/tempo.yaml",
			"-mem-ballast-size-mbs=1024",
		},
		Ports: []corev1.ContainerPort{
			{Name: "tempo-http", ContainerPort: defaultHTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "tempo-grpc", ContainerPort: defaultGRPCPort, Protocol: corev1.ProtocolTCP},
			{Name: "otlp-grpc", ContainerPort: defaultOTLPGRPCPort, Protocol: corev1.ProtocolTCP},
			{Name: "otlp-http", ContainerPort: defaultOTLPHTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "jaeger-compact", ContainerPort: defaultJaegerThriftCompactPort, Protocol: corev1.ProtocolUDP},
			{Name: "jaeger-binary", ContainerPort: defaultJaegerThriftBinaryPort, Protocol: corev1.ProtocolUDP},
			{Name: "jaeger-http", ContainerPort: defaultJaegerThriftHTTPPort, Protocol: corev1.ProtocolTCP},
			{Name: "jaeger-grpc", ContainerPort: defaultJaegerGRPCPort, Protocol: corev1.ProtocolTCP},
			{Name: "zipkin", ContainerPort: defaultZipkinPort, Protocol: corev1.ProtocolTCP},
		},
		VolumeMounts: volumeMounts,
		Resources:    tempoSpec.Resources,
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ready",
					Port: intstr.FromInt(defaultHTTPPort),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
			TimeoutSeconds:      1,
			FailureThreshold:    3,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ready",
					Port: intstr.FromInt(defaultHTTPPort),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			TimeoutSeconds:      1,
			FailureThreshold:    3,
		},
	}
	
	// Create StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", platform.Name, componentName),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: fmt.Sprintf("%s-%s-headless", platform.Name, componentName),
			Replicas:    &tempoSpec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.getLabels(platform),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: m.getLabels(platform),
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   fmt.Sprintf("%d", defaultHTTPPort),
						"prometheus.io/path":   "/metrics",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("%s-operator", platform.Name),
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:      func(i int64) *int64 { return &i }(10001),
						RunAsUser:    func(i int64) *int64 { return &i }(10001),
						RunAsNonRoot: func(b bool) *bool { return &b }(true),
					},
					Containers: []corev1.Container{container},
					Volumes:    volumes,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "storage",
						Labels: m.getLabels(platform),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: tempoSpec.Storage.StorageClassName,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(tempoSpec.Storage.Size),
							},
						},
					},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
		},
	}
	
	// Set controller reference
	if err := controllerutil.SetControllerReference(platform, sts, m.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	
	// Create or update the StatefulSet
	if err := m.createOrUpdate(ctx, sts); err != nil {
		return fmt.Errorf("failed to create/update StatefulSet: %w", err)
	}
	
	log.V(1).Info("Successfully reconciled StatefulSet")
	return nil
}

// reconcilePodDisruptionBudget creates or updates the PodDisruptionBudget
func (m *TempoManager) reconcilePodDisruptionBudget(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, tempoSpec *observabilityv1beta1.TempoSpec) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	minAvailable := intstr.FromInt(1)
	if tempoSpec.Replicas > 2 {
		minAvailable = intstr.FromString("50%")
	}
	
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-pdb", platform.Name, componentName),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.getLabels(platform),
			},
		},
	}
	
	// Set controller reference
	if err := controllerutil.SetControllerReference(platform, pdb, m.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}
	
	// Create or update the PodDisruptionBudget
	if err := m.createOrUpdate(ctx, pdb); err != nil {
		return fmt.Errorf("failed to create/update PodDisruptionBudget: %w", err)
	}
	
	log.V(1).Info("Successfully reconciled PodDisruptionBudget")
	return nil
}

// generateTempoConfig generates the tempo.yaml configuration
func (m *TempoManager) generateTempoConfig(platform *observabilityv1beta1.ObservabilityPlatform, tempoSpec *observabilityv1beta1.TempoSpec) string {
	var sb strings.Builder
	
	// Server configuration
	sb.WriteString("server:\n")
	sb.WriteString(fmt.Sprintf("  http_listen_port: %d\n", defaultHTTPPort))
	sb.WriteString(fmt.Sprintf("  grpc_listen_port: %d\n", defaultGRPCPort))
	sb.WriteString("\n")
	
	// Distributor configuration
	sb.WriteString("distributor:\n")
	sb.WriteString("  receivers:\n")
	sb.WriteString("    otlp:\n")
	sb.WriteString("      protocols:\n")
	sb.WriteString("        grpc:\n")
	sb.WriteString(fmt.Sprintf("          endpoint: 0.0.0.0:%d\n", defaultOTLPGRPCPort))
	sb.WriteString("        http:\n")
	sb.WriteString(fmt.Sprintf("          endpoint: 0.0.0.0:%d\n", defaultOTLPHTTPPort))
	sb.WriteString("    jaeger:\n")
	sb.WriteString("      protocols:\n")
	sb.WriteString("        thrift_compact:\n")
	sb.WriteString(fmt.Sprintf("          endpoint: 0.0.0.0:%d\n", defaultJaegerThriftCompactPort))
	sb.WriteString("        thrift_binary:\n")
	sb.WriteString(fmt.Sprintf("          endpoint: 0.0.0.0:%d\n", defaultJaegerThriftBinaryPort))
	sb.WriteString("        thrift_http:\n")
	sb.WriteString(fmt.Sprintf("          endpoint: 0.0.0.0:%d\n", defaultJaegerThriftHTTPPort))
	sb.WriteString("        grpc:\n")
	sb.WriteString(fmt.Sprintf("          endpoint: 0.0.0.0:%d\n", defaultJaegerGRPCPort))
	sb.WriteString("    zipkin:\n")
	sb.WriteString(fmt.Sprintf("      endpoint: 0.0.0.0:%d\n", defaultZipkinPort))
	sb.WriteString("\n")
	
	// Ingester configuration
	sb.WriteString("ingester:\n")
	sb.WriteString("  max_block_duration: 5m\n")
	sb.WriteString("\n")
	
	// Compactor configuration
	sb.WriteString("compactor:\n")
	sb.WriteString("  compaction:\n")
	sb.WriteString("    block_retention: 1h\n")
	sb.WriteString("\n")
	
	// Metrics generator configuration
	sb.WriteString("metrics_generator:\n")
	sb.WriteString("  registry:\n")
	sb.WriteString("    external_labels:\n")
	for k, v := range platform.Spec.Global.ExternalLabels {
		sb.WriteString(fmt.Sprintf("      %s: %s\n", k, v))
	}
	sb.WriteString("  storage:\n")
	sb.WriteString("    path: " + defaultDataPath + "/generator/wal\n")
	sb.WriteString("    remote_write:\n")
	sb.WriteString("      - url: http://" + fmt.Sprintf("%s-prometheus:%d/api/v1/write", platform.Name, 9090) + "\n")
	sb.WriteString("\n")
	
	// Storage configuration
	sb.WriteString("storage:\n")
	sb.WriteString("  trace:\n")
	
	if tempoSpec.S3 != nil {
		// S3 backend configuration
		sb.WriteString("    backend: s3\n")
		sb.WriteString("    s3:\n")
		sb.WriteString(fmt.Sprintf("      bucket: %s\n", tempoSpec.S3.BucketName))
		sb.WriteString(fmt.Sprintf("      endpoint: %s\n", tempoSpec.S3.Endpoint))
		sb.WriteString(fmt.Sprintf("      region: %s\n", tempoSpec.S3.Region))
		sb.WriteString("      insecure: false\n")
		
		// Add access key configuration if provided
		if tempoSpec.S3.AccessKeyID != "" {
			sb.WriteString(fmt.Sprintf("      access_key: %s\n", tempoSpec.S3.AccessKeyID))
		}
		if tempoSpec.S3.SecretAccessKey != "" {
			sb.WriteString(fmt.Sprintf("      secret_key: %s\n", tempoSpec.S3.SecretAccessKey))
		}
	} else {
		// Local storage backend
		sb.WriteString("    backend: local\n")
		sb.WriteString("    local:\n")
		sb.WriteString(fmt.Sprintf("      path: %s/traces\n", defaultDataPath))
	}
	
	sb.WriteString("    wal:\n")
	sb.WriteString(fmt.Sprintf("      path: %s/wal\n", defaultDataPath))
	sb.WriteString("    cache: filesystem\n")
	sb.WriteString("    filesystem_cache:\n")
	sb.WriteString(fmt.Sprintf("      directory: %s/cache\n", defaultDataPath))
	sb.WriteString("\n")
	
	// Query frontend configuration
	sb.WriteString("query_frontend:\n")
	sb.WriteString("  search:\n")
	sb.WriteString("    max_duration: 0s\n")
	sb.WriteString("\n")
	
	// Querier configuration
	sb.WriteString("querier:\n")
	sb.WriteString("  max_concurrent_queries: 20\n")
	sb.WriteString("\n")
	
	// Overrides configuration for retention
	sb.WriteString("overrides:\n")
	sb.WriteString("  max_traces_per_user: 10000\n")
	sb.WriteString(fmt.Sprintf("  max_search_duration: %s\n", tempoSpec.Retention))
	
	return sb.String()
}

// Helper methods

func (m *TempoManager) getLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app":                          labelComponent,
		"app.kubernetes.io/name":       labelComponent,
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  labelComponent,
		"app.kubernetes.io/part-of":    "observability-platform",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
	}
}

func (m *TempoManager) getImage(tempoSpec *observabilityv1beta1.TempoSpec) string {
	// Remove 'v' prefix if present
	version := strings.TrimPrefix(tempoSpec.Version, "v")
	return fmt.Sprintf("%s:%s", defaultImage, version)
}

func (m *TempoManager) createOrUpdate(ctx context.Context, obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)
	existing := obj.DeepCopyObject().(client.Object)
	
	err := m.Get(ctx, key, existing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Object doesn't exist, create it
		return m.Create(ctx, obj)
	}
	
	// Object exists, update it
	obj.SetResourceVersion(existing.GetResourceVersion())
	return m.Update(ctx, obj)
}
