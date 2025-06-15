package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// GitOps metrics
var (
	// GitOps deployment metrics
	GitOpsDeploymentTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitops_deployments_total",
			Help: "Total number of GitOps deployments",
		},
		[]string{"namespace", "sync_provider", "phase"},
	)

	GitOpsSyncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_sync_duration_seconds",
			Help:    "Duration of GitOps sync operations in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~100s
		},
		[]string{"namespace", "name", "sync_provider"},
	)

	GitOpsSyncTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_sync_total",
			Help: "Total number of GitOps sync operations",
		},
		[]string{"namespace", "name", "sync_provider", "status"},
	)

	GitOpsSyncErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_sync_errors_total",
			Help: "Total number of GitOps sync errors",
		},
		[]string{"namespace", "name", "sync_provider", "error_type"},
	)

	// Drift detection metrics
	GitOpsDriftDetected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_drift_detected_total",
			Help: "Total number of drift detections",
		},
		[]string{"namespace", "name", "resource_type"},
	)

	GitOpsDriftCorrected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_drift_corrected_total",
			Help: "Total number of drift corrections",
		},
		[]string{"namespace", "name", "resource_type"},
	)

	GitOpsDriftSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_drift_size_resources",
			Help:    "Number of resources with drift",
			Buckets: prometheus.LinearBuckets(0, 1, 20), // 0 to 20 resources
		},
		[]string{"namespace", "name"},
	)

	// Rollback metrics
	GitOpsRollbackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_rollback_total",
			Help: "Total number of rollback operations",
		},
		[]string{"namespace", "name", "reason"},
	)

	GitOpsRollbackDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_rollback_duration_seconds",
			Help:    "Duration of rollback operations in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~1000s
		},
		[]string{"namespace", "name"},
	)

	GitOpsHealthCheckFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_health_check_failures_total",
			Help: "Total number of health check failures",
		},
		[]string{"namespace", "name", "check_type"},
	)

	// Promotion metrics
	GitOpsPromotionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_promotion_total",
			Help: "Total number of promotion operations",
		},
		[]string{"from_env", "to_env", "strategy", "status"},
	)

	GitOpsPromotionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_promotion_duration_seconds",
			Help:    "Duration of promotion operations in seconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 10), // 10s to ~10000s
		},
		[]string{"from_env", "to_env", "strategy"},
	)

	GitOpsPromotionApprovalTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_promotion_approval_time_seconds",
			Help:    "Time taken for promotion approval in seconds",
			Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~17hours
		},
		[]string{"from_env", "to_env"},
	)

	// Webhook metrics
	GitOpsWebhookTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_webhook_total",
			Help: "Total number of webhook events received",
		},
		[]string{"provider", "event_type", "status"},
	)

	GitOpsWebhookLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_webhook_latency_seconds",
			Help:    "Latency of webhook processing in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s
		},
		[]string{"provider", "event_type"},
	)

	// ArgoCD specific metrics
	GitOpsArgoCDAppTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitops_argocd_applications_total",
			Help: "Total number of ArgoCD applications managed",
		},
		[]string{"namespace", "project", "sync_status"},
	)

	GitOpsArgoCDSyncTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_argocd_sync_total",
			Help: "Total number of ArgoCD sync operations",
		},
		[]string{"namespace", "name", "status"},
	)

	// Flux specific metrics
	GitOpsFluxKustomizationTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitops_flux_kustomizations_total",
			Help: "Total number of Flux Kustomizations managed",
		},
		[]string{"namespace", "ready"},
	)

	GitOpsFluxReconcileTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_flux_reconcile_total",
			Help: "Total number of Flux reconciliation operations",
		},
		[]string{"namespace", "name", "status"},
	)

	// Git sync metrics
	GitOpsGitCloneTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_git_clone_total",
			Help: "Total number of git clone operations",
		},
		[]string{"namespace", "name", "status"},
	)

	GitOpsGitCloneDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_git_clone_duration_seconds",
			Help:    "Duration of git clone operations in seconds",
			Buckets: prometheus.ExponentialBuckets(0.5, 2, 10), // 0.5s to ~500s
		},
		[]string{"namespace", "name"},
	)

	GitOpsLastSyncTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitops_last_sync_timestamp_seconds",
			Help: "Timestamp of last successful sync",
		},
		[]string{"namespace", "name"},
	)

	// Resource metrics
	GitOpsManagedResourcesTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitops_managed_resources_total",
			Help: "Total number of resources managed by GitOps",
		},
		[]string{"namespace", "name", "resource_type"},
	)

	GitOpsResourceErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_resource_errors_total",
			Help: "Total number of resource errors",
		},
		[]string{"namespace", "name", "resource_type", "error_type"},
	)

	// Performance metrics
	GitOpsReconcileQueueLength = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitops_reconcile_queue_length",
			Help: "Current length of the reconciliation queue",
		},
		[]string{"controller"},
	)

	GitOpsReconcileDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_reconcile_duration_seconds",
			Help:    "Duration of reconciliation operations in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms to ~40s
		},
		[]string{"controller", "namespace", "name"},
	)

	// Approval metrics
	GitOpsApprovalRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_approval_requests_total",
			Help: "Total number of approval requests created",
		},
		[]string{"namespace", "type", "status"},
	)

	GitOpsApprovalResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_approval_response_time_seconds",
			Help:    "Time taken to respond to approval requests",
			Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1min to ~68hours
		},
		[]string{"namespace", "type"},
	)

	// Security metrics
	GitOpsSecurityViolations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_security_violations_total",
			Help: "Total number of security policy violations",
		},
		[]string{"namespace", "name", "violation_type"},
	)

	GitOpsWebhookAuthFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_webhook_auth_failures_total",
			Help: "Total number of webhook authentication failures",
		},
		[]string{"provider", "reason"},
	)
)

