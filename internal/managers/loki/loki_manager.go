/*
Copyright 2025.

Licensed under the MIT License.
*/

package loki

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
	componentName = "loki"
	
	// Default values
	defaultHTTPPort       = 3100
	defaultGRPCPort       = 9095
	defaultDataPath       = "/loki"
	defaultWALPath        = "/wal"
	defaultImage          = "grafana/loki"
	defaultCompactorImage = "grafana/loki"
	
	// Labels
	labelComponent = "loki"
	
	// Default retention
	defaultRetention = "168h" // 7 days
)

// LokiManager manages Loki deployments
type LokiManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// NewLokiManager creates a new Loki manager
func NewLokiManager(client client.Client, scheme *runtime.Scheme) managers.LokiManager {
	return &LokiManager{
		Client: client,
		Scheme: scheme,
	}
}

// Reconcile reconciles the Loki component
func (m *LokiManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return m.ReconcileWithConfig(ctx, platform, nil)
}

// ReconcileWithConfig reconciles the Loki component with provided configuration
func (m *LokiManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Check if Loki is enabled
	if platform.Spec.Components.Loki == nil || !platform.Spec.Components.Loki.Enabled {
		log.V(1).Info("Loki is disabled, skipping reconciliation")
		return nil
	}
	
	lokiSpec := platform.Spec.Components.Loki
	log.Info("Reconciling Loki", "version", lokiSpec.Version)
	
	// 1. Create S3 secret if configured
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		if err := m.reconcileS3Secret(ctx, platform, lokiSpec); err != nil {
			return fmt.Errorf("failed to reconcile S3 secret: %w", err)
		}
	}
	
	// 2. Create ConfigMap
	if err := m.reconcileConfigMap(ctx, platform, lokiSpec); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMap: %w", err)
	}
	
	// 3. Create Services
	if err := m.reconcileServices(ctx, platform); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}
	
	// 4. Create StatefulSet
	if err := m.reconcileStatefulSet(ctx, platform, lokiSpec); err != nil {
		return fmt.Errorf("failed to reconcile StatefulSet: %w", err)
	}
	
	// 5. Create Compactor deployment if enabled
	if lokiSpec.CompactorEnabled {
		if err := m.reconcileCompactor(ctx, platform, lokiSpec); err != nil {
			return fmt.Errorf("failed to reconcile Compactor: %w", err)
		}
	}
	
	// 6. Create PodDisruptionBudget for HA
	if lokiSpec.Replicas > 1 {
		if err := m.reconcilePodDisruptionBudget(ctx, platform, lokiSpec); err != nil {
			return fmt.Errorf("failed to reconcile PodDisruptionBudget: %w", err)
		}
	}
	
	log.Info("Loki reconciliation completed successfully")
	return nil
}

