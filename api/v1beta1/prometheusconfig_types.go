/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PrometheusConfigSpec defines the desired state of PrometheusConfig
type PrometheusConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// TargetPlatform references the ObservabilityPlatform this config applies to
	TargetPlatform corev1.LocalObjectReference `json:"targetPlatform"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Paused indicates whether this configuration should be applied
	Paused bool `json:"paused,omitempty"`

	// +kubebuilder:validation:Optional
	// ServiceDiscovery defines service discovery configurations
	ServiceDiscovery *ServiceDiscoveryConfig `json:"serviceDiscovery,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteWrite defines remote write endpoints
	RemoteWrite []RemoteWriteEndpoint `json:"remoteWrite,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteRead defines remote read endpoints
	RemoteRead []RemoteReadEndpoint `json:"remoteRead,omitempty"`

	// +kubebuilder:validation:Optional
	// RecordingRules defines recording rules
	RecordingRules []RecordingRuleGroup `json:"recordingRules,omitempty"`

	// +kubebuilder:validation:Optional
	// AlertingRules defines alerting rules
	AlertingRules []PrometheusAlertingRuleGroup `json:"alertingRules,omitempty"`

	// +kubebuilder:validation:Optional
	// ScrapeConfigs defines additional scrape configurations
	ScrapeConfigs []ScrapeConfig `json:"scrapeConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// GlobalConfig defines global Prometheus configuration
	GlobalConfig *GlobalPrometheusConfig `json:"globalConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// StorageConfig defines storage-related configuration
	StorageConfig *PrometheusStorageConfig `json:"storageConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// TracingConfig defines distributed tracing configuration
	TracingConfig *PrometheusTracingConfig `json:"tracingConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ExemplarConfig defines exemplar storage configuration
	ExemplarConfig *ExemplarConfig `json:"exemplarConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryConfig defines query-related configuration
	QueryConfig *QueryConfig `json:"queryConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// WALConfig defines Write-Ahead Log configuration
	WALConfig *WALConfig `json:"walConfig,omitempty"`
}

// ServiceDiscoveryConfig defines service discovery configurations
type ServiceDiscoveryConfig struct {
	// +kubebuilder:validation:Optional
	// KubernetesSD defines Kubernetes service discovery
	KubernetesSD []KubernetesSDConfig `json:"kubernetesSD,omitempty"`

	// +kubebuilder:validation:Optional
	// ConsulSD defines Consul service discovery
	ConsulSD []ConsulSDConfig `json:"consulSD,omitempty"`

	// +kubebuilder:validation:Optional
	// DNSSD defines DNS service discovery
	DNSSD []DNSSDConfig `json:"dnsSD,omitempty"`

	// +kubebuilder:validation:Optional
	// FileSD defines file-based service discovery
	FileSD []FileSDConfig `json:"fileSD,omitempty"`

	// +kubebuilder:validation:Optional
	// StaticConfigs defines static target configurations
	StaticConfigs []StaticConfig `json:"staticConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// EC2SD defines AWS EC2 service discovery
	EC2SD []EC2SDConfig `json:"ec2SD,omitempty"`

	// +kubebuilder:validation:Optional
	// AzureSD defines Azure service discovery
	AzureSD []AzureSDConfig `json:"azureSD,omitempty"`

	// +kubebuilder:validation:Optional
	// GCESD defines Google Compute Engine service discovery
	GCESD []GCESDConfig `json:"gceSD,omitempty"`
}

// KubernetesSDConfig defines Kubernetes service discovery configuration
type KubernetesSDConfig struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=node;service;pod;endpoints;endpointslice;ingress
	// Role defines the Kubernetes objects to discover
	Role string `json:"role"`

	// +kubebuilder:validation:Optional
	// Namespaces to discover targets from
	Namespaces *NamespaceDiscovery `json:"namespaces,omitempty"`

	// +kubebuilder:validation:Optional
	// Selectors to filter discovered targets
	Selectors []K8SSelector `json:"selectors,omitempty"`

	// +kubebuilder:validation:Optional
	// KubeConfig for out-of-cluster discovery
	KubeConfig string `json:"kubeConfig,omitempty"`
}

