/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertingRuleSpec defines the desired state of AlertingRule
type AlertingRuleSpec struct {
	// +kubebuilder:validation:Required
	// TargetPlatform references the ObservabilityPlatform this rule applies to
	TargetPlatform corev1.LocalObjectReference `json:"targetPlatform"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Paused indicates whether alerting rules should be evaluated
	Paused bool `json:"paused,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=prometheus;loki;multi
	// +kubebuilder:default="prometheus"
	// RuleType specifies the type of alerting rules
	RuleType string `json:"ruleType,omitempty"`

	// +kubebuilder:validation:Optional
	// PrometheusRules defines Prometheus alerting rules
	PrometheusRules []AlertingRuleGroup `json:"prometheusRules,omitempty"`

	// +kubebuilder:validation:Optional
	// LokiRules defines Loki alerting rules
	LokiRules []LokiAlertingRuleGroup `json:"lokiRules,omitempty"`

	// +kubebuilder:validation:Optional
	// GlobalAnnotations to add to all alerts
	GlobalAnnotations map[string]string `json:"globalAnnotations,omitempty"`

	// +kubebuilder:validation:Optional
	// GlobalLabels to add to all alerts
	GlobalLabels map[string]string `json:"globalLabels,omitempty"`

	// +kubebuilder:validation:Optional
	// TenantID for multi-tenant environments
	TenantID string `json:"tenantId,omitempty"`

	// +kubebuilder:validation:Optional
	// RoutingConfig defines alert routing configuration
	RoutingConfig *AlertRoutingConfig `json:"routingConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Templates for dynamic value substitution
	Templates []AlertTemplate `json:"templates,omitempty"`

	// +kubebuilder:validation:Optional
	// ValidationConfig for rule testing
	ValidationConfig *RuleValidationConfig `json:"validationConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// NotificationConfig for alert notifications
	NotificationConfig *NotificationConfig `json:"notificationConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Metadata for organizational purposes
	Metadata *AlertingRuleMetadata `json:"metadata,omitempty"`
}

// AlertingRuleGroup defines a group of alerting rules
type AlertingRuleGroup struct {
	// +kubebuilder:validation:Required
	// Name of the rule group
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// Interval at which to evaluate rules
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// Limit for the rule group
	Limit int32 `json:"limit,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// Rules in this group
	Rules []AlertRule `json:"rules"`

	// +kubebuilder:validation:Optional
	// PartialResponseStrategy for handling partial responses
	// +kubebuilder:validation:Enum=warn;abort
	// +kubebuilder:default="warn"
	PartialResponseStrategy string `json:"partialResponseStrategy,omitempty"`
}

// AlertRule defines a single alerting rule
type AlertRule struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^[a-zA-Z_][a-zA-Z0-9_]*$"
	// Alert name (must be valid metric name)
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

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=critical;warning;info;debug
	// Severity level of the alert
	Severity string `json:"severity,omitempty"`

	// +kubebuilder:validation:Optional
	// Priority for alert ordering (lower is higher priority)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=999
	Priority int32 `json:"priority,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates if this rule is active
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Optional
	// TestData for rule validation
	TestData *AlertRuleTestData `json:"testData,omitempty"`

	// +kubebuilder:validation:Optional
	// DocumentationURL link for this alert
	DocumentationURL string `json:"documentationUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// Runbook link for incident response
	RunbookURL string `json:"runbookUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags for categorization
	Tags []string `json:"tags,omitempty"`
}

// LokiAlertingRuleGroup defines a group of Loki alerting rules
type LokiAlertingRuleGroup struct {
	// +kubebuilder:validation:Required
	// Name of the rule group
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1m"
	// Interval at which to evaluate rules
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// Limit for the rule group
	Limit int32 `json:"limit,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// Rules in this group
	Rules []LokiAlertRule `json:"rules"`
}