// Delete removes the Loki component resources
func (m *LokiManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	log.Info("Deleting Loki resources")
	
	// Delete in reverse order of creation
	resources := []client.Object{
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getPDBName(platform),
				Namespace: platform.Namespace,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getCompactorName(platform),
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
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getHeadlessServiceName(platform),
				Namespace: platform.Namespace,
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getConfigMapName(platform),
				Namespace: platform.Namespace,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.getS3SecretName(platform),
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

// GetStatus returns the current status of the Loki component
func (m *LokiManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	status := &observabilityv1beta1.ComponentStatus{
		Phase: "Unknown",
	}
	
	// Check if Loki is enabled
	if platform.Spec.Components.Loki == nil || !platform.Spec.Components.Loki.Enabled {
		status.Phase = "Disabled"
		status.Message = "Loki is disabled"
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
	status.Version = platform.Spec.Components.Loki.Version
	status.Replicas = sts.Status.Replicas
	status.Ready = sts.Status.ReadyReplicas
	
	if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
		status.Phase = "Ready"
		status.Message = fmt.Sprintf("All %d replicas are ready", status.Ready)
		
		// Check compactor if enabled
		if platform.Spec.Components.Loki.CompactorEnabled {
			compactor := &appsv1.Deployment{}
			err := m.Client.Get(ctx, types.NamespacedName{
				Name:      m.getCompactorName(platform),
				Namespace: platform.Namespace,
			}, compactor)
			
			if err != nil || compactor.Status.ReadyReplicas != *compactor.Spec.Replicas {
				status.Phase = "Degraded"
				status.Message = "Loki is ready but compactor is not"
			}
		}
	} else if sts.Status.ReadyReplicas > 0 {
		status.Phase = "Degraded"
		status.Message = fmt.Sprintf("%d/%d replicas are ready", status.Ready, status.Replicas)
	} else {
		status.Phase = "NotReady"
		status.Message = "No replicas are ready"
	}
	
	log.V(1).Info("Got Loki status", "phase", status.Phase)
	return status, nil
}

// Validate validates the Loki configuration
func (m *LokiManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.Components.Loki == nil {
		return nil
	}
	
	loki := platform.Spec.Components.Loki
	
	// Validate version format
	if !strings.HasPrefix(loki.Version, "v") && !strings.Contains(loki.Version, ".") {
		return fmt.Errorf("loki version must be in format 'v2.9.0' or '2.9.0', got: %s", loki.Version)
	}
	
	// Validate resources
	if loki.Resources.Requests != nil {
		if _, err := resource.ParseQuantity(loki.Resources.Requests.Memory().String()); err != nil {
			return fmt.Errorf("invalid memory request: %w", err)
		}
		if _, err := resource.ParseQuantity(loki.Resources.Requests.Cpu().String()); err != nil {
			return fmt.Errorf("invalid CPU request: %w", err)
		}
	}
	
	// Validate retention
	if loki.Retention != "" {
		// Simple validation - should be a duration string like "168h"
		if !strings.HasSuffix(loki.Retention, "h") &&
			!strings.HasSuffix(loki.Retention, "d") &&
			!strings.HasSuffix(loki.Retention, "m") {
			return fmt.Errorf("invalid retention format: %s (expected format like '168h', '7d')", loki.Retention)
		}
	}
	
	// Validate S3 configuration
	if loki.S3 != nil && loki.S3.Enabled {
		if loki.S3.BucketName == "" {
			return fmt.Errorf("S3 bucket name is required when S3 is enabled")
		}
		if loki.S3.Region == "" {
			return fmt.Errorf("S3 region is required when S3 is enabled")
		}
	}
	
	return nil
}

// GetServiceURL returns the service URL for Loki
func (m *LokiManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", 
		m.getServiceName(platform), 
		platform.Namespace, 
		defaultHTTPPort)
}

// ConfigureStorage configures storage backend for Loki
func (m *LokiManager) ConfigureStorage(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Update S3 secret if needed
	if platform.Spec.Components.Loki.S3 != nil && platform.Spec.Components.Loki.S3.Enabled {
		if err := m.reconcileS3Secret(ctx, platform, platform.Spec.Components.Loki); err != nil {
			return fmt.Errorf("failed to update S3 secret: %w", err)
		}
	}
	
	// Update ConfigMap with new storage configuration
	if err := m.reconcileConfigMap(ctx, platform, platform.Spec.Components.Loki); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}
	
	// Restart Loki pods to pick up changes
	sts := &appsv1.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      m.getStatefulSetName(platform),
		Namespace: platform.Namespace,
	}, sts)
	
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	
	// Add annotation to force pod restart
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["observability.io/storage-update"] = metav1.Now().Format(metav1.RFC3339)
	
	if err := m.Client.Update(ctx, sts); err != nil {
		return fmt.Errorf("failed to update StatefulSet: %w", err)
	}
	
	log.Info("Storage configuration updated")
	return nil
}

// UpdateRetention updates log retention policies
func (m *LokiManager) UpdateRetention(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("component", componentName)
	
	// Update ConfigMap with new retention
	if err := m.reconcileConfigMap(ctx, platform, platform.Spec.Components.Loki); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}
	
	// Restart Loki pods to pick up changes
	sts := &appsv1.StatefulSet{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      m.getStatefulSetName(platform),
		Namespace: platform.Namespace,
	}, sts)
	
	if err != nil {
		return fmt.Errorf("failed to get StatefulSet: %w", err)
	}
	
	// Add annotation to force pod restart
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["observability.io/retention-update"] = metav1.Now().Format(metav1.RFC3339)
	
	if err := m.Client.Update(ctx, sts); err != nil {
		return fmt.Errorf("failed to update StatefulSet: %w", err)
	}
	
	log.Info("Retention configuration updated", "retention", platform.Spec.Components.Loki.Retention)
	return nil
}

