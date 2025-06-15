/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

// RetentionSpec defines retention configuration for logs and traces
type RetentionSpec struct {
	// Period defines how long to retain data
	// +kubebuilder:validation:Pattern=`^\d+[hdwmy]$`
	Period string `json:"period"`

	// CompactionInterval defines how often to run compaction
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[hm]$`
	CompactionInterval string `json:"compactionInterval,omitempty"`

	// DeleteDelay defines the delay before deleting old data
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+[hm]$`
	DeleteDelay string `json:"deleteDelay,omitempty"`

	// RetentionDeletesEnabled enables deletion of old data
	// +optional
	// +kubebuilder:default=true
	RetentionDeletesEnabled *bool `json:"retentionDeletesEnabled,omitempty"`
}

// AlertmanagerConfig defines Alertmanager configuration
type AlertmanagerConfig struct {
	// Global configuration
	// +optional
	Global *AlertmanagerGlobalConfig `json:"global,omitempty"`

	// Route is the top-level route
	Route *Route `json:"route"`

	// Receivers is the list of notification receivers
	Receivers []Receiver `json:"receivers"`

	// InhibitRules is the list of inhibition rules
	// +optional
	InhibitRules []InhibitRule `json:"inhibitRules,omitempty"`

	// Templates is the list of template files
	// +optional
	Templates []string `json:"templates,omitempty"`
}

// AlertmanagerGlobalConfig defines global Alertmanager configuration
type AlertmanagerGlobalConfig struct {
	// ResolveTimeout is the time after which an alert is declared resolved
	// +optional
	// +kubebuilder:default="5m"
	ResolveTimeout string `json:"resolveTimeout,omitempty"`

	// SMTPFrom is the default SMTP From header field
	// +optional
	SMTPFrom string `json:"smtpFrom,omitempty"`

	// SMTPSmarthost is the default SMTP smarthost
	// +optional
	SMTPSmarthost string `json:"smtpSmarthost,omitempty"`

	// SMTPAuthUsername is the SMTP authentication username
	// +optional
	SMTPAuthUsername string `json:"smtpAuthUsername,omitempty"`

	// SMTPAuthPassword is the SMTP authentication password
	// +optional
	SMTPAuthPassword string `json:"smtpAuthPassword,omitempty"`

	// SMTPRequireTLS requires TLS for SMTP
	// +optional
	// +kubebuilder:default=true
	SMTPRequireTLS *bool `json:"smtpRequireTLS,omitempty"`

	// SlackAPIURL is the Slack API URL
	// +optional
	SlackAPIURL string `json:"slackApiUrl,omitempty"`

	// PagerdutyURL is the PagerDuty URL
	// +optional
	PagerdutyURL string `json:"pagerdutyUrl,omitempty"`
}

// Route defines an Alertmanager route
type Route struct {
	// GroupBy are the labels by which to group alerts
	// +optional
	GroupBy []string `json:"groupBy,omitempty"`

	// GroupWait is how long to initially wait to send a notification
	// +optional
	// +kubebuilder:default="10s"
	GroupWait string `json:"groupWait,omitempty"`

	// GroupInterval is how long to wait before sending an updated notification
	// +optional
	// +kubebuilder:default="5m"
	GroupInterval string `json:"groupInterval,omitempty"`

	// RepeatInterval is how long to wait before repeating a notification
	// +optional
	// +kubebuilder:default="12h"
	RepeatInterval string `json:"repeatInterval,omitempty"`

	// Receiver is the name of the receiver for this route
	Receiver string `json:"receiver"`

	// Continue indicates whether to continue matching subsequent routes
	// +optional
	Continue bool `json:"continue,omitempty"`

	// Routes are the child routes
	// +optional
	Routes []Route `json:"routes,omitempty"`

	// Matchers is a list of matchers for this route
	// +optional
	Matchers []Matcher `json:"matchers,omitempty"`
}

// Matcher defines a matcher for routes
type Matcher struct {
	// Name is the label name
	Name string `json:"name"`

	// Value is the label value
	Value string `json:"value"`

	// MatchType is the type of match (=, !=, =~, !~)
	// +optional
	// +kubebuilder:validation:Enum="=";"!=";"=~";"!~"
	// +kubebuilder:default="="
	MatchType string `json:"matchType,omitempty"`
}

// Receiver defines a notification receiver
type Receiver struct {
	// Name is the name of the receiver
	Name string `json:"name"`

	// EmailConfigs is the list of email configurations
	// +optional
	EmailConfigs []EmailConfig `json:"emailConfigs,omitempty"`

	// PagerdutyConfigs is the list of PagerDuty configurations
	// +optional
	PagerdutyConfigs []PagerdutyConfig `json:"pagerdutyConfigs,omitempty"`

	// SlackConfigs is the list of Slack configurations
	// +optional
	SlackConfigs []SlackConfig `json:"slackConfigs,omitempty"`

	// WebhookConfigs is the list of webhook configurations
	// +optional
	WebhookConfigs []WebhookConfig `json:"webhookConfigs,omitempty"`

	// OpsgenieConfigs is the list of OpsGenie configurations
	// +optional
	OpsgenieConfigs []OpsgenieConfig `json:"opsgenieConfigs,omitempty"`
}

