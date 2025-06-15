/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package quota

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var log = logf.Log.WithName("resource-quota-validator")

// ResourceQuotaValidator validates resource requests against namespace and cluster quotas
type ResourceQuotaValidator struct {
	Client client.Client
	// ClusterResourceLimits defines cluster-wide resource limits
	ClusterResourceLimits *ClusterResourceLimits
}

// ClusterResourceLimits represents cluster-wide resource constraints
type ClusterResourceLimits struct {
	MaxCPUPerNamespace    resource.Quantity
	MaxMemoryPerNamespace resource.Quantity
	MaxStoragePerPVC      resource.Quantity
	MaxPVCPerNamespace    int
	MaxPodsPerNamespace   int
}

// ComponentResources represents aggregated resources for a component
type ComponentResources struct {
	CPU     resource.Quantity
	Memory  resource.Quantity
	Storage resource.Quantity
	Pods    int
	PVCs    int
}

// NamespaceResourceUsage represents current resource usage in a namespace
type NamespaceResourceUsage struct {
	UsedCPU     resource.Quantity
	UsedMemory  resource.Quantity
	UsedStorage resource.Quantity
	UsedPods    int
	UsedPVCs    int
}

// ValidateResourceQuota validates the platform's resource requirements against quotas
func (v *ResourceQuotaValidator) ValidateResourceQuota(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	
	log.V(1).Info("Validating resource quota", "platform", platform.Name, "namespace", platform.Namespace)
	
	// Calculate total resources required by the platform
	totalResources, err := v.calculatePlatformResources(platform)
	if err != nil {
		allErrs = append(allErrs, field.InternalError(field.NewPath("spec"), err))
		return allErrs
	}
	
	// Check namespace resource quotas
	if errs := v.validateNamespaceQuotas(ctx, platform.Namespace, totalResources); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	
	// Check cluster-wide limits if configured
	if v.ClusterResourceLimits != nil {
		if errs := v.validateClusterLimits(ctx, platform.Namespace, totalResources); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	
	// Check for multi-tenancy constraints
	if errs := v.validateMultiTenancy(ctx, platform); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}
	
	return allErrs
}

// calculatePlatformResources calculates total resources required by all components
func (v *ResourceQuotaValidator) calculatePlatformResources(platform *observabilityv1beta1.ObservabilityPlatform) (*ComponentResources, error) {
	total := &ComponentResources{
		CPU:     resource.Quantity{},
		Memory:  resource.Quantity{},
		Storage: resource.Quantity{},
		Pods:    0,
		PVCs:    0,
	}
	
	// Calculate Prometheus resources
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		prom := platform.Spec.Components.Prometheus
		if err := v.addComponentResources(total, prom.Resources, prom.Replicas, prom.Storage); err != nil {
			return nil, fmt.Errorf("calculating prometheus resources: %w", err)
		}
		// Prometheus uses 1 PVC per replica
		total.PVCs += int(prom.Replicas)
	}
	
	// Calculate Grafana resources
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		grafana := platform.Spec.Components.Grafana
		if err := v.addComponentResources(total, grafana.Resources, grafana.Replicas, nil); err != nil {
			return nil, fmt.Errorf("calculating grafana resources: %w", err)
		}
		// Grafana uses 1 PVC if persistence is enabled
		if grafana.Persistence != nil && grafana.Persistence.Enabled {
			total.PVCs++
		}
	}
	
	// Calculate Loki resources
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		loki := platform.Spec.Components.Loki
		resources := loki.Resources
		replicas := loki.Replicas
		if replicas == 0 {
			replicas = 1
		}
		
		var storage *observabilityv1beta1.StorageSpec
		if loki.Storage != nil {
			storage = &observabilityv1beta1.StorageSpec{
				Size: loki.Storage.Size,
			}
		}
		
		if err := v.addComponentResources(total, resources, replicas, storage); err != nil {
			return nil, fmt.Errorf("calculating loki resources: %w", err)
		}
		// Loki uses 1 PVC if not using S3
		if loki.Storage != nil && (loki.Storage.S3 == nil || !loki.Storage.S3.Enabled) {
			total.PVCs++
		}
	}
	
	// Calculate Tempo resources
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		tempo := platform.Spec.Components.Tempo
		replicas := tempo.Replicas
		if replicas == 0 {
			replicas = 1
		}
		
		if err := v.addComponentResources(total, tempo.Resources, replicas, tempo.Storage); err != nil {
			return nil, fmt.Errorf("calculating tempo resources: %w", err)
		}
		// Tempo uses 1 PVC
		total.PVCs++
	}
	
	// Calculate Alertmanager resources if enabled
	if platform.Spec.Alerting != nil && platform.Spec.Alerting.Alertmanager != nil && platform.Spec.Alerting.Alertmanager.Enabled {
		am := platform.Spec.Alerting.Alertmanager
		if err := v.addComponentResources(total, am.Resources, am.Replicas, am.Storage); err != nil {
			return nil, fmt.Errorf("calculating alertmanager resources: %w", err)
		}
		// Alertmanager uses 1 PVC per replica
		total.PVCs += int(am.Replicas)
	}
	
	log.V(2).Info("Calculated platform resources", 
		"cpu", total.CPU.String(),
		"memory", total.Memory.String(),
		"storage", total.Storage.String(),
		"pods", total.Pods,
		"pvcs", total.PVCs)
	
	return total, nil
}