// LokiAlertRule defines a single Loki alerting rule
type LokiAlertRule struct {
	// +kubebuilder:validation:Required
	// Alert name
	Alert string `json:"alert"`

	// +kubebuilder:validation:Required
	// Expr is the LogQL expression
	Expr string `json:"expr"`

	// +kubebuilder:validation:Optional
	// For duration before firing
	For string `json:"for,omitempty"`

	// +kubebuilder:validation:Optional
	// Labels to add to the alert
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	// Annotations to add to the alert
	Annotations map[string]string `json:"annotations,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=critical;warning;info;debug
	// Severity level of the alert
	Severity string `json:"severity,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enabled indicates if this rule is active
	Enabled bool `json:"enabled,omitempty"`
}

// AlertRoutingConfig defines alert routing configuration
type AlertRoutingConfig struct {
	// +kubebuilder:validation:Optional
	// DefaultReceiver for alerts
	DefaultReceiver string `json:"defaultReceiver,omitempty"`

	// +kubebuilder:validation:Optional
	// Routes define routing rules
	Routes []AlertRoute `json:"routes,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// GroupWait time before sending grouped alerts
	GroupWait string `json:"groupWait,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// GroupInterval between sending grouped alerts
	GroupInterval string `json:"groupInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="4h"
	// RepeatInterval for re-sending resolved alerts
	RepeatInterval string `json:"repeatInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// GroupBy labels for alert grouping
	GroupBy []string `json:"groupBy,omitempty"`

	// +kubebuilder:validation:Optional
	// MuteTimeIntervals references mute time definitions
	MuteTimeIntervals []string `json:"muteTimeIntervals,omitempty"`

	// +kubebuilder:validation:Optional
	// ActiveTimeIntervals when this route is active
	ActiveTimeIntervals []string `json:"activeTimeIntervals,omitempty"`
}

// AlertRoute defines an alert routing rule
type AlertRoute struct {
	// +kubebuilder:validation:Optional
	// Receiver for this route
	Receiver string `json:"receiver,omitempty"`

	// +kubebuilder:validation:Optional
	// Match conditions (all must match)
	Match map[string]string `json:"match,omitempty"`

	// +kubebuilder:validation:Optional
	// MatchRe regular expression conditions
	MatchRe map[string]string `json:"matchRe,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Continue matching subsequent routes
	Continue bool `json:"continue,omitempty"`

	// +kubebuilder:validation:Optional
	// Routes nested routes
	Routes []AlertRoute `json:"routes,omitempty"`

	// +kubebuilder:validation:Optional
	// GroupWait override for this route
	GroupWait string `json:"groupWait,omitempty"`

	// +kubebuilder:validation:Optional
	// GroupInterval override for this route
	GroupInterval string `json:"groupInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// RepeatInterval override for this route
	RepeatInterval string `json:"repeatInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// GroupBy override for this route
	GroupBy []string `json:"groupBy,omitempty"`

	// +kubebuilder:validation:Optional
	// MuteTimeIntervals for this route
	MuteTimeIntervals []string `json:"muteTimeIntervals,omitempty"`

	// +kubebuilder:validation:Optional
	// ActiveTimeIntervals for this route
	ActiveTimeIntervals []string `json:"activeTimeIntervals,omitempty"`
}

// AlertTemplate defines a template for dynamic value substitution
type AlertTemplate struct {
	// +kubebuilder:validation:Required
	// Name of the template
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// Template content (Go template syntax)
	Template string `json:"template"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=annotation;label
	// +kubebuilder:default="annotation"
	// Type of template
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// Description of the template
	Description string `json:"description,omitempty"`
}

