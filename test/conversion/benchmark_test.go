/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"encoding/json"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// BenchmarkV1Alpha1ToV1Beta1Conversion benchmarks conversion from v1alpha1 to v1beta1
func BenchmarkV1Alpha1ToV1Beta1Conversion(b *testing.B) {
	benchmarks := []struct {
		name     string
		platform *v1alpha1.ObservabilityPlatform
	}{
		{
			name:     "minimal",
			platform: createMinimalV1Alpha1Platform(),
		},
		{
			name:     "typical",
			platform: createTypicalV1Alpha1Platform(),
		},
		{
			name:     "complex",
			platform: createComplexV1Alpha1Platform(),
		},
		{
			name:     "large",
			platform: createLargeV1Alpha1Platform(),
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				_ = bm.platform.ConvertTo(v1beta1Platform)
			}
		})
	}
}

// BenchmarkV1Beta1ToV1Alpha1Conversion benchmarks conversion from v1beta1 to v1alpha1
func BenchmarkV1Beta1ToV1Alpha1Conversion(b *testing.B) {
	benchmarks := []struct {
		name     string
		platform *v1beta1.ObservabilityPlatform
	}{
		{
			name:     "minimal",
			platform: createMinimalV1Beta1Platform(),
		},
		{
			name:     "typical",
			platform: createTypicalV1Beta1Platform(),
		},
		{
			name:     "complex",
			platform: createComplexV1Beta1Platform(),
		},
		{
			name:     "withLostFields",
			platform: createV1Beta1PlatformWithFieldsToLose(),
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
				_ = v1alpha1Platform.ConvertFrom(bm.platform)
			}
		})
	}
}

// BenchmarkRoundTripConversion benchmarks round-trip conversion
func BenchmarkRoundTripConversion(b *testing.B) {
	platforms := []struct {
		name     string
		platform *v1alpha1.ObservabilityPlatform
	}{
		{
			name:     "minimal",
			platform: createMinimalV1Alpha1Platform(),
		},
		{
			name:     "typical",
			platform: createTypicalV1Alpha1Platform(),
		},
		{
			name:     "complex",
			platform: createComplexV1Alpha1Platform(),
		},
	}

	for _, p := range platforms {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Convert to v1beta1
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				_ = p.platform.ConvertTo(v1beta1Platform)

				// Convert back to v1alpha1
				roundtrip := &v1alpha1.ObservabilityPlatform{}
				_ = roundtrip.ConvertFrom(v1beta1Platform)
			}
		})
	}
}

// BenchmarkJSONSerialization benchmarks JSON serialization of converted objects
func BenchmarkJSONSerialization(b *testing.B) {
	// Create and convert a complex platform
	v1alpha1Platform := createComplexV1Alpha1Platform()
	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	_ = v1alpha1Platform.ConvertTo(v1beta1Platform)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(v1beta1Platform)
		}
	})

	// Marshal once for unmarshal benchmark
	data, _ := json.Marshal(v1beta1Platform)

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var platform v1beta1.ObservabilityPlatform
			_ = json.Unmarshal(data, &platform)
		}
	})

	b.Run("marshal-unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			data, _ := json.Marshal(v1beta1Platform)
			var platform v1beta1.ObservabilityPlatform
			_ = json.Unmarshal(data, &platform)
		}
	})
}

// BenchmarkResourceConversion benchmarks resource quantity conversion
func BenchmarkResourceConversion(b *testing.B) {
	resources := v1alpha1.ResourceRequirements{
		Requests: v1alpha1.ResourceList{
			CPU:    "100m",
			Memory: "128Mi",
		},
		Limits: v1alpha1.ResourceList{
			CPU:    "2000m",
			Memory: "8Gi",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Convert to Kubernetes resources
		requests := make(map[string]resource.Quantity)
		if resources.Requests.CPU != "" {
			requests["cpu"] = resource.MustParse(resources.Requests.CPU)
		}
		if resources.Requests.Memory != "" {
			requests["memory"] = resource.MustParse(resources.Requests.Memory)
		}

		limits := make(map[string]resource.Quantity)
		if resources.Limits.CPU != "" {
			limits["cpu"] = resource.MustParse(resources.Limits.CPU)
		}
		if resources.Limits.Memory != "" {
			limits["memory"] = resource.MustParse(resources.Limits.Memory)
		}
	}
}

// BenchmarkBatchConversion benchmarks converting multiple objects
func BenchmarkBatchConversion(b *testing.B) {
	sizes := []int{1, 10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("batch-%d", size), func(b *testing.B) {
			// Create batch of platforms
			platforms := make([]*v1alpha1.ObservabilityPlatform, size)
			for i := 0; i < size; i++ {
				platforms[i] = createTypicalV1Alpha1Platform()
				platforms[i].Name = fmt.Sprintf("platform-%d", i)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Convert all platforms
				for _, platform := range platforms {
					v1beta1Platform := &v1beta1.ObservabilityPlatform{}
					_ = platform.ConvertTo(v1beta1Platform)
				}
			}
		})
	}
}

