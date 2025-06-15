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
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Name           string
	Healthy        bool
	LastChecked    time.Time
	Message        string
	AvailableReplicas int32
	DesiredReplicas   int32
}

// HealthCheckManager manages health checks for observability platform components
type HealthCheckManager struct {
	client         client.Client
	checkInterval  time.Duration
	timeout        time.Duration
	healthMu       sync.RWMutex
	componentHealth map[string]*ComponentHealth
	httpClient     *http.Client
}

// NewHealthCheckManager creates a new health check manager
func NewHealthCheckManager(client client.Client) *HealthCheckManager {
	return &HealthCheckManager{
		client:          client,
		checkInterval:   30 * time.Second,
		timeout:         10 * time.Second,
		componentHealth: make(map[string]*ComponentHealth),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckComponentHealth checks the health of all components for a platform
func (m *HealthCheckManager) CheckComponentHealth(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*observabilityv1beta1.HealthStatus, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Checking component health", "platform", platform.Name)

	healthStatus := &observabilityv1beta1.HealthStatus{
		Healthy:     true,
		LastChecked: metav1.Now(),
		Components:  make(map[string]observabilityv1beta1.ComponentHealthStatus),
	}

	// Check each enabled component
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		prometheusHealth := m.checkPrometheusHealth(ctx, platform)
		healthStatus.Components["prometheus"] = prometheusHealth
		if !prometheusHealth.Healthy {
			healthStatus.Healthy = false
		}
	}

	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		grafanaHealth := m.checkGrafanaHealth(ctx, platform)
		healthStatus.Components["grafana"] = grafanaHealth
		if !grafanaHealth.Healthy {
			healthStatus.Healthy = false
		}
	}

	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		lokiHealth := m.checkLokiHealth(ctx, platform)
		healthStatus.Components["loki"] = lokiHealth
		if !lokiHealth.Healthy {
			healthStatus.Healthy = false
		}
	}

	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		tempoHealth := m.checkTempoHealth(ctx, platform)
		healthStatus.Components["tempo"] = tempoHealth
		if !tempoHealth.Healthy {
			healthStatus.Healthy = false
		}
	}

	// Update overall health message
	if healthStatus.Healthy {
		healthStatus.Message = "All components are healthy"
	} else {
		unhealthyCount := 0
		for _, comp := range healthStatus.Components {
			if !comp.Healthy {
				unhealthyCount++
			}
		}
		healthStatus.Message = fmt.Sprintf("%d component(s) unhealthy", unhealthyCount)
	}

	return healthStatus, nil
}

// checkPrometheusHealth checks the health of Prometheus
func (m *HealthCheckManager) checkPrometheusHealth(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) observabilityv1beta1.ComponentHealthStatus {
	log := log.FromContext(ctx)
	componentName := fmt.Sprintf("%s-prometheus", platform.Name)

	// Check deployment status
	deployment := &appsv1.Deployment{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      componentName,
		Namespace: platform.Namespace,
	}, deployment)

	if err != nil {
		if errors.IsNotFound(err) {
			return observabilityv1beta1.ComponentHealthStatus{
				Healthy:        false,
				Message:        "Deployment not found",
				LastChecked:    metav1.Now(),
			}
		}
		log.Error(err, "Failed to get Prometheus deployment")
		return observabilityv1beta1.ComponentHealthStatus{
			Healthy:        false,
			Message:        fmt.Sprintf("Failed to check deployment: %v", err),
			LastChecked:    metav1.Now(),
		}
	}

	// Check if deployment is ready
	isReady := deployment.Status.ReadyReplicas >= deployment.Status.Replicas && deployment.Status.Replicas > 0

	// Check service endpoint
	service := &corev1.Service{}
	err = m.client.Get(ctx, types.NamespacedName{
		Name:      componentName,
		Namespace: platform.Namespace,
	}, service)

	if err != nil {
		log.Error(err, "Failed to get Prometheus service")
		return observabilityv1beta1.ComponentHealthStatus{
			Healthy:           false,
			Message:           fmt.Sprintf("Service check failed: %v", err),
			LastChecked:       metav1.Now(),
			AvailableReplicas: deployment.Status.ReadyReplicas,
			DesiredReplicas:   deployment.Status.Replicas,
		}
	}

	// Check Prometheus API health endpoint
	healthEndpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:9090/-/healthy", componentName, platform.Namespace)
	healthy, message := m.checkHTTPEndpoint(ctx, healthEndpoint)

	return observabilityv1beta1.ComponentHealthStatus{
		Healthy:           isReady && healthy,
		Message:           message,
		LastChecked:       metav1.Now(),
		AvailableReplicas: deployment.Status.ReadyReplicas,
		DesiredReplicas:   deployment.Status.Replicas,
	}
}

