/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

// AlertmanagerConfig defines Alertmanager configuration
type AlertmanagerConfig struct {
	// Route is the top-level route
	Route *Route `json:"route,omitempty"`

	// Receivers is the list of receivers
	Receivers []Receiver `json:"receivers,omitempty"`

	// InhibitRules is the list of inhibition rules
	// +optional
	InhibitRules []InhibitRule `json:"inhibitRules,omitempty"`

	// Global configuration
	// +optional
	Global *GlobalConfig `json:"global,omitempty"`
}

// Route defines an Alertmanager route
type Route struct {
	// GroupBy is a list of labels to group by
	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// GroupWait is how long to wait before sending notification
	// +optional
	GroupWait string `json:"groupWait,omitempty"`

	// GroupInterval is how long to wait before sending notification about new alerts
	// +optional
	GroupInterval string `json:"groupInterval,omitempty"`

	// RepeatInterval is how long to wait before repeating notification
	// +optional
	RepeatInterval string `json:"repeatInterval,omitempty"`

	// Receiver is the name of the receiver to send notifications to
	Receiver string `json:"receiver"`

	// Routes are child routes
	// +optional
	Routes []Route `json:"routes,omitempty"`

	// Matchers is a list of matchers
	// +optional
	Matchers []Matcher `json:"matchers,omitempty"`
}

// Receiver defines an Alertmanager receiver
type Receiver struct {
	// Name of the receiver
	Name string `json:"name"`

	// EmailConfigs is email configurations
	// +optional
	EmailConfigs []EmailConfig `json:"emailConfigs,omitempty"`

	// SlackConfigs is Slack configurations
	// +optional
	SlackConfigs []SlackConfig `json:"slackConfigs,omitempty"`

	// WebhookConfigs is webhook configurations
	// +optional
	WebhookConfigs []WebhookConfig `json:"webhookConfigs,omitempty"`

	// PagerdutyConfigs is PagerDuty configurations
	// +optional
	PagerdutyConfigs []PagerdutyConfig `json:"pagerdutyConfigs,omitempty"`
}

// EmailConfig defines email configuration
type EmailConfig struct {
	// To is the email address to send notifications to
	To string `json:"to"`

	// From is the sender address
	// +optional
	From string `json:"from,omitempty"`

	// SmartHost is the SMTP host:port
	// +optional
	SmartHost string `json:"smartHost,omitempty"`

	// AuthUsername is the SMTP auth username
	// +optional
	AuthUsername string `json:"authUsername,omitempty"`

	// AuthPassword is the SMTP auth password
	// +optional
	AuthPassword string `json:"authPassword,omitempty"`
}

// SlackConfig defines Slack configuration
type SlackConfig struct {
	// APIURL is the Slack webhook URL
	APIURL string `json:"apiURL"`

	// Channel is the channel or user to send notifications to
	// +optional
	Channel string `json:"channel,omitempty"`

	// Username is the bot username
	// +optional
	Username string `json:"username,omitempty"`

	// IconEmoji is the emoji icon for the bot
	// +optional
	IconEmoji string `json:"iconEmoji,omitempty"`

	// IconURL is the icon URL for the bot
	// +optional
	IconURL string `json:"iconURL,omitempty"`

	// Title is the notification title
	// +optional
	Title string `json:"title,omitempty"`

	// Text is the notification text
	// +optional
	Text string `json:"text,omitempty"`
}

// WebhookConfig defines webhook configuration
type WebhookConfig struct {
	// URL is the webhook URL
	URL string `json:"url"`

	// MaxAlerts is the maximum number of alerts to send
	// +optional
	MaxAlerts int32 `json:"maxAlerts,omitempty"`
}

// PagerdutyConfig defines PagerDuty configuration
type PagerdutyConfig struct {
	// ServiceKey is the PagerDuty service key
	ServiceKey string `json:"serviceKey"`

	// URL is the PagerDuty API URL
	// +optional
	URL string `json:"url,omitempty"`

	// Client is the client identification
	// +optional
	Client string `json:"client,omitempty"`

	// ClientURL is the client URL
	// +optional
	ClientURL string `json:"clientURL,omitempty"`

	// Description is the incident description
	// +optional
	Description string `json:"description,omitempty"`
}

