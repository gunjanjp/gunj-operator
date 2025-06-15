/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DashboardSpec defines the desired state of Dashboard
type DashboardSpec struct {
	// +kubebuilder:validation:Required
	// TargetPlatform references the ObservabilityPlatform this dashboard applies to
	TargetPlatform corev1.LocalObjectReference `json:"targetPlatform"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Paused indicates whether this dashboard should be synced
	Paused bool `json:"paused,omitempty"`

	// +kubebuilder:validation:Optional
	// Title of the dashboard
	Title string `json:"title,omitempty"`

	// +kubebuilder:validation:Optional
	// Description of the dashboard
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags for categorizing the dashboard
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// Folder to place the dashboard in
	Folder string `json:"folder,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Editable indicates if the dashboard can be edited in Grafana UI
	Editable bool `json:"editable,omitempty"`

	// +kubebuilder:validation:Optional
	// TimeSettings for the dashboard
	TimeSettings *TimeSettings `json:"timeSettings,omitempty"`

	// +kubebuilder:validation:Optional
	// Variables define dashboard template variables
	Variables []DashboardVariable `json:"variables,omitempty"`

	// +kubebuilder:validation:Optional
	// Panels define the visualization panels
	Panels []Panel `json:"panels,omitempty"`

	// +kubebuilder:validation:Optional
	// Annotations define event annotations
	Annotations []Annotation `json:"annotations,omitempty"`

	// +kubebuilder:validation:Optional
	// Links define dashboard links
	Links []DashboardLink `json:"links,omitempty"`

	// +kubebuilder:validation:Optional
	// AccessControl defines permissions for the dashboard
	AccessControl *AccessControl `json:"accessControl,omitempty"`

	// +kubebuilder:validation:Optional
	// Version information for the dashboard
	Version *VersionInfo `json:"version,omitempty"`

	// +kubebuilder:validation:Optional
	// ImportConfig for importing existing dashboards
	ImportConfig *ImportConfig `json:"importConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ExportConfig for exporting dashboard
	ExportConfig *ExportConfig `json:"exportConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Layout configuration for panels
	Layout *LayoutConfig `json:"layout,omitempty"`

	// +kubebuilder:validation:Optional
	// Metadata for organizational purposes
	Metadata *DashboardMetadata `json:"metadata,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSources explicitly used by this dashboard
	DataSources []DataSourceRef `json:"dataSources,omitempty"`

	// +kubebuilder:validation:Optional
	// JSONModel allows providing raw Grafana dashboard JSON
	JSONModel string `json:"jsonModel,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=dark;light;auto
	// +kubebuilder:default="auto"
	// Theme preference for the dashboard
	Theme string `json:"theme,omitempty"`
}

// TimeSettings defines time-related settings for the dashboard
type TimeSettings struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="now-6h"
	// From time range start
	From string `json:"from,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="now"
	// To time range end
	To string `json:"to,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	// RefreshInterval for auto-refresh
	RefreshInterval string `json:"refreshInterval,omitempty"`

	// +kubebuilder:validation:Optional
	// TimeZone for display
	TimeZone string `json:"timezone,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// HideTimePicker hides the time picker
	HideTimePicker bool `json:"hideTimePicker,omitempty"`

	// +kubebuilder:validation:Optional
	// QuickRanges defines custom quick time ranges
	QuickRanges []QuickRange `json:"quickRanges,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// WeekStart on Monday
	WeekStart bool `json:"weekStart,omitempty"`

	// +kubebuilder:validation:Optional
	// FiscalYearStartMonth (1-12)
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=12
	FiscalYearStartMonth int32 `json:"fiscalYearStartMonth,omitempty"`
}

// QuickRange defines a custom quick time range
type QuickRange struct {
	// +kubebuilder:validation:Required
	// Display name for the range
	Display string `json:"display"`

	// +kubebuilder:validation:Required
	// From time
	From string `json:"from"`

	// +kubebuilder:validation:Required
	// To time
	To string `json:"to"`

	// +kubebuilder:validation:Optional
	// Section to group the range in
	Section string `json:"section,omitempty"`
}

