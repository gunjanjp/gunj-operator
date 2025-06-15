package autoscaling

import (
	"context"
	"testing"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHPAManager(t *testing.T) {
	// Create test platform
	platform := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	// Create test policy
	policy := ScalingPolicy{
		Name:                 "test-policy",
		Type:                 HorizontalScaling,
		MinReplicas:          2,
		MaxReplicas:          10,
		TargetCPUPercent:     int32Ptr(70),
		TargetMemoryPercent:  int32Ptr(80),
		CustomMetrics:        []CustomMetric{},
		ScaleDownStabilizationWindow: &[]time.Duration{5 * time.Minute}[0],
	}
	
	// Create fake client
	scheme := runtime.NewScheme()
	_ = autoscalingv2.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	
	// Create HPA manager
	hpaManager := NewHPAManager(fakeClient)
	
	ctx := context.Background()
	
	t.Run("Create HPA", func(t *testing.T) {
		err := hpaManager.CreateOrUpdateHPA(ctx, platform, v1beta1.ComponentPrometheus, policy)
		require.NoError(t, err)
		
		// Verify HPA was created
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-platform-prometheus-hpa",
			Namespace: "test-namespace",
		}, hpa)
		require.NoError(t, err)
		
		assert.Equal(t, int32(2), *hpa.Spec.MinReplicas)
		assert.Equal(t, int32(10), hpa.Spec.MaxReplicas)
		assert.Len(t, hpa.Spec.Metrics, 2) // CPU and Memory
	})
	
	t.Run("Update HPA", func(t *testing.T) {
		// Update policy
		policy.MaxReplicas = 15
		policy.TargetCPUPercent = int32Ptr(60)
		
		err := hpaManager.CreateOrUpdateHPA(ctx, platform, v1beta1.ComponentPrometheus, policy)
		require.NoError(t, err)
		
		// Verify HPA was updated
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-platform-prometheus-hpa",
			Namespace: "test-namespace",
		}, hpa)
		require.NoError(t, err)
		
		assert.Equal(t, int32(15), hpa.Spec.MaxReplicas)
		assert.Equal(t, int32(60), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	})
	
	t.Run("Delete HPA", func(t *testing.T) {
		err := hpaManager.DeleteHPA(ctx, platform, v1beta1.ComponentPrometheus)
		require.NoError(t, err)
		
		// Verify HPA was deleted
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-platform-prometheus-hpa",
			Namespace: "test-namespace",
		}, hpa)
		assert.True(t, client.IgnoreNotFound(err) == nil)
	})
}

func TestVPAManager(t *testing.T) {
	// Create test platform
	platform := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	// Create test policy
	policy := ScalingPolicy{
		Name:           "test-vpa-policy",
		Type:           VerticalScaling,
		ResourceBuffer: 20,
	}
	
	// Create fake client
	scheme := runtime.NewScheme()
	_ = vpav1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	
	// Create VPA manager
	vpaManager := NewVPAManager(fakeClient)
	
	ctx := context.Background()
	
	t.Run("Create VPA", func(t *testing.T) {
		err := vpaManager.CreateOrUpdateVPA(ctx, platform, v1beta1.ComponentGrafana, policy)
		require.NoError(t, err)
		
		// Verify VPA was created
		vpa := &vpav1.VerticalPodAutoscaler{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "test-platform-grafana-vpa",
			Namespace: "test-namespace",
		}, vpa)
		require.NoError(t, err)
		
		assert.Equal(t, "grafana-deployment", vpa.Spec.TargetRef.Name)
		assert.Equal(t, vpav1.UpdateModeOff, *vpa.Spec.UpdatePolicy.UpdateMode)
		assert.Len(t, vpa.Spec.ResourcePolicy.ContainerPolicies, 1)
	})
	
	t.Run("Get Resource Bounds", func(t *testing.T) {
		vpaManager := NewVPAManager(nil)
		
		minAllowed, maxAllowed := vpaManager.getResourceBounds(v1beta1.ComponentPrometheus, policy)
		
		assert.Equal(t, "100m", minAllowed[corev1.ResourceCPU].String())
		assert.Equal(t, "512Mi", minAllowed[corev1.ResourceMemory].String())
		
		// With 20% buffer
		expectedMaxCPU := resource.MustParse("4.8")
		expectedMaxMemory := resource.MustParse("19456Mi") // ~19Gi
		
		assert.True(t, maxAllowed[corev1.ResourceCPU].Equal(expectedMaxCPU))
		assert.True(t, maxAllowed[corev1.ResourceMemory].Equal(expectedMaxMemory) || 
			       maxAllowed[corev1.ResourceMemory].Equal(resource.MustParse("19.2Gi")))
	})
}

