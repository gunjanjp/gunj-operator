package autoscaling

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Scaling event metrics
	scalingEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gunj_operator_scaling_events_total",
			Help: "Total number of scaling events",
		},
		[]string{"component", "type", "success"},
	)
	
	scalingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gunj_operator_scaling_duration_seconds",
			Help:    "Time taken to complete scaling operation",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"component", "type"},
	)
	
	// Resource utilization metrics
	componentCPUUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_cpu_utilization_percent",
			Help: "Current CPU utilization percentage for components",
		},
		[]string{"platform", "component"},
	)
	
	componentMemoryUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_memory_utilization_percent",
			Help: "Current memory utilization percentage for components",
		},
		[]string{"platform", "component"},
	)
	
	// Replica metrics
	componentReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_replicas",
			Help: "Current number of replicas for components",
		},
		[]string{"platform", "component"},
	)
	
	componentTargetReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_target_replicas",
			Help: "Target number of replicas for components",
		},
		[]string{"platform", "component"},
	)
	
	// Cost metrics
	costSavings = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_cost_savings_monthly_usd",
			Help: "Monthly cost savings in USD from autoscaling",
		},
		[]string{"component"},
	)
	
	componentMonthlyCost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_monthly_cost_usd",
			Help: "Current monthly cost in USD for components",
		},
		[]string{"platform", "component"},
	)
	
	// HPA metrics
	hpaActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_hpa_active",
			Help: "Whether HPA is active for a component (1 = active, 0 = inactive)",
		},
		[]string{"platform", "component"},
	)
	
	hpaMinReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_hpa_min_replicas",
			Help: "Minimum replicas configured for HPA",
		},
		[]string{"platform", "component"},
	)
	
	hpaMaxReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_hpa_max_replicas",
			Help: "Maximum replicas configured for HPA",
		},
		[]string{"platform", "component"},
	)
	
	// VPA metrics
	vpaActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_vpa_active",
			Help: "Whether VPA is active for a component (1 = active, 0 = inactive)",
		},
		[]string{"platform", "component"},
	)
	
	vpaRecommendedCPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_vpa_recommended_cpu_cores",
			Help: "VPA recommended CPU in cores",
		},
		[]string{"platform", "component", "container"},
	)
	
	vpaRecommendedMemory = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_vpa_recommended_memory_bytes",
			Help: "VPA recommended memory in bytes",
		},
		[]string{"platform", "component", "container"},
	)
	
	// Predictive scaling metrics
	predictedLoad = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_predicted_load",
			Help: "Predicted load for components",
		},
		[]string{"platform", "component", "horizon"},
	)
	
	predictionAccuracy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_prediction_accuracy_percent",
			Help: "Accuracy of predictive scaling model",
		},
		[]string{"component", "model_type"},
	)
	
	// Workload draining metrics
	drainingPods = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_draining_pods",
			Help: "Number of pods currently being drained",
		},
		[]string{"platform", "component"},
	)
	
	drainDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gunj_operator_drain_duration_seconds",
			Help:    "Time taken to drain a pod",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"component"},
	)
	
	// Autoscaling policy metrics
	policyViolations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gunj_operator_policy_violations_total",
			Help: "Total number of autoscaling policy violations",
		},
		[]string{"platform", "component", "policy", "reason"},
	)
	
	// Resource efficiency metrics
	resourceEfficiency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_resource_efficiency_percent",
			Help: "Resource efficiency percentage (actual usage / allocated)",
		},
		[]string{"platform", "component", "resource"},
	)
	
	// Scaling queue metrics
	scalingQueueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_scaling_queue_size",
			Help: "Number of pending scaling operations in queue",
		},
		[]string{"type"},
	)
	
	scalingQueueLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gunj_operator_scaling_queue_latency_seconds",
			Help:    "Time spent waiting in scaling queue",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)
)

