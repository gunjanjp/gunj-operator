package autoscaling

import (
	"context"
	"fmt"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkloadDrainer handles graceful workload draining during scale-down
type WorkloadDrainer struct {
	client    client.Client
	clientset kubernetes.Interface
	config    WorkloadDrainConfig
}

// NewWorkloadDrainer creates a new workload drainer
func NewWorkloadDrainer(client client.Client, clientset kubernetes.Interface, config WorkloadDrainConfig) *WorkloadDrainer {
	// Set defaults if not specified
	if config.DrainTimeout == 0 {
		config.DrainTimeout = 5 * time.Minute
	}
	if config.GracePeriod == 0 {
		config.GracePeriod = 30 * time.Second
	}
	if config.MaxUnavailable == 0 {
		config.MaxUnavailable = 1
	}
	
	return &WorkloadDrainer{
		client:    client,
		clientset: clientset,
		config:    config,
	}
}

// DrainPod gracefully drains a pod before termination
func (d *WorkloadDrainer) DrainPod(ctx context.Context, pod *corev1.Pod) error {
	log := log.FromContext(ctx)
	log.Info("Starting pod drain", "pod", pod.Name, "namespace", pod.Namespace)
	
	// Check if pod is already terminating
	if pod.DeletionTimestamp != nil {
		log.Info("Pod is already terminating", "pod", pod.Name)
		return nil
	}
	
	// Create eviction
	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: &d.config.GracePeriod,
		},
	}
	
	// Cordon the node to prevent new pods
	if err := d.cordonNode(ctx, pod.Spec.NodeName); err != nil {
		log.Error(err, "Failed to cordon node", "node", pod.Spec.NodeName)
		// Continue with eviction anyway
	}
	
	// Wait for connections to drain
	if err := d.waitForConnectionDrain(ctx, pod); err != nil {
		log.Error(err, "Error waiting for connection drain")
		// Continue with eviction
	}
	
	// Perform eviction
	err := d.clientset.CoreV1().Pods(pod.Namespace).EvictV1(ctx, eviction)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Pod already gone
		}
		return fmt.Errorf("failed to evict pod: %w", err)
	}
	
	// Wait for pod deletion
	if err := d.waitForPodDeletion(ctx, pod); err != nil {
		return fmt.Errorf("pod deletion timeout: %w", err)
	}
	
	// Uncordon the node
	if err := d.uncordonNode(ctx, pod.Spec.NodeName); err != nil {
		log.Error(err, "Failed to uncordon node", "node", pod.Spec.NodeName)
	}
	
	log.Info("Pod drained successfully", "pod", pod.Name)
	return nil
}

// DrainComponent drains all pods for a component
func (d *WorkloadDrainer) DrainComponent(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, targetReplicas int32) error {
	log := log.FromContext(ctx)
	log.Info("Draining component", "component", component, "targetReplicas", targetReplicas)
	
	// List pods for the component
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/name":     string(component),
		"app.kubernetes.io/instance": platform.Name,
	})
	
	if err := d.client.List(ctx, podList, &client.ListOptions{
		Namespace:     platform.Namespace,
		LabelSelector: labelSelector,
	}); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}
	
	currentReplicas := int32(len(podList.Items))
	if currentReplicas <= targetReplicas {
		log.Info("No pods to drain", "currentReplicas", currentReplicas, "targetReplicas", targetReplicas)
		return nil
	}
	
	// Calculate how many pods to drain
	podsToDrain := currentReplicas - targetReplicas
	
	// Sort pods by creation time (drain newest first)
	pods := d.sortPodsByAge(podList.Items)
	
	// Drain pods respecting maxUnavailable
	drainedCount := int32(0)
	for i := 0; i < len(pods) && drainedCount < podsToDrain; i++ {
		pod := &pods[i]
		
		// Check if we can drain more pods
		if !d.canDrainPod(ctx, platform, component, currentReplicas-drainedCount) {
			log.Info("Cannot drain more pods due to availability constraints")
			break
		}
		
		// Drain the pod
		if err := d.DrainPod(ctx, pod); err != nil {
			log.Error(err, "Failed to drain pod", "pod", pod.Name)
			continue
		}
		
		drainedCount++
		
		// Wait between drains to maintain availability
		if drainedCount < podsToDrain {
			time.Sleep(10 * time.Second)
		}
	}
	
	log.Info("Component drain completed", "component", component, "drainedPods", drainedCount)
	return nil
}

// canDrainPod checks if we can safely drain another pod
func (d *WorkloadDrainer) canDrainPod(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, remainingReplicas int32) bool {
	// Check PodDisruptionBudget
	pdb := &policyv1.PodDisruptionBudget{}
	pdbName := fmt.Sprintf("%s-%s-pdb", platform.Name, component)
	
	err := d.client.Get(ctx, types.NamespacedName{
		Name:      pdbName,
		Namespace: platform.Namespace,
	}, pdb)
	
	if err == nil {
		// PDB exists, check if we can disrupt
		if pdb.Status.CurrentHealthy <= pdb.Status.DesiredHealthy {
			return false
		}
	}
	
	// Check maxUnavailable constraint
	minAvailable := remainingReplicas - d.config.MaxUnavailable
	if minAvailable < 1 {
		minAvailable = 1 // Always keep at least one pod
	}
	
	return remainingReplicas > minAvailable
}