// NamespaceDiscovery defines namespace discovery configuration
type NamespaceDiscovery struct {
	// +kubebuilder:validation:Optional
	// Names lists specific namespaces
	Names []string `json:"names,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// IncludeOwnNamespace includes the namespace where Prometheus is running
	IncludeOwnNamespace bool `json:"includeOwnNamespace,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// AllNamespaces discovers from all namespaces
	AllNamespaces bool `json:"allNamespaces,omitempty"`
}

// K8SSelector defines Kubernetes label/field selectors
type K8SSelector struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=service;pod;node
	// Role to apply selector to
	Role string `json:"role,omitempty"`

	// +kubebuilder:validation:Optional
	// Label selector
	Label string `json:"label,omitempty"`

	// +kubebuilder:validation:Optional
	// Field selector
	Field string `json:"field,omitempty"`
}

// ConsulSDConfig defines Consul service discovery configuration
type ConsulSDConfig struct {
	// +kubebuilder:validation:Required
	// Server is the Consul server address
	Server string `json:"server"`

	// +kubebuilder:validation:Optional
	// Token for authentication
	Token string `json:"token,omitempty"`

	// +kubebuilder:validation:Optional
	// Datacenter to query
	Datacenter string `json:"datacenter,omitempty"`

	// +kubebuilder:validation:Optional
	// Services to discover
	Services []string `json:"services,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags to filter services
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// RefreshInterval for re-discovering services
	RefreshInterval string `json:"refreshInterval,omitempty"`
}

// DNSSDConfig defines DNS service discovery configuration
type DNSSDConfig struct {
	// +kubebuilder:validation:Required
	// Names to query
	Names []string `json:"names"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=A;AAAA;SRV
	// +kubebuilder:default="A"
	// Type of DNS query
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// Port to use for discovered hosts
	Port int32 `json:"port,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// RefreshInterval for DNS queries
	RefreshInterval string `json:"refreshInterval,omitempty"`
}

// FileSDConfig defines file-based service discovery configuration
type FileSDConfig struct {
	// +kubebuilder:validation:Required
	// Files containing target groups
	Files []string `json:"files"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// RefreshInterval for re-reading files
	RefreshInterval string `json:"refreshInterval,omitempty"`
}

// StaticConfig defines static target configuration
type StaticConfig struct {
	// +kubebuilder:validation:Required
	// Targets list
	Targets []string `json:"targets"`

	// +kubebuilder:validation:Optional
	// Labels to add to all targets
	Labels map[string]string `json:"labels,omitempty"`
}