// RuleValidationConfig defines rule validation configuration
type RuleValidationConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// ValidateOnCreate validates rules on creation
	ValidateOnCreate bool `json:"validateOnCreate,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// ValidateOnUpdate validates rules on update
	ValidateOnUpdate bool `json:"validateOnUpdate,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// StrictMode fails on any validation warning
	StrictMode bool `json:"strictMode,omitempty"`

	// +kubebuilder:validation:Optional
	// TestDataSources for validation
	TestDataSources []TestDataSource `json:"testDataSources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// ValidationTimeout for rule testing
	ValidationTimeout string `json:"validationTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// DryRun mode for testing without applying
	DryRun bool `json:"dryRun,omitempty"`
}

// TestDataSource defines a data source for testing
type TestDataSource struct {
	// +kubebuilder:validation:Required
	// Name of the data source
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=prometheus;loki;file
	// Type of data source
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// PrometheusEndpoint for testing
	PrometheusEndpoint string `json:"prometheusEndpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// LokiEndpoint for testing
	LokiEndpoint string `json:"lokiEndpoint,omitempty"`

	// +kubebuilder:validation:Optional
	// FileData for file-based testing
	FileData string `json:"fileData,omitempty"`
}

// AlertRuleTestData defines test data for a rule
type AlertRuleTestData struct {
	// +kubebuilder:validation:Optional
	// InputSeries defines test time series
	InputSeries []TestTimeSeries `json:"inputSeries,omitempty"`

	// +kubebuilder:validation:Optional
	// ExpectedAlerts defines expected alert outcomes
	ExpectedAlerts []ExpectedAlert `json:"expectedAlerts,omitempty"`

	// +kubebuilder:validation:Optional
	// EvaluationInterval for testing
	EvaluationInterval string `json:"evaluationInterval,omitempty"`
}

// TestTimeSeries defines a test time series
type TestTimeSeries struct {
	// +kubebuilder:validation:Required
	// Series identifier
	Series string `json:"series"`

	// +kubebuilder:validation:Required
	// Values with timestamps
	Values string `json:"values"`
}

// ExpectedAlert defines an expected alert outcome
type ExpectedAlert struct {
	// +kubebuilder:validation:Required
	// ExpLabels expected on the alert
	ExpLabels map[string]string `json:"expLabels"`

	// +kubebuilder:validation:Required
	// ExpAnnotations expected on the alert
	ExpAnnotations map[string]string `json:"expAnnotations"`

	// +kubebuilder:validation:Required
	// EvalTime when alert should fire
	EvalTime string `json:"evalTime"`
}

// NotificationConfig defines notification configuration
type NotificationConfig struct {
	// +kubebuilder:validation:Optional
	// Receivers define notification endpoints
	Receivers []NotificationReceiver `json:"receivers,omitempty"`

	// +kubebuilder:validation:Optional
	// InhibitRules for alert suppression
	InhibitRules []InhibitRule `json:"inhibitRules,omitempty"`

	// +kubebuilder:validation:Optional
	// TimeIntervals define time-based configurations
	TimeIntervals []TimeInterval `json:"timeIntervals,omitempty"`

	// +kubebuilder:validation:Optional
	// GlobalConfig for notifications
	GlobalConfig *NotificationGlobalConfig `json:"globalConfig,omitempty"`
}

// NotificationReceiver defines a notification endpoint
type NotificationReceiver struct {
	// +kubebuilder:validation:Required
	// Name of the receiver
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// EmailConfigs for email notifications
	EmailConfigs []EmailConfig `json:"emailConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// SlackConfigs for Slack notifications
	SlackConfigs []SlackConfig `json:"slackConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// WebhookConfigs for webhook notifications
	WebhookConfigs []WebhookConfig `json:"webhookConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// PagerDutyConfigs for PagerDuty notifications
	PagerDutyConfigs []PagerDutyConfig `json:"pagerdutyConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// OpsGenieConfigs for OpsGenie notifications
	OpsGenieConfigs []OpsGenieConfig `json:"opsgenieConfigs,omitempty"`

	// +kubebuilder:validation:Optional
	// MSTeamsConfigs for Microsoft Teams notifications
	MSTeamsConfigs []MSTeamsConfig `json:"msteamsConfigs,omitempty"`
}

// EmailConfig defines email notification configuration
type EmailConfig struct {
	// +kubebuilder:validation:Required
	// To email addresses
	To []string `json:"to"`

	// +kubebuilder:validation:Optional
	// From email address
	From string `json:"from,omitempty"`

	// +kubebuilder:validation:Optional
	// Smarthost SMTP server
	Smarthost string `json:"smarthost,omitempty"`

	// +kubebuilder:validation:Optional
	// AuthUsername for SMTP
	AuthUsername string `json:"authUsername,omitempty"`

	// +kubebuilder:validation:Optional
	// AuthPassword for SMTP
	AuthPassword corev1.SecretKeySelector `json:"authPassword,omitempty"`

	// +kubebuilder:validation:Optional
	// Headers to add
	Headers map[string]string `json:"headers,omitempty"`

	// +kubebuilder:validation:Optional
	// HTML email body
	HTML string `json:"html,omitempty"`

	// +kubebuilder:validation:Optional
	// Text email body
	Text string `json:"text,omitempty"`

	// +kubebuilder:validation:Optional
	// RequireTLS for SMTP
	RequireTLS bool `json:"requireTls,omitempty"`
}

// SlackConfig defines Slack notification configuration
type SlackConfig struct {
	// +kubebuilder:validation:Optional
	// APIURL for Slack webhook
	APIURL corev1.SecretKeySelector `json:"apiUrl,omitempty"`

	// +kubebuilder:validation:Required
	// Channel to send to
	Channel string `json:"channel"`

	// +kubebuilder:validation:Optional
	// Username for bot
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:Optional
	// Color of message
	Color string `json:"color,omitempty"`

	// +kubebuilder:validation:Optional
	// Title of message
	Title string `json:"title,omitempty"`

	// +kubebuilder:validation:Optional
	// TitleLink URL
	TitleLink string `json:"titleLink,omitempty"`

	// +kubebuilder:validation:Optional
	// Pretext before message
	Pretext string `json:"pretext,omitempty"`

	// +kubebuilder:validation:Optional
	// Text message body
	Text string `json:"text,omitempty"`

	// +kubebuilder:validation:Optional
	// Fields to include
	Fields []SlackField `json:"fields,omitempty"`

	// +kubebuilder:validation:Optional
	// ShortFields display
	ShortFields bool `json:"shortFields,omitempty"`

	// +kubebuilder:validation:Optional
	// Footer text
	Footer string `json:"footer,omitempty"`

	// +kubebuilder:validation:Optional
	// Fallback text
	Fallback string `json:"fallback,omitempty"`

	// +kubebuilder:validation:Optional
	// CallbackID for interactive messages
	CallbackID string `json:"callbackId,omitempty"`

	// +kubebuilder:validation:Optional
	// IconEmoji for bot
	IconEmoji string `json:"iconEmoji,omitempty"`

	// +kubebuilder:validation:Optional
	// IconURL for bot
	IconURL string `json:"iconUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// ImageURL to include
	ImageURL string `json:"imageUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// ThumbURL to include
	ThumbURL string `json:"thumbUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// LinkNames to linkify
	LinkNames bool `json:"linkNames,omitempty"`

	// +kubebuilder:validation:Optional
	// MrkdwnIn fields
	MrkdwnIn []string `json:"mrkdwnIn,omitempty"`
}

