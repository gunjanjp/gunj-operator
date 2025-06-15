/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Component health metric - 1 for healthy, 0 for unhealthy
	componentHealthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_health",
			Help: "Health status of observability platform components (1 = healthy, 0 = unhealthy)",
		},
		[]string{"platform", "namespace", "component"},
	)

	// Component ready replicas metric
	componentReadyReplicasGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_ready_replicas",
			Help: "Number of ready replicas for observability platform components",
		},
		[]string{"platform", "namespace", "component"},
	)

	// Component desired replicas metric
	componentDesiredReplicasGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_component_desired_replicas",
			Help: "Number of desired replicas for observability platform components",
		},
		[]string{"platform", "namespace", "component"},
	)

	// Health check duration metric
	healthCheckDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gunj_operator_health_check_duration_seconds",
			Help:    "Duration of health checks in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"platform", "namespace"},
	)

	// Health check errors counter
	healthCheckErrorsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gunj_operator_health_check_errors_total",
			Help: "Total number of health check errors",
		},
		[]string{"platform", "namespace", "component", "error_type"},
	)

	// Last health check timestamp
	lastHealthCheckTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_last_health_check_timestamp",
			Help: "Timestamp of the last health check (Unix time)",
		},
		[]string{"platform", "namespace", "component"},
	)
)

func init() {
	// Register metrics
	metrics.Registry.MustRegister(
		componentHealthGauge,
		componentReadyReplicasGauge,
		componentDesiredReplicasGauge,
		healthCheckDurationHistogram,
		healthCheckErrorsCounter,
		lastHealthCheckTimestamp,
	)
}

// UpdateHealthMetrics updates Prometheus metrics based on health check results
func UpdateHealthMetrics(platform, namespace string, healthStatus map[string]*ComponentHealth) {
	for componentName, health := range healthStatus {
		// Update health gauge
		healthValue := 0.0
		if health.Healthy {
			healthValue = 1.0
		}
		componentHealthGauge.WithLabelValues(platform, namespace, componentName).Set(healthValue)

		// Update replica metrics
		componentReadyReplicasGauge.WithLabelValues(platform, namespace, componentName).Set(float64(health.AvailableReplicas))
		componentDesiredReplicasGauge.WithLabelValues(platform, namespace, componentName).Set(float64(health.DesiredReplicas))

		// Update last check timestamp
		lastHealthCheckTimestamp.WithLabelValues(platform, namespace, componentName).Set(float64(health.LastChecked.Unix()))
	}
}

// RecordHealthCheckDuration records the duration of a health check
func RecordHealthCheckDuration(platform, namespace string, duration float64) {
	healthCheckDurationHistogram.WithLabelValues(platform, namespace).Observe(duration)
}

// RecordHealthCheckError increments the health check error counter
func RecordHealthCheckError(platform, namespace, component, errorType string) {
	healthCheckErrorsCounter.WithLabelValues(platform, namespace, component, errorType).Inc()
}