// DashboardVariable defines a template variable
type DashboardVariable struct {
	// +kubebuilder:validation:Required
	// Name of the variable
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// Label display name
	Label string `json:"label,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=query;custom;textbox;constant;datasource;interval;adhoc
	// Type of variable
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Query for query-type variables
	Query *VariableQuery `json:"query,omitempty"`

	// +kubebuilder:validation:Optional
	// Options for custom variables
	Options []VariableOption `json:"options,omitempty"`

	// +kubebuilder:validation:Optional
	// Current selected value
	Current *VariableOption `json:"current,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Hide variable from UI
	Hide bool `json:"hide,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Multi-select enabled
	Multi bool `json:"multi,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// IncludeAll option
	IncludeAll bool `json:"includeAll,omitempty"`

	// +kubebuilder:validation:Optional
	// AllValue when include all is selected
	AllValue string `json:"allValue,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=disabled;onDashLoad;onTimeRangeChanged
	// +kubebuilder:default="disabled"
	// Refresh behavior
	Refresh string `json:"refresh,omitempty"`

	// +kubebuilder:validation:Optional
	// Regex for filtering values
	Regex string `json:"regex,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=alphabetical;numerical;alphabeticalCaseInsensitive
	// Sort order for options
	Sort string `json:"sort,omitempty"`

	// +kubebuilder:validation:Optional
	// Description of the variable
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	// SkipUrlSync prevents URL updates
	SkipUrlSync bool `json:"skipUrlSync,omitempty"`
}

// VariableQuery defines a query for variable values
type VariableQuery struct {
	// +kubebuilder:validation:Required
	// Query string
	Query string `json:"query"`

	// +kubebuilder:validation:Optional
	// DataSource reference
	DataSource *DataSourceRef `json:"dataSource,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=label_values;metrics;query_result;tag_values
	// QueryType for the variable
	QueryType string `json:"queryType,omitempty"`

	// +kubebuilder:validation:Optional
	// MetricName for label_values query
	MetricName string `json:"metricName,omitempty"`

	// +kubebuilder:validation:Optional
	// LabelName for label_values query
	LabelName string `json:"labelName,omitempty"`

	// +kubebuilder:validation:Optional
	// Stream selector for Loki queries
	Stream string `json:"stream,omitempty"`
}

// VariableOption defines an option for a variable
type VariableOption struct {
	// +kubebuilder:validation:Required
	// Text display value
	Text string `json:"text"`

	// +kubebuilder:validation:Required
	// Value actual value
	Value string `json:"value"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Selected indicates if this is selected
	Selected bool `json:"selected,omitempty"`
}

// Panel defines a visualization panel
type Panel struct {
	// +kubebuilder:validation:Required
	// ID unique identifier for the panel
	ID int32 `json:"id"`

	// +kubebuilder:validation:Required
	// Title of the panel
	Title string `json:"title"`

	// +kubebuilder:validation:Optional
	// Description of the panel
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=graph;stat;gauge;bargauge;table;timeseries;text;heatmap;alertlist;dashlist;news;nodeGraph;pieChart;histogram;logs;traces
	// Type of panel
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// GridPos defines position and size
	GridPos *GridPos `json:"gridPos,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSource for the panel
	DataSource *DataSourceRef `json:"dataSource,omitempty"`

	// +kubebuilder:validation:Optional
	// Targets define queries for the panel
	Targets []Target `json:"targets,omitempty"`

	// +kubebuilder:validation:Optional
	// Options specific to panel type
	Options map[string]interface{} `json:"options,omitempty"`

	// +kubebuilder:validation:Optional
	// FieldConfig for the panel
	FieldConfig *FieldConfig `json:"fieldConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// Transform data transformations
	Transformations []Transformation `json:"transformations,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Transparent background
	Transparent bool `json:"transparent,omitempty"`

	// +kubebuilder:validation:Optional
	// Interval for queries
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxDataPoints to return
	MaxDataPoints int32 `json:"maxDataPoints,omitempty"`

	// +kubebuilder:validation:Optional
	// Links from the panel
	Links []PanelLink `json:"links,omitempty"`

	// +kubebuilder:validation:Optional
	// RepeatFor variable
	RepeatFor string `json:"repeatFor,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=h;v
	// RepeatDirection horizontal or vertical
	RepeatDirection string `json:"repeatDirection,omitempty"`

	// +kubebuilder:validation:Optional
	// Alert configuration for graph panels
	Alert *PanelAlert `json:"alert,omitempty"`

	// +kubebuilder:validation:Optional
	// ThresholdStyle for the panel
	ThresholdStyle *ThresholdStyle `json:"thresholdStyle,omitempty"`

	// +kubebuilder:validation:Optional
	// LibraryPanel reference
	LibraryPanel *LibraryPanelRef `json:"libraryPanel,omitempty"`

	// +kubebuilder:validation:Optional
	// CacheTimeout override
	CacheTimeout string `json:"cacheTimeout,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryCachingTTL for query results
	QueryCachingTTL int32 `json:"queryCachingTTL,omitempty"`
}

// GridPos defines panel position and size
type GridPos struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// X position
	X int32 `json:"x"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// Y position
	Y int32 `json:"y"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=24
	// W width in grid units
	W int32 `json:"w"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// H height in grid units
	H int32 `json:"h"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Static prevents dragging/resizing
	Static bool `json:"static,omitempty"`
}

// DataSourceRef references a data source
type DataSourceRef struct {
	// +kubebuilder:validation:Optional
	// Name of the data source
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	// UID of the data source
	UID string `json:"uid,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=prometheus;loki;tempo;elasticsearch;influxdb;graphite;cloudwatch;azuremonitor;mysql;postgres
	// Type of data source
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// IsDefault data source
	IsDefault bool `json:"isDefault,omitempty"`
}

