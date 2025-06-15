/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// helmClient implements the HelmClient interface
type helmClient struct {
	settings   *cli.EnvSettings
	restConfig *rest.Config
	namespace  string
}

// NewHelmClient creates a new Helm client
func NewHelmClient(restConfig *rest.Config, namespace string) (HelmClient, error) {
	settings := cli.New()
	
	return &helmClient{
		settings:   settings,
		restConfig: restConfig,
		namespace:  namespace,
	}, nil
}

// getActionConfig creates a new Helm action configuration
func (c *helmClient) getActionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	
	// Create a RESTClientGetter
	restClientGetter := &restClientGetter{
		namespace:  namespace,
		restConfig: c.restConfig,
	}
	
	// Initialize the action configuration
	if err := actionConfig.Init(restClientGetter, namespace, os.Getenv("HELM_DRIVER"), log.Log.V(4).Info); err != nil {
		return nil, errors.Wrap(err, "failed to initialize Helm action configuration")
	}
	
	return actionConfig, nil
}

// InstallChart installs a Helm chart
func (c *helmClient) InstallChart(ctx context.Context, opts *InstallOptions) (*ReleaseInfo, error) {
	actionConfig, err := c.getActionConfig(opts.Namespace)
	if err != nil {
		return nil, err
	}
	
	install := action.NewInstall(actionConfig)
	install.ReleaseName = opts.ReleaseName
	install.Namespace = opts.Namespace
	install.CreateNamespace = opts.CreateNamespace
	install.Wait = opts.Wait
	install.WaitForJobs = opts.WaitForJobs
	install.Timeout = opts.Timeout
	install.DisableHooks = opts.DisableHooks
	install.DisableOpenAPIValidation = opts.DisableOpenAPI
	install.SkipCRDs = opts.SkipCRDs
	install.SubNotes = opts.SubNotes
	install.Description = opts.Description
	install.Atomic = opts.Atomic
	install.Force = opts.Force
	install.DependencyUpdate = opts.DependencyUpdate
	
	// Set chart version if specified
	if opts.Version != "" {
		install.Version = opts.Version
	}
	
	// Load the chart
	chartPath, err := install.LocateChart(opts.ChartRef, c.settings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to locate chart")
	}
	
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart")
	}
	
	// Merge values
	values := opts.Values
	if values == nil {
		values = make(map[string]interface{})
	}
	
	// Load values from files
	for _, file := range opts.ValuesFiles {
		fileValues, err := c.loadValuesFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load values file %s", file)
		}
		values = mergeMaps(values, fileValues)
	}
	
	// Run the installation
	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, errors.Wrap(err, "failed to install chart")
	}
	
	return c.releaseToInfo(rel), nil
}

// UpgradeChart upgrades an existing Helm release
func (c *helmClient) UpgradeChart(ctx context.Context, opts *UpgradeOptions) (*ReleaseInfo, error) {
	actionConfig, err := c.getActionConfig(opts.Namespace)
	if err != nil {
		return nil, err
	}
	
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = opts.Namespace
	upgrade.ReuseValues = opts.ReuseValues
	upgrade.ResetValues = opts.ResetValues
	upgrade.Wait = opts.Wait
	upgrade.WaitForJobs = opts.WaitForJobs
	upgrade.Timeout = opts.Timeout
	upgrade.DisableHooks = opts.DisableHooks
	upgrade.DisableOpenAPIValidation = opts.DisableOpenAPI
	upgrade.SkipCRDs = opts.SkipCRDs
	upgrade.SubNotes = opts.SubNotes
	upgrade.Description = opts.Description
	upgrade.Atomic = opts.Atomic
	upgrade.CleanupOnFail = opts.CleanupOnFail
	upgrade.Force = opts.Force
	upgrade.Install = opts.Install
	upgrade.MaxHistory = opts.MaxHistory
	
	// Set chart version if specified
	if opts.Version != "" {
		upgrade.Version = opts.Version
	}
	
	// Load the chart
	chartPath, err := upgrade.LocateChart(opts.ChartRef, c.settings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to locate chart")
	}
	
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart")
	}
	
	// Merge values
	values := opts.Values
	if values == nil {
		values = make(map[string]interface{})
	}
	
	// Load values from files
	for _, file := range opts.ValuesFiles {
		fileValues, err := c.loadValuesFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load values file %s", file)
		}
		values = mergeMaps(values, fileValues)
	}
	
	// Run the upgrade
	rel, err := upgrade.RunWithContext(ctx, opts.ReleaseName, chart, values)
	if err != nil {
		return nil, errors.Wrap(err, "failed to upgrade chart")
	}
	
	return c.releaseToInfo(rel), nil
}

// UninstallChart uninstalls a Helm release
func (c *helmClient) UninstallChart(ctx context.Context, releaseName, namespace string) error {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return err
	}
	
	uninstall := action.NewUninstall(actionConfig)
	uninstall.KeepHistory = false
	
	_, err = uninstall.Run(releaseName)
	if err != nil {
		return errors.Wrap(err, "failed to uninstall release")
	}
	
	return nil
}

