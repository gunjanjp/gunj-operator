// Package example demonstrates the Golang coding standards for the Gunj Operator.
// This file serves as a practical reference for developers to understand how
// to apply the coding standards in real code.
package example

import (
    "context"
    "fmt"
    "time"
    
    "github.com/go-logr/logr"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// Constants follow PascalCase naming
const (
    // DefaultReconcileTimeout is the default timeout for reconciliation operations
    DefaultReconcileTimeout = 2 * time.Minute
    
    // MaxRetries defines the maximum number of retry attempts
    MaxRetries = 5
    
    // FinalizerName is the name of the finalizer used by this controller
    FinalizerName = "observability.io/finalizer"
)

// Error variables are defined at package level with descriptive names
var (
    // ErrPlatformNotReady indicates the platform is not in a ready state
    ErrPlatformNotReady = fmt.Errorf("platform is not ready")
    
    // ErrInvalidConfiguration indicates the configuration is invalid
    ErrInvalidConfiguration = fmt.Errorf("invalid configuration")
)

// ComponentManager defines the interface for managing observability components.
// It follows the interface segregation principle with focused methods.
type ComponentManager interface {
    // Deploy creates or updates the component in the cluster
    Deploy(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
    
    // Delete removes the component from the cluster
    Delete(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
    
    // GetStatus returns the current status of the component
    GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ComponentStatus, error)
}

// ComponentStatus represents the status of an observability component.
// It uses clear field names without abbreviations.
type ComponentStatus struct {
    // Ready indicates if the component is ready to serve traffic
    Ready bool
    
    // Message provides additional information about the status
    Message string
    
    // LastUpdateTime is when the status was last updated
    LastUpdateTime time.Time
}

// PrometheusManager manages Prometheus deployments for the Gunj Operator.
// It implements the ComponentManager interface.
type PrometheusManager struct {
    client client.Client
    scheme *runtime.Scheme
    log    logr.Logger
}

// NewPrometheusManager creates a new PrometheusManager instance.
// Constructor functions use the "New" prefix convention.
func NewPrometheusManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *PrometheusManager {
    return &PrometheusManager{
        client: client,
        scheme: scheme,
        log:    log.WithName("prometheus-manager"),
    }
}

// Deploy ensures Prometheus is deployed according to the platform specification.
// It demonstrates proper error handling and logging practices.
func (m *PrometheusManager) Deploy(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
    m.log.Info("Deploying Prometheus", "platform", platform.Name, "namespace", platform.Namespace)
    
    // Validate input
    if platform == nil {
        return fmt.Errorf("platform cannot be nil")
    }
    
    if platform.Spec.Components.Prometheus == nil || !platform.Spec.Components.Prometheus.Enabled {
        m.log.Info("Prometheus is not enabled, skipping deployment")
        return nil
    }
    
    // Build the deployment
    deployment, err := m.buildDeployment(platform)
    if err != nil {
        return fmt.Errorf("building Prometheus deployment: %w", err)
    }
    
    // Set owner reference for garbage collection
    if err := ctrl.SetControllerReference(platform, deployment, m.scheme); err != nil {
        return fmt.Errorf("setting controller reference: %w", err)
    }
    
    // Create or update the deployment
    if err := m.createOrUpdate(ctx, deployment); err != nil {
        return fmt.Errorf("creating or updating Prometheus deployment: %w", err)
    }
    
    // Create associated resources
    if err := m.createService(ctx, platform); err != nil {
        return fmt.Errorf("creating Prometheus service: %w", err)
    }
    
    if err := m.createConfigMap(ctx, platform); err != nil {
        return fmt.Errorf("creating Prometheus config: %w", err)
    }
    
    m.log.Info("Successfully deployed Prometheus", "platform", platform.Name)
    return nil
}

// buildDeployment constructs a Deployment resource for Prometheus.
// It demonstrates the builder pattern and proper resource construction.
func (m *PrometheusManager) buildDeployment(platform *observabilityv1.ObservabilityPlatform) (*appsv1.Deployment, error) {
    prometheusSpec := platform.Spec.Components.Prometheus
    
    // Use a pointer for replicas as it's optional in the API
    replicas := int32(1)
    if prometheusSpec.Replicas != nil {
        replicas = *prometheusSpec.Replicas
    }
    
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-prometheus", platform.Name),
            Namespace: platform.Namespace,
            Labels:    m.getLabels(platform),
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: m.getSelectorLabels(platform),
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: m.getSelectorLabels(platform),
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "prometheus",
                            Image: fmt.Sprintf("prom/prometheus:%s", prometheusSpec.Version),
                            Ports: []corev1.ContainerPort{
                                {
                                    Name:          "web",
                                    ContainerPort: 9090,
                                    Protocol:      corev1.ProtocolTCP,
                                },
                            },
                            Resources: prometheusSpec.Resources,
                            VolumeMounts: []corev1.VolumeMount{
                                {
                                    Name:      "config",
                                    MountPath: "/etc/prometheus",
                                },
                                {
                                    Name:      "data",
                                    MountPath: "/prometheus",
                                },
                            },
                        },
                    },
                    Volumes: m.buildVolumes(platform),
                },
            },
        },
    }
    
    return deployment, nil
}

