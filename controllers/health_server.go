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
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HealthServer provides health check endpoints for the operator
type HealthServer struct {
	server          *http.Server
	ready           bool
	readyMu         sync.RWMutex
	lastHealthCheck time.Time
	healthMu        sync.RWMutex
	healthManager   *HealthCheckManager
}

// NewHealthServer creates a new health server
func NewHealthServer(port string, healthManager *HealthCheckManager) *HealthServer {
	hs := &HealthServer{
		ready:         false,
		healthManager: healthManager,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", hs.livenessHandler)
	mux.HandleFunc("/readyz", hs.readinessHandler)
	mux.HandleFunc("/metrics/health", hs.healthMetricsHandler)

	hs.server = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return hs
}

// Start starts the health server
func (hs *HealthServer) Start(ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Starting health server", "address", hs.server.Addr)

	go func() {
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "Failed to start health server")
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info("Shutting down health server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return hs.server.Shutdown(shutdownCtx)
}

// SetReady marks the operator as ready
func (hs *HealthServer) SetReady(ready bool) {
	hs.readyMu.Lock()
	defer hs.readyMu.Unlock()
	hs.ready = ready
}

// IsReady returns the ready state
func (hs *HealthServer) IsReady() bool {
	hs.readyMu.RLock()
	defer hs.readyMu.RUnlock()
	return hs.ready
}

// UpdateLastHealthCheck updates the last health check timestamp
func (hs *HealthServer) UpdateLastHealthCheck() {
	hs.healthMu.Lock()
	defer hs.healthMu.Unlock()
	hs.lastHealthCheck = time.Now()
}

// livenessHandler handles liveness probe requests
func (hs *HealthServer) livenessHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the operator is alive (basic check)
	// This could include checking if the reconciler goroutine is still running
	
	hs.healthMu.RLock()
	lastCheck := hs.lastHealthCheck
	hs.healthMu.RUnlock()

	// If no health check in the last 5 minutes, consider unhealthy
	if time.Since(lastCheck) > 5*time.Minute && !lastCheck.IsZero() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Liveness check failed: No recent health checks"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readinessHandler handles readiness probe requests
func (hs *HealthServer) readinessHandler(w http.ResponseWriter, r *http.Request) {
	if !hs.IsReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not ready"))
		return
	}

	// Additional readiness checks can be added here
	// For example, checking if the Kubernetes API is accessible

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// healthMetricsHandler provides detailed health metrics
func (hs *HealthServer) healthMetricsHandler(w http.ResponseWriter, r *http.Request) {
	componentHealth := hs.healthManager.GetComponentHealth()

	healthResponse := struct {
		Status     string                          `json:"status"`
		Components map[string]*ComponentHealth     `json:"components"`
		Timestamp  time.Time                       `json:"timestamp"`
	}{
		Status:     "healthy",
		Components: componentHealth,
		Timestamp:  time.Now(),
	}

	// Determine overall status
	for _, health := range componentHealth {
		if !health.Healthy {
			healthResponse.Status = "unhealthy"
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
		log.Log.Error(err, "Failed to encode health response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
