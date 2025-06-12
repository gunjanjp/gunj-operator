package cache

import (
	"context"
	"fmt"
	"time"
)

// RateLimiter provides rate limiting functionality using Redis
type RateLimiter struct {
	client Client
	prefix string
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(client Client, prefix string) *RateLimiter {
	return &RateLimiter{
		client: client,
		prefix: prefix,
	}
}

// Allow checks if a request is allowed under the rate limit
// Uses sliding window algorithm with Redis sorted sets
func (r *RateLimiter) Allow(ctx context.Context, identifier string, limit int, window time.Duration) (bool, int, error) {
	now := time.Now()
	windowStart := now.Add(-window).Unix()
	nowUnix := now.Unix()
	
	key := fmt.Sprintf("%s:%s", r.prefix, identifier)
	
	// Lua script for atomic rate limit check
	script := `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local window_start = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local window_ms = tonumber(ARGV[4])
		
		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)
		
		-- Count current entries
		local current = redis.call('ZCARD', key)
		
		if current < limit then
			-- Add new entry
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, window_ms)
			return {1, limit - current - 1}
		else
			return {0, 0}
		end
	`
	
	result, err := r.client.Eval(ctx, script, []string{key}, limit, windowStart, nowUnix, int(window.Seconds()))
	if err != nil {
		return false, 0, fmt.Errorf("rate limit check failed: %w", err)
	}
	
	// Parse result
	res, ok := result.([]interface{})
	if !ok || len(res) != 2 {
		return false, 0, fmt.Errorf("unexpected result format")
	}
	
	allowed := res[0].(int64) == 1
	remaining := int(res[1].(int64))
	
	return allowed, remaining, nil
}

// Reset clears the rate limit for an identifier
func (r *RateLimiter) Reset(ctx context.Context, identifier string) error {
	key := fmt.Sprintf("%s:%s", r.prefix, identifier)
	return r.client.Delete(ctx, key)
}

// SessionStore provides session management using Redis
type SessionStore struct {
	client Client
	prefix string
	ttl    time.Duration
}

// NewSessionStore creates a new session store
func NewSessionStore(client Client, prefix string, ttl time.Duration) *SessionStore {
	return &SessionStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// Session represents a user session
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Roles     []string               `json:"roles"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Data      map[string]interface{} `json:"data"`
}

// Create creates a new session
func (s *SessionStore) Create(ctx context.Context, session *Session) error {
	key := fmt.Sprintf("%s:%s", s.prefix, session.ID)
	
	// Set expiration time
	session.ExpiresAt = time.Now().Add(s.ttl)
	
	// Store session data
	err := s.client.HSet(ctx, key,
		"user_id", session.UserID,
		"username", session.Username,
		"roles", fmt.Sprintf("%v", session.Roles),
		"created_at", session.CreatedAt.Format(time.RFC3339),
		"expires_at", session.ExpiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("storing session: %w", err)
	}
	
	// Set expiration
	return s.client.Set(ctx, key, "", s.ttl)
}

// Get retrieves a session by ID
func (s *SessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("%s:%s", s.prefix, sessionID)
	
	// Check if session exists
	exists, err := s.client.Exists(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("checking session existence: %w", err)
	}
	if exists == 0 {
		return nil, fmt.Errorf("session not found")
	}
	
	// Get session data
	data, err := s.client.HGetAll(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving session: %w", err)
	}
	
	// Parse session
	session := &Session{
		ID:       sessionID,
		UserID:   data["user_id"],
		Username: data["username"],
		Data:     make(map[string]interface{}),
	}
	
	// Parse timestamps
	if createdAt, err := time.Parse(time.RFC3339, data["created_at"]); err == nil {
		session.CreatedAt = createdAt
	}
	if expiresAt, err := time.Parse(time.RFC3339, data["expires_at"]); err == nil {
		session.ExpiresAt = expiresAt
	}
	
	// TODO: Parse roles from string representation
	
	return session, nil
}

// Refresh extends a session's TTL
func (s *SessionStore) Refresh(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("%s:%s", s.prefix, sessionID)
	
	// Update expiration time
	expiresAt := time.Now().Add(s.ttl)
	err := s.client.HSet(ctx, key, "expires_at", expiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("updating session expiry: %w", err)
	}
	
	// Reset Redis key expiration
	return s.client.Set(ctx, key, "", s.ttl)
}

// Delete removes a session
func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("%s:%s", s.prefix, sessionID)
	return s.client.Delete(ctx, key)
}

// APICache provides caching for API responses
type APICache struct {
	client Client
	prefix string
}

// NewAPICache creates a new API cache
func NewAPICache(client Client, prefix string) *APICache {
	return &APICache{
		client: client,
		prefix: prefix,
	}
}

// Get retrieves a cached response
func (a *APICache) Get(ctx context.Context, key string) ([]byte, error) {
	cacheKey := fmt.Sprintf("%s:%s", a.prefix, key)
	data, err := a.client.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}
	return []byte(data), nil
}

// Set stores a response in cache
func (a *APICache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:%s", a.prefix, key)
	return a.client.Set(ctx, cacheKey, data, ttl)
}

// Invalidate removes cached responses matching a pattern
func (a *APICache) Invalidate(ctx context.Context, pattern string) error {
	// In a real implementation, this would use SCAN to find matching keys
	// For now, we'll just delete the exact key
	cacheKey := fmt.Sprintf("%s:%s", a.prefix, pattern)
	return a.client.Delete(ctx, cacheKey)
}

// MetricsBuffer provides buffering for metrics using Redis Streams
type MetricsBuffer struct {
	client     Client
	streamName string
	maxLen     int64
}

// NewMetricsBuffer creates a new metrics buffer
func NewMetricsBuffer(client Client, streamName string, maxLen int64) *MetricsBuffer {
	return &MetricsBuffer{
		client:     client,
		streamName: streamName,
		maxLen:     maxLen,
	}
}

// Metric represents a metric data point
type Metric struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
}

// Add adds a metric to the buffer
func (m *MetricsBuffer) Add(ctx context.Context, metric *Metric) error {
	values := map[string]interface{}{
		"name":      metric.Name,
		"value":     metric.Value,
		"timestamp": metric.Timestamp.UnixNano(),
	}
	
	// Add labels
	for k, v := range metric.Labels {
		values[fmt.Sprintf("label_%s", k)] = v
	}
	
	return m.client.XAdd(ctx, m.streamName, values)
}

// WebSocketRegistry manages WebSocket connections using Redis
type WebSocketRegistry struct {
	client Client
	prefix string
	ttl    time.Duration
}

// NewWebSocketRegistry creates a new WebSocket registry
func NewWebSocketRegistry(client Client, prefix string, ttl time.Duration) *WebSocketRegistry {
	return &WebSocketRegistry{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// Register registers a WebSocket connection
func (w *WebSocketRegistry) Register(ctx context.Context, connectionID, userID, nodeID string) error {
	key := fmt.Sprintf("%s:connections:%s", w.prefix, connectionID)
	err := w.client.HSet(ctx, key,
		"user_id", userID,
		"node_id", nodeID,
		"connected_at", time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	
	// Add to user's connection set
	userKey := fmt.Sprintf("%s:users:%s", w.prefix, userID)
	return w.client.Set(ctx, userKey, connectionID, w.ttl)
}

// Unregister removes a WebSocket connection
func (w *WebSocketRegistry) Unregister(ctx context.Context, connectionID string) error {
	key := fmt.Sprintf("%s:connections:%s", w.prefix, connectionID)
	
	// Get user ID before deleting
	userID, err := w.client.HGet(ctx, key, "user_id")
	if err == nil && userID != "" {
		// Remove from user's connection set
		userKey := fmt.Sprintf("%s:users:%s", w.prefix, userID)
		_ = w.client.Delete(ctx, userKey)
	}
	
	return w.client.Delete(ctx, key)
}

// Broadcast publishes a message to all connections
func (w *WebSocketRegistry) Broadcast(ctx context.Context, channel string, message interface{}) error {
	return w.client.Publish(ctx, channel, message)
}