// reconcileS3Secret creates or updates the S3 credentials secret
func (m *LokiManager) reconcileS3Secret(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) error {
	log := log.FromContext(ctx)
	
	// Skip if no credentials provided (using IAM role)
	if lokiSpec.S3.AccessKeyID == "" && lokiSpec.S3.SecretAccessKey == "" {
		log.V(1).Info("No S3 credentials provided, assuming IAM role")
		return nil
	}
	
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getS3SecretName(platform),
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
		
		// Set data
		secret.StringData = map[string]string{
			"access_key_id":     lokiSpec.S3.AccessKeyID,
			"secret_access_key": lokiSpec.S3.SecretAccessKey,
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update S3 secret: %w", err)
	}
	
	log.V(1).Info("S3 secret reconciled", "name", secret.Name)
	return nil
}

// reconcileConfigMap creates or updates the Loki ConfigMap
func (m *LokiManager) reconcileConfigMap(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) error {
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
		
		// Generate loki.yaml
		lokiYAML := m.generateLokiConfig(platform, lokiSpec)
		
		configMap.Data = map[string]string{
			"loki.yaml": lokiYAML,
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update ConfigMap: %w", err)
	}
	
	log.V(1).Info("ConfigMap reconciled", "name", configMap.Name)
	return nil
}

// reconcileServices creates or updates the Loki Services
func (m *LokiManager) reconcileServices(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	
	// Create main service
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
					Port:       defaultHTTPPort,
					TargetPort: intstr.FromInt(defaultHTTPPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "grpc",
					Port:       defaultGRPCPort,
					TargetPort: intstr.FromInt(defaultGRPCPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update Service: %w", err)
	}
	
	// Create headless service for StatefulSet
	headlessService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getHeadlessServiceName(platform),
			Namespace: platform.Namespace,
		},
	}
	
	_, err = controllerutil.CreateOrUpdate(ctx, m.Client, headlessService, func() error {
		// Set labels
		headlessService.Labels = m.getLabels(platform)
		
		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, headlessService, m.Scheme); err != nil {
			return err
		}
		
		// Set spec for headless service
		headlessService.Spec = corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Selector:  m.getSelectorLabels(platform),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       defaultHTTPPort,
					TargetPort: intstr.FromInt(defaultHTTPPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "grpc",
					Port:       defaultGRPCPort,
					TargetPort: intstr.FromInt(defaultGRPCPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update headless Service: %w", err)
	}
	
	log.V(1).Info("Services reconciled")
	return nil
}

// reconcileStatefulSet creates or updates the Loki StatefulSet
func (m *LokiManager) reconcileStatefulSet(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) error {
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
		sts.Spec = m.buildStatefulSetSpec(platform, lokiSpec)
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update StatefulSet: %w", err)
	}
	
	log.V(1).Info("StatefulSet reconciled", "name", sts.Name)
	return nil
}

// reconcileCompactor creates or updates the Loki Compactor deployment
func (m *LokiManager) reconcileCompactor(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) error {
	log := log.FromContext(ctx)
	
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getCompactorName(platform),
			Namespace: platform.Namespace,
		},
	}
	
	_, err := controllerutil.CreateOrUpdate(ctx, m.Client, deployment, func() error {
		// Set labels
		deployment.Labels = m.getLabels(platform)
		deployment.Labels["app.kubernetes.io/component"] = "compactor"
		
		// Set owner reference
		if err := controllerutil.SetControllerReference(platform, deployment, m.Scheme); err != nil {
			return err
		}
		
		// Build Deployment spec for compactor
		deployment.Spec = m.buildCompactorSpec(platform, lokiSpec)
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create/update Compactor deployment: %w", err)
	}
	
	log.V(1).Info("Compactor deployment reconciled", "name", deployment.Name)
	return nil
}

