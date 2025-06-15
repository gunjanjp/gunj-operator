/*
Copyright 2025.

Licensed under the MIT License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	observabilityv1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/controllers"
	"github.com/gunjanjp/gunj-operator/internal/managers"
	"github.com/gunjanjp/gunj-operator/internal/metrics"
	"github.com/gunjanjp/gunj-operator/internal/webhooks"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	// Version information
	version   = "unknown"
	gitCommit = "unknown"
	buildDate = "unknown"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(observabilityv1alpha1.AddToScheme(scheme))
	utilruntime.Must(observabilityv1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var webhookPort int
	var certDir string
	var maxConcurrentReconciles int
	var requeueDuration time.Duration
	var printVersion bool
	var namespace string
	var watchNamespace string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.IntVar(&webhookPort, "webhook-port", 9443, "The port the webhook server binds to.")
	flag.StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs", "The directory that contains the server key and certificate.")
	flag.IntVar(&maxConcurrentReconciles, "max-concurrent-reconciles", 3, "Maximum number of concurrent reconciles.")
	flag.DurationVar(&requeueDuration, "requeue-duration", 5*time.Minute, "Duration after which to requeue successful reconciliations.")
	flag.BoolVar(&printVersion, "version", false, "Print version information and exit.")
	flag.StringVar(&namespace, "namespace", "", "Namespace to watch for resources. If empty, all namespaces are watched.")
	flag.StringVar(&watchNamespace, "watch-namespace", "", "Namespace to watch for resources. If empty, all namespaces are watched.")

	opts := zap.Options{
		Development: true,
		TimeEncoder: zap.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Set up logging
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	klog.SetLogger(klogr.New())

	// Print version information
	if printVersion {
		fmt.Printf("gunj-operator version information:\n")
		fmt.Printf("  Version:    %s\n", version)
		fmt.Printf("  Git Commit: %s\n", gitCommit)
		fmt.Printf("  Build Date: %s\n", buildDate)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  Platform:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	setupLog.Info("Starting Gunj Operator",
		"version", version,
		"git-commit", gitCommit,
		"build-date", buildDate,
		"go-version", runtime.Version(),
	)

	// Get REST config
	restConfig := ctrl.GetConfigOrDie()

	// Set up manager options
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			CertDir: certDir,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "gunj-operator.observability.io",
		Cache:                  getCacheOptions(namespace, watchNamespace),
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Create manager factory with REST config for Helm support
	managerFactory := managers.NewDefaultManagerFactoryWithConfig(mgr.GetClient(), mgr.GetScheme(), restConfig)

	// Create component managers
	prometheusManager := managerFactory.CreatePrometheusManager()
	grafanaManager := managerFactory.CreateGrafanaManager()
	lokiManager := managerFactory.CreateLokiManager()
	tempoManager := managerFactory.CreateTempoManager()

	// Create the controller
	if err = (&controllers.ObservabilityPlatformReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  mgr.GetScheme(),
		RestConfig:              restConfig,
		Recorder:                mgr.GetEventRecorderFor("observabilityplatform-controller"),
		PrometheusManager:       prometheusManager,
		GrafanaManager:          grafanaManager,
		LokiManager:             lokiManager,
		TempoManager:            tempoManager,
		Metrics:                 metricsCollector,
		MaxConcurrentReconciles: maxConcurrentReconciles,
		RequeueDuration:         requeueDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ObservabilityPlatform")
		os.Exit(1)
	}

	// Set up webhooks
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&observabilityv1beta1.ObservabilityPlatform{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "ObservabilityPlatform")
			os.Exit(1)
		}
		
		// Set up conversion webhook
		if err = webhooks.SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create conversion webhook")
			os.Exit(1)
		}
		setupLog.Info("Conversion webhook enabled")
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start a goroutine to log memory stats periodically
	go logMemoryStats()

	setupLog.Info("starting manager")
	ctx := ctrl.SetupSignalHandler()
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getCacheOptions returns cache options based on namespace configuration
func getCacheOptions(namespace, watchNamespace string) cache.Options {
	opts := cache.Options{}

	// Determine which namespace to watch
	ns := namespace
	if watchNamespace != "" {
		ns = watchNamespace
	}

	if ns != "" {
		// Watch a specific namespace
		opts.DefaultNamespaces = map[string]cache.Config{
			ns: {},
		}
		setupLog.Info("Watching namespace", "namespace", ns)
	} else {
		// Watch all namespaces
		setupLog.Info("Watching all namespaces")
	}

	return opts
}

// logMemoryStats logs memory statistics periodically for debugging
func logMemoryStats() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		setupLog.V(1).Info("Memory stats",
			"alloc_mb", m.Alloc/1024/1024,
			"total_alloc_mb", m.TotalAlloc/1024/1024,
			"sys_mb", m.Sys/1024/1024,
			"num_gc", m.NumGC,
			"goroutines", runtime.NumGoroutine(),
		)
	}
}