// SlackField defines a Slack message field
type SlackField struct {
	// +kubebuilder:validation:Required
	// Title of field
	Title string `json:"title"`

	// +kubebuilder:validation:Required
	// Value of field
	Value string `json:"value"`

	// +kubebuilder:validation:Optional
	// Short display
	Short bool `json:"short,omitempty"`
}

// WebhookConfig defines webhook notification configuration
type WebhookConfig struct {
	// +kubebuilder:validation:Required
	// URL to send webhook to
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SendResolved alerts
	SendResolved bool `json:"sendResolved,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPConfig for the webhook
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// MaxAlerts to send per webhook
	MaxAlerts int32 `json:"maxAlerts,omitempty"`
}

// HTTPConfig defines HTTP client configuration
type HTTPConfig struct {
	// +kubebuilder:validation:Optional
	// BasicAuth configuration
	BasicAuth *BasicAuthConfig `json:"basicAuth,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerToken for auth
	BearerToken string `json:"bearerToken,omitempty"`

	// +kubebuilder:validation:Optional
	// BearerTokenFile for auth
	BearerTokenFile string `json:"bearerTokenFile,omitempty"`

	// +kubebuilder:validation:Optional
	// TLSConfig for HTTPS
	TLSConfig *TLSConfigSpec `json:"tlsConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ProxyURL to use
	ProxyURL string `json:"proxyUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// FollowRedirects policy
	FollowRedirects bool `json:"followRedirects,omitempty"`

	// +kubebuilder:validation:Optional
	// OAuth2 configuration
	OAuth2 *OAuth2Config `json:"oauth2,omitempty"`
}