// reconcilePodDisruptionBudget creates or updates the PodDisruptionBudget
func (m *LokiManager) reconcilePodDisruptionBudget(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) error {
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
		if lokiSpec.Replicas > 2 {
			minAvailable = lokiSpec.Replicas / 2
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
func (m *LokiManager) buildStatefulSetSpec(platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) appsv1.StatefulSetSpec {
	replicas := lokiSpec.Replicas
	labels := m.getSelectorLabels(platform)
	
	// Determine image tag
	imageTag := lokiSpec.Version
	if !strings.HasPrefix(imageTag, "v") {
		imageTag = lokiSpec.Version
	}
	
	// Build container
	container := corev1.Container{
		Name:  componentName,
		Image: fmt.Sprintf("%s:%s", defaultImage, imageTag),
		Args: []string{
			"-config.file=/etc/loki/loki.yaml",
			"-target=all,table-manager",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: defaultHTTPPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "grpc",
				ContainerPort: defaultGRPCPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: lokiSpec.Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config",
				MountPath: "/etc/loki",
			},
			{
				Name:      "data",
				MountPath: defaultDataPath,
			},
			{
				Name:      "wal",
				MountPath: defaultWALPath,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ready",
					Port: intstr.FromInt(defaultHTTPPort),
				},
			},
			InitialDelaySeconds: 45,
			PeriodSeconds:       10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ready",
					Port: intstr.FromInt(defaultHTTPPort),
				},
			},
			InitialDelaySeconds: 45,
			PeriodSeconds:       10,
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             &[]bool{true}[0],
			RunAsUser:                &[]int64{10001}[0], // Loki user
			AllowPrivilegeEscalation: &[]bool{false}[0],
			ReadOnlyRootFilesystem:   &[]bool{false}[0], // Loki needs to write to disk
		},
	}
	
	// Add environment variables for S3 if configured
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled && lokiSpec.S3.AccessKeyID != "" {
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.getS3SecretName(platform),
						},
						Key: "access_key_id",
					},
				},
			},
			corev1.EnvVar{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.getS3SecretName(platform),
						},
						Key: "secret_access_key",
					},
				},
			},
		)
		
		if lokiSpec.S3.Region != "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "AWS_REGION",
				Value: lokiSpec.S3.Region,
			})
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
		{
			Name: "wal",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	
	// Build volume claim templates
	var volumeClaimTemplates []corev1.PersistentVolumeClaim
	if lokiSpec.Storage.Size.String() != "" {
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
						corev1.ResourceStorage: lokiSpec.Storage.Size,
					},
				},
			},
		}
		
		if lokiSpec.Storage.StorageClassName != "" {
			pvc.Spec.StorageClassName = &lokiSpec.Storage.StorageClassName
		}
		
		volumeClaimTemplates = append(volumeClaimTemplates, pvc)
	}
	
	// Build pod spec
	podSpec := corev1.PodSpec{
		ServiceAccountName: fmt.Sprintf("%s-observability", platform.Name),
		Containers:         []corev1.Container{container},
		Volumes:            volumes,
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:      &[]int64{10001}[0],
			RunAsNonRoot: &[]bool{true}[0],
			RunAsUser:    &[]int64{10001}[0],
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
		ServiceName: m.getHeadlessServiceName(platform),
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

// buildCompactorSpec builds the Compactor deployment specification
func (m *LokiManager) buildCompactorSpec(platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) appsv1.DeploymentSpec {
	replicas := int32(1) // Compactor should be a singleton
	labels := m.getCompactorLabels(platform)
	
	// Determine image tag
	imageTag := lokiSpec.Version
	if !strings.HasPrefix(imageTag, "v") {
		imageTag = lokiSpec.Version
	}
	
	// Build container
	container := corev1.Container{
		Name:  "compactor",
		Image: fmt.Sprintf("%s:%s", defaultCompactorImage, imageTag),
		Args: []string{
			"-config.file=/etc/loki/loki.yaml",
			"-target=compactor",
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config",
				MountPath: "/etc/loki",
			},
			{
				Name:      "data",
				MountPath: defaultDataPath,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             &[]bool{true}[0],
			RunAsUser:                &[]int64{10001}[0],
			AllowPrivilegeEscalation: &[]bool{false}[0],
			ReadOnlyRootFilesystem:   &[]bool{false}[0],
		},
	}
	
	// Add environment variables for S3 if configured
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled && lokiSpec.S3.AccessKeyID != "" {
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.getS3SecretName(platform),
						},
						Key: "access_key_id",
					},
				},
			},
			corev1.EnvVar{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.getS3SecretName(platform),
						},
						Key: "secret_access_key",
					},
				},
			},
		)
		
		if lokiSpec.S3.Region != "" {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  "AWS_REGION",
				Value: lokiSpec.S3.Region,
			})
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
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	
	// Build pod spec
	podSpec := corev1.PodSpec{
		ServiceAccountName: fmt.Sprintf("%s-observability", platform.Name),
		Containers:         []corev1.Container{container},
		Volumes:            volumes,
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:      &[]int64{10001}[0],
			RunAsNonRoot: &[]bool{true}[0],
			RunAsUser:    &[]int64{10001}[0],
		},
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