// BenchmarkStatusConversion benchmarks status conversion
func BenchmarkStatusConversion(b *testing.B) {
	status := v1alpha1.ObservabilityPlatformStatus{
		Phase:              "Ready",
		ObservedGeneration: 10,
		Message:            "All components are healthy",
		ComponentStatus: map[string]v1alpha1.ComponentStatus{
			"prometheus": {
				Phase:         "Ready",
				Version:       "v2.48.0",
				ReadyReplicas: 3,
				TotalReplicas: 3,
			},
			"grafana": {
				Phase:         "Ready",
				Version:       "10.2.0",
				ReadyReplicas: 2,
				TotalReplicas: 2,
			},
			"loki": {
				Phase:         "Ready",
				Version:       "2.9.0",
				ReadyReplicas: 1,
				TotalReplicas: 1,
			},
			"tempo": {
				Phase:         "Ready",
				Version:       "2.3.0",
				ReadyReplicas: 1,
				TotalReplicas: 1,
			},
		},
		Conditions: []metav1.Condition{
			{
				Type:    "Ready",
				Status:  metav1.ConditionTrue,
				Reason:  "AllComponentsReady",
				Message: "All observability components are ready",
			},
			{
				Type:    "Progressing",
				Status:  metav1.ConditionFalse,
				Reason:  "Stable",
				Message: "No changes in progress",
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Convert status
		v1beta1Status := v1beta1.ObservabilityPlatformStatus{
			Phase:              status.Phase,
			ObservedGeneration: status.ObservedGeneration,
			Message:            status.Message,
			Conditions:         status.Conditions,
		}

		// Convert component status
		if len(status.ComponentStatus) > 0 {
			v1beta1Status.ComponentStatus = make(map[string]v1beta1.ComponentStatus)
			for name, cs := range status.ComponentStatus {
				v1beta1Status.ComponentStatus[name] = v1beta1.ComponentStatus{
					Phase:    cs.Phase,
					Version:  cs.Version,
					Ready:    cs.ReadyReplicas,
					Replicas: cs.TotalReplicas,
					Message:  cs.Message,
				}
			}
		}
	}
}

// BenchmarkMapConversion benchmarks map conversion operations
func BenchmarkMapConversion(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("map-%d", size), func(b *testing.B) {
			// Create a map with the specified size
			labels := make(map[string]string, size)
			for i := 0; i < size; i++ {
				labels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Copy map (simulating what happens during conversion)
				newLabels := make(map[string]string, len(labels))
				for k, v := range labels {
					newLabels[k] = v
				}
			}
		})
	}
}

// Helper functions to create test platforms

func createMinimalV1Alpha1Platform() *v1alpha1.ObservabilityPlatform {
	return &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minimal",
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
}

func createTypicalV1Alpha1Platform() *v1alpha1.ObservabilityPlatform {
	return &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "typical",
			Namespace: "monitoring",
			Labels: map[string]string{
				"app":         "observability",
				"environment": "production",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "1",
							Memory: "4Gi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "2",
							Memory: "8Gi",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "100Gi",
						StorageClassName: "fast-ssd",
					},
					Retention: "30d",
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Enabled:  true,
					Version:  "10.2.0",
					Replicas: 2,
				},
			},
			Global: v1alpha1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster": "production",
					"region":  "us-east-1",
				},
				LogLevel: "info",
			},
		},
	}
}

