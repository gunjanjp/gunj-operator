package autoscaling

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/utils"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AutoScalingManager implements the main autoscaling logic
type AutoScalingManager struct {
	client          client.Client
	clientset       kubernetes.Interface
	scheme          *runtime.Scheme
	recorder        record.EventRecorder
	
	// Sub-managers
	hpaManager      *HPAManager
	vpaManager      *VPAManager
	metricsProvider MetricsProvider
	costProvider    CostProvider
	predictiveEngine *PredictiveScalingEngine
	eventTracker    *EventTracker
	workloadDrainer *WorkloadDrainer
	
	// Configuration
	config          AutoScalingConfig
	
	// State
	mu              sync.RWMutex
	activeScaling   map[string]bool // Track active scaling operations
}

// NewAutoScalingManager creates a new autoscaling manager
func NewAutoScalingManager(
	client client.Client,
	clientset kubernetes.Interface,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config AutoScalingConfig,
) (*AutoScalingManager, error) {
	
	// Create sub-managers
	hpaManager := NewHPAManager(client)
	vpaManager := NewVPAManager(client)
	
	// Create metrics provider
	metricsProvider, err := NewPrometheusMetricsProvider(
		client,
		config.CustomMetricsAdapter,
		"", // Will be set per platform
		"", // Will be set per platform
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics provider: %w", err)
	}
	
	// Create cost provider
	costProvider := NewCostProvider(config.CostProvider, "us-east-1") // TODO: Make region configurable
	
	// Create predictive engine
	predictiveEngine := NewPredictiveScalingEngine("linear")
	
	// Create event tracker
	eventStorage := NewInMemoryEventStorage() // TODO: Make storage configurable
	eventTracker := NewEventTracker(client, recorder, eventStorage)
	
	// Create workload drainer
	workloadDrainer := NewWorkloadDrainer(client, clientset, config.DrainConfig)
	
	return &AutoScalingManager{
		client:           client,
		clientset:        clientset,
		scheme:           scheme,
		recorder:         recorder,
		hpaManager:       hpaManager,
		vpaManager:       vpaManager,
		metricsProvider:  metricsProvider,
		costProvider:     costProvider,
		predictiveEngine: predictiveEngine,
		eventTracker:     eventTracker,
		workloadDrainer:  workloadDrainer,
		config:           config,
		activeScaling:    make(map[string]bool),
	}, nil
}

// ReconcileAutoscaling reconciles autoscaling for a platform
func (m *AutoScalingManager) ReconcileAutoscaling(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	
	if !m.config.Enabled {
		log.V(2).Info("Autoscaling is disabled")
		return nil
	}
	
	// Update metrics provider with platform info
	if promProvider, ok := m.metricsProvider.(*PrometheusMetricsProvider); ok {
		promProvider.namespace = platform.Namespace
		promProvider.platformName = platform.Name
	}
	
	// Process each component
	components := m.getEnabledComponents(platform)
	
	for _, component := range components {
		// Check if scaling is already in progress
		if m.isScalingInProgress(platform, component) {
			log.V(1).Info("Scaling already in progress, skipping", "component", component)
			continue
		}
		
		// Get component policy
		policy, exists := m.config.Policies[component]
		if !exists {
			log.V(2).Info("No scaling policy defined for component", "component", component)
			continue
		}
		
		// Reconcile autoscaling for the component
		if err := m.reconcileComponentAutoscaling(ctx, platform, component, policy); err != nil {
			log.Error(err, "Failed to reconcile autoscaling", "component", component)
			m.recorder.Eventf(platform, corev1.EventTypeWarning, "AutoscalingFailed",
				"Failed to reconcile autoscaling for %s: %v", component, err)
		}
	}
	
	// Clean up old events periodically
	if err := m.eventTracker.CleanupOldEvents(ctx, m.config.MetricsRetention); err != nil {
		log.Error(err, "Failed to cleanup old events")
	}
	
	return nil
}

