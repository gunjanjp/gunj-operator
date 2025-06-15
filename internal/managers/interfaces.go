/*
Copyright 2025.

Licensed under the MIT License.
*/

package managers

import (
	"context"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// ComponentManager is the base interface for all component managers
type ComponentManager interface {
	// Reconcile reconciles the component for the given platform
	Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error

	// ReconcileWithConfig reconciles the component with provided configuration
	ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error

	// Delete removes the component resources
	Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error

	// GetStatus returns the current status of the component
	GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error)

	// Validate validates the component configuration
	Validate(platform *observabilityv1beta1.ObservabilityPlatform) error

	// GetServiceURL returns the service URL for the component
	GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string
}

// PrometheusManager manages Prometheus deployments
type PrometheusManager interface {
	ComponentManager

	// UpdateConfiguration updates Prometheus configuration without restart
	UpdateConfiguration(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error

	// ScaleReplicas scales Prometheus replicas
	ScaleReplicas(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, replicas int32) error
}

// GrafanaManager manages Grafana deployments
type GrafanaManager interface {
	ComponentManager

	// ImportDashboards imports dashboards into Grafana
	ImportDashboards(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error

	// ConfigureDataSources configures data sources in Grafana
	ConfigureDataSources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
}

// LokiManager manages Loki deployments
type LokiManager interface {
	ComponentManager

	// ConfigureStorage configures storage backend for Loki
	ConfigureStorage(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error

	// UpdateRetention updates log retention policies
	UpdateRetention(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
}

// TempoManager manages Tempo deployments
type TempoManager interface {
	ComponentManager

	// ConfigureReceivers configures trace receivers
	ConfigureReceivers(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error

	// UpdateSampling updates sampling configuration
	UpdateSampling(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
}

// CostManager manages cost optimization for observability platforms
type CostManager interface {
	// AnalyzePlatformCosts analyzes costs for all platform components
	AnalyzePlatformCosts(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*CostAnalysis, error)

	// GetRecommendations generates cost optimization recommendations
	GetRecommendations(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*OptimizationRecommendations, error)

	// ApplyOptimizations applies cost optimization recommendations
	ApplyOptimizations(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, opts *OptimizationOptions) error

	// SetBudgetAlert configures budget alerts for a platform
	SetBudgetAlert(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, budget *BudgetConfig) error
}

// ManagerFactory creates component managers
type ManagerFactory interface {
	// CreatePrometheusManager creates a new Prometheus manager
	CreatePrometheusManager() PrometheusManager

	// CreateGrafanaManager creates a new Grafana manager
	CreateGrafanaManager() GrafanaManager

	// CreateLokiManager creates a new Loki manager
	CreateLokiManager() LokiManager

	// CreateTempoManager creates a new Tempo manager
	CreateTempoManager() TempoManager

	// CreateCostManager creates a new Cost manager
	CreateCostManager() CostManager
}

// Cost-related types

// CostAnalysis represents the cost analysis for a platform
type CostAnalysis struct {
	Platform  string    `json:"platform"`
	Namespace string    `json:"namespace"`
	TotalCost float64   `json:"totalCost"`
	Currency  string    `json:"currency"`
}

// OptimizationRecommendations contains cost optimization suggestions
type OptimizationRecommendations struct {
	ResourceRecommendations map[string]interface{}    `json:"resourceRecommendations"`
	SpotInstanceCandidates  map[string]bool           `json:"spotInstanceCandidates"`
	ScalingPolicies         map[string]interface{}    `json:"scalingPolicies"`
	StorageOptimizations    map[string]interface{}    `json:"storageOptimizations"`
	TotalSavings            float64                   `json:"totalSavings"`
}

// OptimizationOptions for applying optimizations
type OptimizationOptions struct {
	ApplyResourceOptimization bool     `json:"applyResourceOptimization"`
	EnableSpotInstances       bool     `json:"enableSpotInstances"`
	ApplyScalingPolicies      bool     `json:"applyScalingPolicies"`
	OptimizeStorage           bool     `json:"optimizeStorage"`
	Categories                []string `json:"categories"`
	DryRun                    bool     `json:"dryRun"`
}

// BudgetConfig for cost alerts
type BudgetConfig struct {
	MonthlyLimit    float64          `json:"monthlyLimit"`
	AlertThreshold  float64          `json:"alertThreshold"`
	AlertRecipients []string         `json:"alertRecipients"`
}
