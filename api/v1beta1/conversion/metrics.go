/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// ConversionMetrics provides Prometheus metrics for conversion operations
type ConversionMetrics struct {
	// Total number of conversions
	conversionsTotal *prometheus.CounterVec
	
	// Conversion duration
	conversionDuration *prometheus.HistogramVec
	
	// Field validation errors
	fieldValidationErrors *prometheus.CounterVec
	
	// Data loss occurrences
	dataLossOccurrences *prometheus.CounterVec
	
	// Deprecated field usage
	deprecatedFieldUsage *prometheus.CounterVec
	
	// Enhanced field conversions
	enhancedFieldConversions *prometheus.CounterVec
	
	// Conversion success rate
	conversionSuccessRate *prometheus.GaugeVec
	
	// Active conversions
	activeConversions *prometheus.GaugeVec
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Track success rates
	successCounts map[string]int
	totalCounts   map[string]int
}

var (
	// Global metrics instance
	metricsInstance *ConversionMetrics
	metricsOnce     sync.Once
)

// GetMetrics returns the global metrics instance
func GetMetrics() *ConversionMetrics {
	metricsOnce.Do(func() {
		metricsInstance = newConversionMetrics()
		metricsInstance.register()
	})
	return metricsInstance
}

// newConversionMetrics creates a new metrics instance
func newConversionMetrics() *ConversionMetrics {
	return &ConversionMetrics{
		conversionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_conversions_total",
				Help: "Total number of API conversions performed",
			},
			[]string{"source_version", "target_version", "resource_type", "result"},
		),
		
		conversionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gunj_operator_conversion_duration_seconds",
				Help:    "Duration of API conversion operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"source_version", "target_version", "resource_type"},
		),
		
		fieldValidationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_field_validation_errors_total",
				Help: "Total number of field validation errors during conversion",
			},
			[]string{"source_version", "target_version", "field_path", "error_type"},
		),
		
		dataLossOccurrences: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_data_loss_occurrences_total",
				Help: "Total number of data loss occurrences during conversion",
			},
			[]string{"source_version", "target_version", "field_path"},
		),
		
		deprecatedFieldUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_deprecated_field_usage_total",
				Help: "Usage count of deprecated fields during conversion",
			},
			[]string{"field_path", "deprecation_version"},
		),
		
		enhancedFieldConversions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gunj_operator_enhanced_field_conversions_total",
				Help: "Number of field enhancements applied during conversion",
			},
			[]string{"source_version", "target_version", "field_path", "enhancement_type"},
		),
		
		conversionSuccessRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_operator_conversion_success_rate",
				Help: "Success rate of conversions (0-1)",
			},
			[]string{"source_version", "target_version"},
		),
		
		activeConversions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gunj_operator_active_conversions",
				Help: "Number of currently active conversions",
			},
			[]string{"source_version", "target_version"},
		),
		
		successCounts: make(map[string]int),
		totalCounts:   make(map[string]int),
	}
}

// register registers all metrics with the controller-runtime metrics registry
func (m *ConversionMetrics) register() {
	metrics.Registry.MustRegister(
		m.conversionsTotal,
		m.conversionDuration,
		m.fieldValidationErrors,
		m.dataLossOccurrences,
		m.deprecatedFieldUsage,
		m.enhancedFieldConversions,
		m.conversionSuccessRate,
		m.activeConversions,
	)
}

// RecordConversion records a conversion operation
func (m *ConversionMetrics) RecordConversion(sourceVersion, targetVersion, resourceType string, duration time.Duration, success bool) {
	result := "success"
	if !success {
		result = "failure"
	}
	
	// Record total conversions
	m.conversionsTotal.WithLabelValues(sourceVersion, targetVersion, resourceType, result).Inc()
	
	// Record duration
	m.conversionDuration.WithLabelValues(sourceVersion, targetVersion, resourceType).Observe(duration.Seconds())
	
	// Update success rate
	m.updateSuccessRate(sourceVersion, targetVersion, success)
}

// RecordFieldValidationError records a field validation error
func (m *ConversionMetrics) RecordFieldValidationError(sourceVersion, targetVersion, fieldPath, errorType string) {
	m.fieldValidationErrors.WithLabelValues(sourceVersion, targetVersion, fieldPath, errorType).Inc()
}

// RecordDataLoss records a data loss occurrence
func (m *ConversionMetrics) RecordDataLoss(sourceVersion, targetVersion, fieldPath string) {
	m.dataLossOccurrences.WithLabelValues(sourceVersion, targetVersion, fieldPath).Inc()
}