// OAuth2Config defines OAuth2 configuration
type OAuth2Config struct {
	// +kubebuilder:validation:Required
	// ClientID for OAuth2
	ClientID string `json:"clientId"`

	// +kubebuilder:validation:Required
	// ClientSecret for OAuth2
	ClientSecret corev1.SecretKeySelector `json:"clientSecret"`

	// +kubebuilder:validation:Required
	// TokenURL for OAuth2
	TokenURL string `json:"tokenUrl"`

	// +kubebuilder:validation:Optional
	// Scopes to request
	Scopes []string `json:"scopes,omitempty"`

	// +kubebuilder:validation:Optional
	// EndpointParams to include
	EndpointParams map[string]string `json:"endpointParams,omitempty"`
}

// PagerDutyConfig defines PagerDuty notification configuration
type PagerDutyConfig struct {
	// +kubebuilder:validation:Required
	// RoutingKey for PagerDuty
	RoutingKey corev1.SecretKeySelector `json:"routingKey"`

	// +kubebuilder:validation:Optional
	// ServiceKey for PagerDuty (deprecated)
	ServiceKey corev1.SecretKeySelector `json:"serviceKey,omitempty"`

	// +kubebuilder:validation:Optional
	// URL for PagerDuty API
	URL string `json:"url,omitempty"`

	// +kubebuilder:validation:Optional
	// Client name
	Client string `json:"client,omitempty"`

	// +kubebuilder:validation:Optional
	// ClientURL link
	ClientURL string `json:"clientUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// Description template
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	// Severity mapping
	Severity string `json:"severity,omitempty"`

	// +kubebuilder:validation:Optional
	// Details to include
	Details map[string]string `json:"details,omitempty"`

	// +kubebuilder:validation:Optional
	// Images to attach
	Images []PagerDutyImage `json:"images,omitempty"`

	// +kubebuilder:validation:Optional
	// Links to include
	Links []PagerDutyLink `json:"links,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SendResolved alerts
	SendResolved bool `json:"sendResolved,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPConfig for API calls
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`
}

// PagerDutyImage defines a PagerDuty image attachment
type PagerDutyImage struct {
	// +kubebuilder:validation:Required
	// Src URL of image
	Src string `json:"src"`

	// +kubebuilder:validation:Optional
	// Href link for image
	Href string `json:"href,omitempty"`

	// +kubebuilder:validation:Optional
	// Alt text for image
	Alt string `json:"alt,omitempty"`
}

// PagerDutyLink defines a PagerDuty link
type PagerDutyLink struct {
	// +kubebuilder:validation:Required
	// Href URL
	Href string `json:"href"`

	// +kubebuilder:validation:Optional
	// Text for link
	Text string `json:"text,omitempty"`
}

