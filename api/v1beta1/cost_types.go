package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CostOptimizationSpec defines cost optimization configuration
type CostOptimizationSpec struct {
	// Enabled indicates whether cost optimization is enabled
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ResourceOptimization enables resource right-sizing
	// +kubebuilder:default=true
	ResourceOptimization bool `json:"resourceOptimization,omitempty"`

	// SpotInstances configuration for using spot instances
	// +optional
	SpotInstances *SpotInstancesConfig `json:"spotInstances,omitempty"`

	// AutoScaling enables automatic scaling based on usage
	// +kubebuilder:default=true
	AutoScaling bool `json:"autoScaling,omitempty"`

	// Budget configuration for cost alerts
	// +optional
	Budget *BudgetConfig `json:"budget,omitempty"`

	// CostAllocation for chargeback/showback
	// +optional
	CostAllocation *CostAllocationConfig `json:"costAllocation,omitempty"`
}

// CostOptimizationStatus defines the observed state of cost optimization
type CostOptimizationStatus struct {
	// LastOptimization timestamp
	LastOptimization metav1.Time `json:"lastOptimization,omitempty"`

	// Applied indicates if optimizations have been applied
	Applied bool `json:"applied,omitempty"`

	// EstimatedSavings in USD per month
	EstimatedSavings float64 `json:"estimatedSavings,omitempty"`

	// CurrentCost in USD per month
	CurrentCost float64 `json:"currentCost,omitempty"`

	// OptimizedCost in USD per month
	OptimizedCost float64 `json:"optimizedCost,omitempty"`

	// Recommendations count
	RecommendationsCount int `json:"recommendationsCount,omitempty"`
}

// SpotInstancesConfig defines spot instances configuration
type SpotInstancesConfig struct {
	// Enabled indicates whether to use spot instances
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// AllowedComponents lists components that can use spot instances
	// +optional
	AllowedComponents []string `json:"allowedComponents,omitempty"`

	// MaxSpotPercentage is the maximum percentage of instances that can be spot
	// +kubebuilder:default=70
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	MaxSpotPercentage int `json:"maxSpotPercentage,omitempty"`
}

// BudgetConfig defines budget and alerting configuration
type BudgetConfig struct {
	// MonthlyLimit in USD
	// +kubebuilder:validation:Minimum=0
	MonthlyLimit float64 `json:"monthlyLimit"`

	// AlertThresholds as percentages of the budget
	// +kubebuilder:default={80,90,100}
	AlertThresholds []int `json:"alertThresholds,omitempty"`

	// AlertChannels for notifications
	// +optional
	AlertChannels []AlertChannel `json:"alertChannels,omitempty"`

	// Actions to take at different thresholds
	// +optional
	Actions []BudgetAction `json:"actions,omitempty"`
}

// AlertChannel defines a notification channel
type AlertChannel struct {
	// Type of channel (email, slack, webhook)
	// +kubebuilder:validation:Enum=email;slack;webhook
	Type string `json:"type"`

	// Target address/URL
	Target string `json:"target"`
}

// BudgetAction defines actions to take when budget thresholds are reached
type BudgetAction struct {
	// Threshold percentage
	Threshold int `json:"threshold"`

	// Action to take
	// +kubebuilder:validation:Enum=alert;scaleDown;disableNonEssential;preventNewDeployments
	Action string `json:"action"`
}

// CostAllocationConfig defines cost allocation configuration
type CostAllocationConfig struct {
	// Enabled indicates whether cost allocation is enabled
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Tags for cost allocation
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// Department for chargeback
	// +optional
	Department string `json:"department,omitempty"`

	// CostCenter for accounting
	// +optional
	CostCenter string `json:"costCenter,omitempty"`

	// Team ownership
	// +optional
	Team string `json:"team,omitempty"`
}