// Target defines a query target
type Target struct {
	// +kubebuilder:validation:Required
	// RefID unique reference ID
	RefID string `json:"refId"`

	// +kubebuilder:validation:Optional
	// DataSource override for this query
	DataSource *DataSourceRef `json:"datasource,omitempty"`

	// +kubebuilder:validation:Optional
	// Query expression (PromQL, LogQL, etc.)
	Expr string `json:"expr,omitempty"`

	// +kubebuilder:validation:Optional
	// LegendFormat for the series
	LegendFormat string `json:"legendFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// Interval override
	Interval string `json:"interval,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Instant query
	Instant bool `json:"instant,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=time_series;table;logs;traces
	// Format of the result
	Format string `json:"format,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Hide this query result
	Hide bool `json:"hide,omitempty"`

	// +kubebuilder:validation:Optional
	// MetricQuery for more complex queries
	MetricQuery *MetricQuery `json:"metricQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// LogQuery for Loki queries
	LogQuery *LogQuery `json:"logQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// TraceQuery for Tempo queries
	TraceQuery *TraceQuery `json:"traceQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// RawQuery for raw JSON queries
	RawQuery string `json:"rawQuery,omitempty"`
}

// MetricQuery defines a metric query
type MetricQuery struct {
	// +kubebuilder:validation:Optional
	// Query PromQL expression
	Query string `json:"query,omitempty"`

	// +kubebuilder:validation:Optional
	// Range query or instant
	Range bool `json:"range,omitempty"`

	// +kubebuilder:validation:Optional
	// Step interval
	Step string `json:"step,omitempty"`

	// +kubebuilder:validation:Optional
	// ExemplarQuery for exemplars
	ExemplarQuery bool `json:"exemplarQuery,omitempty"`

	// +kubebuilder:validation:Optional
	// QueryType specific to datasource
	QueryType string `json:"queryType,omitempty"`
}

// LogQuery defines a log query
type LogQuery struct {
	// +kubebuilder:validation:Optional
	// Query LogQL expression
	Query string `json:"query,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=range;instant;stream
	// QueryType for logs
	QueryType string `json:"queryType,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxLines to return
	MaxLines int32 `json:"maxLines,omitempty"`

	// +kubebuilder:validation:Optional
	// LegendFormat for metrics queries
	LegendFormat string `json:"legendFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// Resolution for metrics queries
	Resolution int64 `json:"resolution,omitempty"`
}

