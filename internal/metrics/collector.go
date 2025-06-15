/*
Copyright 2025.

Licensed under the MIT License.
*/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Collector handles metrics collection for the operator
type Collector struct {
	reconcileTotal    *prometheus.CounterVec
	reconcileErrors   *prometheus.CounterVec
	reconcileDuration *prometheus.HistogramVec
	platformsTotal    *prometheus.GaugeVec
	componentStatus   *prometheus.GaugeVec
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	collector := &Collector{
		reconcileTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_reconcile_total",
				Help: "Total number of reconciliations per controller",
			},
			[]string{"controller"},
		),
		reconcileErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_reconcile_errors_total",
				Help: "Total number of reconciliation errors per controller",
			},
			[]string{"controller"},
		),
		reconcileDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gunj_operator_reconcile_duration_seconds",
				Help:    "Time taken for reconciliations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"controller"},
		),
		platformsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_operator_platforms_total",
				Help: "Total number of platforms by phase",
			},
			[]string{"phase"},
		),
		componentStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_operator_component_status",
				Help: "Component status (1 = ready, 0 = not ready)",
			},
			[]string{"platform", "namespace", "component"},
		),
	}

	// Register metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		collector.reconcileTotal,
		collector.reconcileErrors,
		collector.reconcileDuration,
		collector.platformsTotal,
		collector.componentStatus,
	)

	return collector
}

// RecordReconciliation records a successful reconciliation
func (c *Collector) RecordReconciliation(controller string, duration float64) {
	c.reconcileTotal.WithLabelValues(controller).Inc()
	c.reconcileDuration.WithLabelValues(controller).Observe(duration)
}

// RecordReconciliationError records a reconciliation error
func (c *Collector) RecordReconciliationError(controller string) {
	c.reconcileErrors.WithLabelValues(controller).Inc()
}

// RecordPlatformStatus records the status of a platform
func (c *Collector) RecordPlatformStatus(name, namespace, phase string) {
	// This would typically query all platforms and update the gauge
	// For now, we'll just increment/decrement based on phase changes
	c.platformsTotal.WithLabelValues(phase).Inc()
}

// RecordComponentStatus records the status of a component
func (c *Collector) RecordComponentStatus(platform, namespace, component string, ready bool) {
	value := 0.0
	if ready {
		value = 1.0
	}
	c.componentStatus.WithLabelValues(platform, namespace, component).Set(value)
}

// RecordPlatformCount updates the total count of platforms by phase
func (c *Collector) RecordPlatformCount(counts map[string]int) {
	for phase, count := range counts {
		c.platformsTotal.WithLabelValues(phase).Set(float64(count))
	}
}
