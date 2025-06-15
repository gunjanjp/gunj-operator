/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// ReconciliationState tracks the state of reconciliation
type ReconciliationState struct {
	Platform           *observabilityv1beta1.ObservabilityPlatform
	ComponentStates    map[string]ComponentState
	DependencyResolver *DependencyResolver
	ConfigManager      *ConfigurationManager
	StartTime          time.Time
}

// ComponentState tracks the state of a component
type ComponentState struct {
	Name         string
	Status       observabilityv1beta1.ComponentStatus
	Ready        bool
	LastError    error
	Dependencies []string
	Resources    []client.Object
}

// DependencyResolver manages component dependencies
type DependencyResolver struct {
	dependencies map[string][]string
	order        []string
}

// ConfigurationManager manages component configurations
type ConfigurationManager struct {
	platform     *observabilityv1beta1.ObservabilityPlatform
	globalConfig map[string]string
	secrets      map[string]*corev1.Secret
	configMaps   map[string]*corev1.ConfigMap
}

// NewReconciliationState creates a new reconciliation state
func NewReconciliationState(platform *observabilityv1beta1.ObservabilityPlatform) *ReconciliationState {
	return &ReconciliationState{
		Platform:           platform,
		ComponentStates:    make(map[string]ComponentState),
		DependencyResolver: NewDependencyResolver(),
		ConfigManager:      NewConfigurationManager(platform),
		StartTime:          time.Now(),
	}
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver() *DependencyResolver {
	return &DependencyResolver{
		dependencies: map[string][]string{
			"prometheus": {},                          // No dependencies
			"loki":       {},                          // No dependencies
			"tempo":      {},                          // No dependencies
			"grafana":    {"prometheus", "loki", "tempo"}, // Depends on data sources
		},
		order: []string{},
	}
}

// NewConfigurationManager creates a new configuration manager
func NewConfigurationManager(platform *observabilityv1beta1.ObservabilityPlatform) *ConfigurationManager {
	return &ConfigurationManager{
		platform:     platform,
		globalConfig: make(map[string]string),
		secrets:      make(map[string]*corev1.Secret),
		configMaps:   make(map[string]*corev1.ConfigMap),
	}
}

// reconcileCommonResources creates common resources like RBAC, NetworkPolicies, etc.
func (r *ObservabilityPlatformReconciler) reconcileCommonResources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Reconciling common resources")

	// Create state for tracking
	state := NewReconciliationState(platform)

	// 1. Reconcile RBAC resources
	if err := r.reconcileRBAC(ctx, state); err != nil {
		return fmt.Errorf("failed to reconcile RBAC: %w", err)
	}

	// 2. Reconcile NetworkPolicies
	if platform.Spec.Security != nil && platform.Spec.Security.NetworkPolicy {
		if err := r.reconcileNetworkPolicies(ctx, state); err != nil {
			return fmt.Errorf("failed to reconcile NetworkPolicies: %w", err)
		}
	}

	// 3. Reconcile shared secrets
	if err := r.reconcileSharedSecrets(ctx, state); err != nil {
		return fmt.Errorf("failed to reconcile shared secrets: %w", err)
	}

	// 4. Reconcile shared ConfigMaps
	if err := r.reconcileSharedConfigMaps(ctx, state); err != nil {
		return fmt.Errorf("failed to reconcile shared ConfigMaps: %w", err)
	}

	log.Info("Common resources reconciled successfully")
	return nil
}