func init() {
	// Register custom metrics with controller-runtime metrics registry
	metrics.Registry.MustRegister(
		GitOpsDeploymentTotal,
		GitOpsSyncDuration,
		GitOpsSyncTotal,
		GitOpsSyncErrors,
		GitOpsDriftDetected,
		GitOpsDriftCorrected,
		GitOpsDriftSize,
		GitOpsRollbackTotal,
		GitOpsRollbackDuration,
		GitOpsHealthCheckFailures,
		GitOpsPromotionTotal,
		GitOpsPromotionDuration,
		GitOpsPromotionApprovalTime,
		GitOpsWebhookTotal,
		GitOpsWebhookLatency,
		GitOpsArgoCDAppTotal,
		GitOpsArgoCDSyncTotal,
		GitOpsFluxKustomizationTotal,
		GitOpsFluxReconcileTotal,
		GitOpsGitCloneTotal,
		GitOpsGitCloneDuration,
		GitOpsLastSyncTime,
		GitOpsManagedResourcesTotal,
		GitOpsResourceErrors,
		GitOpsReconcileQueueLength,
		GitOpsReconcileDuration,
		GitOpsApprovalRequestsTotal,
		GitOpsApprovalResponseTime,
		GitOpsSecurityViolations,
		GitOpsWebhookAuthFailures,
	)
}

// MetricsRecorder provides methods to record GitOps metrics
type MetricsRecorder struct{}

// NewMetricsRecorder creates a new MetricsRecorder
func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{}
}

// RecordSync records a sync operation
func (m *MetricsRecorder) RecordSync(namespace, name, provider, status string, duration float64) {
	GitOpsSyncTotal.WithLabelValues(namespace, name, provider, status).Inc()
	if duration > 0 {
		GitOpsSyncDuration.WithLabelValues(namespace, name, provider).Observe(duration)
	}
	if status == "success" {
		GitOpsLastSyncTime.WithLabelValues(namespace, name).SetToCurrentTime()
	}
}

// RecordSyncError records a sync error
func (m *MetricsRecorder) RecordSyncError(namespace, name, provider, errorType string) {
	GitOpsSyncErrors.WithLabelValues(namespace, name, provider, errorType).Inc()
}

// RecordDrift records drift detection
func (m *MetricsRecorder) RecordDrift(namespace, name, resourceType string, driftSize float64, corrected bool) {
	GitOpsDriftDetected.WithLabelValues(namespace, name, resourceType).Inc()
	GitOpsDriftSize.WithLabelValues(namespace, name).Observe(driftSize)
	if corrected {
		GitOpsDriftCorrected.WithLabelValues(namespace, name, resourceType).Inc()
	}
}

// RecordRollback records a rollback operation
func (m *MetricsRecorder) RecordRollback(namespace, name, reason string, duration float64) {
	GitOpsRollbackTotal.WithLabelValues(namespace, name, reason).Inc()
	if duration > 0 {
		GitOpsRollbackDuration.WithLabelValues(namespace, name).Observe(duration)
	}
}