// EC2SDConfig defines AWS EC2 service discovery configuration
type EC2SDConfig struct {
	// +kubebuilder:validation:Required
	// Region to discover instances from
	Region string `json:"region"`

	// +kubebuilder:validation:Optional
	// AccessKey for AWS authentication
	AccessKey string `json:"accessKey,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretKey for AWS authentication
	SecretKey corev1.SecretKeySelector `json:"secretKey,omitempty"`

	// +kubebuilder:validation:Optional
	// RoleARN for IAM role assumption
	RoleARN string `json:"roleArn,omitempty"`

	// +kubebuilder:validation:Optional
	// Port to scrape metrics from
	Port int32 `json:"port,omitempty"`

	// +kubebuilder:validation:Optional
	// Filters to apply
	Filters []EC2Filter `json:"filters,omitempty"`
}

// EC2Filter defines EC2 instance filters
type EC2Filter struct {
	// +kubebuilder:validation:Required
	// Name of the filter
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Values for the filter
	Values []string `json:"values"`
}

// AzureSDConfig defines Azure service discovery configuration
type AzureSDConfig struct {
	// +kubebuilder:validation:Required
	// SubscriptionID for Azure
	SubscriptionID string `json:"subscriptionId"`

	// +kubebuilder:validation:Required
	// TenantID for Azure AD
	TenantID string `json:"tenantId"`

	// +kubebuilder:validation:Required
	// ClientID for authentication
	ClientID string `json:"clientId"`

	// +kubebuilder:validation:Required
	// ClientSecret for authentication
	ClientSecret corev1.SecretKeySelector `json:"clientSecret"`

	// +kubebuilder:validation:Optional
	// ResourceGroup to discover VMs from
	ResourceGroup string `json:"resourceGroup,omitempty"`

	// +kubebuilder:validation:Optional
	// Port to scrape metrics from
	Port int32 `json:"port,omitempty"`
}

// GCESDConfig defines Google Compute Engine service discovery configuration
type GCESDConfig struct {
	// +kubebuilder:validation:Required
	// Project ID
	Project string `json:"project"`

	// +kubebuilder:validation:Required
	// Zone to discover instances from
	Zone string `json:"zone"`

	// +kubebuilder:validation:Optional
	// Port to scrape metrics from
	Port int32 `json:"port,omitempty"`

	// +kubebuilder:validation:Optional
	// Filter to apply
	Filter string `json:"filter,omitempty"`

	// +kubebuilder:validation:Optional
	// TagSeparator for concatenating tags
	TagSeparator string `json:"tagSeparator,omitempty"`
}

// RemoteWriteEndpoint defines a remote write endpoint configuration
type RemoteWriteEndpoint struct {
	// +kubebuilder:validation:Required
	// URL of the remote write endpoint
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// RemoteTimeout for requests
	RemoteTimeout string `json:"remoteTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// Headers to add to requests
	Headers map[string]string `json:"headers,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuth configuration
	BasicAuth *BasicAuthConfig `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerToken for authentication
	BearerToken string `json:"bearerToken,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerTokenFile for authentication
	BearerTokenFile string `json:"bearerTokenFile,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSConfig for the endpoint
	TLSConfig *TLSConfigSpec `json:"tlsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ProxyURL for the endpoint
	ProxyURL string `json:"proxyUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// QueueConfig for tuning the remote write queue
	QueueConfig *QueueConfig `json:"queueConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// MetadataConfig for sending metadata
	MetadataConfig *MetadataConfig `json:"metadataConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// WriteRelabelConfigs to apply before sending
	WriteRelabelConfigs []RelabelConfig `json:"writeRelabelConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// Name of the remote write config
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SendExemplars enables sending of exemplars
	SendExemplars bool `json:"sendExemplars,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SendNativeHistograms enables sending of native histograms
	SendNativeHistograms bool `json:"sendNativeHistograms,omitempty"`
}

// RemoteReadEndpoint defines a remote read endpoint configuration
type RemoteReadEndpoint struct {
	// +kubebuilder:validation:Required
	// URL of the remote read endpoint
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// Name of the remote read config
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	// RequiredMatchers that must be present in queries
	RequiredMatchers map[string]string `json:"requiredMatchers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// RemoteTimeout for requests
	RemoteTimeout string `json:"remoteTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// Headers to add to requests
	Headers map[string]string `json:"headers,omitempty"`

	// +kubebuilder:validation:Optional
	// ReadRecent allows reading recent samples
	ReadRecent bool `json:"readRecent,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuth configuration
	BasicAuth *BasicAuthConfig `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerToken for authentication
	BearerToken string `json:"bearerToken,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerTokenFile for authentication
	BearerTokenFile string `json:"bearerTokenFile,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSConfig for the endpoint
	TLSConfig *TLSConfigSpec `json:"tlsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ProxyURL for the endpoint
	ProxyURL string `json:"proxyUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// FilterExternalLabels to remove before querying
	FilterExternalLabels bool `json:"filterExternalLabels,omitempty"`
}