// reconcileRBAC creates RBAC resources
func (r *ObservabilityPlatformReconciler) reconcileRBAC(ctx context.Context, state *ReconciliationState) error {
	log := log.FromContext(ctx)
	platform := state.Platform

	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-observability", platform.Name),
			Namespace: platform.Namespace,
			Labels:    r.commonLabels(platform),
		},
	}

	if err := r.createOrUpdate(ctx, sa, platform, func() error {
		// ServiceAccount is immutable after creation
		return nil
	}); err != nil {
		return err
	}

	// Create ClusterRole
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s-observability", platform.Namespace, platform.Name),
			Labels: r.commonLabels(platform),
		},
	}

	if err := r.createOrUpdate(ctx, clusterRole, platform, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "nodes", "nodes/proxy", "services", "endpoints", "pods", "pods/log"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "daemonsets", "replicasets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}
		return nil
	}); err != nil {
		return err
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s-observability", platform.Namespace, platform.Name),
			Labels: r.commonLabels(platform),
		},
	}

	if err := r.createOrUpdate(ctx, clusterRoleBinding, platform, func() error {
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		}
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		}
		return nil
	}); err != nil {
		return err
	}

	log.V(1).Info("RBAC resources reconciled", "serviceAccount", sa.Name)
	return nil
}

// reconcileNetworkPolicies creates network policies
func (r *ObservabilityPlatformReconciler) reconcileNetworkPolicies(ctx context.Context, state *ReconciliationState) error {
	log := log.FromContext(ctx)
	platform := state.Platform

	// Create default network policy
	netpol := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-observability", platform.Name),
			Namespace: platform.Namespace,
			Labels:    r.commonLabels(platform),
		},
	}

	if err := r.createOrUpdate(ctx, netpol, platform, func() error {
		netpol.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"app.kubernetes.io/instance":   platform.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					// Allow traffic from same namespace
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
					},
				},
				{
					// Allow traffic from ingress controller
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"name": "ingress-nginx",
								},
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Allow all egress traffic
					To: []networkingv1.NetworkPolicyPeer{},
				},
			},
		}
		return nil
	}); err != nil {
		return err
	}

	log.V(1).Info("NetworkPolicy created", "name", netpol.Name)
	return nil
}

// reconcileSharedSecrets creates shared secrets
func (r *ObservabilityPlatformReconciler) reconcileSharedSecrets(ctx context.Context, state *ReconciliationState) error {
	log := log.FromContext(ctx)
	platform := state.Platform

	// Create admin credentials secret if not exists
	adminSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-admin-credentials", platform.Name),
			Namespace: platform.Namespace,
			Labels:    r.commonLabels(platform),
		},
	}

	if err := r.createOrUpdate(ctx, adminSecret, platform, func() error {
		if adminSecret.Data == nil {
			adminSecret.Type = corev1.SecretTypeOpaque
			adminSecret.Data = map[string][]byte{
				"admin-user":     []byte("admin"),
				"admin-password": []byte(generatePassword()),
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Store in config manager
	state.ConfigManager.secrets["admin-credentials"] = adminSecret

	log.V(1).Info("Shared secrets reconciled")
	return nil
}

// reconcileSharedConfigMaps creates shared config maps
func (r *ObservabilityPlatformReconciler) reconcileSharedConfigMaps(ctx context.Context, state *ReconciliationState) error {
	log := log.FromContext(ctx)
	platform := state.Platform

	// Create global configuration ConfigMap
	globalConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-global-config", platform.Name),
			Namespace: platform.Namespace,
			Labels:    r.commonLabels(platform),
		},
	}

	if err := r.createOrUpdate(ctx, globalConfig, platform, func() error {
		if globalConfig.Data == nil {
			globalConfig.Data = make(map[string]string)
		}

		// Set global configuration
		globalConfig.Data["cluster.name"] = platform.Name
		// Set default retention if not specified
		globalConfig.Data["retention.days"] = "30" // Default 30 days
		globalConfig.Data["log.level"] = platform.Spec.Global.LogLevel

		// Add external labels
		for k, v := range platform.Spec.Global.ExternalLabels {
			globalConfig.Data[fmt.Sprintf("external.label.%s", k)] = v
		}

		return nil
	}); err != nil {
		return err
	}

	// Store in config manager
	state.ConfigManager.configMaps["global-config"] = globalConfig

	log.V(1).Info("Shared ConfigMaps reconciled")
	return nil
}

// GetReconciliationOrder returns the order in which components should be reconciled
func (d *DependencyResolver) GetReconciliationOrder(enabledComponents map[string]bool) []string {
	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize graph with enabled components
	for component, enabled := range enabledComponents {
		if enabled {
			graph[component] = []string{}
			inDegree[component] = 0
		}
	}

	// Add dependencies
	for component, deps := range d.dependencies {
		if !enabledComponents[component] {
			continue
		}
		for _, dep := range deps {
			if enabledComponents[dep] {
				graph[dep] = append(graph[dep], component)
				inDegree[component]++
			}
		}
	}

	// Topological sort using Kahn's algorithm
	queue := []string{}
	for component, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, component)
		}
	}

	order := []string{}
	for len(queue) > 0 {
		// Sort queue for deterministic order
		sort.Strings(queue)
		
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return order
}

