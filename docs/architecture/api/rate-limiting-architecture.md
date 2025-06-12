# Gunj Operator Rate Limiting & Quota Management Architecture

**Version**: 1.0.0  
**Date**: June 12, 2025  
**Status**: Draft  
**Author**: Gunjan JP  

## Table of Contents
1. [Overview](#overview)
2. [Rate Limiting Strategies](#rate-limiting-strategies)
3. [Rate Limit Tiers](#rate-limit-tiers)
4. [Quota Management](#quota-management)
5. [GraphQL Complexity Limiting](#graphql-complexity-limiting)
6. [Distributed Rate Limiting](#distributed-rate-limiting)
7. [Implementation Architecture](#implementation-architecture)
8. [Configuration Management](#configuration-management)
9. [Monitoring & Alerting](#monitoring--alerting)
10. [Best Practices](#best-practices)

## Overview

The Gunj Operator implements a sophisticated rate limiting and quota management system to:

- **Protect APIs**: Prevent abuse and denial-of-service attacks
- **Ensure Fair Usage**: Distribute resources fairly among users
- **Maintain Stability**: Prevent system overload
- **Control Costs**: Limit resource consumption
- **Enable SLAs**: Support different service tiers

### Design Principles

1. **Flexible**: Support multiple rate limiting algorithms
2. **Granular**: Per-user, per-endpoint, per-resource limits
3. **Distributed**: Work across multiple API instances
4. **Performant**: Minimal latency impact
5. **Observable**: Comprehensive metrics and alerting
6. **Configurable**: Runtime configuration updates

## Rate Limiting Strategies

### 1. Token Bucket Algorithm (Primary)

The token bucket algorithm provides smooth rate limiting with burst capability.

```yaml
# Token Bucket Configuration
token_bucket:
  # Tokens are added at a constant rate
  refill_rate: 100      # tokens per minute
  bucket_capacity: 500  # maximum tokens
  
  # Each request consumes tokens
  cost_per_request:
    GET: 1
    POST: 5
    PUT: 3
    DELETE: 10
    
  # Burst handling
  burst_multiplier: 2.0  # Allow 2x normal rate for bursts
  burst_duration: 60s    # Burst window
```

### 2. Sliding Window Counter

For precise rate limiting over time windows.

```yaml
# Sliding Window Configuration
sliding_window:
  window_size: 60s      # Time window
  max_requests: 1000    # Requests per window
  
  # Sub-windows for accuracy
  sub_windows: 12       # 5-second sub-windows
  precision: 0.1        # 10% tolerance
```

### 3. Fixed Window Counter

Simple and efficient for basic rate limiting.

```yaml
# Fixed Window Configuration
fixed_window:
  window_size: 60s
  max_requests: 1000
  
  # Window alignment
  align_to_minute: true
  reset_offset: 0s
```

### 4. Adaptive Rate Limiting

Dynamic limits based on system load.

```yaml
# Adaptive Configuration
adaptive:
  base_rate: 1000           # Normal rate
  
  # Load-based adjustments
  cpu_threshold: 70         # CPU percentage
  memory_threshold: 80      # Memory percentage
  
  # Reduction factors
  high_load_factor: 0.5     # 50% reduction
  critical_load_factor: 0.2 # 80% reduction
  
  # Recovery
  recovery_rate: 1.1        # 10% increase per minute
  max_rate: 2000           # Maximum allowed rate
```

## Rate Limit Tiers

### User-Based Tiers

```yaml
# Rate Limit Tiers
tiers:
  # Unauthenticated users
  anonymous:
    requests_per_hour: 100
    burst_size: 20
    concurrent_requests: 5
    
  # Free tier users
  free:
    requests_per_hour: 1000
    burst_size: 100
    concurrent_requests: 10
    
  # Authenticated users
  standard:
    requests_per_hour: 5000
    burst_size: 500
    concurrent_requests: 20
    daily_quota: 100000
    
  # Premium users
  premium:
    requests_per_hour: 20000
    burst_size: 2000
    concurrent_requests: 50
    daily_quota: 500000
    
  # Enterprise users
  enterprise:
    requests_per_hour: 100000
    burst_size: 10000
    concurrent_requests: 100
    daily_quota: unlimited
    
  # Admin users
  admin:
    requests_per_hour: unlimited
    burst_size: unlimited
    concurrent_requests: unlimited
    daily_quota: unlimited
```

### API Key Tiers

```yaml
# API Key Rate Limits
api_key_tiers:
  # Development keys
  development:
    requests_per_hour: 1000
    burst_size: 100
    
  # Production keys
  production:
    requests_per_hour: 10000
    burst_size: 1000
    
  # High-volume keys
  high_volume:
    requests_per_hour: 100000
    burst_size: 10000
    custom_limits: true  # Can be customized per key
```

### Endpoint-Specific Limits

```yaml
# Endpoint Rate Limits
endpoints:
  # Platform operations
  "/api/v1/platforms":
    GET:
      weight: 1
      tier_multiplier: 1.0
    POST:
      weight: 10
      tier_multiplier: 0.1  # 10% of tier limit
      
  # Expensive operations
  "/api/v1/platforms/*/backup":
    POST:
      weight: 100
      max_per_hour: 10
      cooldown: 300s  # 5 minutes between backups
      
  # Metrics queries
  "/api/v1/platforms/*/metrics":
    GET:
      weight: 5
      cache_ttl: 60s
      
  # Webhooks
  "/api/v1/webhooks":
    POST:
      weight: 5
      max_total: 100  # Max 100 webhooks per user
```

## Quota Management

### Resource Quotas

```yaml
# Resource Creation Quotas
quotas:
  # Platforms per namespace
  platforms:
    default:
      max: 10
      warning_threshold: 8
    production:
      max: 50
      warning_threshold: 40
      
  # Components per platform
  components:
    prometheus:
      max_instances: 5
      max_replicas: 10
      max_storage: 1Ti
      
    grafana:
      max_instances: 3
      max_dashboards: 1000
      
  # Global limits
  global:
    max_platforms_per_user: 100
    max_api_keys_per_user: 50
    max_webhooks_per_user: 100
    max_backup_storage: 10Ti
```

### Usage Tracking

```yaml
# Usage Metrics
usage_tracking:
  # API calls
  api_calls:
    track_by: [user, endpoint, method]
    retention: 90d
    aggregations: [minute, hour, day]
    
  # Resource usage
  resources:
    track_by: [user, namespace, resource_type]
    metrics:
      - cpu_hours
      - memory_gb_hours
      - storage_gb_days
      - network_gb
      
  # Cost tracking
  costs:
    calculate_per: [user, namespace, platform]
    billing_period: monthly
    currency: USD
```

## GraphQL Complexity Limiting

### Query Complexity Calculation

```graphql
# Complexity points for GraphQL fields
type Platform @complexity(value: 1) {
  metadata: ObjectMeta @complexity(value: 1)
  spec: PlatformSpec @complexity(value: 2)
  status: PlatformStatus @complexity(value: 1)
  
  # Expensive fields
  metrics(range: TimeRangeInput!): PlatformMetrics @complexity(value: 10)
  events(limit: Int): [Event!] @complexity(multiplier: "limit")
  cost(period: CostPeriod): CostAnalysis @complexity(value: 20)
}

# Query complexity limits by tier
complexity_limits:
  anonymous: 100
  free: 500
  standard: 1000
  premium: 5000
  enterprise: 10000
  admin: unlimited
```

### Query Depth Limiting

```yaml
# GraphQL Query Depth Limits
query_depth:
  max_depth:
    anonymous: 5
    authenticated: 10
    admin: 20
    
  # Introspection limits
  introspection:
    enabled: true
    rate_limit: 10/hour
    max_depth: 3
```

### Field-Level Rate Limiting

```graphql
# Field-specific rate limits
type Query {
  # Standard queries
  platforms: [Platform!] @rateLimit(max: 100, window: "1m")
  
  # Expensive queries
  costAnalysis: CostAnalysis @rateLimit(max: 10, window: "5m")
  
  # Real-time data
  metrics: Metrics @rateLimit(max: 60, window: "1m") @cache(ttl: "10s")
}

type Mutation {
  # Resource creation
  createPlatform: Platform @rateLimit(max: 10, window: "1h")
  
  # Dangerous operations
  deletePlatform: Boolean @rateLimit(max: 5, window: "1h") @cooldown(duration: "5m")
}
```

## Distributed Rate Limiting

### Redis-Based Implementation

```yaml
# Redis Configuration
redis:
  mode: cluster          # standalone, sentinel, cluster
  
  # Connection settings
  addresses:
    - redis-1:6379
    - redis-2:6379
    - redis-3:6379
    
  # Performance settings
  pool_size: 100
  min_idle_conns: 10
  max_retries: 3
  
  # Rate limit keys
  key_prefix: "gunj:ratelimit:"
  key_ttl: 3600  # 1 hour
  
  # Sliding window settings
  window_script: |
    -- Lua script for atomic sliding window
    local key = KEYS[1]
    local now = tonumber(ARGV[1])
    local window = tonumber(ARGV[2])
    local limit = tonumber(ARGV[3])
    
    -- Remove old entries
    redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
    
    -- Count current entries
    local current = redis.call('ZCARD', key)
    
    if current < limit then
        -- Add new entry
        redis.call('ZADD', key, now, now)
        redis.call('EXPIRE', key, window)
        return 1  -- Allowed
    else
        return 0  -- Denied
    end
```

### Synchronization Strategy

```yaml
# Distributed Synchronization
synchronization:
  # Local cache
  local_cache:
    enabled: true
    ttl: 5s
    size: 10000
    
  # Sync protocol
  sync_interval: 1s
  batch_size: 100
  
  # Consistency model
  consistency: eventual
  max_drift: 10%  # Allow 10% over limit for eventual consistency
  
  # Failover
  fallback_to_local: true
  local_limit_multiplier: 0.8  # Use 80% of limit when Redis unavailable
```

## Implementation Architecture

### Rate Limiter Components

```go
// pkg/ratelimit/limiter.go
package ratelimit

type RateLimiter interface {
    // Check if request is allowed
    Allow(ctx context.Context, key string) (bool, error)
    
    // Check with custom cost
    AllowN(ctx context.Context, key string, n int) (bool, error)
    
    // Get current limit info
    Status(ctx context.Context, key string) (*LimitStatus, error)
    
    // Reset limits for key
    Reset(ctx context.Context, key string) error
}

type LimitStatus struct {
    Limit     int           `json:"limit"`
    Remaining int           `json:"remaining"`
    Reset     time.Time     `json:"reset"`
    RetryAfter *time.Duration `json:"retry_after,omitempty"`
}

// Multi-strategy limiter
type MultiStrategyLimiter struct {
    strategies map[string]RateLimiter
    selector   StrategySelector
    store      Store
}

func (m *MultiStrategyLimiter) Allow(ctx context.Context, key string) (bool, error) {
    // Select strategy based on context
    strategy := m.selector.Select(ctx, key)
    
    // Apply rate limit
    allowed, err := m.strategies[strategy].Allow(ctx, key)
    if err != nil {
        return false, err
    }
    
    // Record metrics
    m.recordMetrics(ctx, key, strategy, allowed)
    
    return allowed, nil
}
```

### Middleware Implementation

```go
// pkg/middleware/ratelimit.go
package middleware

func RateLimitMiddleware(limiter ratelimit.RateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Build rate limit key
        key := buildRateLimitKey(c)
        
        // Check rate limit
        allowed, err := limiter.Allow(c.Request.Context(), key)
        if err != nil {
            c.AbortWithStatusJSON(500, gin.H{
                "error": "Rate limit check failed",
            })
            return
        }
        
        if !allowed {
            // Get limit status
            status, _ := limiter.Status(c.Request.Context(), key)
            
            // Set rate limit headers
            c.Header("X-RateLimit-Limit", strconv.Itoa(status.Limit))
            c.Header("X-RateLimit-Remaining", strconv.Itoa(status.Remaining))
            c.Header("X-RateLimit-Reset", strconv.FormatInt(status.Reset.Unix(), 10))
            
            if status.RetryAfter != nil {
                c.Header("Retry-After", strconv.Itoa(int(status.RetryAfter.Seconds())))
            }
            
            c.AbortWithStatusJSON(429, gin.H{
                "error": "Rate limit exceeded",
                "message": "Too many requests",
                "retry_after": status.RetryAfter,
            })
            return
        }
        
        // Set rate limit headers for successful requests
        if status, err := limiter.Status(c.Request.Context(), key); err == nil {
            c.Header("X-RateLimit-Limit", strconv.Itoa(status.Limit))
            c.Header("X-RateLimit-Remaining", strconv.Itoa(status.Remaining))
            c.Header("X-RateLimit-Reset", strconv.FormatInt(status.Reset.Unix(), 10))
        }
        
        c.Next()
    }
}

func buildRateLimitKey(c *gin.Context) string {
    user := auth.GetUser(c)
    if user != nil {
        return fmt.Sprintf("user:%s", user.ID)
    }
    
    if apiKey := auth.GetAPIKey(c); apiKey != "" {
        return fmt.Sprintf("apikey:%s", hashAPIKey(apiKey))
    }
    
    // Fall back to IP-based limiting
    return fmt.Sprintf("ip:%s", c.ClientIP())
}
```

### GraphQL Complexity Limiter

```go
// pkg/graphql/complexity/limiter.go
package complexity

type ComplexityLimiter struct {
    limits map[string]int
    calculator ComplexityCalculator
}

func (l *ComplexityLimiter) Middleware(next graphql.Handler) graphql.Handler {
    return graphql.HandlerFunc(func(ctx context.Context, query string, variables map[string]interface{}) *graphql.Result {
        // Parse query
        doc, err := parser.Parse(query)
        if err != nil {
            return graphql.ErrorResult(err)
        }
        
        // Calculate complexity
        complexity := l.calculator.Calculate(doc, variables)
        
        // Get user limit
        user := auth.GetUserFromContext(ctx)
        limit := l.getUserLimit(user)
        
        // Check complexity
        if complexity > limit {
            return graphql.ErrorResult(ErrComplexityExceeded{
                Complexity: complexity,
                Limit:      limit,
            })
        }
        
        // Add complexity to context for logging
        ctx = context.WithValue(ctx, "query_complexity", complexity)
        
        return next.Handle(ctx, query, variables)
    })
}
```

## Configuration Management

### Dynamic Configuration

```yaml
# Rate Limit Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: gunj-operator-ratelimits
data:
  config.yaml: |
    # Global settings
    global:
      enabled: true
      default_strategy: token_bucket
      redis_required: false
      
    # User tiers
    tiers:
      anonymous:
        rph: 100
        burst: 20
        
      standard:
        rph: 5000
        burst: 500
        
    # Endpoint overrides
    endpoints:
      - path: "/api/v1/platforms"
        method: POST
        tier_multiplier: 0.1
        
      - path: "/api/v1/metrics"
        method: GET
        cache_ttl: 60s
        
    # Feature flags
    features:
      adaptive_limits: true
      graphql_complexity: true
      distributed_limiting: true
```

### Runtime Updates

```go
// pkg/ratelimit/config/watcher.go
package config

type ConfigWatcher struct {
    source ConfigSource
    limiter *MultiStrategyLimiter
    interval time.Duration
}

func (w *ConfigWatcher) Watch(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            newConfig, err := w.source.Load()
            if err != nil {
                log.Error("Failed to load config", "error", err)
                continue
            }
            
            if w.hasChanged(newConfig) {
                log.Info("Updating rate limit configuration")
                w.limiter.UpdateConfig(newConfig)
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

## Monitoring & Alerting

### Metrics Collection

```yaml
# Prometheus Metrics
metrics:
  # Rate limit metrics
  - name: gunj_ratelimit_requests_total
    type: counter
    labels: [user_tier, endpoint, method, result]
    
  - name: gunj_ratelimit_remaining
    type: gauge
    labels: [user_tier, limit_key]
    
  - name: gunj_ratelimit_violations_total
    type: counter
    labels: [user_tier, endpoint, reason]
    
  # Quota metrics
  - name: gunj_quota_usage
    type: gauge
    labels: [user, resource_type, namespace]
    
  - name: gunj_quota_limit
    type: gauge
    labels: [user, resource_type, namespace]
    
  # Performance metrics
  - name: gunj_ratelimit_check_duration_seconds
    type: histogram
    buckets: [0.001, 0.005, 0.01, 0.05, 0.1]
```

### Alert Rules

```yaml
# Alerting Rules
groups:
  - name: rate_limiting
    rules:
      # High violation rate
      - alert: HighRateLimitViolations
        expr: |
          rate(gunj_ratelimit_violations_total[5m]) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High rate limit violations"
          description: "{{ $value }} violations per second"
          
      # Quota exhaustion
      - alert: QuotaExhaustion
        expr: |
          gunj_quota_usage / gunj_quota_limit > 0.9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Quota near exhaustion"
          description: "{{ $labels.user }} at {{ $value | humanizePercentage }} of quota"
          
      # Redis unavailable
      - alert: RateLimitRedisDown
        expr: |
          up{job="redis-ratelimit"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Rate limit Redis is down"
          description: "Falling back to local rate limiting"
```

### Dashboard Queries

```sql
-- Top API consumers
SELECT 
    user_id,
    COUNT(*) as request_count,
    SUM(CASE WHEN allowed THEN 1 ELSE 0 END) as allowed_count,
    SUM(CASE WHEN allowed THEN 0 ELSE 1 END) as denied_count,
    AVG(response_time_ms) as avg_response_time
FROM api_requests
WHERE timestamp > NOW() - INTERVAL '1 hour'
GROUP BY user_id
ORDER BY request_count DESC
LIMIT 20;

-- Rate limit violations by endpoint
SELECT 
    endpoint,
    method,
    COUNT(*) as violation_count,
    COUNT(DISTINCT user_id) as unique_users
FROM rate_limit_violations
WHERE timestamp > NOW() - INTERVAL '1 hour'
GROUP BY endpoint, method
ORDER BY violation_count DESC;

-- Quota usage trends
SELECT 
    date_trunc('hour', timestamp) as hour,
    resource_type,
    MAX(usage) as max_usage,
    AVG(usage) as avg_usage,
    COUNT(DISTINCT user_id) as active_users
FROM quota_usage
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY hour, resource_type
ORDER BY hour DESC;
```

## Best Practices

### 1. Graceful Degradation

```yaml
# Degradation Strategy
degradation:
  # When Redis is unavailable
  redis_failure:
    strategy: local_limiting
    limit_reduction: 20%  # Reduce limits by 20%
    
  # When system is overloaded
  high_load:
    cpu_threshold: 80%
    memory_threshold: 85%
    
    actions:
      - reduce_limits: 50%
      - disable_expensive_endpoints: true
      - enable_caching: aggressive
```

### 2. Client Guidelines

```javascript
// Client-side rate limit handling
class APIClient {
    async request(url, options = {}) {
        const response = await fetch(url, options);
        
        // Check rate limit headers
        const remaining = parseInt(response.headers.get('X-RateLimit-Remaining'));
        const reset = parseInt(response.headers.get('X-RateLimit-Reset'));
        
        if (response.status === 429) {
            const retryAfter = response.headers.get('Retry-After');
            
            // Exponential backoff
            const delay = retryAfter 
                ? parseInt(retryAfter) * 1000 
                : this.calculateBackoff();
                
            await this.delay(delay);
            
            // Retry request
            return this.request(url, options);
        }
        
        // Proactive rate limiting
        if (remaining < 10) {
            const now = Date.now() / 1000;
            const waitTime = Math.max(0, reset - now);
            console.warn(`Rate limit nearly exhausted. Reset in ${waitTime}s`);
        }
        
        return response;
    }
    
    calculateBackoff() {
        this.retryCount = (this.retryCount || 0) + 1;
        return Math.min(1000 * Math.pow(2, this.retryCount), 30000);
    }
}
```

### 3. Testing Rate Limits

```go
// test/ratelimit/integration_test.go
func TestRateLimiting(t *testing.T) {
    // Test burst capacity
    t.Run("BurstCapacity", func(t *testing.T) {
        limiter := NewTokenBucketLimiter(100, 500) // 100/min, 500 burst
        
        // Should allow burst
        for i := 0; i < 500; i++ {
            allowed, _ := limiter.Allow(ctx, "test-user")
            assert.True(t, allowed, "Request %d should be allowed", i)
        }
        
        // 501st request should be denied
        allowed, _ := limiter.Allow(ctx, "test-user")
        assert.False(t, allowed, "Request should be denied after burst")
    })
    
    // Test refill rate
    t.Run("RefillRate", func(t *testing.T) {
        limiter := NewTokenBucketLimiter(60, 60) // 1/sec
        
        // Exhaust tokens
        for i := 0; i < 60; i++ {
            limiter.Allow(ctx, "test-user")
        }
        
        // Wait for refill
        time.Sleep(2 * time.Second)
        
        // Should have 2 tokens
        allowed1, _ := limiter.Allow(ctx, "test-user")
        allowed2, _ := limiter.Allow(ctx, "test-user")
        allowed3, _ := limiter.Allow(ctx, "test-user")
        
        assert.True(t, allowed1)
        assert.True(t, allowed2)
        assert.False(t, allowed3)
    })
}
```

### 4. Rate Limit Headers

```http
# Successful request
HTTP/1.1 200 OK
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 4999
X-RateLimit-Reset: 1718240400
X-RateLimit-Resource: /api/v1/platforms
X-RateLimit-Used: 1

# Rate limited request
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 5000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1718240400
Retry-After: 3600
Content-Type: application/json

{
  "error": "Rate limit exceeded",
  "message": "API rate limit of 5000 requests per hour exceeded",
  "retry_after": 3600,
  "upgrade_url": "https://gunj-operator.com/pricing"
}
```

## Implementation Checklist

### Phase 1: Core Rate Limiting (Week 1)
- [ ] Token bucket implementation
- [ ] Basic middleware
- [ ] User tier management
- [ ] Rate limit headers
- [ ] Local storage

### Phase 2: Distributed System (Week 2)
- [ ] Redis integration
- [ ] Lua scripts
- [ ] Synchronization protocol
- [ ] Failover handling
- [ ] Performance optimization

### Phase 3: Advanced Features (Week 3)
- [ ] GraphQL complexity limiting
- [ ] Quota management
- [ ] Adaptive rate limiting
- [ ] Cost tracking
- [ ] Usage analytics

### Phase 4: Operations (Week 4)
- [ ] Monitoring dashboards
- [ ] Alert rules
- [ ] Documentation
- [ ] Client libraries
- [ ] Load testing