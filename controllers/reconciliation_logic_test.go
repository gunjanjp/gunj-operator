/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/managers"
	"github.com/gunjanjp/gunj-operator/internal/metrics"
)

func TestReconciliationLogic(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1beta1.AddToScheme(scheme)

	// Create test platform
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Prometheus: &observabilityv1beta1.PrometheusSpec{
					Enabled: true,
					Version: "v2.48.0",
					Resources: observabilityv1beta1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    "100m",
							corev1.ResourceMemory: "256Mi",
						},
					},
				},
				Grafana: &observabilityv1beta1.GrafanaSpec{
					Enabled: true,
					Version: "10.2.0",
				},
				Loki: &observabilityv1beta1.LokiSpec{
					Enabled: true,
					Version: "2.9.0",
				},
				Tempo: &observabilityv1beta1.TempoSpec{
					Enabled: true,
					Version: "2.3.0",
				},
			},
			Global: observabilityv1beta1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster":     "production",
					"environment": "prod",
				},
				LogLevel: observabilityv1beta1.LogLevelInfo,
				Retention: observabilityv1beta1.RetentionConfig{
					Days: 30,
					Size: "100Gi",
				},
			},
		},
	}

	// Create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: platform.Namespace,
		},
	}

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(namespace, platform).
		WithStatusSubresource(&observabilityv1beta1.ObservabilityPlatform{}).
		Build()

	// Track component reconciliation order
	reconcileOrder := []string{}

	// Create reconciler with mock managers
	reconciler := &ObservabilityPlatformReconciler{
		Client: client,
		Scheme: scheme,
		Log:    zap.New(zap.UseDevMode(true)),
		PrometheusManager: &managers.MockPrometheusManager{
			ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
				reconcileOrder = append(reconcileOrder, "prometheus")
				return nil
			},
		},
		GrafanaManager: &managers.MockGrafanaManager{
			ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
				reconcileOrder = append(reconcileOrder, "grafana")
				return nil
			},
		},
		LokiManager: &managers.MockLokiManager{
			ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
				reconcileOrder = append(reconcileOrder, "loki")
				return nil
			},
		},
		TempoManager: &managers.MockTempoManager{
			ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
				reconcileOrder = append(reconcileOrder, "tempo")
				return nil
			},
		},
		Metrics: metrics.NewCollector(),
	}

	// Test reconciliation
	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      platform.Name,
			Namespace: platform.Namespace,
		},
	}

	// Run reconciliation
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconciliation failed: %v", err)
	}

	// Verify result
	if result.RequeueAfter == 0 {
		t.Error("Expected requeue after duration to be set")
	}

	// Verify dependency order
	t.Logf("Component reconciliation order: %v", reconcileOrder)

	// Prometheus, Loki, and Tempo should be reconciled before Grafana
	grafanaIndex := -1
	for i, comp := range reconcileOrder {
		if comp == "grafana" {
			grafanaIndex = i
			break
		}
	}

	if grafanaIndex == -1 {
		t.Error("Grafana was not reconciled")
	}

	// Check that data sources were reconciled before Grafana
	foundPrometheus := false
	foundLoki := false
	foundTempo := false
	for i := 0; i < grafanaIndex; i++ {
		switch reconcileOrder[i] {
		case "prometheus":
			foundPrometheus = true
		case "loki":
			foundLoki = true
		case "tempo":
			foundTempo = true
		}
	}

	if !foundPrometheus || !foundLoki || !foundTempo {
		t.Errorf("Not all data sources were reconciled before Grafana. Order: %v", reconcileOrder)
	}

	// Test configuration management
	state := NewReconciliationState(platform)
	
	// Test Prometheus configuration
	promConfig := state.ConfigManager.BuildConfiguration("prometheus")
	t.Logf("Prometheus configuration: %+v", promConfig)
	
	// Verify global config is included
	globalConfig, ok := promConfig["global"].(map[string]interface{})
	if !ok {
		t.Error("Global configuration not found in Prometheus config")
	} else {
		if globalConfig["clusterName"] != platform.Name {
			t.Errorf("Expected cluster name %s, got %v", platform.Name, globalConfig["clusterName"])
		}
	}

	// Test Grafana configuration
	grafanaConfig := state.ConfigManager.BuildConfiguration("grafana")
	t.Logf("Grafana configuration: %+v", grafanaConfig)
	
	// Verify data sources are configured
	if grafanaSpec, ok := grafanaConfig["grafana"].(map[string]interface{}); ok {
		if dataSources, ok := grafanaSpec["dataSources"].([]map[string]interface{}); ok {
			t.Logf("Grafana data sources: %d configured", len(dataSources))
			for _, ds := range dataSources {
				t.Logf("  - %s: %s", ds["name"], ds["url"])
			}
		}
	}

	// Test dependency resolver
	enabledComponents := map[string]bool{
		"prometheus": true,
		"grafana":    true,
		"loki":       true,
		"tempo":      true,
	}
	order := state.DependencyResolver.GetReconciliationOrder(enabledComponents)
	t.Logf("Dependency resolution order: %v", order)

	// Test common resources
	if err := reconciler.reconcileCommonResources(ctx, platform); err != nil {
		t.Fatalf("Failed to reconcile common resources: %v", err)
	}

	// Verify RBAC was created
	sa := &corev1.ServiceAccount{}
	saName := types.NamespacedName{
		Name:      platform.Name + "-observability",
		Namespace: platform.Namespace,
	}
	if err := client.Get(ctx, saName, sa); err != nil {
		t.Errorf("ServiceAccount not created: %v", err)
	} else {
		t.Logf("ServiceAccount created: %s", sa.Name)
	}

	// Verify shared ConfigMap was created
	cm := &corev1.ConfigMap{}
	cmName := types.NamespacedName{
		Name:      platform.Name + "-global-config",
		Namespace: platform.Namespace,
	}
	if err := client.Get(ctx, cmName, cm); err != nil {
		t.Errorf("Global ConfigMap not created: %v", err)
	} else {
		t.Logf("Global ConfigMap created with %d entries", len(cm.Data))
		for k, v := range cm.Data {
			t.Logf("  %s: %s", k, v)
		}
	}

	// Verify shared Secret was created
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      platform.Name + "-admin-credentials",
		Namespace: platform.Namespace,
	}
	if err := client.Get(ctx, secretName, secret); err != nil {
		t.Errorf("Admin credentials secret not created: %v", err)
	} else {
		t.Logf("Admin credentials secret created")
	}

	t.Log("Reconciliation logic test completed successfully!")
}