// BasicAuthConfig defines basic authentication configuration
type BasicAuthConfig struct {
	// +kubebuilder:validation:Required
	// Username for basic auth
	Username string `json:"username"`

	// +kubebuilder:validation:Optional
	// Password for basic auth
	Password corev1.SecretKeySelector `json:"password,omitempty"`

	// +kubebuilder:validation:Optional
	// PasswordFile for basic auth
	PasswordFile string `json:"passwordFile,omitempty"`
}

// TLSConfigSpec defines TLS configuration
type TLSConfigSpec struct {
	// +kubebuilder:validation:Optional
	// CA certificate
	CA string `json:"ca,omitempty"`

	// +kubebuilder:validation:Optional
	// CAFile path
	CAFile string `json:"caFile,omitempty"`

	// +kubebuilder:validation:Optional
	// Cert certificate
	Cert string `json:"cert,omitempty"`

	// +kubebuilder:validation:Optional
	// CertFile path
	CertFile string `json:"certFile,omitempty"`

	// +kubebuilder:validation:Optional
	// Key for the certificate
	Key corev1.SecretKeySelector `json:"key,omitempty"`

	// +kubebuilder:validation:Optional
	// KeyFile path
	KeyFile string `json:"keyFile,omitempty"`

	// +kubebuilder:validation:Optional
	// ServerName for SNI
	ServerName string `json:"serverName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// InsecureSkipVerify disables verification
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// +kubebuilder:validation:Optional
	// MinVersion of TLS
	MinVersion string `json:"minVersion,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxVersion of TLS
	MaxVersion string `json:"maxVersion,omitempty"`
}

// QueueConfig defines remote write queue configuration
type QueueConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10000
	// Capacity of the queue
	Capacity int32 `json:"capacity,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// MaxShards maximum number of shards
	MaxShards int32 `json:"maxShards,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// MinShards minimum number of shards
	MinShards int32 `json:"minShards,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=100
	// MaxSamplesPerSend per batch
	MaxSamplesPerSend int32 `json:"maxSamplesPerSend,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5s"
	// BatchSendDeadline time limit for batch sending
	BatchSendDeadline string `json:"batchSendDeadline,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// MinBackoff for retries
	MinBackoff string `json:"minBackoff,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// MaxBackoff for retries
	MaxBackoff string `json:"maxBackoff,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// RetryOnRateLimit enables retry on 429
	RetryOnRateLimit int32 `json:"retryOnRateLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0.2
	// SampleAgeLimit for dropping old samples (0 = no limit)
	SampleAgeLimit string `json:"sampleAgeLimit,omitempty"`
}

// MetadataConfig defines metadata sending configuration
type MetadataConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Send enables metadata sending
	Send bool `json:"send,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// SendInterval for metadata
	SendInterval string `json:"sendInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// MaxSamplesPerSend for metadata
	MaxSamplesPerSend int32 `json:"maxSamplesPerSend,omitempty"`
}

// RelabelConfig defines relabeling configuration
type RelabelConfig struct {
	// +kubebuilder:validation:Optional
	// SourceLabels to concatenate
	SourceLabels []string `json:"sourceLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=";"
	// Separator for concatenation
	Separator string `json:"separator,omitempty"`

	// +kubebuilder:validation:Optional
	// TargetLabel to populate
	TargetLabel string `json:"targetLabel,omitempty"`

	// +kubebuilder:validation:Optional
	// Regex to match against
	Regex string `json:"regex,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="$1"
	// Replacement value
	Replacement string `json:"replacement,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=replace;keep;drop;hashmod;labelmap;labeldrop;labelkeep
	// +kubebuilder:default="replace"
	// Action to perform
	Action string `json:"action,omitempty"`

	// +kubebuilder:validation:Optional
	// Modulus for hashmod action
	Modulus uint64 `json:"modulus,omitempty"`
}

// RecordingRuleGroup defines a group of recording rules
type RecordingRuleGroup struct {
	// +kubebuilder:validation:Required
	// Name of the rule group
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// Interval at which to evaluate rules
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// Limit for rule group
	Limit int32 `json:"limit,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// Rules in this group
	Rules []RecordingRule `json:"rules"`
}