// TraceQuery defines a trace query
type TraceQuery struct {
	// +kubebuilder:validation:Optional
	// Query for traces
	Query string `json:"query,omitempty"`

	// +kubebuilder:validation:Optional
	// TraceID specific trace lookup
	TraceID string `json:"traceId,omitempty"`

	// +kubebuilder:validation:Optional
	// ServiceName filter
	ServiceName string `json:"serviceName,omitempty"`

	// +kubebuilder:validation:Optional
	// OperationName filter
	OperationName string `json:"operationName,omitempty"`

	// +kubebuilder:validation:Optional
	// MinDuration filter
	MinDuration string `json:"minDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// MaxDuration filter
	MaxDuration string `json:"maxDuration,omitempty"`

	// +kubebuilder:validation:Optional
	// Limit number of traces
	Limit int32 `json:"limit,omitempty"`
}

// FieldConfig defines field configuration
type FieldConfig struct {
	// +kubebuilder:validation:Optional
	// Defaults for all fields
	Defaults *FieldDefaults `json:"defaults,omitempty"`

	// +kubebuilder:validation:Optional
	// Overrides per field
	Overrides []FieldOverride `json:"overrides,omitempty"`
}

// FieldDefaults defines default field configuration
type FieldDefaults struct {
	// +kubebuilder:validation:Optional
	// Unit for values
	Unit string `json:"unit,omitempty"`

	// +kubebuilder:validation:Optional
	// Decimals to display
	Decimals *int32 `json:"decimals,omitempty"`

	// +kubebuilder:validation:Optional
	// Min value
	Min *float64 `json:"min,omitempty"`

	// +kubebuilder:validation:Optional
	// Max value
	Max *float64 `json:"max,omitempty"`

	// +kubebuilder:validation:Optional
	// DisplayName override
	DisplayName string `json:"displayName,omitempty"`

	// +kubebuilder:validation:Optional
	// NoValue text
	NoValue string `json:"noValue,omitempty"`

	// +kubebuilder:validation:Optional
	// Thresholds configuration
	Thresholds *ThresholdsConfig `json:"thresholds,omitempty"`

	// +kubebuilder:validation:Optional
	// Mappings value mappings
	Mappings []ValueMapping `json:"mappings,omitempty"`

	// +kubebuilder:validation:Optional
	// Links field links
	Links []DataLink `json:"links,omitempty"`

	// +kubebuilder:validation:Optional
	// Color configuration
	Color *ColorConfig `json:"color,omitempty"`

	// +kubebuilder:validation:Optional
	// Custom additional settings
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// FieldOverride defines field-specific overrides
type FieldOverride struct {
	// +kubebuilder:validation:Required
	// Matcher to identify fields
	Matcher *FieldMatcher `json:"matcher"`

	// +kubebuilder:validation:Required
	// Properties to override
	Properties []FieldProperty `json:"properties"`
}

// FieldMatcher identifies fields to override
type FieldMatcher struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=byName;byRegexp;byType;byValue
	// ID of the matcher type
	ID string `json:"id"`

	// +kubebuilder:validation:Optional
	// Options for the matcher
	Options interface{} `json:"options,omitempty"`
}

// FieldProperty defines a property override
type FieldProperty struct {
	// +kubebuilder:validation:Required
	// ID of the property
	ID string `json:"id"`

	// +kubebuilder:validation:Optional
	// Value of the property
	Value interface{} `json:"value,omitempty"`
}

// ThresholdsConfig defines threshold configuration
type ThresholdsConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=absolute;percentage
	// +kubebuilder:default="absolute"
	// Mode of thresholds
	Mode string `json:"mode,omitempty"`

	// +kubebuilder:validation:Optional
	// Steps define threshold steps
	Steps []ThresholdStep `json:"steps,omitempty"`
}

// ThresholdStep defines a threshold step
type ThresholdStep struct {
	// +kubebuilder:validation:Optional
	// Value threshold
	Value *float64 `json:"value,omitempty"`

	// +kubebuilder:validation:Required
	// Color for this threshold
	Color string `json:"color"`
}

// ValueMapping defines value mappings
type ValueMapping struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=value;range;regex;special
	// Type of mapping
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Options for the mapping
	Options map[string]interface{} `json:"options,omitempty"`
}

// DataLink defines a data link
type DataLink struct {
	// +kubebuilder:validation:Required
	// Title of the link
	Title string `json:"title"`

	// +kubebuilder:validation:Required
	// URL of the link
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// TargetBlank opens in new window
	TargetBlank bool `json:"targetBlank,omitempty"`
}

// ColorConfig defines color configuration
type ColorConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=thresholds;value;fixed;palette-classic;continuous-GrYlRd;continuous-RdYlGr;continuous-BlYlRd;continuous-YlRd;continuous-BlPu;continuous-YlBl;continuous-blues;continuous-reds;continuous-greens;continuous-purples
	// Mode of coloring
	Mode string `json:"mode,omitempty"`

	// +kubebuilder:validation:Optional
	// FixedColor when mode is fixed
	FixedColor string `json:"fixedColor,omitempty"`

	// +kubebuilder:validation:Optional
	// SeriesBy field
	SeriesBy string `json:"seriesBy,omitempty"`
}

// Transformation defines a data transformation
type Transformation struct {
	// +kubebuilder:validation:Required
	// ID of the transformation
	ID string `json:"id"`

	// +kubebuilder:validation:Optional
	// Options for the transformation
	Options map[string]interface{} `json:"options,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Disabled transformation
	Disabled bool `json:"disabled,omitempty"`

	// +kubebuilder:validation:Optional
	// Filter query to limit transformation
	Filter *TransformationFilter `json:"filter,omitempty"`
}

