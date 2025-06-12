// Package nats provides NATS implementation of the EventBus interface
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/gunjanjp/gunj-operator/internal/eventbus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Config holds NATS configuration
type Config struct {
	// Connection settings
	URL           string        `json:"url" yaml:"url"`
	ClusterID     string        `json:"clusterId" yaml:"clusterId"`
	ClientID      string        `json:"clientId" yaml:"clientId"`
	
	// JetStream settings
	EnableJetStream bool   `json:"enableJetStream" yaml:"enableJetStream"`
	StreamName      string `json:"streamName" yaml:"streamName"`
	
	// Storage settings
	StorageDir     string `json:"storageDir" yaml:"storageDir"`
	MaxMemoryStore int64  `json:"maxMemoryStore" yaml:"maxMemoryStore"` // in bytes
	MaxFileStore   int64  `json:"maxFileStore" yaml:"maxFileStore"`     // in bytes
	
	// Connection options
	MaxReconnects   int           `json:"maxReconnects" yaml:"maxReconnects"`
	ReconnectWait   time.Duration `json:"reconnectWait" yaml:"reconnectWait"`
	Timeout         time.Duration `json:"timeout" yaml:"timeout"`
	
	// TLS settings
	EnableTLS      bool   `json:"enableTls" yaml:"enableTls"`
	CertFile       string `json:"certFile,omitempty" yaml:"certFile,omitempty"`
	KeyFile        string `json:"keyFile,omitempty" yaml:"keyFile,omitempty"`
	CAFile         string `json:"caFile,omitempty" yaml:"caFile,omitempty"`
}

// DefaultConfig returns default NATS configuration
func DefaultConfig() *Config {
	return &Config{
		URL:             "nats://localhost:4222",
		ClusterID:       "gunj-operator",
		ClientID:        "gunj-operator-controller",
		EnableJetStream: true,
		StreamName:      "GUNJ_EVENTS",
		MaxMemoryStore:  1 << 30, // 1GB
		MaxFileStore:    10 << 30, // 10GB
		MaxReconnects:   10,
		ReconnectWait:   2 * time.Second,
		Timeout:         30 * time.Second,
	}
}

// NATSEventBus implements EventBus using NATS with JetStream
type NATSEventBus struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	config *Config
	logger log.Logger
}

// New creates a new NATS event bus
func New(config *Config) (*NATSEventBus, error) {
	opts := []nats.Option{
		nats.Name(config.ClientID),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.Timeout(config.Timeout),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Log.Error(err, "NATS error", "subject", sub.Subject)
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Log.Info("NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Log.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
	}

	// Add TLS if enabled
	if config.EnableTLS {
		tlsConfig, err := createTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("creating TLS config: %w", err)
		}
		opts = append(opts, nats.Secure(tlsConfig))
	}

	// Connect to NATS
	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS: %w", err)
	}

	bus := &NATSEventBus{
		nc:     nc,
		config: config,
		logger: log.Log.WithName("eventbus.nats"),
	}

	// Initialize JetStream if enabled
	if config.EnableJetStream {
		if err := bus.initJetStream(); err != nil {
			nc.Close()
			return nil, fmt.Errorf("initializing JetStream: %w", err)
		}
	}

	return bus, nil
}

// initJetStream initializes JetStream and creates streams
func (n *NATSEventBus) initJetStream() error {
	js, err := n.nc.JetStream()
	if err != nil {
		return fmt.Errorf("creating JetStream context: %w", err)
	}
	n.js = js

	// Create or update main event stream
	streamConfig := &nats.StreamConfig{
		Name:     n.config.StreamName,
		Subjects: []string{
			"platform.>",
			"component.>",
			"jobs.>",
			"cluster.>",
			"webhooks.>",
			"metrics.>",
		},
		Retention:    nats.LimitsPolicy,
		MaxAge:       7 * 24 * time.Hour, // 7 days
		MaxBytes:     n.config.MaxFileStore,
		MaxMsgs:      1000000,
		MaxConsumers: 100,
		Replicas:     3, // For HA in production
		Storage:      nats.FileStorage,
	}

	_, err = n.js.AddStream(streamConfig)
	if err != nil {
		// Try updating if stream exists
		_, err = n.js.UpdateStream(streamConfig)
		if err != nil {
			return fmt.Errorf("creating/updating stream: %w", err)
		}
	}

	// Create job queue stream with work queue policy
	jobStreamConfig := &nats.StreamConfig{
		Name:         "GUNJ_JOBS",
		Subjects:     []string{"jobs.queue"},
		Retention:    nats.WorkQueuePolicy,
		MaxAge:       24 * time.Hour,
		MaxMsgs:      10000,
		MaxConsumers: 10,
		Replicas:     3,
		Storage:      nats.FileStorage,
	}

	_, err = n.js.AddStream(jobStreamConfig)
	if err != nil {
		_, err = n.js.UpdateStream(jobStreamConfig)
		if err != nil {
			return fmt.Errorf("creating/updating job stream: %w", err)
		}
	}

	return nil
}

// Event stream configurations for different use cases
const (
	// Streams
	MainEventStream = "GUNJ_EVENTS"
	JobQueueStream  = "GUNJ_JOBS"
	
	// Consumer durables
	APIConsumer      = "api-server"
	UIConsumer       = "ui-websocket"
	MetricsConsumer  = "metrics-collector"
	WebhookConsumer  = "webhook-dispatcher"
	AuditConsumer    = "audit-logger"
)
