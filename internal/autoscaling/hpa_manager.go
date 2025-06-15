package autoscaling

import (
	"context"
	"fmt"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/utils"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HPAManager manages horizontal pod autoscalers
type HPAManager struct {
	client client.Client
}

// NewHPAManager creates a new HPA manager
func NewHPAManager(client client.Client) *HPAManager {
	return &HPAManager{
		client: client,
	}
}

// CreateOrUpdateHPA creates or updates an HPA for a component
func (m *HPAManager) CreateOrUpdateHPA(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, policy ScalingPolicy) error {
	log := log.FromContext(ctx)
	
	hpa := m.buildHPA(platform, component, policy)
	
	// Check if HPA already exists
	existing := &autoscalingv2.HorizontalPodAutoscaler{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      hpa.Name,
		Namespace: hpa.Namespace,
	}, existing)
	
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new HPA
			log.Info("Creating HPA", "component", component, "name", hpa.Name)
			if err := m.client.Create(ctx, hpa); err != nil {
				return fmt.Errorf("failed to create HPA: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get existing HPA: %w", err)
	}
	
	// Update existing HPA
	existing.Spec = hpa.Spec
	existing.Labels = hpa.Labels
	existing.Annotations = hpa.Annotations
	
	log.Info("Updating HPA", "component", component, "name", hpa.Name)
	if err := m.client.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update HPA: %w", err)
	}
	
	return nil
}

// DeleteHPA deletes an HPA for a component
func (m *HPAManager) DeleteHPA(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) error {
	log := log.FromContext(ctx)
	
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getHPAName(platform, component),
			Namespace: platform.Namespace,
		},
	}
	
	log.Info("Deleting HPA", "component", component, "name", hpa.Name)
	if err := m.client.Delete(ctx, hpa); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete HPA: %w", err)
	}
	
	return nil
}

// GetHPAStatus gets the status of an HPA
func (m *HPAManager) GetHPAStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      m.getHPAName(platform, component),
		Namespace: platform.Namespace,
	}, hpa)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get HPA status: %w", err)
	}
	
	return hpa, nil
}

// buildHPA builds an HPA resource
func (m *HPAManager) buildHPA(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, policy ScalingPolicy) *autoscalingv2.HorizontalPodAutoscaler {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getHPAName(platform, component),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform, component),
			Annotations: map[string]string{
				"observability.io/managed-by": "gunj-operator",
				"observability.io/component":  string(component),
			},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       m.getDeploymentName(component),
			},
			MinReplicas: &policy.MinReplicas,
			MaxReplicas: policy.MaxReplicas,
			Metrics:     m.buildMetrics(policy),
			Behavior:    m.buildBehavior(policy),
		},
	}
	
	// Set owner reference
	if err := utils.SetOwnerReference(platform, hpa, platform.GetScheme()); err == nil {
		// Error is logged in SetOwnerReference
	}
	
	return hpa
}

// buildMetrics builds the metrics for HPA
func (m *HPAManager) buildMetrics(policy ScalingPolicy) []autoscalingv2.MetricSpec {
	metrics := []autoscalingv2.MetricSpec{}
	
	// CPU metric
	if policy.TargetCPUPercent != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: policy.TargetCPUPercent,
				},
			},
		})
	}
	
	// Memory metric
	if policy.TargetMemoryPercent != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: policy.TargetMemoryPercent,
				},
			},
		})
	}
	
	// Custom metrics
	for _, cm := range policy.CustomMetrics {
		metric := autoscalingv2.MetricSpec{}
		
		switch cm.Type {
		case "prometheus":
			quantity, _ := resource.ParseQuantity(cm.TargetValue)
			metric = autoscalingv2.MetricSpec{
				Type: autoscalingv2.PodsMetricSourceType,
				Pods: &autoscalingv2.PodsMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: cm.Name,
					},
					Target: autoscalingv2.MetricTarget{
						Type:         cm.MetricType,
						AverageValue: &quantity,
					},
				},
			}
		case "external":
			quantity, _ := resource.ParseQuantity(cm.TargetValue)
			metric = autoscalingv2.MetricSpec{
				Type: autoscalingv2.ExternalMetricSourceType,
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: cm.Name,
					},
					Target: autoscalingv2.MetricTarget{
						Type:  cm.MetricType,
						Value: &quantity,
					},
				},
			}
		}
		
		metrics = append(metrics, metric)
	}
	
	return metrics
}

// buildBehavior builds the scaling behavior
func (m *HPAManager) buildBehavior(policy ScalingPolicy) *autoscalingv2.HorizontalPodAutoscalerBehavior {
	behavior := &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleUp: &autoscalingv2.HPAScalingRules{
			StabilizationWindowSeconds: int32Ptr(60), // 1 minute
			SelectPolicy:               &[]autoscalingv2.ScalingPolicySelect{autoscalingv2.MaxChangePolicySelect}[0],
			Policies: []autoscalingv2.HPAScalingPolicy{
				{
					Type:          autoscalingv2.PercentScalingPolicy,
					Value:         100, // Double the pods
					PeriodSeconds: 60,
				},
				{
					Type:          autoscalingv2.PodsScalingPolicy,
					Value:         4, // Add max 4 pods
					PeriodSeconds: 60,
				},
			},
		},
		ScaleDown: &autoscalingv2.HPAScalingRules{
			SelectPolicy: &[]autoscalingv2.ScalingPolicySelect{autoscalingv2.MinChangePolicySelect}[0],
			Policies: []autoscalingv2.HPAScalingPolicy{
				{
					Type:          autoscalingv2.PercentScalingPolicy,
					Value:         10, // Scale down by 10%
					PeriodSeconds: 60,
				},
			},
		},
	}
	
	// Apply custom stabilization window for scale down
	if policy.ScaleDownStabilizationWindow != nil {
		behavior.ScaleDown.StabilizationWindowSeconds = int32Ptr(int32(policy.ScaleDownStabilizationWindow.Seconds()))
	} else {
		behavior.ScaleDown.StabilizationWindowSeconds = int32Ptr(300) // Default 5 minutes
	}
	
	return behavior
}

// getHPAName gets the HPA name for a component
func (m *HPAManager) getHPAName(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) string {
	return fmt.Sprintf("%s-%s-hpa", platform.Name, component)
}

// getDeploymentName gets the deployment name for a component
func (m *HPAManager) getDeploymentName(component v1beta1.ComponentType) string {
	// This should match the deployment names created by component managers
	return fmt.Sprintf("%s-deployment", component)
}

// getLabels gets the labels for HPA
func (m *HPAManager) getLabels(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       string(component),
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "hpa",
		"app.kubernetes.io/part-of":    "gunj-operator",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
		"observability.io/component":   string(component),
	}
}

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}
