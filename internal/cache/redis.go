package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/redis/go-redis/v9"
)

// Client defines the interface for cache operations
type Client interface {
	// Basic operations
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	
	// Hash operations
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	
	// List operations
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPop(ctx context.Context, key string) (string, error)
	
	// Pub/Sub operations
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) PubSub
	
	// Stream operations
	XAdd(ctx context.Context, stream string, values map[string]interface{}) error
	XRead(ctx context.Context, streams []string, count int64, block time.Duration) ([]StreamMessage, error)
	
	// Advanced operations
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	Pipeline() Pipeline
	
	// Health check
	Ping(ctx context.Context) error
	Close() error
}

// PubSub defines the interface for pub/sub operations
type PubSub interface {
	Channel() <-chan *redis.Message
	Close() error
}

// Pipeline defines the interface for pipeline operations
type Pipeline interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Exec(ctx context.Context) ([]redis.Cmder, error)
}

// StreamMessage represents a message from Redis Streams
type StreamMessage struct {
	ID     string
	Values map[string]interface{}
}

// Config holds Redis configuration
type Config struct {
	// Connection settings
	Mode          string   `json:"mode" yaml:"mode"`                   // "standalone" or "sentinel"
	Addresses     []string `json:"addresses" yaml:"addresses"`         // Redis addresses
	MasterName    string   `json:"masterName" yaml:"masterName"`       // Master name for Sentinel
	Password      string   `json:"password" yaml:"password"`           // Redis password
	Username      string   `json:"username" yaml:"username"`           // Redis username (ACL)
	DB            int      `json:"db" yaml:"db"`                       // Database number
	
	// Connection pool
	PoolSize      int      `json:"poolSize" yaml:"poolSize"`           // Maximum connections
	MinIdleConns  int      `json:"minIdleConns" yaml:"minIdleConns"`   // Minimum idle connections
	MaxRetries    int      `json:"maxRetries" yaml:"maxRetries"`       // Maximum retry attempts
	
	// Timeouts
	DialTimeout   time.Duration `json:"dialTimeout" yaml:"dialTimeout"`     // Connection timeout
	ReadTimeout   time.Duration `json:"readTimeout" yaml:"readTimeout"`     // Read timeout
	WriteTimeout  time.Duration `json:"writeTimeout" yaml:"writeTimeout"`   // Write timeout
	PoolTimeout   time.Duration `json:"poolTimeout" yaml:"poolTimeout"`     // Pool timeout
	
	// TLS settings
	TLSEnabled    bool     `json:"tlsEnabled" yaml:"tlsEnabled"`       // Enable TLS
	TLSSkipVerify bool     `json:"tlsSkipVerify" yaml:"tlsSkipVerify"` // Skip TLS verification
	TLSCertFile   string   `json:"tlsCertFile" yaml:"tlsCertFile"`     // TLS certificate file
	TLSKeyFile    string   `json:"tlsKeyFile" yaml:"tlsKeyFile"`       // TLS key file
	TLSCAFile     string   `json:"tlsCAFile" yaml:"tlsCAFile"`         // TLS CA file
}

// DefaultConfig returns default Redis configuration
func DefaultConfig() Config {
	return Config{
		Mode:         "standalone",
		Addresses:    []string{"localhost:6379"},
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		TLSEnabled:   false,
	}
}

// redisClient implements the Client interface
type redisClient struct {
	client redis.UniversalClient
	log    logr.Logger
}

// NewClient creates a new Redis client
func NewClient(cfg Config, log logr.Logger) (Client, error) {
	var tlsConfig *tls.Config
	if cfg.TLSEnabled {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
		// Load certificates if provided
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
			if err != nil {
				return nil, fmt.Errorf("loading TLS certificates: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}
	
	var client redis.UniversalClient
	
	switch cfg.Mode {
	case "sentinel":
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       cfg.MasterName,
			SentinelAddrs:    cfg.Addresses,
			Password:         cfg.Password,
			Username:         cfg.Username,
			DB:               cfg.DB,
			PoolSize:         cfg.PoolSize,
			MinIdleConns:     cfg.MinIdleConns,
			MaxRetries:       cfg.MaxRetries,
			DialTimeout:      cfg.DialTimeout,
			ReadTimeout:      cfg.ReadTimeout,
			WriteTimeout:     cfg.WriteTimeout,
			PoolTimeout:      cfg.PoolTimeout,
			TLSConfig:        tlsConfig,
		})
	default:
		client = redis.NewClient(&redis.Options{
			Addr:             cfg.Addresses[0],
			Password:         cfg.Password,
			Username:         cfg.Username,
			DB:               cfg.DB,
			PoolSize:         cfg.PoolSize,
			MinIdleConns:     cfg.MinIdleConns,
			MaxRetries:       cfg.MaxRetries,
			DialTimeout:      cfg.DialTimeout,
			ReadTimeout:      cfg.ReadTimeout,
			WriteTimeout:     cfg.WriteTimeout,
			PoolTimeout:      cfg.PoolTimeout,
			TLSConfig:        tlsConfig,
		})
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to Redis: %w", err)
	}
	
	log.Info("Connected to Redis", "mode", cfg.Mode, "addresses", cfg.Addresses)
	
	return &redisClient{
		client: client,
		log:    log,
	}, nil
}

// Get retrieves a value by key
func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a value with optional expiration
func (r *redisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Handle different value types
	var val interface{}
	switch v := value.(type) {
	case string, []byte:
		val = v
	default:
		// Serialize to JSON for complex types
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshaling value: %w", err)
		}
		val = data
	}
	
	return r.client.Set(ctx, key, val, expiration).Err()
}

// Delete removes one or more keys
func (r *redisClient) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist
func (r *redisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// HGet gets a field from a hash
func (r *redisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

// HSet sets fields in a hash
func (r *redisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HSet(ctx, key, values...).Err()
}

// HGetAll gets all fields from a hash
func (r *redisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// LPush pushes values to the left of a list
func (r *redisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

// RPop pops a value from the right of a list
func (r *redisClient) RPop(ctx context.Context, key string) (string, error) {
	return r.client.RPop(ctx, key).Result()
}

// Publish publishes a message to a channel
func (r *redisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to channels
func (r *redisClient) Subscribe(ctx context.Context, channels ...string) PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// XAdd adds a message to a stream
func (r *redisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err()
}

// XRead reads messages from streams
func (r *redisClient) XRead(ctx context.Context, streams []string, count int64, block time.Duration) ([]StreamMessage, error) {
	// Implementation would convert Redis XStream to our StreamMessage type
	// This is a simplified version
	return nil, fmt.Errorf("not implemented")
}

// Eval executes a Lua script
func (r *redisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return r.client.Eval(ctx, script, keys, args...).Result()
}

// Pipeline creates a pipeline for batch operations
func (r *redisClient) Pipeline() Pipeline {
	return r.client.Pipeline()
}

// Ping checks if Redis is reachable
func (r *redisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *redisClient) Close() error {
	return r.client.Close()
}