// checkGrafanaHealth checks the health of Grafana
func (m *HealthCheckManager) checkGrafanaHealth(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) observabilityv1beta1.ComponentHealthStatus {
	log := log.FromContext(ctx)
	componentName := fmt.Sprintf("%s-grafana", platform.Name)

	// Check deployment status
	deployment := &appsv1.Deployment{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      componentName,
		Namespace: platform.Namespace,
	}, deployment)

	if err != nil {
		if errors.IsNotFound(err) {
			return observabilityv1beta1.ComponentHealthStatus{
				Healthy:        false,
				Message:        "Deployment not found",
				LastChecked:    metav1.Now(),
			}
		}
		log.Error(err, "Failed to get Grafana deployment")
		return observabilityv1beta1.ComponentHealthStatus{
			Healthy:        false,
			Message:        fmt.Sprintf("Failed to check deployment: %v", err),
			LastChecked:    metav1.Now(),
		}
	}

	// Check if deployment is ready
	isReady := deployment.Status.ReadyReplicas >= deployment.Status.Replicas && deployment.Status.Replicas > 0

	// Check Grafana API health endpoint
	healthEndpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:3000/api/health", componentName, platform.Namespace)
	healthy, message := m.checkHTTPEndpoint(ctx, healthEndpoint)

	return observabilityv1beta1.ComponentHealthStatus{
		Healthy:           isReady && healthy,
		Message:           message,
		LastChecked:       metav1.Now(),
		AvailableReplicas: deployment.Status.ReadyReplicas,
		DesiredReplicas:   deployment.Status.Replicas,
	}
}

// checkLokiHealth checks the health of Loki
func (m *HealthCheckManager) checkLokiHealth(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) observabilityv1beta1.ComponentHealthStatus {
	log := log.FromContext(ctx)
	componentName := fmt.Sprintf("%s-loki", platform.Name)

	// Check statefulset status for Loki
	statefulSet := &appsv1.StatefulSet{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      componentName,
		Namespace: platform.Namespace,
	}, statefulSet)

	if err != nil {
		if errors.IsNotFound(err) {
			return observabilityv1beta1.ComponentHealthStatus{
				Healthy:        false,
				Message:        "StatefulSet not found",
				LastChecked:    metav1.Now(),
			}
		}
		log.Error(err, "Failed to get Loki statefulset")
		return observabilityv1beta1.ComponentHealthStatus{
			Healthy:        false,
			Message:        fmt.Sprintf("Failed to check statefulset: %v", err),
			LastChecked:    metav1.Now(),
		}
	}

	// Check if statefulset is ready
	isReady := statefulSet.Status.ReadyReplicas >= statefulSet.Status.Replicas && statefulSet.Status.Replicas > 0

	// Check Loki ready endpoint
	healthEndpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:3100/ready", componentName, platform.Namespace)
	healthy, message := m.checkHTTPEndpoint(ctx, healthEndpoint)

	return observabilityv1beta1.ComponentHealthStatus{
		Healthy:           isReady && healthy,
		Message:           message,
		LastChecked:       metav1.Now(),
		AvailableReplicas: statefulSet.Status.ReadyReplicas,
		DesiredReplicas:   statefulSet.Status.Replicas,
	}
}