// OpsGenieConfig defines OpsGenie notification configuration
type OpsGenieConfig struct {
	// +kubebuilder:validation:Required
	// APIKey for OpsGenie
	APIKey corev1.SecretKeySelector `json:"apiKey"`

	// +kubebuilder:validation:Optional
	// APIURL for OpsGenie
	APIURL string `json:"apiUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// Message template
	Message string `json:"message,omitempty"`

	// +kubebuilder:validation:Optional
	// Description template
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	// Source of alert
	Source string `json:"source,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags to add
	Tags string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// Note to add
	Note string `json:"note,omitempty"`

	// +kubebuilder:validation:Optional
	// Priority level
	Priority string `json:"priority,omitempty"`

	// +kubebuilder:validation:Optional
	// Details to include
	Details map[string]string `json:"details,omitempty"`

	// +kubebuilder:validation:Optional
	// Responders to notify
	Responders []OpsGenieResponder `json:"responders,omitempty"`

	// +kubebuilder:validation:Optional
	// Actions available
	Actions string `json:"actions,omitempty"`

	// +kubebuilder:validation:Optional
	// Entity affected
	Entity string `json:"entity,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SendResolved alerts
	SendResolved bool `json:"sendResolved,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPConfig for API calls
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`
}

// OpsGenieResponder defines an OpsGenie responder
type OpsGenieResponder struct {
	// +kubebuilder:validation:Required
	// ID of responder
	ID string `json:"id"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=team;user;escalation;schedule
	// Type of responder
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Name of responder
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	// Username of responder
	Username string `json:"username,omitempty"`
}

// MSTeamsConfig defines Microsoft Teams notification configuration
type MSTeamsConfig struct {
	// +kubebuilder:validation:Required
	// WebhookURL for Teams channel
	WebhookURL corev1.SecretKeySelector `json:"webhookUrl"`

	// +kubebuilder:validation:Optional
	// Title of message
	Title string `json:"title,omitempty"`

	// +kubebuilder:validation:Optional
	// Text of message
	Text string `json:"text,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SendResolved alerts
	SendResolved bool `json:"sendResolved,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPConfig for webhook
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`
}

// InhibitRule defines an alert inhibition rule
type InhibitRule struct {
	// +kubebuilder:validation:Optional
	// TargetMatch conditions for target alerts
	TargetMatch map[string]string `json:"targetMatch,omitempty"`

	// +kubebuilder:validation:Optional
	// TargetMatchRe regex conditions for target alerts
	TargetMatchRe map[string]string `json:"targetMatchRe,omitempty"`

	// +kubebuilder:validation:Optional
	// SourceMatch conditions for source alerts
	SourceMatch map[string]string `json:"sourceMatch,omitempty"`

	// +kubebuilder:validation:Optional
	// SourceMatchRe regex conditions for source alerts
	SourceMatchRe map[string]string `json:"sourceMatchRe,omitempty"`

	// +kubebuilder:validation:Optional
	// Equal labels that must be equal
	Equal []string `json:"equal,omitempty"`
}

// TimeInterval defines a time interval configuration
type TimeInterval struct {
	// +kubebuilder:validation:Required
	// Name of the time interval
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// TimeIntervals when active
	TimeIntervals []TimeRange `json:"timeIntervals,omitempty"`
}

// TimeRange defines a time range
type TimeRange struct {
	// +kubebuilder:validation:Optional
	// Times of day
	Times []TimeOfDay `json:"times,omitempty"`

	// +kubebuilder:validation:Optional
	// Weekdays active
	Weekdays []string `json:"weekdays,omitempty"`

	// +kubebuilder:validation:Optional
	// DaysOfMonth active
	DaysOfMonth []int32 `json:"daysOfMonth,omitempty"`

	// +kubebuilder:validation:Optional
	// Months active
	Months []string `json:"months,omitempty"`

	// +kubebuilder:validation:Optional
	// Years active
	Years []int32 `json:"years,omitempty"`

	// +kubebuilder:validation:Optional
	// Location timezone
	Location string `json:"location,omitempty"`
}

// TimeOfDay defines a time of day range
type TimeOfDay struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$"
	// StartTime in HH:MM format
	StartTime string `json:"startTime"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$"
	// EndTime in HH:MM format
	EndTime string `json:"endTime"`
}