// createOrUpdate creates a resource if it doesn't exist, or updates it if it does.
// This demonstrates proper Kubernetes client usage and error handling.
func (m *PrometheusManager) createOrUpdate(ctx context.Context, obj client.Object) error {
    key := client.ObjectKeyFromObject(obj)
    existing := obj.DeepCopyObject().(client.Object)
    
    // Check if the resource exists
    err := m.client.Get(ctx, key, existing)
    if err != nil {
        if errors.IsNotFound(err) {
            // Resource doesn't exist, create it
            m.log.Info("Creating resource", "kind", obj.GetObjectKind(), "name", obj.GetName())
            return m.client.Create(ctx, obj)
        }
        return fmt.Errorf("getting existing resource: %w", err)
    }
    
    // Resource exists, update it
    m.log.Info("Updating resource", "kind", obj.GetObjectKind(), "name", obj.GetName())
    return m.client.Update(ctx, obj)
}

// getLabels returns the standard labels for Prometheus resources.
// It demonstrates consistent labeling practices.
func (m *PrometheusManager) getLabels(platform *observabilityv1.ObservabilityPlatform) map[string]string {
    return map[string]string{
        "app.kubernetes.io/name":       "prometheus",
        "app.kubernetes.io/instance":   platform.Name,
        "app.kubernetes.io/component":  "monitoring",
        "app.kubernetes.io/part-of":    "gunj-operator",
        "app.kubernetes.io/managed-by": "gunj-operator",
    }
}

// getSelectorLabels returns the labels used for pod selection.
// Selector labels must be a subset of labels and remain stable.
func (m *PrometheusManager) getSelectorLabels(platform *observabilityv1.ObservabilityPlatform) map[string]string {
    return map[string]string{
        "app.kubernetes.io/name":     "prometheus",
        "app.kubernetes.io/instance": platform.Name,
    }
}

// buildVolumes constructs the volume specifications for Prometheus.
// It shows proper volume configuration for stateful applications.
func (m *PrometheusManager) buildVolumes(platform *observabilityv1.ObservabilityPlatform) []corev1.Volume {
    volumes := []corev1.Volume{
        {
            Name: "config",
            VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: fmt.Sprintf("%s-prometheus-config", platform.Name),
                    },
                },
            },
        },
    }
    
    // Add data volume based on storage configuration
    if platform.Spec.Components.Prometheus.Storage != nil {
        volumes = append(volumes, corev1.Volume{
            Name: "data",
            VolumeSource: corev1.VolumeSource{
                PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
                    ClaimName: fmt.Sprintf("%s-prometheus-data", platform.Name),
                },
            },
        })
    } else {
        // Use emptyDir for testing or non-production environments
        volumes = append(volumes, corev1.Volume{
            Name: "data",
            VolumeSource: corev1.VolumeSource{
                EmptyDir: &corev1.EmptyDirVolumeSource{},
            },
        })
    }
    
    return volumes
}