// addComponentResources adds a component's resources to the total
func (v *ResourceQuotaValidator) addComponentResources(total *ComponentResources, resources *observabilityv1beta1.ResourceRequirements, replicas int32, storage *observabilityv1beta1.StorageSpec) error {
	if replicas <= 0 {
		replicas = 1
	}
	
	// Add CPU
	if resources != nil && resources.Requests != nil && resources.Requests.CPU != "" {
		cpuQuantity, err := resource.ParseQuantity(resources.Requests.CPU)
		if err != nil {
			return fmt.Errorf("parsing CPU quantity: %w", err)
		}
		// Multiply by replicas
		for i := int32(0); i < replicas; i++ {
			total.CPU.Add(cpuQuantity)
		}
	}
	
	// Add Memory
	if resources != nil && resources.Requests != nil && resources.Requests.Memory != "" {
		memQuantity, err := resource.ParseQuantity(resources.Requests.Memory)
		if err != nil {
			return fmt.Errorf("parsing memory quantity: %w", err)
		}
		// Multiply by replicas
		for i := int32(0); i < replicas; i++ {
			total.Memory.Add(memQuantity)
		}
	}
	
	// Add Storage
	if storage != nil && storage.Size != "" {
		storageQuantity, err := resource.ParseQuantity(storage.Size)
		if err != nil {
			return fmt.Errorf("parsing storage quantity: %w", err)
		}
		total.Storage.Add(storageQuantity)
	}
	
	// Add pod count
	total.Pods += int(replicas)
	
	return nil
}

