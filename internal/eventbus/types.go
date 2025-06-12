// Package eventbus provides event-driven communication for the Gunj Operator
package eventbus

import (
	"context"
	"time"
)

// EventBus defines the interface for event-driven communication
type EventBus interface {
	// Publish platform state change events
	PublishPlatformEvent(ctx context.Context, event PlatformEvent) error
	
	// Publish component state changes
	PublishComponentEvent(ctx context.Context, event ComponentEvent) error
	
	// Subscribe to events with filters
	Subscribe(ctx context.Context, subject string, handler EventHandler) (Subscription, error)
	
	// Enqueue async jobs
	EnqueueJob(ctx context.Context, job AsyncJob) error
	
	// Process job queue
	ProcessJobs(ctx context.Context, handler JobHandler) error
	
	// Stream events for real-time updates
	StreamEvents(ctx context.Context, filter EventFilter) (<-chan Event, error)
	
	// Close the event bus connection
	Close() error
}

// Event is the base event interface
type Event interface {
	Subject() string
	Timestamp() time.Time
	Serialize() ([]byte, error)
}

// PlatformEvent represents platform lifecycle events
type PlatformEvent struct {
	Type      EventType `json:"type"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Phase     string    `json:"phase"`
	Message   string    `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	EventTime time.Time `json:"timestamp"`
}

// ComponentEvent represents component state changes
type ComponentEvent struct {
	Platform  string    `json:"platform"`
	Namespace string    `json:"namespace"`
	Component string    `json:"component"` // prometheus, grafana, loki, tempo
	State     string    `json:"state"`
	Ready     bool      `json:"ready"`
	Message   string    `json:"message"`
	EventTime time.Time `json:"timestamp"`
}

// AsyncJob represents background jobs
type AsyncJob struct {
	ID        string                 `json:"id"`
	Type      JobType                `json:"type"`
	Platform  string                 `json:"platform"`
	Namespace string                 `json:"namespace"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
	MaxRetries int                   `json:"max_retries"`
}

// EventType represents types of platform events
type EventType string

const (
	EventTypeCreated EventType = "created"
	EventTypeUpdated EventType = "updated"
	EventTypeDeleted EventType = "deleted"
	EventTypeScaled  EventType = "scaled"
	EventTypeBackup  EventType = "backup"
	EventTypeRestore EventType = "restore"
)

// JobType represents types of async jobs
type JobType string

const (
	JobTypeBackup      JobType = "backup"
	JobTypeRestore     JobType = "restore"
	JobTypeHealthCheck JobType = "health_check"
	JobTypeCleanup     JobType = "cleanup"
	JobTypeUpgrade     JobType = "upgrade"
)

// EventHandler processes events
type EventHandler func(ctx context.Context, event []byte) error

// JobHandler processes async jobs
type JobHandler func(ctx context.Context, job AsyncJob) error

// Subscription represents an event subscription
type Subscription interface {
	Unsubscribe() error
}

// EventFilter for streaming events
type EventFilter struct {
	Namespaces []string
	Platforms  []string
	Components []string
	EventTypes []EventType
}

// Subject patterns for NATS
const (
	SubjectPlatformEvents   = "platform.*.*.events"
	SubjectComponentState   = "component.*.*.*.state"
	SubjectJobQueue         = "jobs.queue"
	SubjectMultiCluster     = "cluster.*.sync"
	SubjectWebhooks         = "webhooks.*.notify"
	SubjectMetrics          = "metrics.*.export"
)

// Implement Event interface for PlatformEvent
func (e PlatformEvent) Subject() string {
	return "platform." + e.Namespace + "." + e.Name + ".events"
}

func (e PlatformEvent) Timestamp() time.Time {
	return e.EventTime
}

// Implement Event interface for ComponentEvent
func (e ComponentEvent) Subject() string {
	return "component." + e.Namespace + "." + e.Platform + "." + e.Component + ".state"
}

func (e ComponentEvent) Timestamp() time.Time {
	return e.EventTime
}