// createService creates the Service resource for Prometheus.
// It demonstrates service creation with proper port configuration.
func (m *PrometheusManager) createService(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-prometheus", platform.Name),
            Namespace: platform.Namespace,
            Labels:    m.getLabels(platform),
        },
        Spec: corev1.ServiceSpec{
            Selector: m.getSelectorLabels(platform),
            Ports: []corev1.ServicePort{
                {
                    Name:       "web",
                    Port:       9090,
                    TargetPort: intstr.FromString("web"),
                    Protocol:   corev1.ProtocolTCP,
                },
            },
            Type: corev1.ServiceTypeClusterIP,
        },
    }
    
    // Set owner reference
    if err := ctrl.SetControllerReference(platform, service, m.scheme); err != nil {
        return fmt.Errorf("setting controller reference: %w", err)
    }
    
    return m.createOrUpdate(ctx, service)
}

// createConfigMap creates the ConfigMap for Prometheus configuration.
// It shows proper ConfigMap creation and configuration management.
func (m *PrometheusManager) createConfigMap(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
    config := m.generatePrometheusConfig(platform)
    
    configMap := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-prometheus-config", platform.Name),
            Namespace: platform.Namespace,
            Labels:    m.getLabels(platform),
        },
        Data: map[string]string{
            "prometheus.yml": config,
        },
    }
    
    // Set owner reference
    if err := ctrl.SetControllerReference(platform, configMap, m.scheme); err != nil {
        return fmt.Errorf("setting controller reference: %w", err)
    }
    
    return m.createOrUpdate(ctx, configMap)
}

// generatePrometheusConfig generates the Prometheus configuration file.
// It demonstrates configuration generation based on platform spec.
func (m *PrometheusManager) generatePrometheusConfig(platform *observabilityv1.ObservabilityPlatform) string {
    // This is a simplified example. In production, use a proper template engine.
    config := fmt.Sprintf(`global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: '%s'
    platform: '%s'

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
    - targets: ['localhost:9090']

  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
    - role: pod
      namespaces:
        names:
        - %s
`, platform.Spec.Global.ExternalLabels["cluster"], platform.Name, platform.Namespace)
    
    return config
}

// Delete removes Prometheus and all associated resources.
// Demonstrates proper cleanup and resource deletion.
func (m *PrometheusManager) Delete(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
    m.log.Info("Deleting Prometheus", "platform", platform.Name, "namespace", platform.Namespace)
    
    // Resources will be automatically deleted due to owner references
    // This method can be used for any additional cleanup if needed
    
    return nil
}

// GetStatus checks the status of Prometheus deployment.
// It demonstrates health checking and status reporting.
func (m *PrometheusManager) GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ComponentStatus, error) {
    deployment := &appsv1.Deployment{}
    key := client.ObjectKey{
        Name:      fmt.Sprintf("%s-prometheus", platform.Name),
        Namespace: platform.Namespace,
    }
    
    if err := m.client.Get(ctx, key, deployment); err != nil {
        if errors.IsNotFound(err) {
            return ComponentStatus{
                Ready:   false,
                Message: "Prometheus deployment not found",
            }, nil
        }
        return ComponentStatus{}, fmt.Errorf("getting Prometheus deployment: %w", err)
    }
    
    // Check if deployment is ready
    ready := deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
    
    status := ComponentStatus{
        Ready:          ready,
        LastUpdateTime: time.Now(),
    }
    
    if ready {
        status.Message = fmt.Sprintf("Prometheus is ready with %d/%d replicas", 
            deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
    } else {
        status.Message = fmt.Sprintf("Prometheus is not ready: %d/%d replicas ready", 
            deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
    }
    
    return status, nil
}