// validateNamespaceQuotas checks against namespace ResourceQuotas
func (v *ResourceQuotaValidator) validateNamespaceQuotas(ctx context.Context, namespace string, required *ComponentResources) field.ErrorList {
	var allErrs field.ErrorList
	quotaPath := field.NewPath("spec").Child("components")
	
	// List all ResourceQuotas in the namespace
	quotaList := &corev1.ResourceQuotaList{}
	if err := v.Client.List(ctx, quotaList, client.InNamespace(namespace)); err != nil {
		log.Error(err, "Failed to list resource quotas", "namespace", namespace)
		// Don't fail validation if we can't check quotas
		return allErrs
	}
	
	if len(quotaList.Items) == 0 {
		log.V(2).Info("No resource quotas found in namespace", "namespace", namespace)
		return allErrs
	}
	
	// Get current usage in the namespace
	currentUsage, err := v.getCurrentNamespaceUsage(ctx, namespace)
	if err != nil {
		log.Error(err, "Failed to get current namespace usage", "namespace", namespace)
		// Don't fail validation if we can't check current usage
		return allErrs
	}
	
	// Check each quota
	for _, quota := range quotaList.Items {
		log.V(2).Info("Checking resource quota", "quota", quota.Name, "namespace", namespace)
		
		// Check CPU
		if hardCPU, ok := quota.Status.Hard[corev1.ResourceRequestsCPU]; ok {
			totalCPU := currentUsage.UsedCPU.DeepCopy()
			totalCPU.Add(required.CPU)
			
			if totalCPU.Cmp(hardCPU) > 0 {
				available := hardCPU.DeepCopy()
				available.Sub(currentUsage.UsedCPU)
				
				allErrs = append(allErrs, field.Invalid(quotaPath, required.CPU.String(),
					fmt.Sprintf("exceeds available CPU quota in namespace %s. Required: %s, Available: %s, Quota: %s",
						namespace, required.CPU.String(), available.String(), hardCPU.String())))
			}
		}
		
		// Check Memory
		if hardMemory, ok := quota.Status.Hard[corev1.ResourceRequestsMemory]; ok {
			totalMemory := currentUsage.UsedMemory.DeepCopy()
			totalMemory.Add(required.Memory)
			
			if totalMemory.Cmp(hardMemory) > 0 {
				available := hardMemory.DeepCopy()
				available.Sub(currentUsage.UsedMemory)
				
				allErrs = append(allErrs, field.Invalid(quotaPath, required.Memory.String(),
					fmt.Sprintf("exceeds available memory quota in namespace %s. Required: %s, Available: %s, Quota: %s",
						namespace, required.Memory.String(), available.String(), hardMemory.String())))
			}
		}
		
		// Check Storage
		if hardStorage, ok := quota.Status.Hard[corev1.ResourceRequestsStorage]; ok {
			totalStorage := currentUsage.UsedStorage.DeepCopy()
			totalStorage.Add(required.Storage)
			
			if totalStorage.Cmp(hardStorage) > 0 {
				available := hardStorage.DeepCopy()
				available.Sub(currentUsage.UsedStorage)
				
				allErrs = append(allErrs, field.Invalid(quotaPath, required.Storage.String(),
					fmt.Sprintf("exceeds available storage quota in namespace %s. Required: %s, Available: %s, Quota: %s",
						namespace, required.Storage.String(), available.String(), hardStorage.String())))
			}
		}
		
		// Check PVC count
		if hardPVCs, ok := quota.Status.Hard[corev1.ResourcePersistentVolumeClaims]; ok {
			totalPVCs := int64(currentUsage.UsedPVCs + required.PVCs)
			
			if totalPVCs > hardPVCs.Value() {
				available := hardPVCs.Value() - int64(currentUsage.UsedPVCs)
				
				allErrs = append(allErrs, field.Invalid(quotaPath, required.PVCs,
					fmt.Sprintf("exceeds available PVC quota in namespace %s. Required: %d, Available: %d, Quota: %d",
						namespace, required.PVCs, available, hardPVCs.Value())))
			}
		}
		
		// Check Pod count
		if hardPods, ok := quota.Status.Hard[corev1.ResourcePods]; ok {
			totalPods := int64(currentUsage.UsedPods + required.Pods)
			
			if totalPods > hardPods.Value() {
				available := hardPods.Value() - int64(currentUsage.UsedPods)
				
				allErrs = append(allErrs, field.Invalid(quotaPath, required.Pods,
					fmt.Sprintf("exceeds available pod quota in namespace %s. Required: %d, Available: %d, Quota: %d",
						namespace, required.Pods, available, hardPods.Value())))
			}
		}
	}
	
	return allErrs
}

// getCurrentNamespaceUsage calculates current resource usage in a namespace
func (v *ResourceQuotaValidator) getCurrentNamespaceUsage(ctx context.Context, namespace string) (*NamespaceResourceUsage, error) {
	usage := &NamespaceResourceUsage{
		UsedCPU:     resource.Quantity{},
		UsedMemory:  resource.Quantity{},
		UsedStorage: resource.Quantity{},
		UsedPods:    0,
		UsedPVCs:    0,
	}
	
	// Get pod usage
	podList := &corev1.PodList{}
	if err := v.Client.List(ctx, podList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}
	
	for _, pod := range podList.Items {
		// Skip terminated pods
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		
		usage.UsedPods++
		
		// Add up resource requests
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests.Cpu(); cpu != nil {
					usage.UsedCPU.Add(*cpu)
				}
				if memory := container.Resources.Requests.Memory(); memory != nil {
					usage.UsedMemory.Add(*memory)
				}
			}
		}
	}
	
	// Get PVC usage
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := v.Client.List(ctx, pvcList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing PVCs: %w", err)
	}
	
	for _, pvc := range pvcList.Items {
		usage.UsedPVCs++
		
		// Add storage requests
		if pvc.Spec.Resources.Requests != nil {
			if storage := pvc.Spec.Resources.Requests.Storage(); storage != nil {
				usage.UsedStorage.Add(*storage)
			}
		}
	}
	
	return usage, nil
}