// generateLokiConfig generates the loki.yaml configuration
func (m *LokiManager) generateLokiConfig(platform *observabilityv1beta1.ObservabilityPlatform, lokiSpec *observabilityv1beta1.LokiSpec) string {
	retention := lokiSpec.Retention
	if retention == "" {
		retention = defaultRetention
	}
	
	config := fmt.Sprintf(`auth_enabled: false

server:
  http_listen_port: %d
  grpc_listen_port: %d
  log_level: %s

common:
  path_prefix: %s
  storage:`, defaultHTTPPort, defaultGRPCPort, platform.Spec.Global.LogLevel, defaultDataPath)
	
	// Configure storage backend
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		config += fmt.Sprintf(`
    s3:
      s3: s3://%s
      bucketnames: %s`, lokiSpec.S3.Region, lokiSpec.S3.BucketName)
		
		if lokiSpec.S3.Endpoint != "" {
			config += fmt.Sprintf(`
      endpoint: %s
      s3forcepathstyle: true`, lokiSpec.S3.Endpoint)
		}
	} else {
		config += `
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules`
	}
	
	config += fmt.Sprintf(`
  replication_factor: 1
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory

compactor:
  working_directory: %s/boltdb-shipper-compactor
  shared_store: %s
  compaction_interval: 10m
  retention_enabled: true
  retention_delete_delay: 2h
  retention_delete_worker_count: 150

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 168h
  ingestion_rate_mb: 16
  ingestion_burst_size_mb: 32
  per_stream_rate_limit: 5MB
  per_stream_rate_limit_burst: 20MB
  retention_period: %s

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper`, defaultDataPath, m.getStorageType(lokiSpec), retention)
	
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		config += `
      object_store: s3`
	} else {
		config += `
      object_store: filesystem`
	}
	
	config += `
      schema: v11
      index:
        prefix: index_
        period: 24h

ingester:
  wal:
    enabled: true
    dir: /wal
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
    final_sleep: 0s
  chunk_idle_period: 1h
  max_chunk_age: 1h
  chunk_target_size: 1048576
  chunk_retain_period: 30s
  max_transfer_retries: 0

storage_config:
  boltdb_shipper:
    active_index_directory: /loki/boltdb-shipper-active
    cache_location: /loki/boltdb-shipper-cache
    cache_ttl: 24h
    shared_store: `
	
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		config += `s3`
	} else {
		config += `filesystem`
	}
	
	config += `

chunk_store_config:
  max_look_back_period: 0s

table_manager:
  retention_deletes_enabled: true
  retention_period: ` + retention + `

ruler:
  storage:
    type: local
    local:
      directory: /loki/rules-temp
  rule_path: /loki/rules-temp
  alertmanager_url: http://alertmanager:9093
  ring:
    kvstore:
      store: inmemory
  enable_api: true

query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 100

frontend:
  compress_responses: true
  log_queries_longer_than: 5s`
	
	return config
}

// getStorageType returns the storage type based on configuration
func (m *LokiManager) getStorageType(lokiSpec *observabilityv1beta1.LokiSpec) string {
	if lokiSpec.S3 != nil && lokiSpec.S3.Enabled {
		return "s3"
	}
	return "filesystem"
}

// Helper methods for resource naming
func (m *LokiManager) getS3SecretName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s-s3", platform.Name)
}

func (m *LokiManager) getConfigMapName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s-config", platform.Name)
}

func (m *LokiManager) getServiceName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s", platform.Name)
}

func (m *LokiManager) getHeadlessServiceName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s-headless", platform.Name)
}

func (m *LokiManager) getStatefulSetName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s", platform.Name)
}

func (m *LokiManager) getCompactorName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s-compactor", platform.Name)
}

func (m *LokiManager) getPDBName(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("loki-%s", platform.Name)
}

// Helper methods for labels
func (m *LokiManager) getLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       componentName,
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
		"app.kubernetes.io/part-of":    "observability-platform",
		"app.kubernetes.io/component":  labelComponent,
		"observability.io/platform":    platform.Name,
	}
}

func (m *LokiManager) getSelectorLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      componentName,
		"app.kubernetes.io/instance":  platform.Name,
		"app.kubernetes.io/component": labelComponent,
	}
}

func (m *LokiManager) getCompactorLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      componentName,
		"app.kubernetes.io/instance":  platform.Name,
		"app.kubernetes.io/component": "compactor",
	}
}
