package autoscaling

import (
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScalingType represents the type of scaling
type ScalingType string

const (
	// HorizontalScaling represents horizontal pod autoscaling
	HorizontalScaling ScalingType = "horizontal"
	// VerticalScaling represents vertical pod autoscaling
	VerticalScaling ScalingType = "vertical"
	// PredictiveScaling represents predictive scaling based on historical data
	PredictiveScaling ScalingType = "predictive"
	// CostAwareScaling represents cost-aware scaling
	CostAwareScaling ScalingType = "cost-aware"
)

// ScalingDecision represents a scaling decision
type ScalingDecision struct {
	Type           ScalingType
	Component      v1beta1.ComponentType
	CurrentReplicas int32
	TargetReplicas  int32
	CurrentResources v1.ResourceRequirements
	TargetResources  v1.ResourceRequirements
	Reason          string
	Timestamp       time.Time
	CostImpact      *CostImpact
}

// CostImpact represents the cost impact of a scaling decision
type CostImpact struct {
	CurrentCost    float64
	ProjectedCost  float64
	MonthlySavings float64
	Currency       string
}

// ScalingPolicy defines the scaling behavior
type ScalingPolicy struct {
	Name               string
	Type               ScalingType
	MinReplicas        int32
	MaxReplicas        int32
	TargetCPUPercent   *int32
	TargetMemoryPercent *int32
	CustomMetrics      []CustomMetric
	CostThreshold      *float64
	ScaleDownStabilizationWindow *time.Duration
	ResourceBuffer     int32 // Percentage buffer for resource recommendations
}

// CustomMetric defines a custom metric for scaling
type CustomMetric struct {
	Name       string
	Type       string // prometheus, external
	Query      string
	TargetValue string
	MetricType autoscalingv2.MetricType
}

// ResourceRecommendation represents a resource recommendation
type ResourceRecommendation struct {
	Component        v1beta1.ComponentType
	Container        string
	CurrentRequests  v1.ResourceList
	CurrentLimits    v1.ResourceList
	RecommendedRequests v1.ResourceList
	RecommendedLimits   v1.ResourceList
	Confidence       float64
	BasedOnDays      int
	CostSavings      *CostImpact
}

// ScalingEvent represents a scaling event
type ScalingEvent struct {
	ID            string
	Type          ScalingType
	Component     v1beta1.ComponentType
	Timestamp     time.Time
	FromReplicas  int32
	ToReplicas    int32
	FromResources *v1.ResourceRequirements
	ToResources   *v1.ResourceRequirements
	Reason        string
	Success       bool
	Error         string
	Duration      time.Duration
	CostImpact    *CostImpact
}

// PredictiveModel represents a predictive scaling model
type PredictiveModel struct {
	Component        v1beta1.ComponentType
	ModelType        string // linear, ml, lstm
	TrainingData     []MetricDataPoint
	Accuracy         float64
	LastUpdated      time.Time
	PredictionWindow time.Duration
}

// MetricDataPoint represents a metric data point for predictive scaling
type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// WorkloadDrainConfig defines how to drain workloads during scale-down
type WorkloadDrainConfig struct {
	MaxUnavailable      int32
	MaxSurge           int32
	DrainTimeout       time.Duration
	GracePeriod        time.Duration
	EvictLocalStorage  bool
	SkipWaitForDeleteTimeout time.Duration
}

// AutoScalingConfig defines the overall autoscaling configuration
type AutoScalingConfig struct {
	Enabled              bool
	HPAEnabled           bool
	VPAEnabled           bool
	PredictiveEnabled    bool
	CostAwareEnabled     bool
	Policies             map[v1beta1.ComponentType]ScalingPolicy
	DrainConfig          WorkloadDrainConfig
	MetricsRetention     time.Duration
	EvaluationInterval   time.Duration
	CostProvider         string // aws, gcp, azure, custom
	CustomMetricsAdapter string // prometheus-adapter endpoint
}

// Manager defines the interface for the autoscaling manager
type Manager interface {
	// HPA Management
	CreateOrUpdateHPA(component v1beta1.ComponentType, policy ScalingPolicy) error
	DeleteHPA(component v1beta1.ComponentType) error
	GetHPAStatus(component v1beta1.ComponentType) (*autoscalingv2.HorizontalPodAutoscaler, error)

	// VPA Management
	CreateOrUpdateVPA(component v1beta1.ComponentType, policy ScalingPolicy) error
	DeleteVPA(component v1beta1.ComponentType) error
	GetVPAStatus(component v1beta1.ComponentType) (*vpav1.VerticalPodAutoscaler, error)

	// Resource Recommendations
	GetResourceRecommendations(component v1beta1.ComponentType) (*ResourceRecommendation, error)
	ApplyResourceRecommendation(component v1beta1.ComponentType, recommendation ResourceRecommendation) error

	// Predictive Scaling
	TrainPredictiveModel(component v1beta1.ComponentType, historicalData []MetricDataPoint) error
	GetPrediction(component v1beta1.ComponentType, duration time.Duration) (*ScalingDecision, error)

	// Cost-Aware Scaling
	GetCostAnalysis(component v1beta1.ComponentType) (*CostImpact, error)
	OptimizeForCost(component v1beta1.ComponentType, budget float64) (*ScalingDecision, error)

	// Event Tracking
	RecordScalingEvent(event ScalingEvent) error
	GetScalingHistory(component v1beta1.ComponentType, duration time.Duration) ([]ScalingEvent, error)

	// Workload Draining
	DrainWorkload(component v1beta1.ComponentType, podName string) error
	GetDrainStatus(component v1beta1.ComponentType, podName string) (bool, error)
}

// MetricsProvider defines the interface for metrics providers
type MetricsProvider interface {
	GetCPUUtilization(component v1beta1.ComponentType) (float64, error)
	GetMemoryUtilization(component v1beta1.ComponentType) (float64, error)
	GetCustomMetric(component v1beta1.ComponentType, metric CustomMetric) (float64, error)
	GetHistoricalMetrics(component v1beta1.ComponentType, duration time.Duration) ([]MetricDataPoint, error)
}

// CostProvider defines the interface for cost providers
type CostProvider interface {
	GetCurrentCost(component v1beta1.ComponentType) (float64, error)
	GetProjectedCost(component v1beta1.ComponentType, replicas int32, resources v1.ResourceRequirements) (float64, error)
	GetOptimalConfiguration(component v1beta1.ComponentType, budget float64) (*ScalingDecision, error)
}

// PredictiveEngine defines the interface for predictive scaling engines
type PredictiveEngine interface {
	Train(data []MetricDataPoint) error
	Predict(horizon time.Duration) ([]MetricDataPoint, error)
	GetAccuracy() float64
	GetModelType() string
}