// validateClusterLimits checks against cluster-wide resource limits
func (v *ResourceQuotaValidator) validateClusterLimits(ctx context.Context, namespace string, required *ComponentResources) field.ErrorList {
	var allErrs field.ErrorList
	quotaPath := field.NewPath("spec").Child("components")
	
	// Get current namespace usage including the new platform
	currentUsage, err := v.getCurrentNamespaceUsage(ctx, namespace)
	if err != nil {
		log.Error(err, "Failed to get current namespace usage for cluster limits", "namespace", namespace)
		return allErrs
	}
	
	// Check CPU limit
	if !v.ClusterResourceLimits.MaxCPUPerNamespace.IsZero() {
		totalCPU := currentUsage.UsedCPU.DeepCopy()
		totalCPU.Add(required.CPU)
		
		if totalCPU.Cmp(v.ClusterResourceLimits.MaxCPUPerNamespace) > 0 {
			available := v.ClusterResourceLimits.MaxCPUPerNamespace.DeepCopy()
			available.Sub(currentUsage.UsedCPU)
			
			allErrs = append(allErrs, field.Invalid(quotaPath, required.CPU.String(),
				fmt.Sprintf("exceeds cluster CPU limit for namespace %s. Required: %s, Available: %s, Limit: %s",
					namespace, required.CPU.String(), available.String(), v.ClusterResourceLimits.MaxCPUPerNamespace.String())))
		}
	}
	
	// Check Memory limit
	if !v.ClusterResourceLimits.MaxMemoryPerNamespace.IsZero() {
		totalMemory := currentUsage.UsedMemory.DeepCopy()
		totalMemory.Add(required.Memory)
		
		if totalMemory.Cmp(v.ClusterResourceLimits.MaxMemoryPerNamespace) > 0 {
			available := v.ClusterResourceLimits.MaxMemoryPerNamespace.DeepCopy()
			available.Sub(currentUsage.UsedMemory)
			
			allErrs = append(allErrs, field.Invalid(quotaPath, required.Memory.String(),
				fmt.Sprintf("exceeds cluster memory limit for namespace %s. Required: %s, Available: %s, Limit: %s",
					namespace, required.Memory.String(), available.String(), v.ClusterResourceLimits.MaxMemoryPerNamespace.String())))
		}
	}
	
	// Check per-PVC storage limit
	if !v.ClusterResourceLimits.MaxStoragePerPVC.IsZero() {
		// Check each component's storage requirement
		platform := &observabilityv1beta1.ObservabilityPlatform{} // This is a simplified check
		
		// Check Prometheus storage
		if required.Storage.Cmp(v.ClusterResourceLimits.MaxStoragePerPVC) > 0 {
			allErrs = append(allErrs, field.Invalid(quotaPath.Child("storage"), required.Storage.String(),
				fmt.Sprintf("exceeds maximum storage per PVC limit. Requested: %s, Limit: %s",
					required.Storage.String(), v.ClusterResourceLimits.MaxStoragePerPVC.String())))
		}
	}
	
	// Check PVC count limit
	if v.ClusterResourceLimits.MaxPVCPerNamespace > 0 {
		totalPVCs := currentUsage.UsedPVCs + required.PVCs
		
		if totalPVCs > v.ClusterResourceLimits.MaxPVCPerNamespace {
			available := v.ClusterResourceLimits.MaxPVCPerNamespace - currentUsage.UsedPVCs
			
			allErrs = append(allErrs, field.Invalid(quotaPath, required.PVCs,
				fmt.Sprintf("exceeds cluster PVC limit for namespace %s. Required: %d, Available: %d, Limit: %d",
					namespace, required.PVCs, available, v.ClusterResourceLimits.MaxPVCPerNamespace)))
		}
	}
	
	// Check Pod count limit
	if v.ClusterResourceLimits.MaxPodsPerNamespace > 0 {
		totalPods := currentUsage.UsedPods + required.Pods
		
		if totalPods > v.ClusterResourceLimits.MaxPodsPerNamespace {
			available := v.ClusterResourceLimits.MaxPodsPerNamespace - currentUsage.UsedPods
			
			allErrs = append(allErrs, field.Invalid(quotaPath, required.Pods,
				fmt.Sprintf("exceeds cluster pod limit for namespace %s. Required: %d, Available: %d, Limit: %d",
					namespace, required.Pods, available, v.ClusterResourceLimits.MaxPodsPerNamespace)))
		}
	}
	
	return allErrs
}