// BuildConfiguration builds the configuration for a component
func (c *ConfigurationManager) BuildConfiguration(component string) map[string]interface{} {
	config := make(map[string]interface{})

	// Add global configuration
	config["global"] = map[string]interface{}{
		"clusterName":    c.platform.Name,
		"namespace":      c.platform.Namespace,
		"externalLabels": c.platform.Spec.Global.ExternalLabels,
		"logLevel":       c.platform.Spec.Global.LogLevel,
	}

	// Add component-specific configuration
	switch component {
	case "prometheus":
		if c.platform.Spec.Components.Prometheus != nil {
			config["prometheus"] = c.buildPrometheusConfig(c.platform.Spec.Components.Prometheus)
		}
	case "grafana":
		if c.platform.Spec.Components.Grafana != nil {
			config["grafana"] = c.buildGrafanaConfig(c.platform.Spec.Components.Grafana)
		}
	case "loki":
		if c.platform.Spec.Components.Loki != nil {
			config["loki"] = c.buildLokiConfig(c.platform.Spec.Components.Loki)
		}
	case "tempo":
		if c.platform.Spec.Components.Tempo != nil {
			config["tempo"] = c.buildTempoConfig(c.platform.Spec.Components.Tempo)
		}
	}

	// Add inter-component URLs
	config["endpoints"] = c.buildEndpoints()

	return config
}

// buildPrometheusConfig builds Prometheus-specific configuration
func (c *ConfigurationManager) buildPrometheusConfig(spec *observabilityv1beta1.PrometheusSpec) map[string]interface{} {
	config := map[string]interface{}{
		"version":  spec.Version,
		"replicas": spec.Replicas,
		"resources": map[string]interface{}{
			"requests": spec.Resources.Requests,
			"limits":   spec.Resources.Limits,
		},
	}

	config["storage"] = map[string]interface{}{
		"size":             spec.Storage.Size,
		"storageClassName": spec.Storage.StorageClassName,
	}

	if spec.Retention != "" {
		config["retention"] = spec.Retention
	}

	return config
}

// buildGrafanaConfig builds Grafana-specific configuration
func (c *ConfigurationManager) buildGrafanaConfig(spec *observabilityv1beta1.GrafanaSpec) map[string]interface{} {
	config := map[string]interface{}{
		"version":  spec.Version,
		"replicas": spec.Replicas,
		"resources": map[string]interface{}{
			"requests": spec.Resources.Requests,
			"limits":   spec.Resources.Limits,
		},
	}

	// Data sources will be configured based on enabled components
	dataSources := []map[string]interface{}{}
	
	if c.platform.Spec.Components.Prometheus != nil && c.platform.Spec.Components.Prometheus.Enabled {
		dataSources = append(dataSources, map[string]interface{}{
			"name":      "Prometheus",
			"type":      "prometheus",
			"url":       c.getPrometheusURL(),
			"isDefault": true,
		})
	}

	if c.platform.Spec.Components.Loki != nil && c.platform.Spec.Components.Loki.Enabled {
		dataSources = append(dataSources, map[string]interface{}{
			"name": "Loki",
			"type": "loki",
			"url":  c.getLokiURL(),
		})
	}

	if c.platform.Spec.Components.Tempo != nil && c.platform.Spec.Components.Tempo.Enabled {
		dataSources = append(dataSources, map[string]interface{}{
			"name": "Tempo",
			"type": "tempo",
			"url":  c.getTempoURL(),
		})
	}

	config["dataSources"] = dataSources

	return config
}