// EmailConfig defines email notification configuration
type EmailConfig struct {
	// To is the email address to send notifications to
	To string `json:"to"`

	// From is the sender address
	// +optional
	From string `json:"from,omitempty"`

	// Smarthost is the SMTP smarthost
	// +optional
	Smarthost string `json:"smarthost,omitempty"`

	// AuthUsername is the SMTP authentication username
	// +optional
	AuthUsername string `json:"authUsername,omitempty"`

	// AuthPassword is the SMTP authentication password
	// +optional
	AuthPassword string `json:"authPassword,omitempty"`

	// Headers is additional email headers
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// HTML is the HTML email body
	// +optional
	HTML string `json:"html,omitempty"`

	// Text is the text email body
	// +optional
	Text string `json:"text,omitempty"`

	// RequireTLS requires TLS
	// +optional
	RequireTLS *bool `json:"requireTls,omitempty"`
}

// PagerdutyConfig defines PagerDuty notification configuration
type PagerdutyConfig struct {
	// ServiceKey is the PagerDuty service key
	ServiceKey string `json:"serviceKey"`

	// URL is the PagerDuty URL
	// +optional
	URL string `json:"url,omitempty"`

	// Client is the client identification
	// +optional
	// +kubebuilder:default="Alertmanager"
	Client string `json:"client,omitempty"`

	// ClientURL is the URL for the client
	// +optional
	ClientURL string `json:"clientUrl,omitempty"`

	// Description is the incident description
	// +optional
	Description string `json:"description,omitempty"`

	// Details are arbitrary key/value pairs
	// +optional
	Details map[string]string `json:"details,omitempty"`
}

// SlackConfig defines Slack notification configuration
type SlackConfig struct {
	// APIURL is the Slack webhook URL
	APIURL string `json:"apiUrl"`

	// Channel is the Slack channel
	// +optional
	Channel string `json:"channel,omitempty"`

	// Username is the username to use
	// +optional
	// +kubebuilder:default="Alertmanager"
	Username string `json:"username,omitempty"`

	// Color is the color of the Slack attachment
	// +optional
	Color string `json:"color,omitempty"`

	// Title is the title of the Slack message
	// +optional
	Title string `json:"title,omitempty"`

	// TitleLink is the link for the title
	// +optional
	TitleLink string `json:"titleLink,omitempty"`

	// Pretext is the pretext of the Slack message
	// +optional
	Pretext string `json:"pretext,omitempty"`

	// Text is the text of the Slack message
	// +optional
	Text string `json:"text,omitempty"`

	// Fields are additional fields
	// +optional
	Fields []SlackField `json:"fields,omitempty"`

	// ShortFields indicates if fields should be short
	// +optional
	ShortFields bool `json:"shortFields,omitempty"`

	// Footer is the footer text
	// +optional
	Footer string `json:"footer,omitempty"`

	// Fallback is the fallback text
	// +optional
	Fallback string `json:"fallback,omitempty"`

	// IconEmoji is the emoji icon
	// +optional
	IconEmoji string `json:"iconEmoji,omitempty"`

	// IconURL is the icon URL
	// +optional
	IconURL string `json:"iconUrl,omitempty"`

	// LinkNames enables @mentions
	// +optional
	LinkNames bool `json:"linkNames,omitempty"`
}

// SlackField defines a Slack field
type SlackField struct {
	// Title is the field title
	Title string `json:"title"`

	// Value is the field value
	Value string `json:"value"`

	// Short indicates if the field is short
	// +optional
	Short bool `json:"short,omitempty"`
}

// WebhookConfig defines webhook notification configuration
type WebhookConfig struct {
	// URL is the webhook URL
	URL string `json:"url"`

	// HTTPConfig is the HTTP configuration
	// +optional
	HTTPConfig *HTTPConfig `json:"httpConfig,omitempty"`

	// MaxAlerts is the maximum number of alerts to send
	// +optional
	// +kubebuilder:default=0
	MaxAlerts int32 `json:"maxAlerts,omitempty"`
}

// OpsgenieConfig defines OpsGenie notification configuration
type OpsgenieConfig struct {
	// APIKey is the OpsGenie API key
	APIKey string `json:"apiKey"`

	// APIURL is the OpsGenie API URL
	// +optional
	APIURL string `json:"apiUrl,omitempty"`

	// Message is the message for OpsGenie
	// +optional
	Message string `json:"message,omitempty"`

	// Description is the description for OpsGenie
	// +optional
	Description string `json:"description,omitempty"`

	// Source is the source field
	// +optional
	Source string `json:"source,omitempty"`

	// Tags are the tags for the alert
	// +optional
	Tags string `json:"tags,omitempty"`

	// Note is additional note
	// +optional
	Note string `json:"note,omitempty"`

	// Priority is the priority
	// +optional
	Priority string `json:"priority,omitempty"`
}

// HTTPConfig defines HTTP client configuration
type HTTPConfig struct {
	// BasicAuth is the basic authentication credentials
	// +optional
	BasicAuth *BasicAuthSpec `json:"basicAuth,omitempty"`

	// BearerToken is the bearer token
	// +optional
	BearerToken string `json:"bearerToken,omitempty"`

	// ProxyURL is the proxy URL
	// +optional
	ProxyURL string `json:"proxyUrl,omitempty"`

	// TLSConfig is the TLS configuration
	// +optional
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
}

// InhibitRule defines an inhibition rule
type InhibitRule struct {
	// SourceMatch is the matchers for the source alerts
	SourceMatch []Matcher `json:"sourceMatch"`

	// TargetMatch is the matchers for the target alerts
	TargetMatch []Matcher `json:"targetMatch"`

	// Equal is the list of labels that must be equal
	Equal []string `json:"equal"`
}