func init() {
	// Register all metrics
	metrics.Registry.MustRegister(
		// Scaling events
		scalingEventTotal,
		scalingDuration,
		
		// Resource utilization
		componentCPUUtilization,
		componentMemoryUtilization,
		
		// Replicas
		componentReplicas,
		componentTargetReplicas,
		
		// Cost
		costSavings,
		componentMonthlyCost,
		
		// HPA
		hpaActive,
		hpaMinReplicas,
		hpaMaxReplicas,
		
		// VPA
		vpaActive,
		vpaRecommendedCPU,
		vpaRecommendedMemory,
		
		// Predictive
		predictedLoad,
		predictionAccuracy,
		
		// Draining
		drainingPods,
		drainDuration,
		
		// Policy
		policyViolations,
		
		// Efficiency
		resourceEfficiency,
		
		// Queue
		scalingQueueSize,
		scalingQueueLatency,
	)
}

// UpdateComponentMetrics updates component-specific metrics
func UpdateComponentMetrics(platform, component string, replicas, targetReplicas int32, cpuUtil, memUtil float64) {
	componentReplicas.WithLabelValues(platform, component).Set(float64(replicas))
	componentTargetReplicas.WithLabelValues(platform, component).Set(float64(targetReplicas))
	componentCPUUtilization.WithLabelValues(platform, component).Set(cpuUtil)
	componentMemoryUtilization.WithLabelValues(platform, component).Set(memUtil)
}

// UpdateHPAMetrics updates HPA-specific metrics
func UpdateHPAMetrics(platform, component string, active bool, minReplicas, maxReplicas int32) {
	activeValue := 0.0
	if active {
		activeValue = 1.0
	}
	hpaActive.WithLabelValues(platform, component).Set(activeValue)
	hpaMinReplicas.WithLabelValues(platform, component).Set(float64(minReplicas))
	hpaMaxReplicas.WithLabelValues(platform, component).Set(float64(maxReplicas))
}

// UpdateVPAMetrics updates VPA-specific metrics
func UpdateVPAMetrics(platform, component, container string, active bool, cpuCores, memoryBytes float64) {
	activeValue := 0.0
	if active {
		activeValue = 1.0
	}
	vpaActive.WithLabelValues(platform, component).Set(activeValue)
	vpaRecommendedCPU.WithLabelValues(platform, component, container).Set(cpuCores)
	vpaRecommendedMemory.WithLabelValues(platform, component, container).Set(memoryBytes)
}

// UpdatePredictiveMetrics updates predictive scaling metrics
func UpdatePredictiveMetrics(platform, component, horizon string, load float64, modelType string, accuracy float64) {
	predictedLoad.WithLabelValues(platform, component, horizon).Set(load)
	predictionAccuracy.WithLabelValues(component, modelType).Set(accuracy * 100)
}

// UpdateDrainMetrics updates workload draining metrics
func UpdateDrainMetrics(platform, component string, drainingCount int) {
	drainingPods.WithLabelValues(platform, component).Set(float64(drainingCount))
}

// RecordDrainDuration records the time taken to drain a pod
func RecordDrainDuration(component string, duration float64) {
	drainDuration.WithLabelValues(component).Observe(duration)
}

// UpdateCostMetrics updates cost-related metrics
func UpdateCostMetrics(platform, component string, monthlyCost, savings float64) {
	componentMonthlyCost.WithLabelValues(platform, component).Set(monthlyCost)
	costSavings.WithLabelValues(component).Set(savings)
}

// RecordPolicyViolation records a policy violation
func RecordPolicyViolation(platform, component, policy, reason string) {
	policyViolations.WithLabelValues(platform, component, policy, reason).Inc()
}

// UpdateResourceEfficiency updates resource efficiency metrics
func UpdateResourceEfficiency(platform, component, resource string, efficiency float64) {
	resourceEfficiency.WithLabelValues(platform, component, resource).Set(efficiency)
}

// UpdateQueueMetrics updates scaling queue metrics
func UpdateQueueMetrics(queueType string, size int) {
	scalingQueueSize.WithLabelValues(queueType).Set(float64(size))
}

// RecordQueueLatency records time spent in scaling queue
func RecordQueueLatency(queueType string, latency float64) {
	scalingQueueLatency.WithLabelValues(queueType).Observe(latency)
}