// buildLokiConfig builds Loki-specific configuration
func (c *ConfigurationManager) buildLokiConfig(spec *observabilityv1beta1.LokiSpec) map[string]interface{} {
	config := map[string]interface{}{
		"version":  spec.Version,
		"replicas": spec.Replicas,
		"resources": map[string]interface{}{
			"requests": spec.Resources.Requests,
			"limits":   spec.Resources.Limits,
		},
	}

	config["storage"] = map[string]interface{}{
		"size":             spec.Storage.Size,
		"storageClassName": spec.Storage.StorageClassName,
	}

	return config
}

// buildTempoConfig builds Tempo-specific configuration
func (c *ConfigurationManager) buildTempoConfig(spec *observabilityv1beta1.TempoSpec) map[string]interface{} {
	config := map[string]interface{}{
		"version":  spec.Version,
		"replicas": spec.Replicas,
		"resources": map[string]interface{}{
			"requests": spec.Resources.Requests,
			"limits":   spec.Resources.Limits,
		},
	}

	config["storage"] = map[string]interface{}{
		"size":             spec.Storage.Size,
		"storageClassName": spec.Storage.StorageClassName,
	}

	return config
}

// buildEndpoints builds inter-component endpoint URLs
func (c *ConfigurationManager) buildEndpoints() map[string]string {
	endpoints := make(map[string]string)
	
	endpoints["prometheus"] = c.getPrometheusURL()
	endpoints["grafana"] = c.getGrafanaURL()
	endpoints["loki"] = c.getLokiURL()
	endpoints["tempo"] = c.getTempoURL()
	
	return endpoints
}

// Helper methods for getting component URLs
func (c *ConfigurationManager) getPrometheusURL() string {
	return fmt.Sprintf("http://prometheus-%s.%s.svc.cluster.local:9090", c.platform.Name, c.platform.Namespace)
}

func (c *ConfigurationManager) getGrafanaURL() string {
	return fmt.Sprintf("http://grafana-%s.%s.svc.cluster.local:3000", c.platform.Name, c.platform.Namespace)
}

func (c *ConfigurationManager) getLokiURL() string {
	return fmt.Sprintf("http://loki-%s.%s.svc.cluster.local:3100", c.platform.Name, c.platform.Namespace)
}

func (c *ConfigurationManager) getTempoURL() string {
	return fmt.Sprintf("http://tempo-%s.%s.svc.cluster.local:3200", c.platform.Name, c.platform.Namespace)
}

// createOrUpdate creates or updates a resource
func (r *ObservabilityPlatformReconciler) createOrUpdate(ctx context.Context, obj client.Object, owner *observabilityv1beta1.ObservabilityPlatform, mutate func() error) error {
	log := log.FromContext(ctx)

	// Set controller reference
	if err := controllerutil.SetControllerReference(owner, obj, r.Scheme); err != nil {
		return err
	}

	// Create or update the object
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, obj, func() error {
		// Set common labels
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for k, v := range r.commonLabels(owner) {
			labels[k] = v
		}
		obj.SetLabels(labels)

		// Call the mutate function
		return mutate()
	})

	if err != nil {
		return err
	}

	log.V(1).Info("Resource reconciled", "operation", op, "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
	return nil
}

// commonLabels returns common labels for all resources
func (r *ObservabilityPlatformReconciler) commonLabels(platform *observabilityv1beta1.ObservabilityPlatform) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "observability-platform",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
		"app.kubernetes.io/part-of":    "observability-platform",
		"app.kubernetes.io/component":  "common",
		"observability.io/platform":    platform.Name,
	}
}

// generatePassword generates a random password
func generatePassword() string {
	// This is a simple implementation. In production, use a proper random generator
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// UpdateComponentStatus is now replaced by StatusManager.SetComponentStatus
// This method is kept for backward compatibility but delegates to StatusManager
func (r *ObservabilityPlatformReconciler) UpdateComponentStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, componentName string, status observabilityv1beta1.ComponentStatus) error {
	// Delegate to StatusManager
	ready := status.Phase == "Ready"
	return r.StatusManager.SetComponentStatus(ctx, platform, componentName, ready, status.Message)
}