// GetRelease retrieves information about a Helm release
func (c *helmClient) GetRelease(ctx context.Context, releaseName, namespace string) (*ReleaseInfo, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	
	get := action.NewGet(actionConfig)
	
	rel, err := get.Run(releaseName)
	if err != nil {
		if err == driver.ErrReleaseNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get release")
	}
	
	return c.releaseToInfo(rel), nil
}

// ListReleases lists all Helm releases in a namespace
func (c *helmClient) ListReleases(ctx context.Context, namespace string) ([]*ReleaseInfo, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	
	list := action.NewList(actionConfig)
	list.All = true
	
	releases, err := list.Run()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list releases")
	}
	
	infos := make([]*ReleaseInfo, len(releases))
	for i, rel := range releases {
		infos[i] = c.releaseToInfo(rel)
	}
	
	return infos, nil
}

// RollbackRelease rolls back a release to a previous version
func (c *helmClient) RollbackRelease(ctx context.Context, releaseName, namespace string, version int) error {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return err
	}
	
	rollback := action.NewRollback(actionConfig)
	rollback.Version = version
	rollback.Wait = true
	rollback.Timeout = 300 * time.Second
	
	if err := rollback.Run(releaseName); err != nil {
		return errors.Wrap(err, "failed to rollback release")
	}
	
	return nil
}

// GetValues retrieves the values for a release
func (c *helmClient) GetValues(ctx context.Context, releaseName, namespace string) (map[string]interface{}, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	
	get := action.NewGetValues(actionConfig)
	get.AllValues = true
	
	values, err := get.Run(releaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get values")
	}
	
	return values, nil
}

// TestRelease runs Helm tests for a release
func (c *helmClient) TestRelease(ctx context.Context, releaseName, namespace string) error {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return err
	}
	
	test := action.NewReleaseTesting(actionConfig)
	test.Timeout = 300 * time.Second
	
	_, err = test.Run(releaseName)
	if err != nil {
		return errors.Wrap(err, "failed to run tests")
	}
	
	return nil
}

// RenderChart renders chart templates without installing
func (c *helmClient) RenderChart(ctx context.Context, opts *RenderOptions) (string, error) {
	actionConfig, err := c.getActionConfig(opts.Namespace)
	if err != nil {
		return "", err
	}
	
	install := action.NewInstall(actionConfig)
	install.DryRun = true
	install.ReleaseName = opts.ReleaseName
	install.Namespace = opts.Namespace
	install.IncludeCRDs = opts.IncludeCRDs
	
	if opts.Version != "" {
		install.Version = opts.Version
	}
	
	// Load the chart
	chartPath, err := install.LocateChart(opts.ChartRef, c.settings)
	if err != nil {
		return "", errors.Wrap(err, "failed to locate chart")
	}
	
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to load chart")
	}
	
	// Merge values
	values := opts.Values
	if values == nil {
		values = make(map[string]interface{})
	}
	
	// Load values from files
	for _, file := range opts.ValuesFiles {
		fileValues, err := c.loadValuesFile(file)
		if err != nil {
			return "", errors.Wrapf(err, "failed to load values file %s", file)
		}
		values = mergeMaps(values, fileValues)
	}
	
	// Run the dry-run installation to render templates
	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return "", errors.Wrap(err, "failed to render chart")
	}
	
	return rel.Manifest, nil
}

// releaseToInfo converts a Helm release to ReleaseInfo
func (c *helmClient) releaseToInfo(rel *release.Release) *ReleaseInfo {
	info := &ReleaseInfo{
		Name:         rel.Name,
		Namespace:    rel.Namespace,
		Version:      rel.Version,
		Status:       rel.Info.Status,
		LastDeployed: rel.Info.LastDeployed.Time,
		Values:       rel.Config,
	}
	
	if rel.Chart != nil {
		info.Chart = &ChartInfo{
			Name:        rel.Chart.Name(),
			Version:     rel.Chart.Metadata.Version,
			Description: rel.Chart.Metadata.Description,
		}
	}
	
	return info
}

// loadValuesFile loads values from a YAML file
func (c *helmClient) loadValuesFile(filename string) (map[string]interface{}, error) {
	values, err := chartutil.ReadValuesFile(filename)
	if err != nil {
		return nil, err
	}
	return values.AsMap(), nil
}

// mergeMaps merges two maps, with values from the second map overriding the first
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy from a
	for k, v := range a {
		result[k] = v
	}
	
	// Override with b
	for k, v := range b {
		result[k] = v
	}
	
	return result
}

// restClientGetter implements RESTClientGetter interface for Helm
type restClientGetter struct {
	namespace  string
	restConfig *rest.Config
}

func (g *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return g.restConfig, nil
}

func (g *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
}

func (g *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(g.restConfig)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(discoveryClient), nil
}

func (g *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	return mapper, nil
}