// TestComponentStatusTracking tests the component status tracking
func TestComponentStatusTracking(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1beta1.AddToScheme(scheme)

	// Create test platform
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Prometheus: &observabilityv1beta1.PrometheusSpec{Enabled: true},
			},
		},
	}

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(platform).
		WithStatusSubresource(&observabilityv1beta1.ObservabilityPlatform{}).
		Build()

	// Create reconciler
	reconciler := &ObservabilityPlatformReconciler{
		Client: client,
		Scheme: scheme,
	}

	ctx := context.Background()

	// Test updating component status
	status := observabilityv1beta1.ComponentStatus{
		Phase:   observabilityv1beta1.ComponentPhaseReady,
		Message: "Prometheus is ready",
		Ready:   true,
	}

	err := reconciler.UpdateComponentStatus(ctx, platform, "prometheus", status)
	if err != nil {
		t.Fatalf("Failed to update component status: %v", err)
	}

	// Retrieve updated platform
	updated := &observabilityv1beta1.ObservabilityPlatform{}
	err = client.Get(ctx, types.NamespacedName{
		Name:      platform.Name,
		Namespace: platform.Namespace,
	}, updated)
	if err != nil {
		t.Fatalf("Failed to get updated platform: %v", err)
	}

	// Verify component status
	if promStatus, ok := updated.Status.ComponentStatuses["prometheus"]; ok {
		if promStatus.Phase != observabilityv1beta1.ComponentPhaseReady {
			t.Errorf("Expected prometheus phase to be Ready, got %s", promStatus.Phase)
		}
		t.Logf("Prometheus status: %s - %s", promStatus.Phase, promStatus.Message)
	} else {
		t.Error("Prometheus status not found")
	}

	// Verify overall platform status
	if updated.Status.Phase != observabilityv1beta1.PhaseReady {
		t.Errorf("Expected platform phase to be Ready, got %s", updated.Status.Phase)
	}

	t.Log("Component status tracking test completed successfully!")
}