// RecordingRule defines a single recording rule
type RecordingRule struct {
	// +kubebuilder:validation:Required
	// Record name for the rule
	Record string `json:"record"`

	// +kubebuilder:validation:Required
	// Expr is the PromQL expression
	Expr string `json:"expr"`

	// +kubebuilder:validation:Optional
	// Labels to add to the recorded metric
	Labels map[string]string `json:"labels,omitempty"`
}

// PrometheusAlertingRuleGroup defines a group of alerting rules
type PrometheusAlertingRuleGroup struct {
	// +kubebuilder:validation:Required
	// Name of the rule group
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// Interval at which to evaluate rules
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// Limit for rule group
	Limit int32 `json:"limit,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// Rules in this group
	Rules []PrometheusAlertingRule `json:"rules"`
}

// PrometheusAlertingRule defines a single alerting rule
type PrometheusAlertingRule struct {
	// +kubebuilder:validation:Required
	// Alert name
	Alert string `json:"alert"`

	// +kubebuilder:validation:Required
	// Expr is the PromQL expression
	Expr string `json:"expr"`

	// +kubebuilder:validation:Optional
	// For duration before firing
	For string `json:"for,omitempty"`

	// +kubebuilder:validation:Optional
	// KeepFiringFor duration after resolution
	KeepFiringFor string `json:"keepFiringFor,omitempty"`

	// +kubebuilder:validation:Optional
	// Labels to add to the alert
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	// Annotations to add to the alert
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ScrapeConfig defines a scrape configuration
type ScrapeConfig struct {
	// +kubebuilder:validation:Required
	// JobName for the scrape config
	JobName string `json:"jobName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// ScrapeInterval for this job
	ScrapeInterval string `json:"scrapeInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// ScrapeTimeout for this job
	ScrapeTimeout string `json:"scrapeTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="/metrics"
	// MetricsPath to scrape
	MetricsPath string `json:"metricsPath,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=http;https
	// +kubebuilder:default="http"
	// Scheme to use
	Scheme string `json:"scheme,omitempty"`

	// +kubebuilder:validation:Optional
	// Params to add to scrape requests
	Params map[string][]string `json:"params,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuth configuration
	BasicAuth *BasicAuthConfig `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerToken for authentication
	BearerToken string `json:"bearerToken,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerTokenFile for authentication
	BearerTokenFile string `json:"bearerTokenFile,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSConfig for the scrape
	TLSConfig *TLSConfigSpec `json:"tlsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ProxyURL for the scrape
	ProxyURL string `json:"proxyUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// HonorLabels controls label conflicts
	HonorLabels bool `json:"honorLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// HonorTimestamps controls timestamp handling
	HonorTimestamps bool `json:"honorTimestamps,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// TrackTimestampsStaleness enables staleness tracking
	TrackTimestampsStaleness bool `json:"trackTimestampsStaleness,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// SampleLimit per scrape (0 = no limit)
	SampleLimit uint64 `json:"sampleLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// TargetLimit per scrape (0 = no limit)
	TargetLimit uint64 `json:"targetLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// LabelLimit per sample (0 = no limit)
	LabelLimit uint64 `json:"labelLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// LabelNameLengthLimit per sample (0 = no limit)
	LabelNameLengthLimit uint64 `json:"labelNameLengthLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// LabelValueLengthLimit per sample (0 = no limit)
	LabelValueLengthLimit uint64 `json:"labelValueLengthLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	// BodySizeLimit for scrape response (0 = no limit)
	BodySizeLimit string `json:"bodySizeLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// MetricRelabelConfigs to apply to scraped metrics
	MetricRelabelConfigs []RelabelConfig `json:"metricRelabelConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// RelabelConfigs to apply to targets
	RelabelConfigs []RelabelConfig `json:"relabelConfigs,omitempty"`

	// +kubebuilder:validation:Required
	// ServiceDiscovery configuration for this job
	ServiceDiscovery ServiceDiscoveryConfig `json:"serviceDiscovery"`
}

// GlobalPrometheusConfig defines global Prometheus configuration
type GlobalPrometheusConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// ScrapeInterval global default
	ScrapeInterval string `json:"scrapeInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	// ScrapeTimeout global default
	ScrapeTimeout string `json:"scrapeTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// EvaluationInterval for rules
	EvaluationInterval string `json:"evaluationInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// ExternalLabels to add to all metrics
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryLogFile to log queries to
	QueryLogFile string `json:"queryLogFile,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// SampleLimit global default (0 = no limit)
	SampleLimit uint64 `json:"sampleLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// TargetLimit global default (0 = no limit)
	TargetLimit uint64 `json:"targetLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// LabelLimit global default (0 = no limit)
	LabelLimit uint64 `json:"labelLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// LabelNameLengthLimit global default (0 = no limit)
	LabelNameLengthLimit uint64 `json:"labelNameLengthLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// LabelValueLengthLimit global default (0 = no limit)
	LabelValueLengthLimit uint64 `json:"labelValueLengthLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0"
	// BodySizeLimit global default (0 = no limit)
	BodySizeLimit string `json:"bodySizeLimit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	// QueryConcurrency maximum concurrent queries
	QueryConcurrency int32 `json:"queryConcurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2m"
	// QueryTimeout maximum query duration
	QueryTimeout string `json:"queryTimeout,omitempty"`
}