// TransformationFilter limits transformation application
type TransformationFilter struct {
	// +kubebuilder:validation:Required
	// ID of the filter
	ID string `json:"id"`

	// +kubebuilder:validation:Optional
	// Options for the filter
	Options interface{} `json:"options,omitempty"`
}

// PanelLink defines a panel link
type PanelLink struct {
	// +kubebuilder:validation:Required
	// Title of the link
	Title string `json:"title"`

	// +kubebuilder:validation:Optional
	// URL of the link
	URL string `json:"url,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=dashboard;absolute;dashboards;panels
	// Type of link
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// Dashboard for dashboard links
	Dashboard string `json:"dashboard,omitempty"`

	// +kubebuilder:validation:Optional
	// Params query parameters
	Params string `json:"params,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// TargetBlank opens in new window
	TargetBlank bool `json:"targetBlank,omitempty"`

	// +kubebuilder:validation:Optional
	// Tooltip for the link
	Tooltip string `json:"tooltip,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// IncludeVars includes template variables
	IncludeVars bool `json:"includeVars,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// KeepTime preserves time range
	KeepTime bool `json:"keepTime,omitempty"`

	// +kubebuilder:validation:Optional
	// Icon for the link
	Icon string `json:"icon,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags to match for dashboard links
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// AsDropdown shows links as dropdown
	AsDropdown bool `json:"asDropdown,omitempty"`
}

// PanelAlert defines alert configuration for a panel
type PanelAlert struct {
	// +kubebuilder:validation:Required
	// Name of the alert
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// Message for the alert
	Message string `json:"message,omitempty"`

	// +kubebuilder:validation:Optional
	// Conditions to evaluate
	Conditions []AlertCondition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="5m"
	// For duration before alerting
	For string `json:"for,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=alerting;no_data;keep_state;ok
	// +kubebuilder:default="alerting"
	// NoDataState behavior
	NoDataState string `json:"noDataState,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=alerting;keep_state;ok
	// +kubebuilder:default="alerting"
	// ExecutionErrorState behavior
	ExecutionErrorState string `json:"executionErrorState,omitempty"`

	// +kubebuilder:validation:Optional
	// Frequency of evaluation
	Frequency string `json:"frequency,omitempty"`

	// +kubebuilder:validation:Optional
	// Handler ID
	Handler int32 `json:"handler,omitempty"`

	// +kubebuilder:validation:Optional
	// Notifications to send
	Notifications []AlertNotification `json:"notifications,omitempty"`

	// +kubebuilder:validation:Optional
	// AlertRuleTags for the alert
	AlertRuleTags map[string]string `json:"alertRuleTags,omitempty"`
}

// AlertCondition defines an alert condition
type AlertCondition struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=query;reducer;evaluator;operator
	// Type of condition
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Query parameters
	Query *AlertQuery `json:"query,omitempty"`

	// +kubebuilder:validation:Optional
	// Reducer parameters
	Reducer *AlertReducer `json:"reducer,omitempty"`

	// +kubebuilder:validation:Optional
	// Evaluator parameters
	Evaluator *AlertEvaluator `json:"evaluator,omitempty"`

	// +kubebuilder:validation:Optional
	// Operator parameters
	Operator *AlertOperator `json:"operator,omitempty"`
}

// AlertQuery defines query parameters for alerts
type AlertQuery struct {
	// +kubebuilder:validation:Optional
	// Model query model
	Model interface{} `json:"model,omitempty"`

	// +kubebuilder:validation:Optional
	// From time offset
	From string `json:"from,omitempty"`

	// +kubebuilder:validation:Optional
	// To time offset
	To string `json:"to,omitempty"`
}

// AlertReducer defines reducer parameters
type AlertReducer struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=avg;min;max;sum;count;last;median;diff;diff_abs;percent_diff;percent_diff_abs;count_non_null
	// Type of reducer
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Params for the reducer
	Params []interface{} `json:"params,omitempty"`
}

// AlertEvaluator defines evaluator parameters
type AlertEvaluator struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=gt;lt;outside_range;within_range;no_value
	// Type of evaluator
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Params for the evaluator
	Params []float64 `json:"params,omitempty"`
}

// AlertOperator defines operator parameters
type AlertOperator struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=and;or
	// Type of operator
	Type string `json:"type"`
}

// AlertNotification defines alert notification
type AlertNotification struct {
	// +kubebuilder:validation:Required
	// UID of the notification channel
	UID string `json:"uid"`

	// +kubebuilder:validation:Optional
	// ID of the notification (deprecated)
	ID int64 `json:"id,omitempty"`
}

// ThresholdStyle defines threshold visualization style
type ThresholdStyle struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=off;line;area;line+area
	// +kubebuilder:default="off"
	// Mode of threshold display
	Mode string `json:"mode,omitempty"`
}

// LibraryPanelRef references a library panel
type LibraryPanelRef struct {
	// +kubebuilder:validation:Required
	// UID of the library panel
	UID string `json:"uid"`

	// +kubebuilder:validation:Required
	// Name of the library panel
	Name string `json:"name"`
}

// Annotation defines dashboard annotations
type Annotation struct {
	// +kubebuilder:validation:Required
	// Name of the annotation
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// DataSource for the annotation
	DataSource *DataSourceRef `json:"datasource,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Enable the annotation
	Enable bool `json:"enable,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// ShowIn panel
	ShowIn int32 `json:"showIn,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=dashboard;tags;alert
	// Type of annotation
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags to filter by
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// Query for the annotation
	Query string `json:"query,omitempty"`

	// +kubebuilder:validation:Optional
	// Expr for query
	Expr string `json:"expr,omitempty"`

	// +kubebuilder:validation:Optional
	// Step interval
	Step string `json:"step,omitempty"`

	// +kubebuilder:validation:Optional
	// TextFormat for display
	TextFormat string `json:"textFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// TitleFormat for display
	TitleFormat string `json:"titleFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// TagsFormat for display
	TagsFormat string `json:"tagsFormat,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="rgba(0, 211, 255, 1)"
	// IconColor for the annotation
	IconColor string `json:"iconColor,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Hide the annotation
	Hide bool `json:"hide,omitempty"`

	// +kubebuilder:validation:Optional
	// Limit number of annotations
	Limit int32 `json:"limit,omitempty"`

	// +kubebuilder:validation:Optional
	// MatchAny tag
	MatchAny bool `json:"matchAny,omitempty"`

	// +kubebuilder:validation:Optional
	// UseValueForTime field
	UseValueForTime bool `json:"useValueForTime,omitempty"`
}