func createComplexV1Alpha1Platform() *v1alpha1.ObservabilityPlatform {
	return &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex",
			Namespace: "monitoring",
			Labels: map[string]string{
				"app":         "observability",
				"environment": "production",
				"team":        "platform",
				"cost-center": "engineering",
			},
			Annotations: map[string]string{
				"description": "Production observability platform",
				"owner":       "platform-team",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Paused: false,
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "2",
							Memory: "8Gi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "4",
							Memory: "16Gi",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "500Gi",
						StorageClassName: "fast-ssd",
					},
					Retention: "90d",
					RemoteWrite: []v1alpha1.RemoteWriteSpec{
						{
							URL:           "https://remote1.example.com/write",
							RemoteTimeout: "30s",
						},
						{
							URL:           "https://remote2.example.com/write",
							RemoteTimeout: "30s",
						},
					},
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Enabled:       true,
					Version:       "10.2.0",
					Replicas:      3,
					AdminPassword: "admin123",
					Ingress: &v1alpha1.IngressConfig{
						Enabled:   true,
						ClassName: "nginx",
						Host:      "grafana.example.com",
					},
					DataSources: []v1alpha1.DataSourceConfig{
						{
							Name:      "Prometheus",
							Type:      "prometheus",
							URL:       "http://prometheus:9090",
							IsDefault: true,
						},
						{
							Name: "Loki",
							Type: "loki",
							URL:  "http://loki:3100",
						},
					},
				},
				Loki: &v1alpha1.LokiSpec{
					Enabled:   true,
					Version:   "2.9.0",
					Retention: "30d",
					Storage: &v1alpha1.StorageConfig{
						Size:             "1Ti",
						StorageClassName: "standard",
					},
					S3: &v1alpha1.S3Config{
						Enabled:    true,
						BucketName: "loki-logs",
						Region:     "us-east-1",
					},
				},
				Tempo: &v1alpha1.TempoSpec{
					Enabled:   true,
					Version:   "2.3.0",
					Retention: "7d",
					Storage: &v1alpha1.StorageConfig{
						Size:             "100Gi",
						StorageClassName: "fast-ssd",
					},
				},
			},
			Global: v1alpha1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster":     "production",
					"region":      "us-east-1",
					"environment": "prod",
					"datacenter":  "dc1",
				},
				LogLevel: "info",
				NodeSelector: map[string]string{
					"node-role": "observability",
					"zone":      "us-east-1a",
				},
				Tolerations: []v1alpha1.Toleration{
					{
						Key:      "observability",
						Operator: "Equal",
						Value:    "true",
						Effect:   "NoSchedule",
					},
				},
			},
			HighAvailability: &v1alpha1.HighAvailabilityConfig{
				Enabled:     true,
				MinReplicas: 3,
			},
			Backup: &v1alpha1.BackupConfig{
				Enabled:       true,
				Schedule:      "0 2 * * *",
				RetentionDays: 30,
			},
			Alerting: &v1alpha1.AlertingConfig{
				AlertManager: &v1alpha1.AlertManagerSpec{
					Enabled:  true,
					Replicas: 3,
				},
			},
		},
		Status: v1alpha1.ObservabilityPlatformStatus{
			Phase:              "Ready",
			ObservedGeneration: 5,
			ComponentStatus: map[string]v1alpha1.ComponentStatus{
				"prometheus": {Phase: "Ready", Version: "v2.48.0", ReadyReplicas: 3},
				"grafana":    {Phase: "Ready", Version: "10.2.0", ReadyReplicas: 3},
				"loki":       {Phase: "Ready", Version: "2.9.0", ReadyReplicas: 1},
				"tempo":      {Phase: "Ready", Version: "2.3.0", ReadyReplicas: 1},
			},
		},
	}
}

