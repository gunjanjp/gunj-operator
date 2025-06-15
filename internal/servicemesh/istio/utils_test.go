package istio

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istioclientv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istioclientv1beta1security "istio.io/client-go/pkg/apis/security/v1beta1"
	istioclientv1alpha1telemetry "istio.io/client-go/pkg/apis/telemetry/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	securityv1beta1 "istio.io/api/security/v1beta1"
	telemetryv1alpha1 "istio.io/api/telemetry/v1alpha1"
)

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "simple service name",
			host:     "myservice",
			expected: "myservice",
		},
		{
			name:     "service with namespace",
			host:     "myservice.namespace",
			expected: "myservice",
		},
		{
			name:     "fully qualified domain name",
			host:     "myservice.namespace.svc.cluster.local",
			expected: "myservice",
		},
		{
			name:     "empty host",
			host:     "",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetServiceName(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNamespace(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "simple service name",
			host:     "myservice",
			expected: "",
		},
		{
			name:     "service with namespace",
			host:     "myservice.namespace",
			expected: "namespace",
		},
		{
			name:     "fully qualified domain name",
			host:     "myservice.namespace.svc.cluster.local",
			expected: "namespace",
		},
		{
			name:     "empty host",
			host:     "",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNamespace(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFQDN(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		namespace string
		expected  string
	}{
		{
			name:      "basic service",
			service:   "myservice",
			namespace: "default",
			expected:  "myservice.default.svc.cluster.local",
		},
		{
			name:      "service with dashes",
			service:   "my-service",
			namespace: "my-namespace",
			expected:  "my-service.my-namespace.svc.cluster.local",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildFQDN(tt.service, tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsIstioNamespace(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name:     "nil labels",
			labels:   nil,
			expected: false,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: false,
		},
		{
			name: "istio-injection enabled",
			labels: map[string]string{
				"istio-injection": "enabled",
			},
			expected: true,
		},
		{
			name: "istio-injection disabled",
			labels: map[string]string{
				"istio-injection": "disabled",
			},
			expected: false,
		},
		{
			name: "istio revision label",
			labels: map[string]string{
				"istio.io/rev": "1-14-3",
			},
			expected: true,
		},
		{
			name: "both labels present",
			labels: map[string]string{
				"istio-injection": "enabled",
				"istio.io/rev":    "1-14-3",
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIstioNamespace(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMTLSMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{
			name:    "valid UNSET",
			mode:    "UNSET",
			wantErr: false,
		},
		{
			name:    "valid DISABLE",
			mode:    "DISABLE",
			wantErr: false,
		},
		{
			name:    "valid PERMISSIVE",
			mode:    "PERMISSIVE",
			wantErr: false,
		},
		{
			name:    "valid STRICT",
			mode:    "STRICT",
			wantErr: false,
		},
		{
			name:    "invalid mode",
			mode:    "INVALID",
			wantErr: true,
		},
		{
			name:    "empty mode",
			mode:    "",
			wantErr: true,
		},
		{
			name:    "lowercase mode",
			mode:    "strict",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMTLSMode(tt.mode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLoadBalancerType(t *testing.T) {
	tests := []struct {
		name    string
		lbType  string
		wantErr bool
	}{
		{
			name:    "valid ROUND_ROBIN",
			lbType:  "ROUND_ROBIN",
			wantErr: false,
		},
		{
			name:    "valid LEAST_REQUEST",
			lbType:  "LEAST_REQUEST",
			wantErr: false,
		},
		{
			name:    "valid RANDOM",
			lbType:  "RANDOM",
			wantErr: false,
		},
		{
			name:    "valid PASSTHROUGH",
			lbType:  "PASSTHROUGH",
			wantErr: false,
		},
		{
			name:    "invalid type",
			lbType:  "WEIGHTED",
			wantErr: true,
		},
		{
			name:    "empty type",
			lbType:  "",
			wantErr: true,
		},
		{
			name:    "lowercase type",
			lbType:  "round_robin",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoadBalancerType(tt.lbType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConvertToIstioLabels(t *testing.T) {
	tests := []struct {
		name           string
		genericLabels  map[string]string
		expectedLabels map[string]string
	}{
		{
			name: "convert standard labels",
			genericLabels: map[string]string{
				"app.kubernetes.io/name":    "myapp",
				"app.kubernetes.io/version": "v1",
				"custom-label":              "value",
			},
			expectedLabels: map[string]string{
				"app":          "myapp",
				"version":      "v1",
				"custom-label": "value",
			},
		},
		{
			name:           "empty labels",
			genericLabels:  map[string]string{},
			expectedLabels: map[string]string{},
		},
		{
			name: "no conversion needed",
			genericLabels: map[string]string{
				"app":     "myapp",
				"version": "v1",
			},
			expectedLabels: map[string]string{
				"app":     "myapp",
				"version": "v1",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToIstioLabels(tt.genericLabels)
			assert.Equal(t, tt.expectedLabels, result)
		})
	}
}

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name     string
		maps     []map[string]string
		expected map[string]string
	}{
		{
			name: "merge two maps",
			maps: []map[string]string{
				{"a": "1", "b": "2"},
				{"c": "3", "d": "4"},
			},
			expected: map[string]string{
				"a": "1", "b": "2", "c": "3", "d": "4",
			},
		},
		{
			name: "override values",
			maps: []map[string]string{
				{"a": "1", "b": "2"},
				{"b": "3", "c": "4"},
			},
			expected: map[string]string{
				"a": "1", "b": "3", "c": "4",
			},
		},
		{
			name:     "empty maps",
			maps:     []map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "nil map in slice",
			maps: []map[string]string{
				{"a": "1"},
				nil,
				{"b": "2"},
			},
			expected: map[string]string{
				"a": "1", "b": "2",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMaps(tt.maps...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateResourceName(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		component string
		suffix    string
		expected  string
	}{
		{
			name:      "all parts",
			prefix:    "gunj",
			component: "prometheus",
			suffix:    "metrics",
			expected:  "gunj-prometheus-metrics",
		},
		{
			name:      "no component",
			prefix:    "gunj",
			component: "",
			suffix:    "config",
			expected:  "gunj-config",
		},
		{
			name:      "no suffix",
			prefix:    "gunj",
			component: "grafana",
			suffix:    "",
			expected:  "gunj-grafana",
		},
		{
			name:      "only prefix",
			prefix:    "gunj",
			component: "",
			suffix:    "",
			expected:  "gunj",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateResourceName(tt.prefix, tt.component, tt.suffix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIstioResourceList(t *testing.T) {
	t.Run("new list is empty", func(t *testing.T) {
		list := NewIstioResourceList()
		assert.NotNil(t, list)
		assert.True(t, list.IsEmpty())
		assert.Equal(t, 0, list.Count())
	})
	
	t.Run("count resources", func(t *testing.T) {
		list := NewIstioResourceList()
		
		// Add resources
		list.VirtualServices = []*istioclientv1beta1.VirtualService{
			{ObjectMeta: metav1.ObjectMeta{Name: "vs1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "vs2"}},
		}
		list.DestinationRules = []*istioclientv1beta1.DestinationRule{
			{ObjectMeta: metav1.ObjectMeta{Name: "dr1"}},
		}
		list.PeerAuthentications = []*istioclientv1beta1security.PeerAuthentication{
			{ObjectMeta: metav1.ObjectMeta{Name: "pa1"}},
		}
		list.Telemetries = []*istioclientv1alpha1telemetry.Telemetry{
			{ObjectMeta: metav1.ObjectMeta{Name: "t1"}},
		}
		
		assert.False(t, list.IsEmpty())
		assert.Equal(t, 5, list.Count())
	})
	
	t.Run("get virtual service by name", func(t *testing.T) {
		list := NewIstioResourceList()
		
		vs1 := &istioclientv1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{Name: "vs1"},
		}
		vs2 := &istioclientv1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{Name: "vs2"},
		}
		list.VirtualServices = []*istioclientv1beta1.VirtualService{vs1, vs2}
		
		// Found
		result := list.GetVirtualServiceByName("vs1")
		assert.Equal(t, vs1, result)
		
		// Not found
		result = list.GetVirtualServiceByName("vs3")
		assert.Nil(t, result)
	})
	
	t.Run("get destination rule by host", func(t *testing.T) {
		list := NewIstioResourceList()
		
		dr1 := &istioclientv1beta1.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{Name: "dr1"},
			Spec: networkingv1beta1.DestinationRule{
				Host: "service1",
			},
		}
		dr2 := &istioclientv1beta1.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{Name: "dr2"},
			Spec: networkingv1beta1.DestinationRule{
				Host: "service2",
			},
		}
		list.DestinationRules = []*istioclientv1beta1.DestinationRule{dr1, dr2}
		
		// Found
		result := list.GetDestinationRuleByHost("service1")
		assert.Equal(t, dr1, result)
		
		// Not found
		result = list.GetDestinationRuleByHost("service3")
		assert.Nil(t, result)
	})
}

func TestValidateVirtualService(t *testing.T) {
	tests := []struct {
		name    string
		vs      *istioclientv1beta1.VirtualService
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid virtual service",
			vs: &istioclientv1beta1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vs"},
				Spec: networkingv1beta1.VirtualService{
					Hosts: []string{"myservice"},
					Http: []*networkingv1beta1.HTTPRoute{
						{
							Route: []*networkingv1beta1.HTTPRouteDestination{
								{
									Destination: &networkingv1beta1.Destination{
										Host: "myservice",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			vs: &istioclientv1beta1.VirtualService{
				Spec: networkingv1beta1.VirtualService{
					Hosts: []string{"myservice"},
				},
			},
			wantErr: true,
			errMsg:  "VirtualService name cannot be empty",
		},
		{
			name: "missing hosts",
			vs: &istioclientv1beta1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vs"},
				Spec:       networkingv1beta1.VirtualService{},
			},
			wantErr: true,
			errMsg:  "VirtualService must have at least one host",
		},
		{
			name: "missing routes",
			vs: &istioclientv1beta1.VirtualService{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vs"},
				Spec: networkingv1beta1.VirtualService{
					Hosts: []string{"myservice"},
				},
			},
			wantErr: true,
			errMsg:  "VirtualService must have at least one route",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVirtualService(tt.vs)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDestinationRule(t *testing.T) {
	tests := []struct {
		name    string
		dr      *istioclientv1beta1.DestinationRule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid destination rule",
			dr: &istioclientv1beta1.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dr"},
				Spec: networkingv1beta1.DestinationRule{
					Host: "myservice",
				},
			},
			wantErr: false,
		},
		{
			name: "valid with subsets",
			dr: &istioclientv1beta1.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dr"},
				Spec: networkingv1beta1.DestinationRule{
					Host: "myservice",
					Subsets: []*networkingv1beta1.Subset{
						{
							Name:   "v1",
							Labels: map[string]string{"version": "v1"},
						},
						{
							Name:   "v2",
							Labels: map[string]string{"version": "v2"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			dr: &istioclientv1beta1.DestinationRule{
				Spec: networkingv1beta1.DestinationRule{
					Host: "myservice",
				},
			},
			wantErr: true,
			errMsg:  "DestinationRule name cannot be empty",
		},
		{
			name: "missing host",
			dr: &istioclientv1beta1.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dr"},
				Spec:       networkingv1beta1.DestinationRule{},
			},
			wantErr: true,
			errMsg:  "DestinationRule host cannot be empty",
		},
		{
			name: "subset without name",
			dr: &istioclientv1beta1.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dr"},
				Spec: networkingv1beta1.DestinationRule{
					Host: "myservice",
					Subsets: []*networkingv1beta1.Subset{
						{
							Labels: map[string]string{"version": "v1"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "subset name cannot be empty",
		},
		{
			name: "subset without labels",
			dr: &istioclientv1beta1.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dr"},
				Spec: networkingv1beta1.DestinationRule{
					Host: "myservice",
					Subsets: []*networkingv1beta1.Subset{
						{
							Name: "v1",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "must have at least one label",
		},
		{
			name: "duplicate subset names",
			dr: &istioclientv1beta1.DestinationRule{
				ObjectMeta: metav1.ObjectMeta{Name: "test-dr"},
				Spec: networkingv1beta1.DestinationRule{
					Host: "myservice",
					Subsets: []*networkingv1beta1.Subset{
						{
							Name:   "v1",
							Labels: map[string]string{"version": "v1"},
						},
						{
							Name:   "v1",
							Labels: map[string]string{"version": "v2"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate subset name: v1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDestinationRule(tt.dr)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTelemetry(t *testing.T) {
	tests := []struct {
		name    string
		t       *istioclientv1alpha1telemetry.Telemetry
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid telemetry",
			t: &istioclientv1alpha1telemetry.Telemetry{
				ObjectMeta: metav1.ObjectMeta{Name: "test-telemetry"},
				Spec: telemetryv1alpha1.Telemetry{
					Metrics: []*telemetryv1alpha1.Metrics{
						{
							Providers: []*telemetryv1alpha1.ProviderRef{
								{Name: "prometheus"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			t: &istioclientv1alpha1telemetry.Telemetry{
				Spec: telemetryv1alpha1.Telemetry{},
			},
			wantErr: true,
			errMsg:  "Telemetry name cannot be empty",
		},
		{
			name: "metrics without providers",
			t: &istioclientv1alpha1telemetry.Telemetry{
				ObjectMeta: metav1.ObjectMeta{Name: "test-telemetry"},
				Spec: telemetryv1alpha1.Telemetry{
					Metrics: []*telemetryv1alpha1.Metrics{
						{
							Providers: []*telemetryv1alpha1.ProviderRef{},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "metrics configuration must have at least one provider",
		},
		{
			name: "invalid tracing sampling rate",
			t: &istioclientv1alpha1telemetry.Telemetry{
				ObjectMeta: metav1.ObjectMeta{Name: "test-telemetry"},
				Spec: telemetryv1alpha1.Telemetry{
					Tracing: []*telemetryv1alpha1.Tracing{
						{
							Providers: []*telemetryv1alpha1.ProviderRef{
								{Name: "jaeger"},
							},
							RandomSamplingPercentage: func() *float64 { v := 150.0; return &v }(),
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0 and 100",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTelemetry(tt.t)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
