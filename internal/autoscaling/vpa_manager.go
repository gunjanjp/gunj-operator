package autoscaling

import (
	"context"
	"fmt"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VPAManager manages vertical pod autoscalers
type VPAManager struct {
	client client.Client
}

// NewVPAManager creates a new VPA manager
func NewVPAManager(client client.Client) *VPAManager {
	return &VPAManager{
		client: client,
	}
}

// CreateOrUpdateVPA creates or updates a VPA for a component
func (m *VPAManager) CreateOrUpdateVPA(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, policy ScalingPolicy) error {
	log := log.FromContext(ctx)
	
	vpa := m.buildVPA(platform, component, policy)
	
	// Check if VPA already exists
	existing := &vpav1.VerticalPodAutoscaler{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      vpa.Name,
		Namespace: vpa.Namespace,
	}, existing)
	
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new VPA
			log.Info("Creating VPA", "component", component, "name", vpa.Name)
			if err := m.client.Create(ctx, vpa); err != nil {
				return fmt.Errorf("failed to create VPA: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get existing VPA: %w", err)
	}
	
	// Update existing VPA
	existing.Spec = vpa.Spec
	existing.Labels = vpa.Labels
	existing.Annotations = vpa.Annotations
	
	log.Info("Updating VPA", "component", component, "name", vpa.Name)
	if err := m.client.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update VPA: %w", err)
	}
	
	return nil
}

// DeleteVPA deletes a VPA for a component
func (m *VPAManager) DeleteVPA(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) error {
	log := log.FromContext(ctx)
	
	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getVPAName(platform, component),
			Namespace: platform.Namespace,
		},
	}
	
	log.Info("Deleting VPA", "component", component, "name", vpa.Name)
	if err := m.client.Delete(ctx, vpa); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete VPA: %w", err)
	}
	
	return nil
}

// GetVPAStatus gets the status of a VPA
func (m *VPAManager) GetVPAStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) (*vpav1.VerticalPodAutoscaler, error) {
	vpa := &vpav1.VerticalPodAutoscaler{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      m.getVPAName(platform, component),
		Namespace: platform.Namespace,
	}, vpa)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get VPA status: %w", err)
	}
	
	return vpa, nil
}

// GetRecommendation gets the resource recommendation from VPA
func (m *VPAManager) GetRecommendation(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) (*ResourceRecommendation, error) {
	vpa, err := m.GetVPAStatus(ctx, platform, component)
	if err != nil {
		return nil, err
	}
	
	if vpa.Status.Recommendation == nil {
		return nil, fmt.Errorf("no recommendation available yet")
	}
	
	recommendation := &ResourceRecommendation{
		Component:   component,
		Confidence:  m.calculateConfidence(vpa),
		BasedOnDays: m.getDataDays(vpa),
	}
	
	// Process container recommendations
	for _, containerRec := range vpa.Status.Recommendation.ContainerRecommendations {
		if containerRec.ContainerName == string(component) {
			recommendation.Container = containerRec.ContainerName
			
			// Current resources (from target)
			if containerRec.Target != nil {
				recommendation.RecommendedRequests = containerRec.Target
			}
			
			// Upper and lower bounds for limits
			if containerRec.UpperBound != nil {
				recommendation.RecommendedLimits = containerRec.UpperBound
			}
			
			// Calculate cost impact
			recommendation.CostSavings = m.calculateCostSavings(recommendation)
			
			break
		}
	}
	
	return recommendation, nil
}

// buildVPA builds a VPA resource
func (m *VPAManager) buildVPA(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, policy ScalingPolicy) *vpav1.VerticalPodAutoscaler {
	updateMode := vpav1.UpdateModeAuto
	if policy.Type == VerticalScaling {
		// For pure vertical scaling, we can be more aggressive
		updateMode = vpav1.UpdateModeAuto
	} else {
		// For mixed scaling, be more conservative
		updateMode = vpav1.UpdateModeOff
	}
	
	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.getVPAName(platform, component),
			Namespace: platform.Namespace,
			Labels:    m.getLabels(platform, component),
			Annotations: map[string]string{
				"observability.io/managed-by": "gunj-operator",
				"observability.io/component":  string(component),
			},
		},
		Spec: vpav1.VerticalPodAutoscalerSpec{
			TargetRef: &vpav1.TargetRef{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       m.getDeploymentName(component),
			},
			UpdatePolicy: &vpav1.PodUpdatePolicy{
				UpdateMode: &updateMode,
			},
			ResourcePolicy: m.buildResourcePolicy(component, policy),
		},
	}
	
	// Set owner reference
	if err := utils.SetOwnerReference(platform, vpa, platform.GetScheme()); err == nil {
		// Error is logged in SetOwnerReference
	}
	
	return vpa
}

// buildResourcePolicy builds the resource policy for VPA
func (m *VPAManager) buildResourcePolicy(component v1beta1.ComponentType, policy ScalingPolicy) *vpav1.PodResourcePolicy {
	resourcePolicy := &vpav1.PodResourcePolicy{
		ContainerPolicies: []vpav1.ContainerResourcePolicy{
			{
				ContainerName: string(component),
				Mode:          &[]vpav1.ContainerScalingMode{vpav1.ContainerScalingModeAuto}[0],
			},
		},
	}
	
	// Set minimum and maximum allowed resources based on component type
	minAllowed, maxAllowed := m.getResourceBounds(component, policy)
	
	resourcePolicy.ContainerPolicies[0].MinAllowed = minAllowed
	resourcePolicy.ContainerPolicies[0].MaxAllowed = maxAllowed
	
	// Add controlled resources
	resourcePolicy.ContainerPolicies[0].ControlledResources = &[]corev1.ResourceName{
		corev1.ResourceCPU,
		corev1.ResourceMemory,
	}
	
	return resourcePolicy
}