// reconcileComponentAutoscaling reconciles autoscaling for a single component
func (m *AutoScalingManager) reconcileComponentAutoscaling(
	ctx context.Context,
	platform *v1beta1.ObservabilityPlatform,
	component v1beta1.ComponentType,
	policy ScalingPolicy,
) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Reconciling autoscaling", "component", component, "policy", policy.Name)
	
	// Mark scaling as in progress
	m.setScalingInProgress(platform, component, true)
	defer m.setScalingInProgress(platform, component, false)
	
	startTime := time.Now()
	
	// Determine scaling type and execute
	var decision *ScalingDecision
	var err error
	
	switch policy.Type {
	case HorizontalScaling:
		decision, err = m.executeHorizontalScaling(ctx, platform, component, policy)
	case VerticalScaling:
		decision, err = m.executeVerticalScaling(ctx, platform, component, policy)
	case PredictiveScaling:
		decision, err = m.executePredictiveScaling(ctx, platform, component, policy)
	case CostAwareScaling:
		decision, err = m.executeCostAwareScaling(ctx, platform, component, policy)
	default:
		// Default to horizontal scaling
		decision, err = m.executeHorizontalScaling(ctx, platform, component, policy)
	}
	
	// Record the scaling event
	if decision != nil {
		event := ScalingEvent{
			Type:      policy.Type,
			Component: component,
			Timestamp: time.Now(),
			Success:   err == nil,
			Duration:  time.Since(startTime),
			Reason:    decision.Reason,
		}
		
		if err != nil {
			event.Error = err.Error()
		} else {
			event.FromReplicas = decision.CurrentReplicas
			event.ToReplicas = decision.TargetReplicas
			event.FromResources = &decision.CurrentResources
			event.ToResources = &decision.TargetResources
			event.CostImpact = decision.CostImpact
		}
		
		if recordErr := m.eventTracker.RecordScalingEvent(ctx, platform, event); recordErr != nil {
			log.Error(recordErr, "Failed to record scaling event")
		}
	}
	
	return err
}

// executeHorizontalScaling executes horizontal pod autoscaling
func (m *AutoScalingManager) executeHorizontalScaling(
	ctx context.Context,
	platform *v1beta1.ObservabilityPlatform,
	component v1beta1.ComponentType,
	policy ScalingPolicy,
) (*ScalingDecision, error) {
	log := log.FromContext(ctx)
	
	// Create or update HPA
	if m.config.HPAEnabled {
		if err := m.hpaManager.CreateOrUpdateHPA(ctx, platform, component, policy); err != nil {
			return nil, fmt.Errorf("failed to manage HPA: %w", err)
		}
		
		// Get HPA status for decision
		hpa, err := m.hpaManager.GetHPAStatus(ctx, platform, component)
		if err != nil {
			log.Error(err, "Failed to get HPA status")
		} else {
			decision := &ScalingDecision{
				Type:            HorizontalScaling,
				Component:       component,
				CurrentReplicas: hpa.Status.CurrentReplicas,
				TargetReplicas:  hpa.Status.DesiredReplicas,
				Reason:          "HPA-based scaling",
				Timestamp:       time.Now(),
			}
			
			// Calculate cost impact
			if m.config.CostAwareEnabled {
				decision.CostImpact = m.calculateCostImpact(ctx, platform, component, 
					hpa.Status.CurrentReplicas, hpa.Status.DesiredReplicas)
			}
			
			return decision, nil
		}
	}
	
	// Manual scaling based on metrics
	return m.manualHorizontalScaling(ctx, platform, component, policy)
}

