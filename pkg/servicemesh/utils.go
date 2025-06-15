// Package servicemesh provides service mesh integration for the Gunj Operator
package servicemesh

import (
	"context"
	"errors"
	"fmt"
	"time"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Common errors for service mesh operations
var (
	// ErrServiceMeshNotEnabled indicates service mesh is not enabled
	ErrServiceMeshNotEnabled = errors.New("service mesh integration is not enabled")
	
	// ErrUnsupportedProvider indicates the service mesh provider is not supported
	ErrUnsupportedProvider = errors.New("unsupported service mesh provider")
	
	// ErrInvalidConfiguration indicates the configuration is invalid
	ErrInvalidConfiguration = errors.New("invalid service mesh configuration")
	
	// ErrServiceMeshNotReady indicates the service mesh is not ready
	ErrServiceMeshNotReady = errors.New("service mesh is not ready")
	
	// ErrResourceNotFound indicates a required resource was not found
	ErrResourceNotFound = errors.New("service mesh resource not found")
	
	// ErrOperationTimeout indicates an operation timed out
	ErrOperationTimeout = errors.New("service mesh operation timed out")
)

// ServiceMeshUtils provides utility functions for service mesh operations
type ServiceMeshUtils struct {
	client client.Client
	log    logr.Logger
}

// NewServiceMeshUtils creates a new service mesh utils instance
func NewServiceMeshUtils(client client.Client, log logr.Logger) *ServiceMeshUtils {
	return &ServiceMeshUtils{
		client: client,
		log:    log,
	}
}

// IsSidecarInjected checks if a pod has sidecar injected
func (u *ServiceMeshUtils) IsSidecarInjected(pod *corev1.Pod, provider MeshProvider) bool {
	switch provider {
	case IstioProvider:
		return u.isIstioSidecarInjected(pod)
	case LinkerdProvider:
		return u.isLinkerdSidecarInjected(pod)
	default:
		return false
	}
}

// isIstioSidecarInjected checks if Istio sidecar is injected
func (u *ServiceMeshUtils) isIstioSidecarInjected(pod *corev1.Pod) bool {
	// Check for Istio sidecar container
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" {
			return true
		}
	}
	
	// Check for Istio annotations
	if pod.Annotations != nil {
		if injected, ok := pod.Annotations["sidecar.istio.io/status"]; ok && injected != "" {
			return true
		}
	}
	
	return false
}

// isLinkerdSidecarInjected checks if Linkerd sidecar is injected
func (u *ServiceMeshUtils) isLinkerdSidecarInjected(pod *corev1.Pod) bool {
	// Check for Linkerd proxy container
	for _, container := range pod.Spec.Containers {
		if container.Name == "linkerd-proxy" {
			return true
		}
	}
	
	// Check for Linkerd annotations
	if pod.Annotations != nil {
		if _, ok := pod.Annotations["linkerd.io/injected"]; ok {
			return true
		}
	}
	
	return false
}

// GetMeshEnabledPods returns all pods with service mesh sidecar injected
func (u *ServiceMeshUtils) GetMeshEnabledPods(ctx context.Context, namespace string, provider MeshProvider) ([]*corev1.Pod, error) {
	podList := &corev1.PodList{}
	if err := u.client.List(ctx, podList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}
	
	var meshPods []*corev1.Pod
	for i := range podList.Items {
		pod := &podList.Items[i]
		if u.IsSidecarInjected(pod, provider) {
			meshPods = append(meshPods, pod)
		}
	}
	
	return meshPods, nil
}

// GetComponentPods returns all pods for a specific component
func (u *ServiceMeshUtils) GetComponentPods(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, component string) ([]*corev1.Pod, error) {
	// Create label selector
	labelSelector := labels.NewSelector()
	
	req1, _ := labels.NewRequirement("app.kubernetes.io/instance", selection.Equals, []string{platform.Name})
	req2, _ := labels.NewRequirement("app.kubernetes.io/component", selection.Equals, []string{component})
	
	labelSelector = labelSelector.Add(*req1, *req2)
	
	// List pods
	podList := &corev1.PodList{}
	if err := u.client.List(ctx, podList, 
		client.InNamespace(platform.Namespace),
		client.MatchingLabelsSelector{Selector: labelSelector},
	); err != nil {
		return nil, fmt.Errorf("listing component pods: %w", err)
	}
	
	result := make([]*corev1.Pod, len(podList.Items))
	for i := range podList.Items {
		result[i] = &podList.Items[i]
	}
	
	return result, nil
}