// getResourceBounds gets the resource bounds for a component
func (m *VPAManager) getResourceBounds(component v1beta1.ComponentType, policy ScalingPolicy) (corev1.ResourceList, corev1.ResourceList) {
	// Default bounds based on component type
	var minCPU, maxCPU, minMemory, maxMemory resource.Quantity
	
	switch component {
	case v1beta1.ComponentPrometheus:
		minCPU = resource.MustParse("100m")
		maxCPU = resource.MustParse("4")
		minMemory = resource.MustParse("512Mi")
		maxMemory = resource.MustParse("16Gi")
	case v1beta1.ComponentGrafana:
		minCPU = resource.MustParse("50m")
		maxCPU = resource.MustParse("2")
		minMemory = resource.MustParse("256Mi")
		maxMemory = resource.MustParse("4Gi")
	case v1beta1.ComponentLoki:
		minCPU = resource.MustParse("100m")
		maxCPU = resource.MustParse("2")
		minMemory = resource.MustParse("512Mi")
		maxMemory = resource.MustParse("8Gi")
	case v1beta1.ComponentTempo:
		minCPU = resource.MustParse("100m")
		maxCPU = resource.MustParse("2")
		minMemory = resource.MustParse("512Mi")
		maxMemory = resource.MustParse("8Gi")
	default:
		minCPU = resource.MustParse("50m")
		maxCPU = resource.MustParse("1")
		minMemory = resource.MustParse("128Mi")
		maxMemory = resource.MustParse("2Gi")
	}
	
	// Apply resource buffer if specified
	if policy.ResourceBuffer > 0 {
		bufferFactor := 1 + float64(policy.ResourceBuffer)/100
		maxCPU = *resource.NewMilliQuantity(int64(float64(maxCPU.MilliValue())*bufferFactor), resource.DecimalSI)
		maxMemory = *resource.NewQuantity(int64(float64(maxMemory.Value())*bufferFactor), resource.BinarySI)
	}
	
	minAllowed := corev1.ResourceList{
		corev1.ResourceCPU:    minCPU,
		corev1.ResourceMemory: minMemory,
	}
	
	maxAllowed := corev1.ResourceList{
		corev1.ResourceCPU:    maxCPU,
		corev1.ResourceMemory: maxMemory,
	}
	
	return minAllowed, maxAllowed
}

// calculateConfidence calculates the confidence level of VPA recommendation
func (m *VPAManager) calculateConfidence(vpa *vpav1.VerticalPodAutoscaler) float64 {
	// Base confidence on the amount of data and stability
	confidence := 0.5
	
	if vpa.Status.Recommendation != nil {
		// Increase confidence based on data points
		// This is a simplified calculation
		confidence = 0.8
	}
	
	return confidence
}

// getDataDays gets the number of days of data used for recommendation
func (m *VPAManager) getDataDays(vpa *vpav1.VerticalPodAutoscaler) int {
	// VPA typically uses 8 days of data by default
	return 8
}

// calculateCostSavings calculates potential cost savings
func (m *VPAManager) calculateCostSavings(recommendation *ResourceRecommendation) *CostImpact {
	// Simplified cost calculation
	// In production, this would integrate with cloud provider pricing APIs
	
	// Assume costs per resource unit (example values)
	cpuCostPerCore := 0.05      // $/hour/core
	memoryCostPerGB := 0.01      // $/hour/GB
	
	currentCPU := recommendation.CurrentRequests[corev1.ResourceCPU].AsApproximateFloat64()
	recommendedCPU := recommendation.RecommendedRequests[corev1.ResourceCPU].AsApproximateFloat64()
	
	currentMemoryGB := recommendation.CurrentRequests[corev1.ResourceMemory].AsApproximateFloat64() / (1024 * 1024 * 1024)
	recommendedMemoryGB := recommendation.RecommendedRequests[corev1.ResourceMemory].AsApproximateFloat64() / (1024 * 1024 * 1024)
	
	currentHourlyCost := (currentCPU * cpuCostPerCore) + (currentMemoryGB * memoryCostPerGB)
	recommendedHourlyCost := (recommendedCPU * cpuCostPerCore) + (recommendedMemoryGB * memoryCostPerGB)
	
	monthlySavings := (currentHourlyCost - recommendedHourlyCost) * 24 * 30
	
	return &CostImpact{
		CurrentCost:    currentHourlyCost * 24 * 30,  // Monthly cost
		ProjectedCost:  recommendedHourlyCost * 24 * 30,
		MonthlySavings: monthlySavings,
		Currency:       "USD",
	}
}

// getVPAName gets the VPA name for a component
func (m *VPAManager) getVPAName(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) string {
	return fmt.Sprintf("%s-%s-vpa", platform.Name, component)
}

// getDeploymentName gets the deployment name for a component
func (m *VPAManager) getDeploymentName(component v1beta1.ComponentType) string {
	// This should match the deployment names created by component managers
	return fmt.Sprintf("%s-deployment", component)
}

// getLabels gets the labels for VPA
func (m *VPAManager) getLabels(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       string(component),
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "vpa",
		"app.kubernetes.io/part-of":    "gunj-operator",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
		"observability.io/component":   string(component),
	}
}