// AutoscalingSpec defines autoscaling configuration
type AutoscalingSpec struct {
	// Enabled indicates whether autoscaling is enabled
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// MinReplicas is the minimum number of replicas
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the maximum number of replicas
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	// Metrics to use for scaling decisions
	// +optional
	Metrics []autoscalingv2.MetricSpec `json:"metrics,omitempty"`

	// Behavior configures scaling behavior
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// Toleration represents a pod toleration
type Toleration struct {
	// Key is the taint key that the toleration applies to
	// +optional
	Key string `json:"key,omitempty"`

	// Operator represents a key's relationship to the value
	// +kubebuilder:validation:Enum=Exists;Equal
	// +kubebuilder:default=Equal
	Operator string `json:"operator,omitempty"`

	// Value is the taint value the toleration matches to
	// +optional
	Value string `json:"value,omitempty"`

	// Effect indicates the taint effect to match
	// +kubebuilder:validation:Enum=NoSchedule;PreferNoSchedule;NoExecute
	// +optional
	Effect string `json:"effect,omitempty"`

	// TolerationSeconds represents the period of time the toleration tolerates the taint
	// +optional
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// ServiceMeshSpec defines service mesh integration configuration
type ServiceMeshSpec struct {
	// Provider specifies the service mesh provider
	// +kubebuilder:validation:Enum=istio;linkerd
	Provider string `json:"provider"`

	// ControlPlane specifies the name of the service mesh control plane
	ControlPlane string `json:"controlPlane"`

	// EnableTracing enables distributed tracing integration
	// +kubebuilder:default=true
	EnableTracing bool `json:"enableTracing,omitempty"`

	// EnableMetrics enables service mesh metrics
	// +kubebuilder:default=true
	EnableMetrics bool `json:"enableMetrics,omitempty"`

	// IstioConfig contains Istio-specific configuration
	// +optional
	IstioConfig *IstioConfig `json:"istioConfig,omitempty"`

	// LinkerdConfig contains Linkerd-specific configuration
	// +optional
	LinkerdConfig *LinkerdConfig `json:"linkerdConfig,omitempty"`
}

// ServiceMeshStatus defines the observed state of service mesh integration
type ServiceMeshStatus struct {
	// Ready indicates if service mesh integration is ready
	Ready bool `json:"ready"`

	// InjectedPods is the number of pods with sidecar injection
	InjectedPods int32 `json:"injectedPods,omitempty"`

	// LastSyncTime is the last time service mesh config was synced
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Message contains any status message
	Message string `json:"message,omitempty"`
}

// IstioConfig defines Istio-specific configuration
type IstioConfig struct {
	// Namespace where Istio is installed
	// +kubebuilder:default="istio-system"
	Namespace string `json:"namespace,omitempty"`

	// MeshID for multi-cluster setups
	// +optional
	MeshID string `json:"meshID,omitempty"`

	// TraceProvider configuration
	// +optional
	TraceProvider *TraceProvider `json:"traceProvider,omitempty"`
}

// LinkerdConfig defines Linkerd-specific configuration
type LinkerdConfig struct {
	// Namespace where Linkerd is installed
	// +kubebuilder:default="linkerd"
	Namespace string `json:"namespace,omitempty"`

	// ClusterName for multi-cluster setups
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
}

// TraceProvider defines trace provider configuration
type TraceProvider struct {
	// Service name of the trace provider
	Service string `json:"service"`

	// Port of the trace provider
	Port int32 `json:"port"`
}

// PrometheusRule defines a Prometheus alerting rule
type PrometheusRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PrometheusRuleSpec `json:"spec"`
}

// PrometheusRuleSpec defines the desired state of PrometheusRule
type PrometheusRuleSpec struct {
	// Groups is a list of rule groups
	Groups []RuleGroup `json:"groups"`
}

// RuleGroup defines a group of alerting rules
type RuleGroup struct {
	// Name of the rule group
	Name string `json:"name"`

	// Interval at which to evaluate rules
	// +optional
	Interval string `json:"interval,omitempty"`

	// Rules in the group
	Rules []Rule `json:"rules"`
}

// Rule defines a single alerting rule
type Rule struct {
	// Alert name
	Alert string `json:"alert"`

	// Expr is the PromQL expression
	Expr string `json:"expr"`

	// For defines how long the condition must be true to fire alert
	// +optional
	For string `json:"for,omitempty"`

	// Labels to add to the alert
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to add to the alert
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}