func TestPredictiveEngine(t *testing.T) {
	ctx := context.Background()
	
	t.Run("Linear Model Training", func(t *testing.T) {
		engine := NewPredictiveScalingEngine("linear")
		
		// Generate test data
		baseTime := time.Now().Add(-24 * time.Hour)
		data := []MetricDataPoint{}
		
		for i := 0; i < 100; i++ {
			data = append(data, MetricDataPoint{
				Timestamp: baseTime.Add(time.Duration(i) * 15 * time.Minute),
				Value:     float64(i) * 0.5 + 10, // Linear growth
			})
		}
		
		err := engine.Train(ctx, data)
		require.NoError(t, err)
		
		assert.True(t, engine.GetAccuracy() > 0.9)
		assert.Equal(t, "linear", engine.GetModelType())
	})
	
	t.Run("Prediction", func(t *testing.T) {
		engine := NewPredictiveScalingEngine("linear")
		
		// Generate training data
		baseTime := time.Now().Add(-24 * time.Hour)
		data := []MetricDataPoint{}
		
		for i := 0; i < 100; i++ {
			data = append(data, MetricDataPoint{
				Timestamp: baseTime.Add(time.Duration(i) * 15 * time.Minute),
				Value:     float64(i) * 0.5 + 10,
			})
		}
		
		err := engine.Train(ctx, data)
		require.NoError(t, err)
		
		// Make prediction
		predictions, err := engine.Predict(ctx, 1*time.Hour)
		require.NoError(t, err)
		
		assert.NotEmpty(t, predictions)
		assert.True(t, predictions[0].Value > data[len(data)-1].Value) // Should predict growth
	})
	
	t.Run("Scaling Decision", func(t *testing.T) {
		engine := NewPredictiveScalingEngine("linear")
		
		// Train with increasing load data
		baseTime := time.Now().Add(-24 * time.Hour)
		data := []MetricDataPoint{}
		
		for i := 0; i < 100; i++ {
			data = append(data, MetricDataPoint{
				Timestamp: baseTime.Add(time.Duration(i) * 15 * time.Minute),
				Value:     float64(i) * 10, // Rapid growth
			})
		}
		
		err := engine.Train(ctx, data)
		require.NoError(t, err)
		
		// Make scaling decision
		decision, err := engine.MakePrediction(ctx, v1beta1.ComponentPrometheus, 3, 1*time.Hour)
		require.NoError(t, err)
		
		assert.Equal(t, PredictiveScaling, decision.Type)
		assert.True(t, decision.TargetReplicas > 3) // Should scale up
		assert.Contains(t, decision.Reason, "Predicted load")
	})
}

func TestCostProvider(t *testing.T) {
	ctx := context.Background()
	
	t.Run("AWS Cost Calculation", func(t *testing.T) {
		provider := NewCostProvider("aws", "us-east-1")
		
		resources := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		}
		
		cost, err := provider.GetCurrentCost(ctx, v1beta1.ComponentPrometheus, 3, resources)
		require.NoError(t, err)
		
		// 2 CPU * $0.0464/hour * 3 replicas * 24 * 30 = ~$200
		// 8GB * $0.0058/hour * 3 replicas * 24 * 30 = ~$100
		// Plus storage costs
		assert.True(t, cost > 300 && cost < 500)
	})
	
	t.Run("Optimal Configuration", func(t *testing.T) {
		provider := NewCostProvider("aws", "us-east-1")
		
		decision, err := provider.GetOptimalConfiguration(ctx, v1beta1.ComponentGrafana, 5, 150.0)
		require.NoError(t, err)
		
		assert.NotNil(t, decision)
		assert.Equal(t, CostAwareScaling, decision.Type)
		assert.True(t, decision.CostImpact.ProjectedCost <= 150.0)
	})
	
	t.Run("Cost Trend Analysis", func(t *testing.T) {
		provider := NewCostProvider("aws", "us-east-1")
		
		history := []ScalingEvent{
			{
				Timestamp:    time.Now().Add(-24 * time.Hour),
				ToReplicas:   2,
				CostImpact:   &CostImpact{ProjectedCost: 100},
			},
			{
				Timestamp:    time.Now().Add(-12 * time.Hour),
				ToReplicas:   4,
				CostImpact:   &CostImpact{ProjectedCost: 200},
			},
			{
				Timestamp:    time.Now(),
				ToReplicas:   3,
				CostImpact:   &CostImpact{ProjectedCost: 150},
			},
		}
		
		analysis, err := provider.AnalyzeCostTrends(ctx, v1beta1.ComponentPrometheus, history)
		require.NoError(t, err)
		
		assert.Equal(t, float64(150), analysis.AverageCost)
		assert.Equal(t, "increasing", analysis.Trend)
		assert.NotEmpty(t, analysis.Recommendations)
	})
}