// NotificationGlobalConfig defines global notification configuration
type NotificationGlobalConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// ResolveTimeout after which to declare alerts resolved
	ResolveTimeout string `json:"resolveTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTPConfig global defaults
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPFrom default sender
	SMTPFrom string `json:"smtpFrom,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPSmarthost default server
	SMTPSmarthost string `json:"smtpSmarthost,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPHello hostname
	SMTPHello string `json:"smtpHello,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPAuthUsername default
	SMTPAuthUsername string `json:"smtpAuthUsername,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPAuthPassword default
	SMTPAuthPassword corev1.SecretKeySelector `json:"smtpAuthPassword,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPAuthIdentity default
	SMTPAuthIdentity string `json:"smtpAuthIdentity,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPAuthSecret default
	SMTPAuthSecret corev1.SecretKeySelector `json:"smtpAuthSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// SMTPRequireTLS default
	SMTPRequireTLS bool `json:"smtpRequireTls,omitempty"`

	// +kubebuilder:validation:Optional
	// SlackAPIURL default
	SlackAPIURL corev1.SecretKeySelector `json:"slackApiUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// PagerDutyURL default
	PagerDutyURL string `json:"pagerdutyUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// OpsGenieAPIURL default
	OpsGenieAPIURL string `json:"opsgenieApiUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// OpsGenieAPIKey default
	OpsGenieAPIKey corev1.SecretKeySelector `json:"opsgenieApiKey,omitempty"`

	// +kubebuilder:validation:Optional
	// VictorOpsAPIURL default
	VictorOpsAPIURL string `json:"victoropsApiUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// VictorOpsAPIKey default
	VictorOpsAPIKey corev1.SecretKeySelector `json:"victoropsApiKey,omitempty"`
}