// WaitForCondition waits for a condition to be met
func (u *ServiceMeshUtils) WaitForCondition(ctx context.Context, checkFunc func() (bool, error), timeout time.Duration) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	timeoutCh := time.After(timeout)
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutCh:
			return ErrOperationTimeout
		case <-ticker.C:
			ready, err := checkFunc()
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		}
	}
}

// CreateServiceEntry creates a service entry for external services
func CreateServiceEntry(name, namespace string, hosts []string, ports []ServicePort, location string) *ServiceEntry {
	return &ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: ServiceEntrySpec{
			Hosts:    hosts,
			Ports:    ports,
			Location: location,
		},
	}
}

// ServiceEntry represents a service entry for external services
type ServiceEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceEntrySpec `json:"spec"`
}

// ServiceEntrySpec defines the specification for a service entry
type ServiceEntrySpec struct {
	Hosts    []string      `json:"hosts"`
	Ports    []ServicePort `json:"ports"`
	Location string        `json:"location"`
}

// ServicePort represents a port for a service
type ServicePort struct {
	Number   uint32 `json:"number"`
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
}

// CreateVirtualService creates a virtual service for traffic management
func CreateVirtualService(name, namespace string, hosts []string, gateways []string, http []HTTPRoute) *VirtualService {
	return &VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: VirtualServiceSpec{
			Hosts:    hosts,
			Gateways: gateways,
			HTTP:     http,
		},
	}
}

// VirtualService represents a virtual service for traffic management
type VirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualServiceSpec `json:"spec"`
}

// VirtualServiceSpec defines the specification for a virtual service
type VirtualServiceSpec struct {
	Hosts    []string    `json:"hosts"`
	Gateways []string    `json:"gateways,omitempty"`
	HTTP     []HTTPRoute `json:"http,omitempty"`
}

// HTTPRoute represents an HTTP route
type HTTPRoute struct {
	Match       []HTTPMatchRequest `json:"match,omitempty"`
	Route       []HTTPRouteDestination `json:"route,omitempty"`
	Redirect    *HTTPRedirect `json:"redirect,omitempty"`
	Rewrite     *HTTPRewrite `json:"rewrite,omitempty"`
	Timeout     *time.Duration `json:"timeout,omitempty"`
	Retries     *HTTPRetry `json:"retries,omitempty"`
	FaultInjection *HTTPFaultInjection `json:"fault,omitempty"`
}

// HTTPMatchRequest represents an HTTP match request
type HTTPMatchRequest struct {
	URI         *StringMatch          `json:"uri,omitempty"`
	Headers     map[string]StringMatch `json:"headers,omitempty"`
	Method      *StringMatch          `json:"method,omitempty"`
	QueryParams map[string]StringMatch `json:"queryParams,omitempty"`
}