// waitForConnectionDrain waits for active connections to drain
func (d *WorkloadDrainer) waitForConnectionDrain(ctx context.Context, pod *corev1.Pod) error {
	log := log.FromContext(ctx)
	
	// Check if pod has readiness gate for connection draining
	for _, condition := range pod.Status.Conditions {
		if condition.Type == "ConnectionsDrained" {
			if condition.Status == corev1.ConditionTrue {
				return nil
			}
			break
		}
	}
	
	// For components that expose metrics, check active connections
	switch pod.Labels["app.kubernetes.io/name"] {
	case string(v1beta1.ComponentPrometheus):
		return d.waitForPrometheusConnectionDrain(ctx, pod)
	case string(v1beta1.ComponentGrafana):
		return d.waitForGrafanaConnectionDrain(ctx, pod)
	case string(v1beta1.ComponentLoki):
		return d.waitForLokiConnectionDrain(ctx, pod)
	}
	
	// Default wait period
	log.Info("Waiting for connections to drain", "duration", "30s")
	time.Sleep(30 * time.Second)
	
	return nil
}

// waitForPrometheusConnectionDrain waits for Prometheus to finish processing queries
func (d *WorkloadDrainer) waitForPrometheusConnectionDrain(ctx context.Context, pod *corev1.Pod) error {
	// In production, this would query Prometheus metrics to check active queries
	// For now, use a fixed wait time
	time.Sleep(20 * time.Second)
	return nil
}

// waitForGrafanaConnectionDrain waits for Grafana active sessions to complete
func (d *WorkloadDrainer) waitForGrafanaConnectionDrain(ctx context.Context, pod *corev1.Pod) error {
	// In production, this would check Grafana's active sessions
	time.Sleep(15 * time.Second)
	return nil
}

// waitForLokiConnectionDrain waits for Loki to flush logs
func (d *WorkloadDrainer) waitForLokiConnectionDrain(ctx context.Context, pod *corev1.Pod) error {
	// In production, this would check Loki's ingestion queue
	time.Sleep(30 * time.Second)
	return nil
}

// waitForPodDeletion waits for a pod to be deleted
func (d *WorkloadDrainer) waitForPodDeletion(ctx context.Context, pod *corev1.Pod) error {
	deadline := time.Now().Add(d.config.DrainTimeout)
	
	for time.Now().Before(deadline) {
		current := &corev1.Pod{}
		err := d.client.Get(ctx, types.NamespacedName{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}, current)
		
		if errors.IsNotFound(err) {
			return nil // Pod is gone
		}
		
		if err != nil {
			return fmt.Errorf("error checking pod status: %w", err)
		}
		
		time.Sleep(5 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for pod deletion")
}

// cordonNode marks a node as unschedulable
func (d *WorkloadDrainer) cordonNode(ctx context.Context, nodeName string) error {
	if nodeName == "" {
		return nil
	}
	
	node := &corev1.Node{}
	if err := d.client.Get(ctx, types.NamespacedName{Name: nodeName}, node); err != nil {
		return err
	}
	
	if node.Spec.Unschedulable {
		return nil // Already cordoned
	}
	
	node.Spec.Unschedulable = true
	return d.client.Update(ctx, node)
}

// uncordonNode marks a node as schedulable
func (d *WorkloadDrainer) uncordonNode(ctx context.Context, nodeName string) error {
	if nodeName == "" {
		return nil
	}
	
	node := &corev1.Node{}
	if err := d.client.Get(ctx, types.NamespacedName{Name: nodeName}, node); err != nil {
		return err
	}
	
	if !node.Spec.Unschedulable {
		return nil // Already uncordoned
	}
	
	node.Spec.Unschedulable = false
	return d.client.Update(ctx, node)
}

// sortPodsByAge sorts pods by creation time (newest first)
func (d *WorkloadDrainer) sortPodsByAge(pods []corev1.Pod) []corev1.Pod {
	// Simple bubble sort for small lists
	for i := 0; i < len(pods)-1; i++ {
		for j := 0; j < len(pods)-i-1; j++ {
			if pods[j].CreationTimestamp.Before(&pods[j+1].CreationTimestamp) {
				pods[j], pods[j+1] = pods[j+1], pods[j]
			}
		}
	}
	return pods
}

// GetDrainStatus gets the drain status for a component
func (d *WorkloadDrainer) GetDrainStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) (*DrainStatus, error) {
	status := &DrainStatus{
		Component:     component,
		DrainingPods:  []string{},
		CompletedPods: []string{},
		PendingPods:   []string{},
	}
	
	// List pods
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/name":     string(component),
		"app.kubernetes.io/instance": platform.Name,
	})
	
	if err := d.client.List(ctx, podList, &client.ListOptions{
		Namespace:     platform.Namespace,
		LabelSelector: labelSelector,
	}); err != nil {
		return nil, err
	}
	
	// Categorize pods
	for _, pod := range podList.Items {
		if pod.DeletionTimestamp != nil {
			status.DrainingPods = append(status.DrainingPods, pod.Name)
		} else if pod.Status.Phase == corev1.PodRunning {
			status.PendingPods = append(status.PendingPods, pod.Name)
		}
	}
	
	// Get recently terminated pods from events
	events := &corev1.EventList{}
	fieldSelector := fields.OneTermEqualSelector("involvedObject.kind", "Pod")
	
	if err := d.client.List(ctx, events, &client.ListOptions{
		Namespace:     platform.Namespace,
		FieldSelector: fieldSelector,
	}); err == nil {
		cutoff := time.Now().Add(-5 * time.Minute)
		for _, event := range events.Items {
			if event.Reason == "Killing" && event.CreationTimestamp.After(cutoff) {
				status.CompletedPods = append(status.CompletedPods, event.InvolvedObject.Name)
			}
		}
	}
	
	status.TotalPods = len(status.DrainingPods) + len(status.PendingPods)
	
	return status, nil
}

// DrainStatus represents the drain status for a component
type DrainStatus struct {
	Component     v1beta1.ComponentType
	TotalPods     int
	DrainingPods  []string
	CompletedPods []string
	PendingPods   []string
}