func createLargeV1Alpha1Platform() *v1alpha1.ObservabilityPlatform {
	platform := createComplexV1Alpha1Platform()
	platform.Name = "large"

	// Add lots of labels
	for i := 0; i < 100; i++ {
		platform.Labels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	// Add lots of annotations
	for i := 0; i < 100; i++ {
		platform.Annotations[fmt.Sprintf("annotation-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	// Add lots of remote writes
	for i := 0; i < 20; i++ {
		platform.Spec.Components.Prometheus.RemoteWrite = append(
			platform.Spec.Components.Prometheus.RemoteWrite,
			v1alpha1.RemoteWriteSpec{
				URL: fmt.Sprintf("https://remote%d.example.com/write", i),
			},
		)
	}

	// Add lots of datasources
	for i := 0; i < 20; i++ {
		platform.Spec.Components.Grafana.DataSources = append(
			platform.Spec.Components.Grafana.DataSources,
			v1alpha1.DataSourceConfig{
				Name: fmt.Sprintf("DataSource-%d", i),
				Type: "prometheus",
				URL:  fmt.Sprintf("http://prometheus-%d:9090", i),
			},
		)
	}

	return platform
}

func createMinimalV1Beta1Platform() *v1beta1.ObservabilityPlatform {
	return &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minimal",
			Namespace: "default",
		},
		Spec: v1beta1.ObservabilityPlatformSpec{
			Components: v1beta1.Components{
				Prometheus: &v1beta1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
}

func createTypicalV1Beta1Platform() *v1beta1.ObservabilityPlatform {
	return &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "typical",
			Namespace: "monitoring",
			Labels: map[string]string{
				"app":         "observability",
				"environment": "production",
			},
		},
		Spec: v1beta1.ObservabilityPlatformSpec{
			Components: v1beta1.Components{
				Prometheus: &v1beta1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
				},
				Grafana: &v1beta1.GrafanaSpec{
					Enabled:  true,
					Version:  "10.2.0",
					Replicas: 2,
				},
			},
			Global: v1beta1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster": "production",
					"region":  "us-east-1",
				},
				LogLevel: "info",
			},
		},
	}
}

func createComplexV1Beta1Platform() *v1beta1.ObservabilityPlatform {
	return &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex",
			Namespace: "monitoring",
		},
		Spec: v1beta1.ObservabilityPlatformSpec{
			Components: v1beta1.Components{
				Prometheus: &v1beta1.PrometheusSpec{
					Enabled:        true,
					Version:        "v2.48.0",
					Replicas:       3,
					ExternalLabels: map[string]string{"env": "prod"},
				},
				Grafana: &v1beta1.GrafanaSpec{
					Enabled: true,
					Version: "10.2.0",
					Plugins: []string{"piechart-panel"},
				},
				Loki: &v1beta1.LokiSpec{
					Enabled:          true,
					Version:          "2.9.0",
					CompactorEnabled: true,
				},
				Tempo: &v1beta1.TempoSpec{
					Enabled:       true,
					Version:       "2.3.0",
					SearchEnabled: true,
				},
			},
			Security: &v1beta1.SecurityConfig{
				TLS: v1beta1.TLSConfig{
					Enabled: true,
				},
			},
		},
	}
}

func createV1Beta1PlatformWithFieldsToLose() *v1beta1.ObservabilityPlatform {
	platform := createComplexV1Beta1Platform()
	
	// Add fields that will be lost
	platform.Spec.Components.Prometheus.WALCompression = &[]bool{true}[0]
	platform.Spec.Components.Prometheus.EnableFeatures = []string{"exemplar-storage"}
	
	platform.Spec.Components.Grafana.SMTP = &v1beta1.SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	}
	
	platform.Spec.Components.Loki.IndexGateway = &v1beta1.IndexGatewayConfig{
		Enabled:  true,
		Replicas: 2,
	}
	
	platform.Spec.Components.Tempo.MetricsGenerator = &v1beta1.MetricsGeneratorConfig{
		Enabled: true,
	}
	
	return platform
}

// BenchmarkMemoryUsage measures memory allocations during conversion
func BenchmarkMemoryUsage(b *testing.B) {
	platforms := []struct {
		name     string
		platform *v1alpha1.ObservabilityPlatform
	}{
		{"minimal", createMinimalV1Alpha1Platform()},
		{"typical", createTypicalV1Alpha1Platform()},
		{"complex", createComplexV1Alpha1Platform()},
		{"large", createLargeV1Alpha1Platform()},
	}

	for _, p := range platforms {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			
			var m runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m)
			allocBefore := m.Alloc

			// Perform conversion
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			_ = p.platform.ConvertTo(v1beta1Platform)

			runtime.ReadMemStats(&m)
			allocAfter := m.Alloc

			b.ReportMetric(float64(allocAfter-allocBefore), "bytes/op")
		})
	}
}

// BenchmarkConversionThroughput measures conversions per second
func BenchmarkConversionThroughput(b *testing.B) {
	platform := createTypicalV1Alpha1Platform()
	
	b.ResetTimer()
	
	start := time.Now()
	for i := 0; i < b.N; i++ {
		v1beta1Platform := &v1beta1.ObservabilityPlatform{}
		_ = platform.ConvertTo(v1beta1Platform)
	}
	elapsed := time.Since(start)
	
	conversionsPerSecond := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(conversionsPerSecond, "conversions/sec")
}