// InhibitRule defines an inhibition rule
type InhibitRule struct {
	// SourceMatchers is a list of matchers for the source alerts
	// +optional
	SourceMatchers []Matcher `json:"sourceMatchers,omitempty"`

	// TargetMatchers is a list of matchers for the target alerts
	// +optional
	TargetMatchers []Matcher `json:"targetMatchers,omitempty"`

	// Equal is a list of labels that must be equal
	// +optional
	Equal []string `json:"equal,omitempty"`
}

// Matcher defines a matcher for labels
type Matcher struct {
	// Name is the label name
	Name string `json:"name"`

	// Value is the label value
	Value string `json:"value"`

	// MatchType is the type of matching (=, !=, =~, !~)
	// +kubebuilder:validation:Enum="=";"!=";"=~";"!~"
	// +optional
	MatchType string `json:"matchType,omitempty"`
}

// GlobalConfig defines global Alertmanager configuration
type GlobalConfig struct {
	// ResolveTimeout is the time after which an alert is declared resolved
	// +optional
	ResolveTimeout string `json:"resolveTimeout,omitempty"`

	// SMTPFrom is the default SMTP From header field
	// +optional
	SMTPFrom string `json:"smtpFrom,omitempty"`

	// SMTPSmartHost is the default SMTP smarthost
	// +optional
	SMTPSmartHost string `json:"smtpSmartHost,omitempty"`

	// SMTPAuthUsername is the SMTP auth username
	// +optional
	SMTPAuthUsername string `json:"smtpAuthUsername,omitempty"`

	// SMTPAuthPassword is the SMTP auth password
	// +optional
	SMTPAuthPassword string `json:"smtpAuthPassword,omitempty"`

	// SMTPRequireTLS is whether to require TLS
	// +optional
	SMTPRequireTLS bool `json:"smtpRequireTLS,omitempty"`

	// SlackAPIURL is the global Slack API URL
	// +optional
	SlackAPIURL string `json:"slackAPIURL,omitempty"`

	// PagerdutyURL is the PagerDuty API URL
	// +optional
	PagerdutyURL string `json:"pagerdutyURL,omitempty"`
}

// RetentionSpec defines retention configuration
type RetentionSpec struct {
	// Days is the number of days to retain data
	// +kubebuilder:validation:Minimum=1
	Days int32 `json:"days"`

	// DeletesEnabled enables deletion of old data
	// +optional
	DeletesEnabled bool `json:"deletesEnabled,omitempty"`

	// CompactionInterval is how often to run compaction
	// +optional
	CompactionInterval string `json:"compactionInterval,omitempty"`
}

// RetentionPolicies defines retention policies for each component
type RetentionPolicies struct {
	// Metrics retention period
	// +optional
	Metrics string `json:"metrics,omitempty"`

	// Logs retention period
	// +optional
	Logs string `json:"logs,omitempty"`

	// Traces retention period
	// +optional
	Traces string `json:"traces,omitempty"`
}

// AntiAffinitySpec defines anti-affinity configuration
type AntiAffinitySpec struct {
	// Type is the anti-affinity type (hard, soft)
	// +kubebuilder:validation:Enum=hard;soft
	Type string `json:"type"`

	// TopologyKey is the topology key for anti-affinity
	// +optional
	// +kubebuilder:default="kubernetes.io/hostname"
	TopologyKey string `json:"topologyKey,omitempty"`
}

// StorageLocation defines backup storage location
type StorageLocation struct {
	// Type is the storage type (s3, azure, gcs, local)
	// +kubebuilder:validation:Enum=s3;azure;gcs;local
	Type string `json:"type"`

	// BucketName is the bucket name for cloud storage
	// +optional
	BucketName string `json:"bucketName,omitempty"`

	// Path is the path within the bucket or local filesystem
	// +optional
	Path string `json:"path,omitempty"`

	// Region is the cloud region
	// +optional
	Region string `json:"region,omitempty"`

	// Endpoint is the S3-compatible endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// SecretName containing credentials
	// +optional
	SecretName string `json:"secretName,omitempty"`
}