// PrometheusStorageConfig defines storage-related configuration
type PrometheusStorageConfig struct {
	// +kubebuilder:validation:Optional
	// TSDBConfig for time series database
	TSDBConfig *TSDBConfig `json:"tsdbConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ExemplarsConfig for exemplar storage
	ExemplarsConfig *ExemplarsStorageConfig `json:"exemplarsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// RemoteFlushDeadline for shutdown
	RemoteFlushDeadline string `json:"remoteFlushDeadline,omitempty"`
}

// TSDBConfig defines TSDB configuration
type TSDBConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="15d"
	// Retention duration
	Retention string `json:"retention,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0"
	// RetentionSize maximum (0 = no limit)
	RetentionSize string `json:"retentionSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// NoLockfile disables lock file
	NoLockfile bool `json:"noLockfile,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// AllowOverlappingBlocks enables overlapping blocks
	AllowOverlappingBlocks bool `json:"allowOverlappingBlocks,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// WALCompression enables WAL compression
	WALCompression bool `json:"walCompression,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=snappy;zstd
	// WALCompressionType to use
	WALCompressionType string `json:"walCompressionType,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="512MB"
	// WALSegmentSize for segments
	WALSegmentSize string `json:"walSegmentSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2h"
	// MinBlockDuration for compaction
	MinBlockDuration string `json:"minBlockDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1d10m"
	// MaxBlockDuration for compaction
	MaxBlockDuration string `json:"maxBlockDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// MaxBlockChunkSegmentSize in MB
	MaxBlockChunkSegmentSize int32 `json:"maxBlockChunkSegmentSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// EnableNativeHistograms enables native histograms
	EnableNativeHistograms bool `json:"enableNativeHistograms,omitempty"`

	// +kubebuilder:validation:Optional
	// OutOfOrderTimeWindow for accepting out-of-order samples
	OutOfOrderTimeWindow string `json:"outOfOrderTimeWindow,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// OutOfOrderCapacity per metric (0 = default)
	OutOfOrderCapacity int32 `json:"outOfOrderCapacity,omitempty"`
}

// ExemplarsStorageConfig defines exemplar storage configuration
type ExemplarsStorageConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=100000
	// MaxExemplars to store
	MaxExemplars int64 `json:"maxExemplars,omitempty"`
}

// PrometheusTracingConfig defines distributed tracing configuration
type PrometheusTracingConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Enabled enables tracing
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// Endpoint for tracing backend
	Endpoint string `json:"endpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=grpc;http
	// +kubebuilder:default="grpc"
	// ClientType for tracing
	ClientType string `json:"clientType,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0.01
	// SamplingFraction for traces
	SamplingFraction float64 `json:"samplingFraction,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Insecure disables TLS
	Insecure bool `json:"insecure,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSConfig for secure connections
	TLSConfig *TLSConfigSpec `json:"tlsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Headers to add to traces
	Headers map[string]string `json:"headers,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=none;gzip
	// +kubebuilder:default="none"
	// Compression to use
	Compression string `json:"compression,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5s"
	// Timeout for trace export
	Timeout string `json:"timeout,omitempty"`
}