// AlertingRuleMetadata defines metadata for organizational purposes
type AlertingRuleMetadata struct {
	// +kubebuilder:validation:Optional
	// Team responsible for these rules
	Team string `json:"team,omitempty"`

	// +kubebuilder:validation:Optional
	// Service these rules monitor
	Service string `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	// Environment these rules apply to
	Environment string `json:"environment,omitempty"`

	// +kubebuilder:validation:Optional
	// Version of the rules
	Version string `json:"version,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags for categorization
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// Description of the rule set
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	// Owner contact information
	Owner string `json:"owner,omitempty"`

	// +kubebuilder:validation:Optional
	// Repository containing rule definitions
	Repository string `json:"repository,omitempty"`

	// +kubebuilder:validation:Optional
	// Documentation link
	DocumentationURL string `json:"documentationUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// SLO references
	SLOReferences []string `json:"sloReferences,omitempty"`
}

// AlertingRuleStatus defines the observed state of AlertingRule
type AlertingRuleStatus struct {
	// +kubebuilder:validation:Enum=Pending;Applied;Failed;Invalid;Testing
	// Phase represents the current phase of the alerting rules
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAppliedTime is when the rules were last applied
	LastAppliedTime *metav1.Time `json:"lastAppliedTime,omitempty"`

	// LastAppliedHash is the hash of the last applied rules
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`

	// ValidationResult contains validation results
	ValidationResult *RuleValidationResult `json:"validationResult,omitempty"`

	// ActiveAlerts currently firing
	ActiveAlerts []ActiveAlert `json:"activeAlerts,omitempty"`

	// RuleStats contains statistics about the rules
	RuleStats *RuleStatistics `json:"ruleStats,omitempty"`

	// SyncStatus shows the synchronization status with target platform
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`
}

// RuleValidationResult contains validation results
type RuleValidationResult struct {
	// +kubebuilder:validation:Enum=Passed;Failed;Warning
	// Status of validation
	Status string `json:"status"`

	// Errors found during validation
	Errors []ValidationError `json:"errors,omitempty"`

	// Warnings found during validation
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// ValidatedAt timestamp
	ValidatedAt *metav1.Time `json:"validatedAt,omitempty"`

	// TestedRules count
	TestedRules int32 `json:"testedRules,omitempty"`

	// PassedRules count
	PassedRules int32 `json:"passedRules,omitempty"`

	// FailedRules count
	FailedRules int32 `json:"failedRules,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	// RuleGroup containing the error
	RuleGroup string `json:"ruleGroup"`

	// Rule with the error
	Rule string `json:"rule"`

	// Error message
	Error string `json:"error"`

	// Line number if applicable
	Line int32 `json:"line,omitempty"`

	// Column number if applicable
	Column int32 `json:"column,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	// RuleGroup containing the warning
	RuleGroup string `json:"ruleGroup"`

	// Rule with the warning
	Rule string `json:"rule"`

	// Warning message
	Warning string `json:"warning"`

	// Suggestion for fixing
	Suggestion string `json:"suggestion,omitempty"`
}

// ActiveAlert represents a currently firing alert
type ActiveAlert struct {
	// Alert name
	Alert string `json:"alert"`

	// Labels on the alert
	Labels map[string]string `json:"labels"`

	// Annotations on the alert
	Annotations map[string]string `json:"annotations"`

	// State of the alert
	State string `json:"state"`

	// ActiveAt timestamp
	ActiveAt *metav1.Time `json:"activeAt"`

	// Value that triggered the alert
	Value string `json:"value,omitempty"`

	// GeneratorURL link to source
	GeneratorURL string `json:"generatorUrl,omitempty"`
}

// RuleStatistics contains statistics about the rules
type RuleStatistics struct {
	// TotalGroups count
	TotalGroups int32 `json:"totalGroups"`

	// TotalRules count
	TotalRules int32 `json:"totalRules"`

	// ActiveRules count
	ActiveRules int32 `json:"activeRules"`

	// DisabledRules count
	DisabledRules int32 `json:"disabledRules"`

	// PrometheusRules count
	PrometheusRules int32 `json:"prometheusRules"`

	// LokiRules count
	LokiRules int32 `json:"lokiRules"`

	// LastEvaluationTime timestamp
	LastEvaluationTime *metav1.Time `json:"lastEvaluationTime,omitempty"`

	// EvaluationDuration in milliseconds
	EvaluationDuration int64 `json:"evaluationDuration,omitempty"`

	// SeverityBreakdown by level
	SeverityBreakdown map[string]int32 `json:"severityBreakdown,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=alert;alerts,categories={observability,alerting}
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetPlatform.name`,description="Target ObservabilityPlatform"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.ruleType`,description="Rule type"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="Current phase"
// +kubebuilder:printcolumn:name="Rules",type=integer,JSONPath=`.status.ruleStats.totalRules`,description="Total rules"
// +kubebuilder:printcolumn:name="Active",type=integer,JSONPath=`.status.ruleStats.activeRules`,description="Active rules"
// +kubebuilder:printcolumn:name="Alerts",type=integer,JSONPath=`.status.activeAlerts[*]`,description="Active alerts",priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Time since creation"

// AlertingRule is the Schema for the alertingrules API
type AlertingRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertingRuleSpec   `json:"spec,omitempty"`
	Status AlertingRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertingRuleList contains a list of AlertingRule
type AlertingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertingRule `json:"items"`
}

// Hub marks this type as a conversion hub.
func (*AlertingRule) Hub() {}

func init() {
	SchemeBuilder.Register(&AlertingRule{}, &AlertingRuleList{})
}