func TestEventTracker(t *testing.T) {
	ctx := context.Background()
	
	// Create fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	
	storage := NewInMemoryEventStorage()
	tracker := NewEventTracker(fakeClient, nil, storage)
	
	platform := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	t.Run("Record Event", func(t *testing.T) {
		event := ScalingEvent{
			Type:            HorizontalScaling,
			Component:       v1beta1.ComponentPrometheus,
			Timestamp:       time.Now(),
			FromReplicas:    2,
			ToReplicas:      4,
			Reason:          "High CPU usage",
			Success:         true,
			Duration:        30 * time.Second,
		}
		
		err := tracker.RecordScalingEvent(ctx, platform, event)
		require.NoError(t, err)
		
		// Verify event was stored
		events, err := tracker.GetScalingHistory(ctx, v1beta1.ComponentPrometheus, 1*time.Hour)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "High CPU usage", events[0].Reason)
	})
	
	t.Run("Generate Report", func(t *testing.T) {
		// Add more events
		events := []ScalingEvent{
			{
				Type:         HorizontalScaling,
				Component:    v1beta1.ComponentPrometheus,
				Timestamp:    time.Now().Add(-30 * time.Minute),
				FromReplicas: 2,
				ToReplicas:   3,
				Success:      true,
				Duration:     20 * time.Second,
				CostImpact:   &CostImpact{MonthlySavings: -50},
			},
			{
				Type:         HorizontalScaling,
				Component:    v1beta1.ComponentGrafana,
				Timestamp:    time.Now().Add(-15 * time.Minute),
				FromReplicas: 1,
				ToReplicas:   2,
				Success:      true,
				Duration:     15 * time.Second,
				CostImpact:   &CostImpact{MonthlySavings: -30},
			},
			{
				Type:         VerticalScaling,
				Component:    v1beta1.ComponentLoki,
				Timestamp:    time.Now().Add(-5 * time.Minute),
				Success:      false,
				Error:        "Insufficient resources",
				Duration:     5 * time.Second,
			},
		}
		
		for _, event := range events {
			_ = tracker.RecordScalingEvent(ctx, platform, event)
		}
		
		report, err := tracker.GenerateReport(ctx, platform, 1*time.Hour)
		require.NoError(t, err)
		
		assert.Equal(t, "test-platform", report.PlatformName)
		assert.True(t, report.TotalEvents >= 3)
		assert.NotNil(t, report.Components[v1beta1.ComponentPrometheus])
		assert.NotNil(t, report.Components[v1beta1.ComponentGrafana])
	})
}

func TestWorkloadDrainer(t *testing.T) {
	ctx := context.Background()
	
	// Create fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	
	config := WorkloadDrainConfig{
		MaxUnavailable: 1,
		DrainTimeout:   2 * time.Minute,
		GracePeriod:    30 * time.Second,
	}
	
	drainer := NewWorkloadDrainer(fakeClient, nil, config)
	
	platform := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	t.Run("Sort Pods By Age", func(t *testing.T) {
		now := time.Now()
		pods := []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-1",
					CreationTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-2",
					CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod-3",
					CreationTimestamp: metav1.Time{Time: now},
				},
			},
		}
		
		sorted := drainer.sortPodsByAge(pods)
		
		assert.Equal(t, "pod-3", sorted[0].Name) // Newest first
		assert.Equal(t, "pod-1", sorted[1].Name)
		assert.Equal(t, "pod-2", sorted[2].Name) // Oldest last
	})
	
	t.Run("Can Drain Pod", func(t *testing.T) {
		// Without PDB, should allow draining
		canDrain := drainer.canDrainPod(ctx, platform, v1beta1.ComponentPrometheus, 5)
		assert.True(t, canDrain)
		
		// With only 1 replica left, should not drain
		canDrain = drainer.canDrainPod(ctx, platform, v1beta1.ComponentPrometheus, 1)
		assert.False(t, canDrain)
	})
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}