// executeVerticalScaling executes vertical pod autoscaling
func (m *AutoScalingManager) executeVerticalScaling(
	ctx context.Context,
	platform *v1beta1.ObservabilityPlatform,
	component v1beta1.ComponentType,
	policy ScalingPolicy,
) (*ScalingDecision, error) {
	log := log.FromContext(ctx)
	
	// Create or update VPA
	if m.config.VPAEnabled {
		if err := m.vpaManager.CreateOrUpdateVPA(ctx, platform, component, policy); err != nil {
			return nil, fmt.Errorf("failed to manage VPA: %w", err)
		}
		
		// Get recommendations
		recommendation, err := m.vpaManager.GetRecommendation(ctx, platform, component)
		if err != nil {
			log.Error(err, "Failed to get VPA recommendation")
			return nil, err
		}
		
		decision := &ScalingDecision{
			Type:             VerticalScaling,
			Component:        component,
			CurrentResources: recommendation.CurrentRequests.DeepCopy(),
			TargetResources: corev1.ResourceRequirements{
				Requests: recommendation.RecommendedRequests,
				Limits:   recommendation.RecommendedLimits,
			},
			Reason:     "VPA recommendation",
			Timestamp:  time.Now(),
			CostImpact: recommendation.CostSavings,
		}
		
		// Apply recommendation if confidence is high enough
		if recommendation.Confidence >= 0.8 {
			if err := m.applyResourceRecommendation(ctx, platform, component, recommendation); err != nil {
				return decision, fmt.Errorf("failed to apply recommendation: %w", err)
			}
		}
		
		return decision, nil
	}
	
	return nil, fmt.Errorf("VPA is not enabled")
}

// executePredictiveScaling executes predictive scaling
func (m *AutoScalingManager) executePredictiveScaling(
	ctx context.Context,
	platform *v1beta1.ObservabilityPlatform,
	component v1beta1.ComponentType,
	policy ScalingPolicy,
) (*ScalingDecision, error) {
	log := log.FromContext(ctx)
	
	if !m.config.PredictiveEnabled {
		return nil, fmt.Errorf("predictive scaling is not enabled")
	}
	
	// Get historical metrics
	historicalData, err := m.metricsProvider.GetHistoricalMetrics(ctx, component, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical metrics: %w", err)
	}
	
	// Train predictive model
	if err := m.predictiveEngine.Train(ctx, historicalData); err != nil {
		return nil, fmt.Errorf("failed to train predictive model: %w", err)
	}
	
	// Get current replicas
	currentReplicas := m.getCurrentReplicas(ctx, platform, component)
	
	// Make prediction
	decision, err := m.predictiveEngine.MakePrediction(ctx, component, currentReplicas, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to make prediction: %w", err)
	}
	
	// Apply scaling if significant change predicted
	if decision.TargetReplicas != currentReplicas {
		log.Info("Applying predictive scaling",
			"component", component,
			"currentReplicas", currentReplicas,
			"targetReplicas", decision.TargetReplicas,
			"reason", decision.Reason,
		)
		
		if err := m.scaleComponent(ctx, platform, component, decision.TargetReplicas); err != nil {
			return decision, fmt.Errorf("failed to scale component: %w", err)
		}
	}
	
	return decision, nil
}

// executeCostAwareScaling executes cost-aware scaling
func (m *AutoScalingManager) executeCostAwareScaling(
	ctx context.Context,
	platform *v1beta1.ObservabilityPlatform,
	component v1beta1.ComponentType,
	policy ScalingPolicy,
) (*ScalingDecision, error) {
	log := log.FromContext(ctx)
	
	if !m.config.CostAwareEnabled {
		return nil, fmt.Errorf("cost-aware scaling is not enabled")
	}
	
	// Get current configuration
	currentReplicas := m.getCurrentReplicas(ctx, platform, component)
	
	// Get cost threshold from policy
	budget := 1000.0 // Default budget
	if policy.CostThreshold != nil {
		budget = *policy.CostThreshold
	}
	
	// Find optimal configuration
	decision, err := m.costProvider.GetOptimalConfiguration(ctx, component, currentReplicas, budget)
	if err != nil {
		return nil, fmt.Errorf("failed to get optimal configuration: %w", err)
	}
	
	// Apply the configuration if it saves money
	if decision.CostImpact != nil && decision.CostImpact.MonthlySavings > 0 {
		log.Info("Applying cost-aware scaling",
			"component", component,
			"savings", decision.CostImpact.MonthlySavings,
			"targetReplicas", decision.TargetReplicas,
		)
		
		// Scale replicas
		if decision.TargetReplicas != currentReplicas {
			if err := m.scaleComponent(ctx, platform, component, decision.TargetReplicas); err != nil {
				return decision, fmt.Errorf("failed to scale component: %w", err)
			}
		}
		
		// Update resources if changed
		if !utils.ResourcesEqual(decision.CurrentResources, decision.TargetResources) {
			if err := m.updateComponentResources(ctx, platform, component, decision.TargetResources); err != nil {
				return decision, fmt.Errorf("failed to update resources: %w", err)
			}
		}
	}
	
	return decision, nil
}