// checkTempoHealth checks the health of Tempo
func (m *HealthCheckManager) checkTempoHealth(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) observabilityv1beta1.ComponentHealthStatus {
	log := log.FromContext(ctx)
	componentName := fmt.Sprintf("%s-tempo", platform.Name)

	// Check deployment status
	deployment := &appsv1.Deployment{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      componentName,
		Namespace: platform.Namespace,
	}, deployment)

	if err != nil {
		if errors.IsNotFound(err) {
			return observabilityv1beta1.ComponentHealthStatus{
				Healthy:        false,
				Message:        "Deployment not found",
				LastChecked:    metav1.Now(),
			}
		}
		log.Error(err, "Failed to get Tempo deployment")
		return observabilityv1beta1.ComponentHealthStatus{
			Healthy:        false,
			Message:        fmt.Sprintf("Failed to check deployment: %v", err),
			LastChecked:    metav1.Now(),
		}
	}

	// Check if deployment is ready
	isReady := deployment.Status.ReadyReplicas >= deployment.Status.Replicas && deployment.Status.Replicas > 0

	// Check Tempo ready endpoint
	healthEndpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:3200/ready", componentName, platform.Namespace)
	healthy, message := m.checkHTTPEndpoint(ctx, healthEndpoint)

	return observabilityv1beta1.ComponentHealthStatus{
		Healthy:           isReady && healthy,
		Message:           message,
		LastChecked:       metav1.Now(),
		AvailableReplicas: deployment.Status.ReadyReplicas,
		DesiredReplicas:   deployment.Status.Replicas,
	}
}

// checkHTTPEndpoint performs an HTTP health check on the given endpoint
func (m *HealthCheckManager) checkHTTPEndpoint(ctx context.Context, endpoint string) (bool, string) {
	log := log.FromContext(ctx)
	
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		log.V(2).Info("HTTP health check failed", "endpoint", endpoint, "error", err)
		return false, fmt.Sprintf("HTTP check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, "Healthy"
	}

	return false, fmt.Sprintf("Unhealthy: HTTP status %d", resp.StatusCode)
}

// StartPeriodicHealthChecks starts periodic health checks for a platform
func (m *HealthCheckManager) StartPeriodicHealthChecks(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) {
	log := log.FromContext(ctx)
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.V(1).Info("Stopping periodic health checks", "platform", platform.Name)
			return
		case <-ticker.C:
			health, err := m.CheckComponentHealth(ctx, platform)
			if err != nil {
				log.Error(err, "Failed to check component health", "platform", platform.Name)
				continue
			}

			// Store health status for metrics
			m.healthMu.Lock()
			for name, status := range health.Components {
				m.componentHealth[fmt.Sprintf("%s-%s", platform.Name, name)] = &ComponentHealth{
					Name:              name,
					Healthy:           status.Healthy,
					LastChecked:       status.LastChecked.Time,
					Message:           status.Message,
					AvailableReplicas: status.AvailableReplicas,
					DesiredReplicas:   status.DesiredReplicas,
				}
			}
			m.healthMu.Unlock()
		}
	}
}

// GetComponentHealth returns the current health status for all components
func (m *HealthCheckManager) GetComponentHealth() map[string]*ComponentHealth {
	m.healthMu.RLock()
	defer m.healthMu.RUnlock()
	
	// Create a copy to avoid race conditions
	healthCopy := make(map[string]*ComponentHealth)
	for k, v := range m.componentHealth {
		healthCopy[k] = &ComponentHealth{
			Name:              v.Name,
			Healthy:           v.Healthy,
			LastChecked:       v.LastChecked,
			Message:           v.Message,
			AvailableReplicas: v.AvailableReplicas,
			DesiredReplicas:   v.DesiredReplicas,
		}
	}
	
	return healthCopy
}
