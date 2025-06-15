package istio

import (
	"context"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1beta1 "istio.io/api/networking/v1beta1"
	istioclientv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	istioclientv1beta1security "istio.io/client-go/pkg/apis/security/v1beta1"
	istioclientv1alpha1telemetry "istio.io/client-go/pkg/apis/telemetry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	
	"github.com/gunjanjp/gunj-operator/internal/servicemesh"
)

func TestIstioManager_Configure(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		config    *servicemesh.ServiceMeshConfig
		objects   []runtime.Object
		wantErr   bool
		validate  func(t *testing.T, c client.Client)
	}{
		{
			name:      "enable sidecar injection",
			namespace: "test-ns",
			config: &servicemesh.ServiceMeshConfig{
				Type:    servicemesh.ServiceMeshTypeIstio,
				Enabled: true,
			},
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				ns := &corev1.Namespace{}
				err := c.Get(context.Background(), types.NamespacedName{Name: "test-ns"}, ns)
				require.NoError(t, err)
				assert.Equal(t, "enabled", ns.Labels["istio-injection"])
			},
		},
		{
			name:      "configure strict mTLS",
			namespace: "test-ns",
			config: &servicemesh.ServiceMeshConfig{
				Type:    servicemesh.ServiceMeshTypeIstio,
				Enabled: true,
				MTLS: servicemesh.MTLSConfig{
					Strict: true,
				},
			},
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				// Check namespace label
				ns := &corev1.Namespace{}
				err := c.Get(context.Background(), types.NamespacedName{Name: "test-ns"}, ns)
				require.NoError(t, err)
				assert.Equal(t, "enabled", ns.Labels["istio-injection"])
				
				// Check PeerAuthentication
				pa := &istioclientv1beta1security.PeerAuthentication{}
				err = c.Get(context.Background(), types.NamespacedName{
					Name:      "default",
					Namespace: "test-ns",
				}, pa)
				require.NoError(t, err)
				assert.NotNil(t, pa.Spec.Mtls)
			},
		},
		{
			name:      "configure telemetry",
			namespace: "test-ns",
			config: &servicemesh.ServiceMeshConfig{
				Type:    servicemesh.ServiceMeshTypeIstio,
				Enabled: true,
				Telemetry: servicemesh.TelemetryConfig{
					Metrics: servicemesh.MetricsConfig{
						Enabled:   true,
						Providers: []string{"prometheus"},
					},
					Tracing: servicemesh.TracingConfig{
						Enabled:      true,
						Provider:     "jaeger",
						SamplingRate: 1.0,
					},
				},
			},
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, c client.Client) {
				telemetry := &istioclientv1alpha1telemetry.Telemetry{}
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "default",
					Namespace: "test-ns",
				}, telemetry)
				require.NoError(t, err)
				assert.Len(t, telemetry.Spec.Metrics, 1)
				assert.Len(t, telemetry.Spec.Tracing, 1)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = clientgoscheme.AddToScheme(scheme)
			_ = istioclientv1beta1.AddToScheme(scheme)
			_ = istioclientv1beta1security.AddToScheme(scheme)
			_ = istioclientv1alpha1telemetry.AddToScheme(scheme)
			
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.objects...).
				Build()
			
			m := NewIstioManager(client)
			err := m.Configure(context.Background(), tt.namespace, tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, client)
				}
			}
		})
	}
}