// DashboardLink defines dashboard-level links
type DashboardLink struct {
	// +kubebuilder:validation:Required
	// Title of the link
	Title string `json:"title"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=link;dashboards
	// Type of link
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	// Icon for the link
	Icon string `json:"icon,omitempty"`

	// +kubebuilder:validation:Optional
	// Tooltip for the link
	Tooltip string `json:"tooltip,omitempty"`

	// +kubebuilder:validation:Optional
	// URL for external links
	URL string `json:"url,omitempty"`

	// +kubebuilder:validation:Optional
	// Tags for dashboard links
	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// AsDropdown shows as dropdown
	AsDropdown bool `json:"asDropdown,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// TargetBlank opens in new window
	TargetBlank bool `json:"targetBlank,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// IncludeVars includes variables
	IncludeVars bool `json:"includeVars,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// KeepTime preserves time range
	KeepTime bool `json:"keepTime,omitempty"`
}

// AccessControl defines dashboard access control
type AccessControl struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Public dashboard
	Public bool `json:"public,omitempty"`

	// +kubebuilder:validation:Optional
	// OrgID organization ID
	OrgID int64 `json:"orgId,omitempty"`

	// +kubebuilder:validation:Optional
	// Permissions for the dashboard
	Permissions []Permission `json:"permissions,omitempty"`

	// +kubebuilder:validation:Optional
	// Teams with access
	Teams []string `json:"teams,omitempty"`

	// +kubebuilder:validation:Optional
	// Users with access
	Users []string `json:"users,omitempty"`

	// +kubebuilder:validation:Optional
	// Editors who can edit
	Editors []string `json:"editors,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// CanEdit for all authenticated users
	CanEdit bool `json:"canEdit,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// CanSave for all authenticated users
	CanSave bool `json:"canSave,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// CanAdmin for all authenticated users
	CanAdmin bool `json:"canAdmin,omitempty"`
}

// Permission defines a permission entry
type Permission struct {
	// +kubebuilder:validation:Optional
	// Role name
	Role string `json:"role,omitempty"`

	// +kubebuilder:validation:Optional
	// UserID for user permission
	UserID int64 `json:"userId,omitempty"`

	// +kubebuilder:validation:Optional
	// TeamID for team permission
	TeamID int64 `json:"teamId,omitempty"`

	// +kubebuilder:validation:Optional
	// UserLogin for user permission
	UserLogin string `json:"userLogin,omitempty"`

	// +kubebuilder:validation:Optional
	// Team for team permission
	Team string `json:"team,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=View;Edit;Admin
	// Permission level
	Permission string `json:"permission"`
}