// RecordDeprecatedFieldUsage records usage of a deprecated field
func (m *ConversionMetrics) RecordDeprecatedFieldUsage(fieldPath, deprecationVersion string) {
	m.deprecatedFieldUsage.WithLabelValues(fieldPath, deprecationVersion).Inc()
}

// RecordEnhancedField records a field enhancement
func (m *ConversionMetrics) RecordEnhancedField(sourceVersion, targetVersion, fieldPath, enhancementType string) {
	m.enhancedFieldConversions.WithLabelValues(sourceVersion, targetVersion, fieldPath, enhancementType).Inc()
}

// StartConversion marks the start of a conversion
func (m *ConversionMetrics) StartConversion(sourceVersion, targetVersion string) {
	m.activeConversions.WithLabelValues(sourceVersion, targetVersion).Inc()
}

// EndConversion marks the end of a conversion
func (m *ConversionMetrics) EndConversion(sourceVersion, targetVersion string) {
	m.activeConversions.WithLabelValues(sourceVersion, targetVersion).Dec()
}

// updateSuccessRate updates the success rate gauge
func (m *ConversionMetrics) updateSuccessRate(sourceVersion, targetVersion string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := sourceVersion + "->" + targetVersion
	
	m.totalCounts[key]++
	if success {
		m.successCounts[key]++
	}
	
	// Calculate and update success rate
	rate := float64(m.successCounts[key]) / float64(m.totalCounts[key])
	m.conversionSuccessRate.WithLabelValues(sourceVersion, targetVersion).Set(rate)
}

// ConversionTimer provides a convenient way to time conversions
type ConversionTimer struct {
	metrics       *ConversionMetrics
	sourceVersion string
	targetVersion string
	resourceType  string
	startTime     time.Time
}

// StartTimer creates and starts a new conversion timer
func (m *ConversionMetrics) StartTimer(sourceVersion, targetVersion, resourceType string) *ConversionTimer {
	m.StartConversion(sourceVersion, targetVersion)
	
	return &ConversionTimer{
		metrics:       m,
		sourceVersion: sourceVersion,
		targetVersion: targetVersion,
		resourceType:  resourceType,
		startTime:     time.Now(),
	}
}

// Complete completes the timer and records the metrics
func (t *ConversionTimer) Complete(success bool) {
	duration := time.Since(t.startTime)
	t.metrics.RecordConversion(t.sourceVersion, t.targetVersion, t.resourceType, duration, success)
	t.metrics.EndConversion(t.sourceVersion, t.targetVersion)
}

// MetricsReporter provides a way to report validation results as metrics
type MetricsReporter struct {
	metrics       *ConversionMetrics
	sourceVersion string
	targetVersion string
}

// NewMetricsReporter creates a new metrics reporter
func NewMetricsReporter(sourceVersion, targetVersion string) *MetricsReporter {
	return &MetricsReporter{
		metrics:       GetMetrics(),
		sourceVersion: sourceVersion,
		targetVersion: targetVersion,
	}
}

// ReportValidationResult reports validation results as metrics
func (r *MetricsReporter) ReportValidationResult(result *ValidationResult) {
	// Report field validation errors
	for _, err := range result.Errors {
		r.metrics.RecordFieldValidationError(
			r.sourceVersion,
			r.targetVersion,
			err.Field,
			string(err.Type),
		)
	}
	
	// Report data loss
	if result.Metrics.DataLossFields > 0 {
		// In real implementation, track specific fields
		r.metrics.RecordDataLoss(r.sourceVersion, r.targetVersion, "multiple_fields")
	}
	
	// Report deprecated field usage
	if result.Metrics.DeprecatedFields > 0 {
		// In real implementation, track specific fields
		r.metrics.RecordDeprecatedFieldUsage("multiple_fields", "v1.0.0")
	}
	
	// Report enhanced fields
	if result.Metrics.EnhancedFields > 0 {
		// In real implementation, track specific fields
		r.metrics.RecordEnhancedField(
			r.sourceVersion,
			r.targetVersion,
			"multiple_fields",
			"feature_addition",
		)
	}
}

// GetConversionMetricsSummary returns a summary of conversion metrics
func GetConversionMetricsSummary() map[string]interface{} {
	metrics := GetMetrics()
	
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()
	
	summary := make(map[string]interface{})
	
	// Add success rates
	successRates := make(map[string]float64)
	for key, total := range metrics.totalCounts {
		if total > 0 {
			successRates[key] = float64(metrics.successCounts[key]) / float64(total)
		}
	}
	summary["success_rates"] = successRates
	
	// Add totals
	summary["total_conversions"] = metrics.totalCounts
	
	return summary
}