func TestIstioManager_CreateVirtualService(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = istioclientv1beta1.AddToScheme(scheme)
	
	tests := []struct {
		name      string
		namespace string
		vsName    string
		spec      *servicemesh.VirtualServiceSpec
		wantErr   bool
		validate  func(t *testing.T, vs *istioclientv1beta1.VirtualService)
	}{
		{
			name:      "create basic virtual service",
			namespace: "test-ns",
			vsName:    "test-vs",
			spec: &servicemesh.VirtualServiceSpec{
				Hosts: []string{"test-service"},
				HTTP: []servicemesh.HTTPRouteSpec{
					{
						Route: []servicemesh.HTTPRouteDestination{
							{
								Destination: &servicemesh.Destination{
									Host: "test-service",
								},
								Weight: 100,
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, vs *istioclientv1beta1.VirtualService) {
				assert.Equal(t, "test-vs", vs.Name)
				assert.Equal(t, "test-ns", vs.Namespace)
				assert.Len(t, vs.Spec.Hosts, 1)
				assert.Equal(t, "test-service", vs.Spec.Hosts[0])
				assert.Len(t, vs.Spec.Http, 1)
			},
		},
		{
			name:      "create canary virtual service",
			namespace: "test-ns",
			vsName:    "canary-vs",
			spec: &servicemesh.VirtualServiceSpec{
				Hosts: []string{"myapp"},
				HTTP: []servicemesh.HTTPRouteSpec{
					{
						Route: []servicemesh.HTTPRouteDestination{
							{
								Destination: &servicemesh.Destination{
									Host:   "myapp",
									Subset: "stable",
								},
								Weight: 80,
							},
							{
								Destination: &servicemesh.Destination{
									Host:   "myapp",
									Subset: "canary",
								},
								Weight: 20,
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, vs *istioclientv1beta1.VirtualService) {
				assert.Len(t, vs.Spec.Http[0].Route, 2)
				assert.Equal(t, int32(80), vs.Spec.Http[0].Route[0].Weight)
				assert.Equal(t, int32(20), vs.Spec.Http[0].Route[1].Weight)
				assert.Equal(t, "stable", vs.Spec.Http[0].Route[0].Destination.Subset)
				assert.Equal(t, "canary", vs.Spec.Http[0].Route[1].Destination.Subset)
			},
		},
		{
			name:      "create virtual service with retry and timeout",
			namespace: "test-ns",
			vsName:    "retry-vs",
			spec: &servicemesh.VirtualServiceSpec{
				Hosts: []string{"retry-service"},
				HTTP: []servicemesh.HTTPRouteSpec{
					{
						Route: []servicemesh.HTTPRouteDestination{
							{
								Destination: &servicemesh.Destination{
									Host: "retry-service",
								},
								Weight: 100,
							},
						},
						Timeout: "10s",
						Retry: &servicemesh.RetryPolicy{
							Attempts:      3,
							PerTryTimeout: "3s",
							RetryOn:       "5xx",
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, vs *istioclientv1beta1.VirtualService) {
				assert.NotNil(t, vs.Spec.Http[0].Timeout)
				assert.NotNil(t, vs.Spec.Http[0].Retries)
				assert.Equal(t, int32(3), vs.Spec.Http[0].Retries.Attempts)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()
			
			m := NewIstioManager(client)
			err := m.CreateVirtualService(context.Background(), tt.namespace, tt.vsName, tt.spec)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Validate created resource
				vs := &istioclientv1beta1.VirtualService{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      tt.vsName,
					Namespace: tt.namespace,
				}, vs)
				require.NoError(t, err)
				
				if tt.validate != nil {
					tt.validate(t, vs)
				}
			}
		})
	}
}

func TestIstioManager_CreateDestinationRule(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = istioclientv1beta1.AddToScheme(scheme)
	
	tests := []struct {
		name      string
		namespace string
		drName    string
		spec      *servicemesh.DestinationRuleSpec
		wantErr   bool
		validate  func(t *testing.T, dr *istioclientv1beta1.DestinationRule)
	}{
		{
			name:      "create basic destination rule",
			namespace: "test-ns",
			drName:    "test-dr",
			spec: &servicemesh.DestinationRuleSpec{
				Host: "test-service",
				TrafficPolicy: &servicemesh.TrafficPolicy{
					LoadBalancer: &servicemesh.LoadBalancerSettings{
						Simple: "ROUND_ROBIN",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, dr *istioclientv1beta1.DestinationRule) {
				assert.Equal(t, "test-service", dr.Spec.Host)
				assert.NotNil(t, dr.Spec.TrafficPolicy)
				assert.NotNil(t, dr.Spec.TrafficPolicy.LoadBalancer)
			},
		},
		{
			name:      "create destination rule with subsets",
			namespace: "test-ns",
			drName:    "subset-dr",
			spec: &servicemesh.DestinationRuleSpec{
				Host: "myapp",
				Subsets: []servicemesh.Subset{
					{
						Name:   "stable",
						Labels: map[string]string{"version": "stable"},
					},
					{
						Name:   "canary",
						Labels: map[string]string{"version": "canary"},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, dr *istioclientv1beta1.DestinationRule) {
				assert.Len(t, dr.Spec.Subsets, 2)
				assert.Equal(t, "stable", dr.Spec.Subsets[0].Name)
				assert.Equal(t, "canary", dr.Spec.Subsets[1].Name)
			},
		},
		{
			name:      "create destination rule with circuit breaker",
			namespace: "test-ns",
			drName:    "cb-dr",
			spec: &servicemesh.DestinationRuleSpec{
				Host: "cb-service",
				TrafficPolicy: &servicemesh.TrafficPolicy{
					ConnectionPool: &servicemesh.ConnectionPoolSettings{
						TCP: &servicemesh.TCPSettings{
							MaxConnections: 100,
							ConnectTimeout: "30s",
						},
					},
					OutlierDetection: &servicemesh.OutlierDetection{
						ConsecutiveErrors:  5,
						Interval:           "30s",
						BaseEjectionTime:   "30s",
						MaxEjectionPercent: 50,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, dr *istioclientv1beta1.DestinationRule) {
				assert.NotNil(t, dr.Spec.TrafficPolicy.ConnectionPool)
				assert.NotNil(t, dr.Spec.TrafficPolicy.OutlierDetection)
				assert.Equal(t, int32(100), dr.Spec.TrafficPolicy.ConnectionPool.Tcp.MaxConnections)
				assert.Equal(t, int32(5), dr.Spec.TrafficPolicy.OutlierDetection.ConsecutiveErrors)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()
			
			m := NewIstioManager(client)
			err := m.CreateDestinationRule(context.Background(), tt.namespace, tt.drName, tt.spec)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Validate created resource
				dr := &istioclientv1beta1.DestinationRule{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      tt.drName,
					Namespace: tt.namespace,
				}, dr)
				require.NoError(t, err)
				
				if tt.validate != nil {
					tt.validate(t, dr)
				}
			}
		})
	}
}

func TestIstioManager_CreatePeerAuthentication(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = istioclientv1beta1security.AddToScheme(scheme)
	
	tests := []struct {
		name      string
		namespace string
		paName    string
		spec      *servicemesh.PeerAuthenticationSpec
		wantErr   bool
		validate  func(t *testing.T, pa *istioclientv1beta1security.PeerAuthentication)
	}{
		{
			name:      "create strict mTLS",
			namespace: "test-ns",
			paName:    "strict-mtls",
			spec: &servicemesh.PeerAuthenticationSpec{
				MutualTLS: &servicemesh.MutualTLS{
					Mode: "STRICT",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, pa *istioclientv1beta1security.PeerAuthentication) {
				assert.NotNil(t, pa.Spec.Mtls)
			},
		},
		{
			name:      "create workload-specific mTLS",
			namespace: "test-ns",
			paName:    "workload-mtls",
			spec: &servicemesh.PeerAuthenticationSpec{
				Selector: &servicemesh.WorkloadSelector{
					MatchLabels: map[string]string{
						"app": "myapp",
					},
				},
				MutualTLS: &servicemesh.MutualTLS{
					Mode: "PERMISSIVE",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, pa *istioclientv1beta1security.PeerAuthentication) {
				assert.NotNil(t, pa.Spec.Selector)
				assert.Equal(t, "myapp", pa.Spec.Selector.MatchLabels["app"])
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()
			
			m := NewIstioManager(client)
			err := m.CreatePeerAuthentication(context.Background(), tt.namespace, tt.paName, tt.spec)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Validate created resource
				pa := &istioclientv1beta1security.PeerAuthentication{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      tt.paName,
					Namespace: tt.namespace,
				}, pa)
				require.NoError(t, err)
				
				if tt.validate != nil {
					tt.validate(t, pa)
				}
			}
		})
	}
}

func TestIstioManager_GetStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = istioclientv1beta1security.AddToScheme(scheme)
	_ = istioclientv1alpha1telemetry.AddToScheme(scheme)
	
	tests := []struct {
		name      string
		namespace string
		objects   []runtime.Object
		want      *servicemesh.ServiceMeshStatus
	}{
		{
			name:      "namespace without injection",
			namespace: "test-ns",
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			want: &servicemesh.ServiceMeshStatus{
				Ready:      false,
				Message:    "Sidecar injection not enabled",
				Components: make(map[string]servicemesh.ComponentStatus),
			},
		},
		{
			name:      "namespace with injection enabled",
			namespace: "test-ns",
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
						Labels: map[string]string{
							"istio-injection": "enabled",
						},
					},
				},
			},
			want: &servicemesh.ServiceMeshStatus{
				Ready:      true,
				Message:    "Istio is configured",
				Components: make(map[string]servicemesh.ComponentStatus),
			},
		},
		{
			name:      "namespace with complete configuration",
			namespace: "test-ns",
			objects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
						Labels: map[string]string{
							"istio-injection": "enabled",
						},
					},
				},
				&istioclientv1beta1security.PeerAuthentication{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "test-ns",
					},
				},
				&istioclientv1alpha1telemetry.Telemetry{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "test-ns",
					},
				},
			},
			want: &servicemesh.ServiceMeshStatus{
				Ready:   true,
				Message: "Istio is configured",
				Components: map[string]servicemesh.ComponentStatus{
					"mTLS": {
						Ready:   true,
						Message: "PeerAuthentication configured",
					},
					"telemetry": {
						Ready:   true,
						Message: "Telemetry configured",
					},
				},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.objects...).
				Build()
			
			m := NewIstioManager(client)
			got, err := m.GetStatus(context.Background(), tt.namespace)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.want.Ready, got.Ready)
			assert.Equal(t, tt.want.Message, got.Message)
			assert.Equal(t, len(tt.want.Components), len(got.Components))
			
			for k, v := range tt.want.Components {
				assert.Equal(t, v.Ready, got.Components[k].Ready)
				assert.Equal(t, v.Message, got.Components[k].Message)
			}
		})
	}
}