// RecordHealthCheckFailure records a health check failure
func (m *MetricsRecorder) RecordHealthCheckFailure(namespace, name, checkType string) {
	GitOpsHealthCheckFailures.WithLabelValues(namespace, name, checkType).Inc()
}

// RecordPromotion records a promotion operation
func (m *MetricsRecorder) RecordPromotion(fromEnv, toEnv, strategy, status string, duration float64) {
	GitOpsPromotionTotal.WithLabelValues(fromEnv, toEnv, strategy, status).Inc()
	if duration > 0 {
		GitOpsPromotionDuration.WithLabelValues(fromEnv, toEnv, strategy).Observe(duration)
	}
}

// RecordApprovalTime records the time taken for approval
func (m *MetricsRecorder) RecordApprovalTime(fromEnv, toEnv string, duration float64) {
	GitOpsPromotionApprovalTime.WithLabelValues(fromEnv, toEnv).Observe(duration)
}

// RecordWebhook records a webhook event
func (m *MetricsRecorder) RecordWebhook(provider, eventType, status string, latency float64) {
	GitOpsWebhookTotal.WithLabelValues(provider, eventType, status).Inc()
	if latency > 0 {
		GitOpsWebhookLatency.WithLabelValues(provider, eventType).Observe(latency)
	}
}

// RecordArgoCD records ArgoCD-specific metrics
func (m *MetricsRecorder) RecordArgoCD(namespace, project, syncStatus string, appCount float64) {
	GitOpsArgoCDAppTotal.WithLabelValues(namespace, project, syncStatus).Set(appCount)
}

// RecordFlux records Flux-specific metrics
func (m *MetricsRecorder) RecordFlux(namespace string, ready bool, kustomizationCount float64) {
	readyStr := "false"
	if ready {
		readyStr = "true"
	}
	GitOpsFluxKustomizationTotal.WithLabelValues(namespace, readyStr).Set(kustomizationCount)
}

// RecordGitOperation records git operations
func (m *MetricsRecorder) RecordGitOperation(namespace, name, status string, duration float64) {
	GitOpsGitCloneTotal.WithLabelValues(namespace, name, status).Inc()
	if duration > 0 {
		GitOpsGitCloneDuration.WithLabelValues(namespace, name).Observe(duration)
	}
}

// RecordManagedResources records the number of managed resources
func (m *MetricsRecorder) RecordManagedResources(namespace, name, resourceType string, count float64) {
	GitOpsManagedResourcesTotal.WithLabelValues(namespace, name, resourceType).Set(count)
}

// RecordResourceError records a resource error
func (m *MetricsRecorder) RecordResourceError(namespace, name, resourceType, errorType string) {
	GitOpsResourceErrors.WithLabelValues(namespace, name, resourceType, errorType).Inc()
}

// RecordReconcileMetrics records reconciliation metrics
func (m *MetricsRecorder) RecordReconcileMetrics(controller, namespace, name string, duration float64, queueLength float64) {
	GitOpsReconcileDuration.WithLabelValues(controller, namespace, name).Observe(duration)
	GitOpsReconcileQueueLength.WithLabelValues(controller).Set(queueLength)
}

// RecordApprovalRequest records approval request metrics
func (m *MetricsRecorder) RecordApprovalRequest(namespace, reqType, status string) {
	GitOpsApprovalRequestsTotal.WithLabelValues(namespace, reqType, status).Inc()
}

// RecordApprovalResponseTime records approval response time
func (m *MetricsRecorder) RecordApprovalResponseTime(namespace, reqType string, duration float64) {
	GitOpsApprovalResponseTime.WithLabelValues(namespace, reqType).Observe(duration)
}

// RecordSecurityViolation records security violations
func (m *MetricsRecorder) RecordSecurityViolation(namespace, name, violationType string) {
	GitOpsSecurityViolations.WithLabelValues(namespace, name, violationType).Inc()
}

// RecordWebhookAuthFailure records webhook authentication failures
func (m *MetricsRecorder) RecordWebhookAuthFailure(provider, reason string) {
	GitOpsWebhookAuthFailures.WithLabelValues(provider, reason).Inc()
}

// UpdateDeploymentGauge updates the deployment gauge metrics
func (m *MetricsRecorder) UpdateDeploymentGauge(namespace, syncProvider, phase string, delta float64) {
	GitOpsDeploymentTotal.WithLabelValues(namespace, syncProvider, phase).Add(delta)
}
