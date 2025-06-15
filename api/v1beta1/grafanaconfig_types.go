/*
Copyright 2025 Gunjan Jalori.

Licensed under the MIT License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GrafanaConfigSpec defines the desired state of GrafanaConfig
type GrafanaConfigSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates whether this GrafanaConfig should be applied
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// TargetRef specifies which ObservabilityPlatform this config applies to
	TargetRef ObservabilityPlatformReference `json:"targetRef,omitempty"`

	// +kubebuilder:validation:Optional
	// Server contains Grafana server configuration
	Server *GrafanaServerConfig `json:"server,omitempty"`

	// +kubebuilder:validation:Optional
	// Security contains security-related configuration
	Security *GrafanaSecurityConfig `json:"security,omitempty"`

	// +kubebuilder:validation:Optional
	// Auth contains authentication configuration
	Auth *GrafanaAuthConfig `json:"auth,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSources contains data source configurations
	DataSources []GrafanaDataSource `json:"dataSources,omitempty"`

	// +kubebuilder:validation:Optional
	// Dashboards contains dashboard provisioning configuration
	Dashboards *GrafanaDashboardConfig `json:"dashboards,omitempty"`

	// +kubebuilder:validation:Optional
	// Plugins contains plugin management configuration
	Plugins *GrafanaPluginConfig `json:"plugins,omitempty"`

	// +kubebuilder:validation:Optional
	// Notifications contains notification channel configurations
	Notifications []GrafanaNotificationChannel `json:"notifications,omitempty"`

	// +kubebuilder:validation:Optional
	// Organizations contains organization management configuration
	Organizations []GrafanaOrganization `json:"organizations,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTP contains email configuration
	SMTP *GrafanaSMTPConfig `json:"smtp,omitempty"`

	// +kubebuilder:validation:Optional
	// Analytics contains analytics and reporting configuration
	Analytics *GrafanaAnalyticsConfig `json:"analytics,omitempty"`

	// +kubebuilder:validation:Optional
	// ExternalImageStorage configures external image storage for sharing
	ExternalImageStorage *GrafanaImageStorageConfig `json:"externalImageStorage,omitempty"`
}

// GrafanaServerConfig contains Grafana server configuration
type GrafanaServerConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="0.0.0.0"
	// HTTPAddr is the IP address to bind to
	HTTPAddr string `json:"httpAddr,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=3000
	// HTTPPort is the port to bind to
	HTTPPort int32 `json:"httpPort,omitempty"`

	// +kubebuilder:validation:Optional
	// Protocol can be http, https, h2, socket
	// +kubebuilder:validation:Enum=http;https;h2;socket
	// +kubebuilder:default="http"
	Protocol string `json:"protocol,omitempty"`

	// +kubebuilder:validation:Optional
	// Domain is used for redirect URLs
	Domain string `json:"domain,omitempty"`

	// +kubebuilder:validation:Optional
	// RootURL is the full URL for accessing Grafana
	RootURL string `json:"rootUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// ServeFromSubPath enables serving from a sub-path
	ServeFromSubPath bool `json:"serveFromSubPath,omitempty"`

	// +kubebuilder:validation:Optional
	// RouterLogging enables router logging
	RouterLogging bool `json:"routerLogging,omitempty"`

	// +kubebuilder:validation:Optional
	// EnableGzip enables gzip compression
	EnableGzip bool `json:"enableGzip,omitempty"`
}

// GrafanaSecurityConfig contains security-related configuration
type GrafanaSecurityConfig struct {
	// +kubebuilder:validation:Optional
	// AdminUser is the default admin username
	AdminUser string `json:"adminUser,omitempty"`

	// +kubebuilder:validation:Optional
	// AdminPassword is the default admin password (will be stored in secret)
	AdminPasswordSecret *corev1.SecretKeySelector `json:"adminPasswordSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretKey is used for signing (will be stored in secret)
	SecretKeySecret *corev1.SecretKeySelector `json:"secretKeySecret,omitempty"`

	// +kubebuilder:validation:Optional
	// DisableGravatar disables gravatar profile images
	DisableGravatar bool `json:"disableGravatar,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSourceProxyWhitelist is a comma-separated list of IP/domain patterns
	DataSourceProxyWhitelist string `json:"dataSourceProxyWhitelist,omitempty"`

	// +kubebuilder:validation:Optional
	// CookieSecure sets the secure flag on cookies
	CookieSecure bool `json:"cookieSecure,omitempty"`

	// +kubebuilder:validation:Optional
	// CookieSameSite sets the SameSite attribute
	// +kubebuilder:validation:Enum=lax;strict;none
	CookieSameSite string `json:"cookieSameSite,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowEmbedding allows embedding Grafana
	AllowEmbedding bool `json:"allowEmbedding,omitempty"`

	// +kubebuilder:validation:Optional
	// StrictTransportSecurity enables HSTS
	StrictTransportSecurity bool `json:"strictTransportSecurity,omitempty"`

	// +kubebuilder:validation:Optional
	// StrictTransportSecurityMaxAge sets HSTS max age
	StrictTransportSecurityMaxAge int32 `json:"strictTransportSecurityMaxAge,omitempty"`
}

// GrafanaAuthConfig contains authentication configuration
type GrafanaAuthConfig struct {
	// +kubebuilder:validation:Optional
	// DisableLoginForm disables the login form
	DisableLoginForm bool `json:"disableLoginForm,omitempty"`

	// +kubebuilder:validation:Optional
	// DisableSignoutMenu disables the signout menu
	DisableSignoutMenu bool `json:"disableSignoutMenu,omitempty"`

	// +kubebuilder:validation:Optional
	// SignoutRedirectURL is the URL to redirect after signout
	SignoutRedirectURL string `json:"signoutRedirectUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// OAuthAutoLogin enables automatic OAuth login
	OAuthAutoLogin bool `json:"oauthAutoLogin,omitempty"`

	// +kubebuilder:validation:Optional
	// Anonymous enables anonymous access
	Anonymous *GrafanaAnonymousAuth `json:"anonymous,omitempty"`

	// +kubebuilder:validation:Optional
	// Basic enables basic authentication
	Basic *GrafanaBasicAuth `json:"basic,omitempty"`

	// +kubebuilder:validation:Optional
	// LDAP enables LDAP authentication
	LDAP *GrafanaLDAPAuth `json:"ldap,omitempty"`

	// +kubebuilder:validation:Optional
	// OAuth enables OAuth authentication
	OAuth *GrafanaOAuthConfig `json:"oauth,omitempty"`

	// +kubebuilder:validation:Optional
	// SAML enables SAML authentication
	SAML *GrafanaSAMLConfig `json:"saml,omitempty"`
}

// GrafanaAnonymousAuth configures anonymous access
type GrafanaAnonymousAuth struct {
	// +kubebuilder:validation:Optional
	// Enabled enables anonymous access
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// OrgName is the organization name for anonymous users
	OrgName string `json:"orgName,omitempty"`

	// +kubebuilder:validation:Optional
	// OrgRole is the role for anonymous users
	// +kubebuilder:validation:Enum=Viewer;Editor;Admin
	OrgRole string `json:"orgRole,omitempty"`
}

// GrafanaBasicAuth configures basic authentication
type GrafanaBasicAuth struct {
	// +kubebuilder:validation:Optional
	// Enabled enables basic auth
	Enabled bool `json:"enabled,omitempty"`
}

// GrafanaLDAPAuth configures LDAP authentication
type GrafanaLDAPAuth struct {
	// +kubebuilder:validation:Optional
	// Enabled enables LDAP auth
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigSecret references a secret containing LDAP configuration
	ConfigSecret *corev1.SecretKeySelector `json:"configSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowSignUp allows LDAP users to sign up
	AllowSignUp bool `json:"allowSignUp,omitempty"`
}

// GrafanaOAuthConfig configures OAuth authentication
type GrafanaOAuthConfig struct {
	// +kubebuilder:validation:Optional
	// Providers contains OAuth provider configurations
	Providers []GrafanaOAuthProvider `json:"providers,omitempty"`
}

// GrafanaOAuthProvider configures an OAuth provider
type GrafanaOAuthProvider struct {
	// +kubebuilder:validation:Required
	// Name is the provider name (e.g., github, google, azure)
	// +kubebuilder:validation:Enum=github;gitlab;google;azuread;okta;generic_oauth
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// Enabled enables this provider
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// ClientID is the OAuth client ID
	ClientID string `json:"clientId,omitempty"`

	// +kubebuilder:validation:Optional
	// ClientSecret references the OAuth client secret
	ClientSecretRef *corev1.SecretKeySelector `json:"clientSecretRef,omitempty"`

	// +kubebuilder:validation:Optional
	// Scopes are the OAuth scopes
	Scopes []string `json:"scopes,omitempty"`

	// +kubebuilder:validation:Optional
	// AuthURL is the authorization URL
	AuthURL string `json:"authUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// TokenURL is the token URL
	TokenURL string `json:"tokenUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// APIURL is the API URL
	APIURL string `json:"apiUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowSignUp allows OAuth users to sign up
	AllowSignUp bool `json:"allowSignUp,omitempty"`

	// +kubebuilder:validation:Optional
	// RoleAttributePath is the JMESPath for role mapping
	RoleAttributePath string `json:"roleAttributePath,omitempty"`
}

// GrafanaSAMLConfig configures SAML authentication
type GrafanaSAMLConfig struct {
	// +kubebuilder:validation:Optional
	// Enabled enables SAML auth
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// CertificateSecret references the SAML certificate
	CertificateSecret *corev1.SecretKeySelector `json:"certificateSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// PrivateKeySecret references the SAML private key
	PrivateKeySecret *corev1.SecretKeySelector `json:"privateKeySecret,omitempty"`

	// +kubebuilder:validation:Optional
	// IdpMetadataURL is the IdP metadata URL
	IdpMetadataURL string `json:"idpMetadataUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// IdpMetadata is the raw IdP metadata
	IdpMetadata string `json:"idpMetadata,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxIssueDelay is the max issue delay
	MaxIssueDelay string `json:"maxIssueDelay,omitempty"`

	// +kubebuilder:validation:Optional
	// MetadataValidDuration is the metadata valid duration
	MetadataValidDuration string `json:"metadataValidDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// AssertionAttributeName is the assertion attribute name
	AssertionAttributeName string `json:"assertionAttributeName,omitempty"`

	// +kubebuilder:validation:Optional
	// AssertionAttributeLogin is the login attribute
	AssertionAttributeLogin string `json:"assertionAttributeLogin,omitempty"`

	// +kubebuilder:validation:Optional
	// AssertionAttributeEmail is the email attribute
	AssertionAttributeEmail string `json:"assertionAttributeEmail,omitempty"`

	// +kubebuilder:validation:Optional
	// AssertionAttributeGroups is the groups attribute
	AssertionAttributeGroups string `json:"assertionAttributeGroups,omitempty"`

	// +kubebuilder:validation:Optional
	// AssertionAttributeRole is the role attribute
	AssertionAttributeRole string `json:"assertionAttributeRole,omitempty"`

	// +kubebuilder:validation:Optional
	// AssertionAttributeOrg is the org attribute
	AssertionAttributeOrg string `json:"assertionAttributeOrg,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowIdpInitiated allows IdP initiated SSO
	AllowIdpInitiated bool `json:"allowIdpInitiated,omitempty"`
}

// GrafanaDataSource configures a data source
type GrafanaDataSource struct {
	// +kubebuilder:validation:Required
	// Name is the data source name
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Type is the data source type
	// +kubebuilder:validation:Enum=prometheus;loki;tempo;elasticsearch;influxdb;graphite;postgres;mysql;mssql;cloudwatch;azuremonitor;stackdriver
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="proxy"
	// Access is the access mode
	// +kubebuilder:validation:Enum=proxy;direct
	Access string `json:"access,omitempty"`

	// +kubebuilder:validation:Required
	// URL is the data source URL
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// OrgID is the organization ID
	OrgID int64 `json:"orgId,omitempty"`

	// +kubebuilder:validation:Optional
	// UID is the unique identifier
	UID string `json:"uid,omitempty"`

	// +kubebuilder:validation:Optional
	// Database is the database name (for SQL data sources)
	Database string `json:"database,omitempty"`

	// +kubebuilder:validation:Optional
	// User is the database user
	User string `json:"user,omitempty"`

	// +kubebuilder:validation:Optional
	// PasswordSecret references the password
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuth enables basic authentication
	BasicAuth bool `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuthUser is the basic auth username
	BasicAuthUser string `json:"basicAuthUser,omitempty"`

	// +kubebuilder:validation:Optional
	// BasicAuthPasswordSecret references the basic auth password
	BasicAuthPasswordSecret *corev1.SecretKeySelector `json:"basicAuthPasswordSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// WithCredentials sends credentials
	WithCredentials bool `json:"withCredentials,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// IsDefault marks as default data source
	IsDefault bool `json:"isDefault,omitempty"`

	// +kubebuilder:validation:Optional
	// JSONData contains type-specific configuration
	JSONData map[string]interface{} `json:"jsonData,omitempty"`

	// +kubebuilder:validation:Optional
	// SecureJSONDataSecret references sensitive configuration
	SecureJSONDataSecret *corev1.LocalObjectReference `json:"secureJsonDataSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Editable allows editing from the UI
	Editable bool `json:"editable,omitempty"`
}

// GrafanaDashboardConfig configures dashboard provisioning
type GrafanaDashboardConfig struct {
	// +kubebuilder:validation:Optional
	// Providers contains dashboard providers
	Providers []GrafanaDashboardProvider `json:"providers,omitempty"`
}

// GrafanaDashboardProvider configures a dashboard provider
type GrafanaDashboardProvider struct {
	// +kubebuilder:validation:Required
	// Name is the provider name
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// OrgID is the organization ID
	OrgID int64 `json:"orgId,omitempty"`

	// +kubebuilder:validation:Optional
	// Folder is the folder name
	Folder string `json:"folder,omitempty"`

	// +kubebuilder:validation:Optional
	// FolderUID is the folder UID
	FolderUID string `json:"folderUid,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="file"
	// Type is the provider type
	// +kubebuilder:validation:Enum=file;configmap
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// DisableDeletion prevents deletion
	DisableDeletion bool `json:"disableDeletion,omitempty"`

	// +kubebuilder:validation:Optional
	// UpdateIntervalSeconds is the update interval
	UpdateIntervalSeconds int64 `json:"updateIntervalSeconds,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowUIUpdates allows UI updates
	AllowUIUpdates bool `json:"allowUiUpdates,omitempty"`

	// +kubebuilder:validation:Optional
	// Options contains provider-specific options
	Options map[string]string `json:"options,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigMapRef references ConfigMaps containing dashboards
	ConfigMapRef *DashboardConfigMapReference `json:"configMapRef,omitempty"`
}

// DashboardConfigMapReference references ConfigMaps containing dashboards
type DashboardConfigMapReference struct {
	// +kubebuilder:validation:Optional
	// Name is the ConfigMap name pattern (supports wildcards)
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	// Namespace is the ConfigMap namespace
	Namespace string `json:"namespace,omitempty"`

	// +kubebuilder:validation:Optional
	// LabelSelector selects ConfigMaps by labels
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// Key is the ConfigMap key containing the dashboard (if not specified, all keys are used)
	Key string `json:"key,omitempty"`
}

// GrafanaPluginConfig configures plugin management
type GrafanaPluginConfig struct {
	// +kubebuilder:validation:Optional
	// InstallPlugins is a comma-separated list of plugins to install
	InstallPlugins []string `json:"installPlugins,omitempty"`

	// +kubebuilder:validation:Optional
	// AllowLoadingUnsignedPlugins is a comma-separated list of unsigned plugins to allow
	AllowLoadingUnsignedPlugins []string `json:"allowLoadingUnsignedPlugins,omitempty"`

	// +kubebuilder:validation:Optional
	// PluginCatalogURL is the plugin catalog URL
	PluginCatalogURL string `json:"pluginCatalogUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// PluginAdminEnabled enables plugin admin
	PluginAdminEnabled bool `json:"pluginAdminEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	// PluginAdminExternalManageEnabled enables external plugin management
	PluginAdminExternalManageEnabled bool `json:"pluginAdminExternalManageEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	// PluginSkipInstall skips plugin installation
	PluginSkipInstall bool `json:"pluginSkipInstall,omitempty"`
}

// GrafanaNotificationChannel configures a notification channel
type GrafanaNotificationChannel struct {
	// +kubebuilder:validation:Required
	// Name is the channel name
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Type is the channel type
	// +kubebuilder:validation:Enum=email;slack;pagerduty;webhook;telegram;teams;discord;googlechat;prometheus-alertmanager;opsgenie;threema
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// UID is the unique identifier
	UID string `json:"uid,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// OrgID is the organization ID
	OrgID int64 `json:"orgId,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// IsDefault marks as default channel
	IsDefault bool `json:"isDefault,omitempty"`

	// +kubebuilder:validation:Optional
	// SendReminder sends reminders
	SendReminder bool `json:"sendReminder,omitempty"`

	// +kubebuilder:validation:Optional
	// Frequency is the reminder frequency
	Frequency string `json:"frequency,omitempty"`

	// +kubebuilder:validation:Optional
	// DisableResolveMessage disables resolve messages
	DisableResolveMessage bool `json:"disableResolveMessage,omitempty"`

	// +kubebuilder:validation:Optional
	// Settings contains channel-specific settings
	Settings map[string]interface{} `json:"settings,omitempty"`

	// +kubebuilder:validation:Optional
	// SecureSettingsSecret references sensitive settings
	SecureSettingsSecret *corev1.LocalObjectReference `json:"secureSettingsSecret,omitempty"`
}

// GrafanaOrganization configures an organization
type GrafanaOrganization struct {
	// +kubebuilder:validation:Required
	// Name is the organization name
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// ID is the organization ID (auto-generated if not specified)
	ID int64 `json:"id,omitempty"`

	// +kubebuilder:validation:Optional
	// Users contains user assignments
	Users []GrafanaOrganizationUser `json:"users,omitempty"`

	// +kubebuilder:validation:Optional
	// Preferences contains organization preferences
	Preferences *GrafanaOrganizationPreferences `json:"preferences,omitempty"`
}

// GrafanaOrganizationUser assigns a user to an organization
type GrafanaOrganizationUser struct {
	// +kubebuilder:validation:Required
	// LoginOrEmail is the user login or email
	LoginOrEmail string `json:"loginOrEmail"`

	// +kubebuilder:validation:Required
	// Role is the user role in the organization
	// +kubebuilder:validation:Enum=Admin;Editor;Viewer
	Role string `json:"role"`
}

// GrafanaOrganizationPreferences contains organization preferences
type GrafanaOrganizationPreferences struct {
	// +kubebuilder:validation:Optional
	// Theme is the UI theme
	// +kubebuilder:validation:Enum=light;dark;system
	Theme string `json:"theme,omitempty"`

	// +kubebuilder:validation:Optional
	// HomeDashboardID is the home dashboard ID
	HomeDashboardID int64 `json:"homeDashboardId,omitempty"`

	// +kubebuilder:validation:Optional
	// Timezone is the default timezone
	Timezone string `json:"timezone,omitempty"`

	// +kubebuilder:validation:Optional
	// WeekStart is the week start day
	WeekStart string `json:"weekStart,omitempty"`
}

// GrafanaSMTPConfig configures SMTP settings
type GrafanaSMTPConfig struct {
	// +kubebuilder:validation:Optional
	// Enabled enables SMTP
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// Host is the SMTP server host:port
	Host string `json:"host,omitempty"`

	// +kubebuilder:validation:Optional
	// User is the SMTP username
	User string `json:"user,omitempty"`

	// +kubebuilder:validation:Optional
	// PasswordSecret references the SMTP password
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// CertFileSecret references the certificate file
	CertFileSecret *corev1.SecretKeySelector `json:"certFileSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// KeyFileSecret references the key file
	KeyFileSecret *corev1.SecretKeySelector `json:"keyFileSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// SkipVerify skips certificate verification
	SkipVerify bool `json:"skipVerify,omitempty"`

	// +kubebuilder:validation:Optional
	// FromAddress is the sender address
	FromAddress string `json:"fromAddress,omitempty"`

	// +kubebuilder:validation:Optional
	// FromName is the sender name
	FromName string `json:"fromName,omitempty"`

	// +kubebuilder:validation:Optional
	// EHLOIdentity is the EHLO identity
	EHLOIdentity string `json:"ehloIdentity,omitempty"`

	// +kubebuilder:validation:Optional
	// StartTLSPolicy is the STARTTLS policy
	// +kubebuilder:validation:Enum=OpportunisticStartTLS;MandatoryStartTLS;NoStartTLS
	StartTLSPolicy string `json:"startTlsPolicy,omitempty"`
}

// GrafanaAnalyticsConfig configures analytics and reporting
type GrafanaAnalyticsConfig struct {
	// +kubebuilder:validation:Optional
	// ReportingEnabled enables usage reporting
	ReportingEnabled bool `json:"reportingEnabled,omitempty"`

	// +kubebuilder:validation:Optional
	// GoogleAnalyticsID is the Google Analytics ID
	GoogleAnalyticsID string `json:"googleAnalyticsId,omitempty"`

	// +kubebuilder:validation:Optional
	// GoogleTagManagerID is the Google Tag Manager ID
	GoogleTagManagerID string `json:"googleTagManagerId,omitempty"`

	// +kubebuilder:validation:Optional
	// RudderstackWriteKeySecret references the Rudderstack write key
	RudderstackWriteKeySecret *corev1.SecretKeySelector `json:"rudderstackWriteKeySecret,omitempty"`

	// +kubebuilder:validation:Optional
	// RudderstackDataPlaneURL is the Rudderstack data plane URL
	RudderstackDataPlaneURL string `json:"rudderstackDataPlaneUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// RudderstackSDKURL is the Rudderstack SDK URL
	RudderstackSDKURL string `json:"rudderstackSdkUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// RudderstackConfigURL is the Rudderstack config URL
	RudderstackConfigURL string `json:"rudderstackConfigUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// IntercomSecretSecret references the Intercom secret
	IntercomSecretSecret *corev1.SecretKeySelector `json:"intercomSecretSecret,omitempty"`
}

// GrafanaImageStorageConfig configures external image storage
type GrafanaImageStorageConfig struct {
	// +kubebuilder:validation:Required
	// Provider is the storage provider
	// +kubebuilder:validation:Enum=s3;gcs;azure;local
	Provider string `json:"provider"`

	// +kubebuilder:validation:Optional
	// S3 contains S3-specific configuration
	S3 *GrafanaS3ImageStorage `json:"s3,omitempty"`

	// +kubebuilder:validation:Optional
	// GCS contains GCS-specific configuration
	GCS *GrafanaGCSImageStorage `json:"gcs,omitempty"`

	// +kubebuilder:validation:Optional
	// Azure contains Azure-specific configuration
	Azure *GrafanaAzureImageStorage `json:"azure,omitempty"`
}

// GrafanaS3ImageStorage configures S3 image storage
type GrafanaS3ImageStorage struct {
	// +kubebuilder:validation:Required
	// Bucket is the S3 bucket name
	Bucket string `json:"bucket"`

	// +kubebuilder:validation:Optional
	// Region is the AWS region
	Region string `json:"region,omitempty"`

	// +kubebuilder:validation:Optional
	// Path is the bucket path
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Optional
	// AccessKeySecret references the access key
	AccessKeySecret *corev1.SecretKeySelector `json:"accessKeySecret,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretKeySecret references the secret key
	SecretKeySecret *corev1.SecretKeySelector `json:"secretKeySecret,omitempty"`
}

// GrafanaGCSImageStorage configures GCS image storage
type GrafanaGCSImageStorage struct {
	// +kubebuilder:validation:Required
	// Bucket is the GCS bucket name
	Bucket string `json:"bucket"`

	// +kubebuilder:validation:Optional
	// Path is the bucket path
	Path string `json:"path,omitempty"`

	// +kubebuilder:validation:Optional
	// KeyFileSecret references the service account key
	KeyFileSecret *corev1.SecretKeySelector `json:"keyFileSecret,omitempty"`
}

// GrafanaAzureImageStorage configures Azure image storage
type GrafanaAzureImageStorage struct {
	// +kubebuilder:validation:Required
	// AccountName is the storage account name
	AccountName string `json:"accountName"`

	// +kubebuilder:validation:Required
	// ContainerName is the container name
	ContainerName string `json:"containerName"`

	// +kubebuilder:validation:Optional
	// AccountKeySecret references the account key
	AccountKeySecret *corev1.SecretKeySelector `json:"accountKeySecret,omitempty"`
}

// GrafanaConfigStatus defines the observed state of GrafanaConfig
type GrafanaConfigStatus struct {
	// +kubebuilder:validation:Optional
	// Phase indicates the current state of the GrafanaConfig
	// +kubebuilder:validation:Enum=Pending;Applying;Applied;Failed
	Phase string `json:"phase,omitempty"`

	// +kubebuilder:validation:Optional
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Optional
	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +kubebuilder:validation:Optional
	// LastAppliedTime indicates when the configuration was last applied
	LastAppliedTime *metav1.Time `json:"lastAppliedTime,omitempty"`

	// +kubebuilder:validation:Optional
	// LastAppliedGeneration indicates the generation that was last applied
	LastAppliedGeneration int64 `json:"lastAppliedGeneration,omitempty"`

	// +kubebuilder:validation:Optional
	// AppliedTo lists the ObservabilityPlatforms this config has been applied to
	AppliedTo []ObservabilityPlatformReference `json:"appliedTo,omitempty"`

	// +kubebuilder:validation:Optional
	// Message provides additional status information
	Message string `json:"message,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSourceStatuses contains status for each configured data source
	DataSourceStatuses []DataSourceStatus `json:"dataSourceStatuses,omitempty"`

	// +kubebuilder:validation:Optional
	// DashboardProviderStatuses contains status for each dashboard provider
	DashboardProviderStatuses []DashboardProviderStatus `json:"dashboardProviderStatuses,omitempty"`

	// +kubebuilder:validation:Optional
	// NotificationChannelStatuses contains status for each notification channel
	NotificationChannelStatuses []NotificationChannelStatus `json:"notificationChannelStatuses,omitempty"`
}

// DataSourceStatus represents the status of a data source
type DataSourceStatus struct {
	// +kubebuilder:validation:Required
	// Name is the data source name
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// UID is the data source UID in Grafana
	UID string `json:"uid,omitempty"`

	// +kubebuilder:validation:Optional
	// ID is the data source ID in Grafana
	ID int64 `json:"id,omitempty"`

	// +kubebuilder:validation:Optional
	// Ready indicates if the data source is working
	Ready bool `json:"ready,omitempty"`

	// +kubebuilder:validation:Optional
	// Message provides status details
	Message string `json:"message,omitempty"`

	// +kubebuilder:validation:Optional
	// LastUpdated indicates when this data source was last updated
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// DashboardProviderStatus represents the status of a dashboard provider
type DashboardProviderStatus struct {
	// +kubebuilder:validation:Required
	// Name is the provider name
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// DashboardsLoaded indicates the number of dashboards loaded
	DashboardsLoaded int32 `json:"dashboardsLoaded,omitempty"`

	// +kubebuilder:validation:Optional
	// LastSync indicates when dashboards were last synced
	LastSync *metav1.Time `json:"lastSync,omitempty"`

	// +kubebuilder:validation:Optional
	// Message provides status details
	Message string `json:"message,omitempty"`
}

// NotificationChannelStatus represents the status of a notification channel
type NotificationChannelStatus struct {
	// +kubebuilder:validation:Required
	// Name is the channel name
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// UID is the channel UID in Grafana
	UID string `json:"uid,omitempty"`

	// +kubebuilder:validation:Optional
	// ID is the channel ID in Grafana
	ID int64 `json:"id,omitempty"`

	// +kubebuilder:validation:Optional
	// Ready indicates if the channel is working
	Ready bool `json:"ready,omitempty"`

	// +kubebuilder:validation:Optional
	// Message provides status details
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories={observability,gunj},shortName=gc
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetRef.name`
// +kubebuilder:printcolumn:name="Applied",type=date,JSONPath=`.status.lastAppliedTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GrafanaConfig is the Schema for the grafanaconfigs API
// It provides advanced configuration options for Grafana instances managed by ObservabilityPlatform
type GrafanaConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GrafanaConfigSpec   `json:"spec,omitempty"`
	Status GrafanaConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GrafanaConfigList contains a list of GrafanaConfig
type GrafanaConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GrafanaConfig `json:"items"`
}

// ObservabilityPlatformReference references an ObservabilityPlatform
type ObservabilityPlatformReference struct {
	// +kubebuilder:validation:Required
	// Name is the ObservabilityPlatform name
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// Namespace is the ObservabilityPlatform namespace (defaults to same namespace)
	Namespace string `json:"namespace,omitempty"`
}

func init() {
	SchemeBuilder.Register(&GrafanaConfig{}, &GrafanaConfigList{})
}