// validateMultiTenancy checks multi-tenancy constraints
func (v *ResourceQuotaValidator) validateMultiTenancy(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	
	// Check if namespace has tenant label
	ns := &corev1.Namespace{}
	if err := v.Client.Get(ctx, client.ObjectKey{Name: platform.Namespace}, ns); err != nil {
		log.Error(err, "Failed to get namespace", "namespace", platform.Namespace)
		return allErrs
	}
	
	// Check for tenant isolation labels
	tenantLabel, hasTenant := ns.Labels["tenant"]
	if hasTenant {
		log.V(2).Info("Validating multi-tenant constraints", "tenant", tenantLabel, "namespace", platform.Namespace)
		
		// List all platforms in tenant's namespaces
		platformList := &observabilityv1beta1.ObservabilityPlatformList{}
		labelSelector := labels.SelectorFromSet(map[string]string{"tenant": tenantLabel})
		
		// Get all namespaces for this tenant
		nsList := &corev1.NamespaceList{}
		if err := v.Client.List(ctx, nsList, client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
			log.Error(err, "Failed to list tenant namespaces", "tenant", tenantLabel)
			return allErrs
		}
		
		// Count existing platforms for this tenant
		platformCount := 0
		for _, ns := range nsList.Items {
			platforms := &observabilityv1beta1.ObservabilityPlatformList{}
			if err := v.Client.List(ctx, platforms, client.InNamespace(ns.Name)); err != nil {
				continue
			}
			platformCount += len(platforms.Items)
		}
		
		// Check tenant platform limit (configurable, defaulting to 5)
		maxPlatformsPerTenant := 5 // This could be configurable
		if platformCount >= maxPlatformsPerTenant {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("metadata").Child("namespace"),
				fmt.Sprintf("tenant %s has reached the maximum number of platforms (%d). Please contact your administrator to increase the limit.",
					tenantLabel, maxPlatformsPerTenant)))
		}
		
		// Check for isolation requirements
		if isolationRequired, ok := ns.Labels["require-isolation"]; ok && isolationRequired == "true" {
			// Ensure platform has appropriate node selectors or taints
			if platform.Spec.NodeSelector == nil || platform.Spec.NodeSelector["tenant"] != tenantLabel {
				allErrs = append(allErrs, field.Required(field.NewPath("spec").Child("nodeSelector"),
					fmt.Sprintf("platforms in isolated tenant namespace must have nodeSelector with tenant=%s", tenantLabel)))
			}
		}
	}
	
	// Check namespace resource allocation fairness
	if err := v.checkResourceFairness(ctx, platform); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec"), "resources",
			fmt.Sprintf("resource allocation violates fairness policy: %v", err)))
	}
	
	return allErrs
}

// checkResourceFairness ensures fair resource distribution across namespaces
func (v *ResourceQuotaValidator) checkResourceFairness(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	// Get all namespaces
	nsList := &corev1.NamespaceList{}
	if err := v.Client.List(ctx, nsList); err != nil {
		return fmt.Errorf("listing namespaces: %w", err)
	}
	
	// Calculate average resource usage per namespace
	totalNamespaces := 0
	totalCPU := resource.Quantity{}
	totalMemory := resource.Quantity{}
	
	for _, ns := range nsList.Items {
		// Skip system namespaces
		if strings.HasPrefix(ns.Name, "kube-") || ns.Name == "default" {
			continue
		}
		
		usage, err := v.getCurrentNamespaceUsage(ctx, ns.Name)
		if err != nil {
			continue
		}
		
		totalNamespaces++
		totalCPU.Add(usage.UsedCPU)
		totalMemory.Add(usage.UsedMemory)
	}
	
	if totalNamespaces == 0 {
		return nil
	}
	
	// Calculate average
	avgCPU := totalCPU.DeepCopy()
	avgCPU.Set(avgCPU.Value() / int64(totalNamespaces))
	
	avgMemory := totalMemory.DeepCopy()
	avgMemory.Set(avgMemory.Value() / int64(totalNamespaces))
	
	// Check if this platform would cause significant imbalance (e.g., 3x average)
	platformResources, err := v.calculatePlatformResources(platform)
	if err != nil {
		return err
	}
	
	// Calculate new namespace total
	currentUsage, err := v.getCurrentNamespaceUsage(ctx, platform.Namespace)
	if err != nil {
		return err
	}
	
	newTotalCPU := currentUsage.UsedCPU.DeepCopy()
	newTotalCPU.Add(platformResources.CPU)
	
	newTotalMemory := currentUsage.UsedMemory.DeepCopy()
	newTotalMemory.Add(platformResources.Memory)
	
	// Check if new total exceeds 3x average
	maxAllowedCPU := avgCPU.DeepCopy()
	maxAllowedCPU.Set(maxAllowedCPU.Value() * 3)
	
	maxAllowedMemory := avgMemory.DeepCopy()
	maxAllowedMemory.Set(maxAllowedMemory.Value() * 3)
	
	if newTotalCPU.Cmp(maxAllowedCPU) > 0 {
		return fmt.Errorf("would exceed fair CPU allocation (3x average). Current average: %s, Would use: %s",
			avgCPU.String(), newTotalCPU.String())
	}
	
	if newTotalMemory.Cmp(maxAllowedMemory) > 0 {
		return fmt.Errorf("would exceed fair memory allocation (3x average). Current average: %s, Would use: %s",
			avgMemory.String(), newTotalMemory.String())
	}
	
	return nil
}