// VersionInfo defines version information
type VersionInfo struct {
	// +kubebuilder:validation:Optional
	// Version number
	Version int32 `json:"version,omitempty"`

	// +kubebuilder:validation:Optional
	// CreatedBy user
	CreatedBy string `json:"createdBy,omitempty"`

	// +kubebuilder:validation:Optional
	// Created timestamp
	Created *metav1.Time `json:"created,omitempty"`

	// +kubebuilder:validation:Optional
	// UpdatedBy user
	UpdatedBy string `json:"updatedBy,omitempty"`

	// +kubebuilder:validation:Optional
	// Updated timestamp
	Updated *metav1.Time `json:"updated,omitempty"`

	// +kubebuilder:validation:Optional
	// Message for this version
	Message string `json:"message,omitempty"`
}

// ImportConfig defines import configuration
type ImportConfig struct {
	// +kubebuilder:validation:Optional
	// DashboardID from Grafana.com
	DashboardID int64 `json:"dashboardId,omitempty"`

	// +kubebuilder:validation:Optional
	// Revision to import
	Revision int32 `json:"revision,omitempty"`

	// +kubebuilder:validation:Optional
	// URL to import from
	URL string `json:"url,omitempty"`

	// +kubebuilder:validation:Optional
	// JSONContent raw JSON
	JSONContent string `json:"jsonContent,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigMapRef to import from
	ConfigMapRef *corev1.ConfigMapKeySelector `json:"configMapRef,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretRef to import from
	SecretRef *corev1.SecretKeySelector `json:"secretRef,omitempty"`

	// +kubebuilder:validation:Optional
	// DataSourceMapping to apply
	DataSourceMapping map[string]string `json:"dataSourceMapping,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Overwrite existing dashboard
	Overwrite bool `json:"overwrite,omitempty"`
}

// ExportConfig defines export configuration
type ExportConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// ExportVariables includes variables
	ExportVariables bool `json:"exportVariables,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// ExportDataSources includes data sources
	ExportDataSources bool `json:"exportDataSources,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigMapName to export to
	ConfigMapName string `json:"configMapName,omitempty"`

	// +kubebuilder:validation:Optional
	// SecretName to export to
	SecretName string `json:"secretName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// GenerateUID generates new UID
	GenerateUID bool `json:"generateUid,omitempty"`
}