// ReconcileWithDependencies reconciles components in dependency order
func (r *ObservabilityPlatformReconciler) ReconcileWithDependencies(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	state := NewReconciliationState(platform)

	// Determine enabled components
	enabledComponents := map[string]bool{
		"prometheus": platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled,
		"grafana":    platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled,
		"loki":       platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled,
		"tempo":      platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled,
	}

	// Get reconciliation order
	order := state.DependencyResolver.GetReconciliationOrder(enabledComponents)
	log.Info("Reconciling components in dependency order", "order", strings.Join(order, " -> "))

	// Update progress for component reconciliation
	r.StatusManager.UpdateProgress(ctx, platform, "component-reconciliation", 0, len(order), "Starting component reconciliation")

	// Reconcile each component in order
	for i, component := range order {
		log.V(1).Info("Reconciling component", "component", component)

		// Update progress
		r.StatusManager.UpdateProgress(ctx, platform, "component-reconciliation", i, len(order), fmt.Sprintf("Reconciling %s", component))

		// Record component deployment start
		r.EventRecorder.RecordComponentEvent(platform, component, EventReasonComponentDeploying, "Starting component deployment")

		// Build configuration for component
		config := state.ConfigManager.BuildConfiguration(component)
		
		// Track deployment time
		startTime := time.Now()
		
		// Reconcile based on component type
		var err error
		
		switch component {
		case "prometheus":
			if r.PrometheusManager != nil {
				err = r.PrometheusManager.ReconcileWithConfig(ctx, platform, config)
			}
		case "grafana":
			if r.GrafanaManager != nil {
				err = r.GrafanaManager.ReconcileWithConfig(ctx, platform, config)
			}
		case "loki":
			if r.LokiManager != nil {
				err = r.LokiManager.ReconcileWithConfig(ctx, platform, config)
			}
		case "tempo":
			if r.TempoManager != nil {
				err = r.TempoManager.ReconcileWithConfig(ctx, platform, config)
			}
		}

		// Record deployment duration
		duration := time.Since(startTime)

		// Update component status based on result
		if err != nil {
			// Component failed
			r.StatusManager.SetComponentStatus(ctx, platform, component, false, fmt.Sprintf("Failed: %v", err))
			r.EventRecorder.RecordOperationEvent(platform, fmt.Sprintf("%s-deployment", component), EventReasonComponentFailed, fmt.Sprintf("Component deployment failed: %v", err), duration)
			log.Error(err, "Failed to reconcile component", "component", component)
			
			// Record component state
			state.ComponentStates[component] = ComponentState{
				Name:      component,
				Status:    observabilityv1beta1.ComponentStatus{
					Phase:   "Failed",
					Message: err.Error(),
				},
				Ready:     false,
				LastError: err,
			}
			
			// Stop on error if component is critical
			if r.isComponentCritical(component) {
				return fmt.Errorf("critical component %s failed: %w", component, err)
			}
		} else {
			// Component succeeded
			r.StatusManager.SetComponentStatus(ctx, platform, component, true, "Component is ready")
			r.EventRecorder.RecordOperationEvent(platform, fmt.Sprintf("%s-deployment", component), EventReasonComponentReady, "Component deployed successfully", duration)
			log.Info("Component reconciled successfully", "component", component)
			
			// Record component state
			state.ComponentStates[component] = ComponentState{
				Name:      component,
				Status:    observabilityv1beta1.ComponentStatus{
					Phase:   "Ready",
					Message: fmt.Sprintf("%s is ready", component),
				},
				Ready:     true,
				LastError: nil,
			}
		}
	}

	// Complete progress
	r.StatusManager.UpdateProgress(ctx, platform, "component-reconciliation", len(order), len(order), "All components reconciled")

	return nil
}

// isComponentCritical determines if a component failure should stop reconciliation
func (r *ObservabilityPlatformReconciler) isComponentCritical(component string) bool {
	// For now, all components are considered critical
	// This can be made configurable in the future
	return true
}