// manualHorizontalScaling performs manual horizontal scaling based on metrics
func (m *AutoScalingManager) manualHorizontalScaling(
	ctx context.Context,
	platform *v1beta1.ObservabilityPlatform,
	component v1beta1.ComponentType,
	policy ScalingPolicy,
) (*ScalingDecision, error) {
	// Get current metrics
	cpuUtil, err := m.metricsProvider.GetCPUUtilization(ctx, component)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU utilization: %w", err)
	}
	
	memUtil, err := m.metricsProvider.GetMemoryUtilization(ctx, component)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory utilization: %w", err)
	}
	
	currentReplicas := m.getCurrentReplicas(ctx, platform, component)
	targetReplicas := currentReplicas
	reason := ""
	
	// Scale up if needed
	if (policy.TargetCPUPercent != nil && cpuUtil > float64(*policy.TargetCPUPercent)) ||
	   (policy.TargetMemoryPercent != nil && memUtil > float64(*policy.TargetMemoryPercent)) {
		targetReplicas = currentReplicas + 1
		reason = fmt.Sprintf("High resource utilization (CPU: %.1f%%, Memory: %.1f%%)", cpuUtil, memUtil)
	}
	
	// Scale down if possible
	if (policy.TargetCPUPercent != nil && cpuUtil < float64(*policy.TargetCPUPercent)*0.5) &&
	   (policy.TargetMemoryPercent != nil && memUtil < float64(*policy.TargetMemoryPercent)*0.5) {
		targetReplicas = currentReplicas - 1
		reason = fmt.Sprintf("Low resource utilization (CPU: %.1f%%, Memory: %.1f%%)", cpuUtil, memUtil)
	}
	
	// Apply bounds
	if targetReplicas < policy.MinReplicas {
		targetReplicas = policy.MinReplicas
	}
	if targetReplicas > policy.MaxReplicas {
		targetReplicas = policy.MaxReplicas
	}
	
	decision := &ScalingDecision{
		Type:            HorizontalScaling,
		Component:       component,
		CurrentReplicas: currentReplicas,
		TargetReplicas:  targetReplicas,
		Reason:          reason,
		Timestamp:       time.Now(),
	}
	
	// Apply scaling if needed
	if targetReplicas != currentReplicas {
		if err := m.scaleComponent(ctx, platform, component, targetReplicas); err != nil {
			return decision, err
		}
	}
	
	return decision, nil
}

// Helper methods

func (m *AutoScalingManager) getEnabledComponents(platform *v1beta1.ObservabilityPlatform) []v1beta1.ComponentType {
	components := []v1beta1.ComponentType{}
	
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		components = append(components, v1beta1.ComponentPrometheus)
	}
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		components = append(components, v1beta1.ComponentGrafana)
	}
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		components = append(components, v1beta1.ComponentLoki)
	}
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		components = append(components, v1beta1.ComponentTempo)
	}
	
	return components
}

func (m *AutoScalingManager) isScalingInProgress(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key := fmt.Sprintf("%s/%s/%s", platform.Namespace, platform.Name, component)
	return m.activeScaling[key]
}