// LayoutConfig defines layout configuration
type LayoutConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=fixed;flow;grid
	// +kubebuilder:default="grid"
	// Type of layout
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	// Justify content
	Justify string `json:"justify,omitempty"`

	// +kubebuilder:validation:Optional
	// Orientation of flow layout
	Orientation string `json:"orientation,omitempty"`

	// +kubebuilder:validation:Optional
	// Wrapping for flow layout
	Wrapping bool `json:"wrapping,omitempty"`
}

// DashboardMetadata defines metadata for organizational purposes
type DashboardMetadata struct {
	// +kubebuilder:validation:Optional
	// Team responsible for the dashboard
	Team string `json:"team,omitempty"`

	// +kubebuilder:validation:Optional
	// Service this dashboard monitors
	Service string `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	// Environment this dashboard is for
	Environment string `json:"environment,omitempty"`

	// +kubebuilder:validation:Optional
	// Owner of the dashboard
	Owner string `json:"owner,omitempty"`

	// +kubebuilder:validation:Optional
	// Purpose of the dashboard
	Purpose string `json:"purpose,omitempty"`

	// +kubebuilder:validation:Optional
	// SLOReferences tracked by this dashboard
	SLOReferences []string `json:"sloReferences,omitempty"`

	// +kubebuilder:validation:Optional
	// DocumentationURL for the dashboard
	DocumentationURL string `json:"documentationUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// RunbookURL for incidents
	RunbookURL string `json:"runbookUrl,omitempty"`

	// +kubebuilder:validation:Optional
	// RelatedDashboards UIDs
	RelatedDashboards []string `json:"relatedDashboards,omitempty"`
}

// DashboardStatus defines the observed state of Dashboard
type DashboardStatus struct {
	// +kubebuilder:validation:Enum=Pending;Synced;Failed;Invalid;OutOfSync
	// Phase represents the current phase of the dashboard
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed spec
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is when the dashboard was last synced
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// LastSyncHash is the hash of the last synced dashboard
	LastSyncHash string `json:"lastSyncHash,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`

	// GrafanaUID is the UID assigned by Grafana
	GrafanaUID string `json:"grafanaUid,omitempty"`

	// GrafanaID is the numeric ID in Grafana
	GrafanaID int64 `json:"grafanaId,omitempty"`

	// GrafanaVersion is the current version in Grafana
	GrafanaVersion int32 `json:"grafanaVersion,omitempty"`

	// URL to access the dashboard
	URL string `json:"url,omitempty"`

	// Slug is the URL slug for the dashboard
	Slug string `json:"slug,omitempty"`

	// ValidationErrors contains any validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// SyncStatus shows the synchronization status with Grafana
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`

	// PanelCount is the number of panels
	PanelCount int32 `json:"panelCount,omitempty"`

	// VariableCount is the number of variables
	VariableCount int32 `json:"variableCount,omitempty"`

	// DataSourceRefs lists referenced data sources
	DataSourceRefs []string `json:"dataSourceRefs,omitempty"`

	// CreatedBy user who created in Grafana
	CreatedBy string `json:"createdBy,omitempty"`

	// UpdatedBy user who last updated in Grafana
	UpdatedBy string `json:"updatedBy,omitempty"`

	// Created timestamp in Grafana
	Created *metav1.Time `json:"created,omitempty"`

	// Updated timestamp in Grafana
	Updated *metav1.Time `json:"updated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dash;dashboards,categories={observability,grafana}
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetPlatform.name`,description="Target ObservabilityPlatform"
// +kubebuilder:printcolumn:name="Title",type=string,JSONPath=`.spec.title`,description="Dashboard title"
// +kubebuilder:printcolumn:name="Folder",type=string,JSONPath=`.spec.folder`,description="Dashboard folder"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="Current phase"
// +kubebuilder:printcolumn:name="Grafana UID",type=string,JSONPath=`.status.grafanaUid`,description="Grafana UID",priority=1
// +kubebuilder:printcolumn:name="Panels",type=integer,JSONPath=`.status.panelCount`,description="Number of panels",priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Time since creation"

// Dashboard is the Schema for the dashboards API
type Dashboard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DashboardSpec   `json:"spec,omitempty"`
	Status DashboardStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DashboardList contains a list of Dashboard
type DashboardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dashboard `json:"items"`
}

// Hub marks this type as a conversion hub.
func (*Dashboard) Hub() {}

func init() {
	SchemeBuilder.Register(&Dashboard{}, &DashboardList{})
}