// StringMatch represents a string match
type StringMatch struct {
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// HTTPRouteDestination represents an HTTP route destination
type HTTPRouteDestination struct {
	Destination Destination `json:"destination"`
	Weight      int32       `json:"weight,omitempty"`
}

// Destination represents a destination
type Destination struct {
	Host   string `json:"host"`
	Subset string `json:"subset,omitempty"`
	Port   *PortSelector `json:"port,omitempty"`
}

// PortSelector represents a port selector
type PortSelector struct {
	Number uint32 `json:"number,omitempty"`
}

// HTTPRedirect represents an HTTP redirect
type HTTPRedirect struct {
	URI          string `json:"uri,omitempty"`
	Authority    string `json:"authority,omitempty"`
	RedirectCode int    `json:"redirectCode,omitempty"`
}

// HTTPRewrite represents an HTTP rewrite
type HTTPRewrite struct {
	URI       string `json:"uri,omitempty"`
	Authority string `json:"authority,omitempty"`
}

// HTTPRetry represents HTTP retry configuration
type HTTPRetry struct {
	Attempts      int32          `json:"attempts"`
	PerTryTimeout *time.Duration `json:"perTryTimeout,omitempty"`
	RetryOn       string         `json:"retryOn,omitempty"`
}

// HTTPFaultInjection represents HTTP fault injection
type HTTPFaultInjection struct {
	Delay *HTTPFaultInjectionDelay `json:"delay,omitempty"`
	Abort *HTTPFaultInjectionAbort `json:"abort,omitempty"`
}

// HTTPFaultInjectionDelay represents delay injection
type HTTPFaultInjectionDelay struct {
	Percentage  int32          `json:"percentage"`
	FixedDelay  *time.Duration `json:"fixedDelay"`
}

// HTTPFaultInjectionAbort represents abort injection
type HTTPFaultInjectionAbort struct {
	Percentage  int32 `json:"percentage"`
	HTTPStatus  int32 `json:"httpStatus"`
}

// CreateDestinationRule creates a destination rule for traffic management
func CreateDestinationRule(name, namespace, host string, subsets []Subset, trafficPolicy *TrafficPolicy) *DestinationRule {
	return &DestinationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: DestinationRuleSpec{
			Host:          host,
			Subsets:       subsets,
			TrafficPolicy: trafficPolicy,
		},
	}
}

// DestinationRule represents a destination rule
type DestinationRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DestinationRuleSpec `json:"spec"`
}

// DestinationRuleSpec defines the specification for a destination rule
type DestinationRuleSpec struct {
	Host          string         `json:"host"`
	Subsets       []Subset       `json:"subsets,omitempty"`
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty"`
}

// Subset represents a subset of endpoints
type Subset struct {
	Name          string            `json:"name"`
	Labels        map[string]string `json:"labels"`
	TrafficPolicy *TrafficPolicy    `json:"trafficPolicy,omitempty"`
}

// GetServiceMeshNamespace returns the namespace where service mesh is installed
func GetServiceMeshNamespace(provider MeshProvider) string {
	switch provider {
	case IstioProvider:
		return "istio-system"
	case LinkerdProvider:
		return "linkerd"
	default:
		return ""
	}
}

// IsServiceMeshInstalled checks if service mesh is installed
func (u *ServiceMeshUtils) IsServiceMeshInstalled(ctx context.Context, provider MeshProvider) (bool, error) {
	namespace := GetServiceMeshNamespace(provider)
	if namespace == "" {
		return false, nil
	}
	
	// Check if namespace exists
	ns := &corev1.Namespace{}
	err := u.client.Get(ctx, client.ObjectKey{Name: namespace}, ns)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return false, fmt.Errorf("checking service mesh namespace: %w", err)
		}
		return false, nil
	}
	
	// Check for provider-specific resources
	switch provider {
	case IstioProvider:
		return u.isIstioInstalled(ctx, namespace)
	case LinkerdProvider:
		return u.isLinkerdInstalled(ctx, namespace)
	default:
		return false, nil
	}
}

// isIstioInstalled checks if Istio is installed
func (u *ServiceMeshUtils) isIstioInstalled(ctx context.Context, namespace string) (bool, error) {
	// Check for istiod deployment
	podList := &corev1.PodList{}
	if err := u.client.List(ctx, podList, 
		client.InNamespace(namespace),
		client.MatchingLabels{"app": "istiod"},
	); err != nil {
		return false, fmt.Errorf("checking istiod pods: %w", err)
	}
	
	// Check if any pods are running
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}
	}
	
	return false, nil
}

// isLinkerdInstalled checks if Linkerd is installed
func (u *ServiceMeshUtils) isLinkerdInstalled(ctx context.Context, namespace string) (bool, error) {
	// Check for linkerd-controller deployment
	podList := &corev1.PodList{}
	if err := u.client.List(ctx, podList, 
		client.InNamespace(namespace),
		client.MatchingLabels{"linkerd.io/control-plane-component": "controller"},
	); err != nil {
		return false, fmt.Errorf("checking linkerd controller pods: %w", err)
	}
	
	// Check if any pods are running
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}
	}
	
	return false, nil
}