func (m *AutoScalingManager) setScalingInProgress(platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, inProgress bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := fmt.Sprintf("%s/%s/%s", platform.Namespace, platform.Name, component)
	if inProgress {
		m.activeScaling[key] = true
	} else {
		delete(m.activeScaling, key)
	}
}

func (m *AutoScalingManager) getCurrentReplicas(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType) int32 {
	// This should get the actual replica count from the deployment
	// For now, return a default
	return 1
}

func (m *AutoScalingManager) scaleComponent(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, replicas int32) error {
	// This should update the deployment replica count
	// Implementation depends on how components are deployed
	return nil
}

func (m *AutoScalingManager) updateComponentResources(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, resources corev1.ResourceRequirements) error {
	// This should update the deployment resource requirements
	// Implementation depends on how components are deployed
	return nil
}

func (m *AutoScalingManager) applyResourceRecommendation(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, recommendation *ResourceRecommendation) error {
	// This should apply the VPA recommendation
	// Implementation depends on how components are deployed
	return nil
}

func (m *AutoScalingManager) calculateCostImpact(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component v1beta1.ComponentType, currentReplicas, targetReplicas int32) *CostImpact {
	// Get current resources (simplified)
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    utils.MustParseQuantity("500m"),
			corev1.ResourceMemory: utils.MustParseQuantity("1Gi"),
		},
	}
	
	currentCost, _ := m.costProvider.GetCurrentCost(ctx, component, currentReplicas, resources)
	projectedCost, _ := m.costProvider.GetProjectedCost(ctx, component, targetReplicas, resources)
	
	return &CostImpact{
		CurrentCost:    currentCost,
		ProjectedCost:  projectedCost,
		MonthlySavings: currentCost - projectedCost,
		Currency:       "USD",
	}
}

// Interface implementations

func (m *AutoScalingManager) CreateOrUpdateHPA(component v1beta1.ComponentType, policy ScalingPolicy) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) DeleteHPA(component v1beta1.ComponentType) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) GetHPAStatus(component v1beta1.ComponentType) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	// This would be called by the controller
	return nil, nil
}

func (m *AutoScalingManager) CreateOrUpdateVPA(component v1beta1.ComponentType, policy ScalingPolicy) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) DeleteVPA(component v1beta1.ComponentType) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) GetVPAStatus(component v1beta1.ComponentType) (*vpav1.VerticalPodAutoscaler, error) {
	// This would be called by the controller
	return nil, nil
}

func (m *AutoScalingManager) GetResourceRecommendations(component v1beta1.ComponentType) (*ResourceRecommendation, error) {
	// This would be called by the controller
	return nil, nil
}

func (m *AutoScalingManager) ApplyResourceRecommendation(component v1beta1.ComponentType, recommendation ResourceRecommendation) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) TrainPredictiveModel(component v1beta1.ComponentType, historicalData []MetricDataPoint) error {
	return m.predictiveEngine.Train(context.Background(), historicalData)
}

func (m *AutoScalingManager) GetPrediction(component v1beta1.ComponentType, duration time.Duration) (*ScalingDecision, error) {
	// This would be called by the controller
	return nil, nil
}

func (m *AutoScalingManager) GetCostAnalysis(component v1beta1.ComponentType) (*CostImpact, error) {
	// This would be called by the controller
	return nil, nil
}

func (m *AutoScalingManager) OptimizeForCost(component v1beta1.ComponentType, budget float64) (*ScalingDecision, error) {
	// This would be called by the controller
	return nil, nil
}

func (m *AutoScalingManager) RecordScalingEvent(event ScalingEvent) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) GetScalingHistory(component v1beta1.ComponentType, duration time.Duration) ([]ScalingEvent, error) {
	return m.eventTracker.GetScalingHistory(context.Background(), component, duration)
}

func (m *AutoScalingManager) DrainWorkload(component v1beta1.ComponentType, podName string) error {
	// This would be called by the controller
	return nil
}

func (m *AutoScalingManager) GetDrainStatus(component v1beta1.ComponentType, podName string) (bool, error) {
	// This would be called by the controller
	return false, nil
}
