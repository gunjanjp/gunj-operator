/*
Copyright 2025.

Licensed under the MIT License.
*/

package managers

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gunjanjp/gunj-operator/internal/managers/cost"
	"github.com/gunjanjp/gunj-operator/internal/managers/grafana"
	"github.com/gunjanjp/gunj-operator/internal/managers/loki"
	"github.com/gunjanjp/gunj-operator/internal/managers/prometheus"
	"github.com/gunjanjp/gunj-operator/internal/managers/tempo"
	"github.com/gunjanjp/gunj-operator/internal/version"
)

// ManagerMode defines the mode for component managers
type ManagerMode string

const (
	// ManagerModeNative uses native Kubernetes resource management
	ManagerModeNative ManagerMode = "native"
	
	// ManagerModeHelm uses Helm for component management
	ManagerModeHelm ManagerMode = "helm"
	
	// Environment variable to control manager mode
	EnvManagerMode = "GUNJ_MANAGER_MODE"
)

// DefaultManagerFactory is the default implementation of ManagerFactory
type DefaultManagerFactory struct {
	client         client.Client
	scheme         *runtime.Scheme
	restConfig     *rest.Config
	mode           ManagerMode
	versionManager *version.Manager
}

// NewDefaultManagerFactory creates a new default manager factory
func NewDefaultManagerFactory(client client.Client, scheme *runtime.Scheme) ManagerFactory {
	return &DefaultManagerFactory{
		client: client,
		scheme: scheme,
		mode:   getManagerMode(),
	}
}

// NewDefaultManagerFactoryWithConfig creates a new default manager factory with REST config
func NewDefaultManagerFactoryWithConfig(client client.Client, scheme *runtime.Scheme, restConfig *rest.Config) ManagerFactory {
	return &DefaultManagerFactory{
		client:     client,
		scheme:     scheme,
		restConfig: restConfig,
		mode:       getManagerMode(),
	}
}

// SetMode sets the manager mode
func (f *DefaultManagerFactory) SetMode(mode ManagerMode) {
	f.mode = mode
}

// SetVersionManager sets the version manager
func (f *DefaultManagerFactory) SetVersionManager(vm *version.Manager) {
	f.versionManager = vm
}

// CreatePrometheusManager creates a new Prometheus manager
func (f *DefaultManagerFactory) CreatePrometheusManager() PrometheusManager {
	switch f.mode {
	case ManagerModeHelm:
		if f.restConfig == nil {
			// Fallback to native if no REST config
			return prometheus.NewPrometheusManager(f.client, f.scheme)
		}
		
		manager, err := prometheus.NewPrometheusManagerHelm(f.client, f.scheme, f.restConfig)
		if err != nil {
			// Log error and fallback to native
			fmt.Printf("Failed to create Helm-based Prometheus manager: %v, falling back to native\n", err)
			return prometheus.NewPrometheusManager(f.client, f.scheme)
		}
		return manager
		
	default:
		return prometheus.NewPrometheusManager(f.client, f.scheme)
	}
}

// CreateGrafanaManager creates a new Grafana manager
func (f *DefaultManagerFactory) CreateGrafanaManager() GrafanaManager {
	switch f.mode {
	case ManagerModeHelm:
		if f.restConfig == nil {
			// Fallback to native if no REST config
			return grafana.NewGrafanaManager(f.client, f.scheme)
		}
		
		manager, err := grafana.NewGrafanaManagerHelm(f.client, f.scheme, f.restConfig)
		if err != nil {
			// Log error and fallback to native
			fmt.Printf("Failed to create Helm-based Grafana manager: %v, falling back to native\n", err)
			return grafana.NewGrafanaManager(f.client, f.scheme)
		}
		return manager
		
	default:
		return grafana.NewGrafanaManager(f.client, f.scheme)
	}
}

// CreateLokiManager creates a new Loki manager
func (f *DefaultManagerFactory) CreateLokiManager() LokiManager {
	switch f.mode {
	case ManagerModeHelm:
		if f.restConfig == nil {
			// Fallback to native if no REST config
			return loki.NewLokiManager(f.client, f.scheme)
		}
		
		manager, err := loki.NewLokiManagerHelm(f.client, f.scheme, f.restConfig)
		if err != nil {
			// Log error and fallback to native
			fmt.Printf("Failed to create Helm-based Loki manager: %v, falling back to native\n", err)
			return loki.NewLokiManager(f.client, f.scheme)
		}
		return manager
		
	default:
		return loki.NewLokiManager(f.client, f.scheme)
	}
}

// CreateTempoManager creates a new Tempo manager
func (f *DefaultManagerFactory) CreateTempoManager() TempoManager {
	switch f.mode {
	case ManagerModeHelm:
		if f.restConfig == nil {
			// Fallback to native if no REST config
			return tempo.NewTempoManager(f.client, f.scheme)
		}
		
		manager, err := tempo.NewTempoManagerHelm(f.client, f.scheme, f.restConfig)
		if err != nil {
			// Log error and fallback to native
			fmt.Printf("Failed to create Helm-based Tempo manager: %v, falling back to native\n", err)
			return tempo.NewTempoManager(f.client, f.scheme)
		}
		return manager
		
	default:
		return tempo.NewTempoManager(f.client, f.scheme)
	}
}

// CreateCostManager creates a new Cost manager
func (f *DefaultManagerFactory) CreateCostManager() CostManager {
	// Cost manager doesn't use Helm, always native
	return cost.NewManager(f.client, f.scheme, ctrl.Log.WithName("cost-manager"))
}

// getManagerMode returns the manager mode from environment or default
func getManagerMode() ManagerMode {
	mode := os.Getenv(EnvManagerMode)
	switch ManagerMode(mode) {
	case ManagerModeHelm:
		return ManagerModeHelm
	case ManagerModeNative:
		return ManagerModeNative
	default:
		// Default to Helm mode for new installations
		return ManagerModeHelm
	}
}

// ManagerFactoryConfig provides configuration for the manager factory
type ManagerFactoryConfig struct {
	Client         client.Client
	Scheme         *runtime.Scheme
	RestConfig     *rest.Config
	Mode           ManagerMode
	VersionManager *version.Manager
}

// NewManagerFactory creates a new manager factory with configuration
func NewManagerFactory(config ManagerFactoryConfig) ManagerFactory {
	mode := config.Mode
	if mode == "" {
		mode = getManagerMode()
	}
	
	return &DefaultManagerFactory{
		client:         config.Client,
		scheme:         config.Scheme,
		restConfig:     config.RestConfig,
		mode:           mode,
		versionManager: config.VersionManager,
	}
}