// ExemplarConfig defines exemplar configuration
type ExemplarConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled enables exemplar storage
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=100000
	// MaxExemplars to store globally
	MaxExemplars int64 `json:"maxExemplars,omitempty"`
}

// QueryConfig defines query-related configuration
type QueryConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=50000000
	// MaxSamples maximum samples a query can load
	MaxSamples int64 `json:"maxSamples,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="2m"
	// Timeout for queries
	Timeout string `json:"timeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=20
	// MaxConcurrency for queries
	MaxConcurrency int32 `json:"maxConcurrency,omitempty"`

	// +kubebuilder:validation:Optional
	// LookbackDelta for instant queries
	LookbackDelta string `json:"lookbackDelta,omitempty"`
}

// WALConfig defines Write-Ahead Log configuration
type WALConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Compression enables WAL compression
	Compression bool `json:"compression,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=snappy;zstd
	// +kubebuilder:default="snappy"
	// CompressionType to use
	CompressionType string `json:"compressionType,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="128MB"
	// SegmentSize for WAL segments
	SegmentSize string `json:"segmentSize,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// TruncateFrequency enables periodic truncation
	TruncateFrequency string `json:"truncateFrequency,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="4h"
	// MinTime to keep in WAL
	MinTime string `json:"minTime,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="19h"
	// MaxTime to keep in WAL
	MaxTime string `json:"maxTime,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// NoLockfile disables lock file
	NoLockfile bool `json:"noLockfile,omitempty"`
}

// PrometheusConfigStatus defines the observed state of PrometheusConfig
type PrometheusConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Enum=Pending;Applied;Failed;Invalid
	// Phase represents the current phase of the configuration
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAppliedTime is when the configuration was last applied
	LastAppliedTime *metav1.Time `json:"lastAppliedTime,omitempty"`

	// LastAppliedHash is the hash of the last applied configuration
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`

	// ValidationErrors contains any validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// SyncStatus shows the synchronization status with target platform
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`
}

// SyncStatus defines synchronization status
type SyncStatus struct {
	// +kubebuilder:validation:Enum=Synced;OutOfSync;Unknown
	// State of synchronization
	State string `json:"state"`

	// LastSyncTime when last synced
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Reason for current state
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=promconfig;promconfigs,categories={observability,prometheus}
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetPlatform.name`,description="Target ObservabilityPlatform"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="Current phase"
// +kubebuilder:printcolumn:name="Sync",type=string,JSONPath=`.status.syncStatus.state`,description="Sync status"
// +kubebuilder:printcolumn:name="Last Applied",type=date,JSONPath=`.status.lastAppliedTime`,description="Last applied time"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Time since creation"

// PrometheusConfig is the Schema for the prometheusconfigs API
type PrometheusConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrometheusConfigSpec   `json:"spec,omitempty"`
	Status PrometheusConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PrometheusConfigList contains a list of PrometheusConfig
type PrometheusConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrometheusConfig `json:"items"`
}

// Hub marks this type as a conversion hub.
func (*PrometheusConfig) Hub() {}

func init() {
	SchemeBuilder.Register(&PrometheusConfig{}, &PrometheusConfigList{})
}
