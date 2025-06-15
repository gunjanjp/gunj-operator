/*
Copyright 2025.

Licensed under the MIT License.
*/

package managers

import (
	"context"
	"fmt"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// MockPrometheusManager is a mock implementation of PrometheusManager for testing
type MockPrometheusManager struct {
	ReconcileFn            func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	DeleteFn               func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	GetStatusFn            func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error)
	ValidateFn             func(platform *observabilityv1beta1.ObservabilityPlatform) error
	UpdateConfigurationFn  func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	ScaleReplicasFn        func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, replicas int32) error
}

func (m *MockPrometheusManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockPrometheusManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, platform)
	}
	return nil
}

func (m *MockPrometheusManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	if m.GetStatusFn != nil {
		return m.GetStatusFn(ctx, platform)
	}
	return &observabilityv1beta1.ComponentStatus{Ready: true}, nil
}

func (m *MockPrometheusManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ValidateFn != nil {
		return m.ValidateFn(platform)
	}
	return nil
}

func (m *MockPrometheusManager) UpdateConfiguration(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.UpdateConfigurationFn != nil {
		return m.UpdateConfigurationFn(ctx, platform)
	}
	return nil
}

func (m *MockPrometheusManager) ScaleReplicas(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, replicas int32) error {
	if m.ScaleReplicasFn != nil {
		return m.ScaleReplicasFn(ctx, platform, replicas)
	}
	return nil
}

func (m *MockPrometheusManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockPrometheusManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://prometheus-%s.%s.svc.cluster.local:9090", platform.Name, platform.Namespace)
}

// MockGrafanaManager is a mock implementation of GrafanaManager for testing
type MockGrafanaManager struct {
	ReconcileFn             func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	DeleteFn                func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	GetStatusFn             func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error)
	ValidateFn              func(platform *observabilityv1beta1.ObservabilityPlatform) error
	ImportDashboardsFn      func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	ConfigureDataSourcesFn  func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
}

func (m *MockGrafanaManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockGrafanaManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, platform)
	}
	return nil
}

func (m *MockGrafanaManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	if m.GetStatusFn != nil {
		return m.GetStatusFn(ctx, platform)
	}
	return &observabilityv1beta1.ComponentStatus{Ready: true}, nil
}

func (m *MockGrafanaManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ValidateFn != nil {
		return m.ValidateFn(platform)
	}
	return nil
}

func (m *MockGrafanaManager) ImportDashboards(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ImportDashboardsFn != nil {
		return m.ImportDashboardsFn(ctx, platform)
	}
	return nil
}

func (m *MockGrafanaManager) ConfigureDataSources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ConfigureDataSourcesFn != nil {
		return m.ConfigureDataSourcesFn(ctx, platform)
	}
	return nil
}

func (m *MockGrafanaManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockGrafanaManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://grafana-%s.%s.svc.cluster.local:3000", platform.Name, platform.Namespace)
}

// MockLokiManager is a mock implementation of LokiManager for testing
type MockLokiManager struct {
	ReconcileFn        func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	DeleteFn           func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	GetStatusFn        func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error)
	ValidateFn         func(platform *observabilityv1beta1.ObservabilityPlatform) error
	ConfigureStorageFn func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	UpdateRetentionFn  func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
}

func (m *MockLokiManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockLokiManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, platform)
	}
	return nil
}

func (m *MockLokiManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	if m.GetStatusFn != nil {
		return m.GetStatusFn(ctx, platform)
	}
	return &observabilityv1beta1.ComponentStatus{Ready: true}, nil
}

func (m *MockLokiManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ValidateFn != nil {
		return m.ValidateFn(platform)
	}
	return nil
}

func (m *MockLokiManager) ConfigureStorage(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ConfigureStorageFn != nil {
		return m.ConfigureStorageFn(ctx, platform)
	}
	return nil
}

func (m *MockLokiManager) UpdateRetention(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.UpdateRetentionFn != nil {
		return m.UpdateRetentionFn(ctx, platform)
	}
	return nil
}

func (m *MockLokiManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockLokiManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://loki-%s.%s.svc.cluster.local:3100", platform.Name, platform.Namespace)
}

// MockTempoManager is a mock implementation of TempoManager for testing
type MockTempoManager struct {
	ReconcileFn          func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	DeleteFn             func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	GetStatusFn          func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error)
	ValidateFn           func(platform *observabilityv1beta1.ObservabilityPlatform) error
	ConfigureReceiversFn func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
	UpdateSamplingFn     func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error
}

func (m *MockTempoManager) Reconcile(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockTempoManager) Delete(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, platform)
	}
	return nil
}

func (m *MockTempoManager) GetStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.ComponentStatus, error) {
	if m.GetStatusFn != nil {
		return m.GetStatusFn(ctx, platform)
	}
	return &observabilityv1beta1.ComponentStatus{Ready: true}, nil
}

func (m *MockTempoManager) Validate(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ValidateFn != nil {
		return m.ValidateFn(platform)
	}
	return nil
}

func (m *MockTempoManager) ConfigureReceivers(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.ConfigureReceiversFn != nil {
		return m.ConfigureReceiversFn(ctx, platform)
	}
	return nil
}

func (m *MockTempoManager) UpdateSampling(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	if m.UpdateSamplingFn != nil {
		return m.UpdateSamplingFn(ctx, platform)
	}
	return nil
}

func (m *MockTempoManager) ReconcileWithConfig(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, config map[string]interface{}) error {
	if m.ReconcileFn != nil {
		return m.ReconcileFn(ctx, platform)
	}
	return nil
}

func (m *MockTempoManager) GetServiceURL(platform *observabilityv1beta1.ObservabilityPlatform) string {
	return fmt.Sprintf("http://tempo-%s.%s.svc.cluster.local:3200", platform.Name, platform.Namespace)
}