// GetQuotaSummary returns a human-readable summary of quota status
func (v *ResourceQuotaValidator) GetQuotaSummary(ctx context.Context, namespace string) (string, error) {
	var summary strings.Builder
	
	// Get namespace quotas
	quotaList := &corev1.ResourceQuotaList{}
	if err := v.Client.List(ctx, quotaList, client.InNamespace(namespace)); err != nil {
		return "", fmt.Errorf("listing quotas: %w", err)
	}
	
	if len(quotaList.Items) == 0 {
		return "No resource quotas defined in namespace", nil
	}
	
	// Get current usage
	usage, err := v.getCurrentNamespaceUsage(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("getting usage: %w", err)
	}
	
	summary.WriteString(fmt.Sprintf("Resource Quota Summary for namespace '%s':\n", namespace))
	summary.WriteString("----------------------------------------\n")
	
	for _, quota := range quotaList.Items {
		summary.WriteString(fmt.Sprintf("\nQuota: %s\n", quota.Name))
		
		// CPU
		if hardCPU, ok := quota.Status.Hard[corev1.ResourceRequestsCPU]; ok {
			summary.WriteString(fmt.Sprintf("  CPU: %s / %s (%.1f%% used)\n",
				usage.UsedCPU.String(), hardCPU.String(),
				float64(usage.UsedCPU.MilliValue())/float64(hardCPU.MilliValue())*100))
		}
		
		// Memory
		if hardMemory, ok := quota.Status.Hard[corev1.ResourceRequestsMemory]; ok {
			summary.WriteString(fmt.Sprintf("  Memory: %s / %s (%.1f%% used)\n",
				usage.UsedMemory.String(), hardMemory.String(),
				float64(usage.UsedMemory.Value())/float64(hardMemory.Value())*100))
		}
		
		// Storage
		if hardStorage, ok := quota.Status.Hard[corev1.ResourceRequestsStorage]; ok {
			summary.WriteString(fmt.Sprintf("  Storage: %s / %s (%.1f%% used)\n",
				usage.UsedStorage.String(), hardStorage.String(),
				float64(usage.UsedStorage.Value())/float64(hardStorage.Value())*100))
		}
		
		// Pods
		if hardPods, ok := quota.Status.Hard[corev1.ResourcePods]; ok {
			summary.WriteString(fmt.Sprintf("  Pods: %d / %d (%.1f%% used)\n",
				usage.UsedPods, hardPods.Value(),
				float64(usage.UsedPods)/float64(hardPods.Value())*100))
		}
		
		// PVCs
		if hardPVCs, ok := quota.Status.Hard[corev1.ResourcePersistentVolumeClaims]; ok {
			summary.WriteString(fmt.Sprintf("  PVCs: %d / %d (%.1f%% used)\n",
				usage.UsedPVCs, hardPVCs.Value(),
				float64(usage.UsedPVCs)/float64(hardPVCs.Value())*100))
		}
	}
	
	return summary.String(), nil
}
