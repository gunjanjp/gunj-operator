# Rate Limiting Implementation Guide

This guide provides practical implementation patterns for the Gunj Operator rate limiting system.

## Token Bucket Implementation

### Core Algorithm

```go
// pkg/ratelimit/tokenbucket/bucket.go
package tokenbucket

import (
    "sync"
    "time"
)

type Bucket struct {
    capacity    float64
    tokens      float64
    refillRate  float64
    lastRefill  time.Time
    mu          sync.Mutex
}

func NewBucket(capacity, refillRate float64) *Bucket {
    return &Bucket{
        capacity:   capacity,
        tokens:     capacity, // Start full
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

func (b *Bucket) Allow(cost float64) bool {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    // Refill tokens based on time elapsed
    b.refill()
    
    // Check if enough tokens available
    if b.tokens >= cost {
        b.tokens -= cost
        return true
    }
    
    return false
}

func (b *Bucket) AllowWait(cost float64) time.Duration {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.refill()
    
    if b.tokens >= cost {
        b.tokens -= cost
        return 0
    }
    
    // Calculate wait time
    tokensNeeded := cost - b.tokens
    waitSeconds := tokensNeeded / b.refillRate
    return time.Duration(waitSeconds * float64(time.Second))
}

func (b *Bucket) refill() {
    now := time.Now()
    elapsed := now.Sub(b.lastRefill).Seconds()
    
    // Add tokens based on elapsed time
    tokensToAdd := elapsed * b.refillRate
    b.tokens = min(b.tokens+tokensToAdd, b.capacity)
    
    b.lastRefill = now
}

func (b *Bucket) Status() (remaining float64, resetAt time.Time) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    b.refill()
    
    // Calculate when bucket will be full
    if b.tokens < b.capacity {
        tokensNeeded := b.capacity - b.tokens
        secondsToFull := tokensNeeded / b.refillRate
        resetAt = time.Now().Add(time.Duration(secondsToFull * float64(time.Second)))
    } else {
        resetAt = time.Now()
    }
    
    return b.tokens, resetAt
}
```

### Distributed Token Bucket

```go
// pkg/ratelimit/distributed/redis_bucket.go
package distributed

import (
    "context"
    "time"
    
    "github.com/go-redis/redis/v8"
)

type RedisBucket struct {
    client     *redis.Client
    keyPrefix  string
    capacity   float64
    refillRate float64
    script     *redis.Script
}

const luaScript = `
-- Token bucket implementation in Lua
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

-- Get current bucket state
local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

-- Calculate refill
local elapsed = now - last_refill
local tokens_to_add = elapsed * refill_rate
tokens = math.min(tokens + tokens_to_add, capacity)

-- Check if request allowed
if tokens >= cost then
    tokens = tokens - cost
    
    -- Update bucket state
    redis.call('HMSET', key, 
        'tokens', tokens,
        'last_refill', now
    )
    redis.call('EXPIRE', key, 3600) -- 1 hour TTL
    
    return {1, tokens} -- Allowed, remaining tokens
else
    -- Update refill time even if denied
    redis.call('HSET', key, 'last_refill', now)
    redis.call('EXPIRE', key, 3600)
    
    -- Calculate wait time
    local tokens_needed = cost - tokens
    local wait_seconds = tokens_needed / refill_rate
    
    return {0, tokens, wait_seconds} -- Denied, remaining, wait time
end
`

func NewRedisBucket(client *redis.Client, keyPrefix string, capacity, refillRate float64) *RedisBucket {
    return &RedisBucket{
        client:     client,
        keyPrefix:  keyPrefix,
        capacity:   capacity,
        refillRate: refillRate,
        script:     redis.NewScript(luaScript),
    }
}

func (rb *RedisBucket) Allow(ctx context.Context, key string, cost float64) (bool, error) {
    fullKey := rb.keyPrefix + key
    now := float64(time.Now().Unix())
    
    result, err := rb.script.Run(ctx, rb.client, 
        []string{fullKey},
        rb.capacity, rb.refillRate, cost, now,
    ).Result()
    
    if err != nil {
        return false, err
    }
    
    vals := result.([]interface{})
    allowed := vals[0].(int64) == 1
    
    return allowed, nil
}
```

## Sliding Window Implementation

```go
// pkg/ratelimit/slidingwindow/window.go
package slidingwindow

import (
    "context"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
)

type SlidingWindow struct {
    client    *redis.Client
    keyPrefix string
    window    time.Duration
    limit     int
}

func (sw *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
    fullKey := sw.keyPrefix + key
    now := time.Now()
    
    // Use Redis pipeline for atomic operations
    pipe := sw.client.Pipeline()
    
    // Remove old entries
    minTime := now.Add(-sw.window).UnixNano()
    pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", minTime))
    
    // Count current entries
    count := pipe.ZCard(ctx, fullKey)
    
    // Execute pipeline
    _, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    
    // Check limit
    currentCount, err := count.Result()
    if err != nil {
        return false, err
    }
    
    if currentCount >= int64(sw.limit) {
        return false, nil
    }
    
    // Add new entry
    err = sw.client.ZAdd(ctx, fullKey, &redis.Z{
        Score:  float64(now.UnixNano()),
        Member: now.UnixNano(),
    }).Err()
    if err != nil {
        return false, err
    }
    
    // Set expiration
    sw.client.Expire(ctx, fullKey, sw.window)
    
    return true, nil
}

func (sw *SlidingWindow) Status(ctx context.Context, key string) (*LimitStatus, error) {
    fullKey := sw.keyPrefix + key
    now := time.Now()
    minTime := now.Add(-sw.window).UnixNano()
    
    // Count current requests
    count, err := sw.client.ZCount(ctx, fullKey, 
        fmt.Sprintf("%d", minTime),
        fmt.Sprintf("%d", now.UnixNano()),
    ).Result()
    if err != nil {
        return nil, err
    }
    
    // Get oldest entry for reset time
    oldest, err := sw.client.ZRangeWithScores(ctx, fullKey, 0, 0).Result()
    
    var resetTime time.Time
    if len(oldest) > 0 {
        oldestTime := int64(oldest[0].Score)
        resetTime = time.Unix(0, oldestTime).Add(sw.window)
    } else {
        resetTime = now.Add(sw.window)
    }
    
    remaining := sw.limit - int(count)
    if remaining < 0 {
        remaining = 0
    }
    
    return &LimitStatus{
        Limit:     sw.limit,
        Remaining: remaining,
        Reset:     resetTime,
    }, nil
}
```

## GraphQL Complexity Calculator

```go
// pkg/graphql/complexity/calculator.go
package complexity

import (
    "github.com/graphql-go/graphql/language/ast"
    "github.com/graphql-go/graphql/language/visitor"
)

type ComplexityCalculator struct {
    schema     *graphql.Schema
    fieldCosts map[string]int
}

func (c *ComplexityCalculator) Calculate(query ast.Document, variables map[string]interface{}) int {
    complexity := 0
    
    visitor.Visit(&query, visitor.VisitFuncs{
        Field: func(p visitor.VisitParams) (string, interface{}) {
            field, ok := p.Node.(*ast.Field)
            if !ok {
                return visitor.ActionNoChange, nil
            }
            
            // Get field cost
            fieldName := field.Name.Value
            cost := c.getFieldCost(fieldName, field.Arguments, variables)
            
            // Check for multiplier directives
            multiplier := c.getMultiplier(field, variables)
            
            complexity += cost * multiplier
            
            return visitor.ActionNoChange, nil
        },
    })
    
    return complexity
}

func (c *ComplexityCalculator) getFieldCost(fieldName string, args []*ast.Argument, variables map[string]interface{}) int {
    // Check custom costs
    if cost, ok := c.fieldCosts[fieldName]; ok {
        return cost
    }
    
    // Default costs based on return type
    switch fieldName {
    case "platforms", "events", "metrics":
        // List fields have higher cost
        limit := c.getLimit(args, variables)
        return 1 + limit
    default:
        return 1
    }
}

func (c *ComplexityCalculator) getLimit(args []*ast.Argument, variables map[string]interface{}) int {
    for _, arg := range args {
        if arg.Name.Value == "limit" {
            switch val := arg.Value.(type) {
            case *ast.IntValue:
                limit, _ := strconv.Atoi(val.Value)
                return limit
            case *ast.Variable:
                if v, ok := variables[val.Name.Value]; ok {
                    if limit, ok := v.(int); ok {
                        return limit
                    }
                }
            }
        }
    }
    return 10 // Default limit
}
```

## Quota Manager

```go
// pkg/quota/manager.go
package quota

import (
    "context"
    "fmt"
    "sync"
    "time"
)

type Manager struct {
    store  Store
    limits map[string]ResourceLimit
    mu     sync.RWMutex
}

type ResourceLimit struct {
    Type      string
    Limit     int64
    Period    time.Duration
    Namespace string
}

type Usage struct {
    Used      int64
    Limit     int64
    Reset     time.Time
    Namespace string
}

func (m *Manager) CheckQuota(ctx context.Context, userID, resourceType string, requested int64) error {
    // Get current usage
    usage, err := m.GetUsage(ctx, userID, resourceType)
    if err != nil {
        return err
    }
    
    // Check if quota would be exceeded
    if usage.Used + requested > usage.Limit {
        return &QuotaExceededError{
            Resource:  resourceType,
            Used:      usage.Used,
            Requested: requested,
            Limit:     usage.Limit,
        }
    }
    
    return nil
}

func (m *Manager) ConsumeQuota(ctx context.Context, userID, resourceType string, amount int64) error {
    // Start transaction
    tx, err := m.store.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Lock user's quota row
    usage, err := tx.GetUsageForUpdate(ctx, userID, resourceType)
    if err != nil {
        return err
    }
    
    // Check quota
    limit := m.getLimit(userID, resourceType)
    if usage + amount > limit {
        return &QuotaExceededError{
            Resource:  resourceType,
            Used:      usage,
            Requested: amount,
            Limit:     limit,
        }
    }
    
    // Update usage
    err = tx.UpdateUsage(ctx, userID, resourceType, usage + amount)
    if err != nil {
        return err
    }
    
    // Commit transaction
    return tx.Commit()
}

func (m *Manager) GetUsage(ctx context.Context, userID, resourceType string) (*Usage, error) {
    // Get from cache first
    cacheKey := fmt.Sprintf("%s:%s", userID, resourceType)
    if cached, ok := m.cache.Get(cacheKey); ok {
        return cached.(*Usage), nil
    }
    
    // Get from store
    used, err := m.store.GetUsage(ctx, userID, resourceType)
    if err != nil {
        return nil, err
    }
    
    limit := m.getLimit(userID, resourceType)
    period := m.getPeriod(resourceType)
    
    usage := &Usage{
        Used:      used,
        Limit:     limit,
        Reset:     m.getResetTime(period),
        Namespace: m.getUserNamespace(userID),
    }
    
    // Cache for 1 minute
    m.cache.Set(cacheKey, usage, time.Minute)
    
    return usage, nil
}
```

## Rate Limit Middleware Examples

### REST API Middleware

```go
// pkg/middleware/ratelimit_rest.go
package middleware

func CreateRateLimitMiddleware(config *Config) gin.HandlerFunc {
    // Create limiters
    limiters := map[string]RateLimiter{
        "token_bucket": NewTokenBucketLimiter(config),
        "sliding_window": NewSlidingWindowLimiter(config),
        "adaptive": NewAdaptiveLimiter(config),
    }
    
    return func(c *gin.Context) {
        // Skip rate limiting for health checks
        if c.Request.URL.Path == "/health" {
            c.Next()
            return
        }
        
        // Get rate limit key
        key := getRateLimitKey(c)
        
        // Get endpoint configuration
        endpoint := fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.Path)
        endpointConfig := config.GetEndpointConfig(endpoint)
        
        // Select limiter
        limiterType := endpointConfig.Strategy
        if limiterType == "" {
            limiterType = config.DefaultStrategy
        }
        
        limiter := limiters[limiterType]
        
        // Apply rate limit
        result, err := limiter.Check(c.Request.Context(), key, endpointConfig)
        if err != nil {
            c.AbortWithStatusJSON(500, gin.H{
                "error": "Rate limit check failed",
                "details": err.Error(),
            })
            return
        }
        
        // Set headers
        c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
        c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
        c.Header("X-RateLimit-Reset", strconv.FormatInt(result.Reset.Unix(), 10))
        
        if !result.Allowed {
            // Calculate retry after
            retryAfter := int(result.Reset.Sub(time.Now()).Seconds())
            if retryAfter < 1 {
                retryAfter = 1
            }
            
            c.Header("Retry-After", strconv.Itoa(retryAfter))
            
            c.AbortWithStatusJSON(429, gin.H{
                "error": "rate_limit_exceeded",
                "message": fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", retryAfter),
                "limit": result.Limit,
                "remaining": 0,
                "reset": result.Reset.Unix(),
            })
            return
        }
        
        // Track metrics
        rateLimitMetrics.WithLabelValues(
            getUserTier(c),
            endpoint,
            "allowed",
        ).Inc()
        
        c.Next()
    }
}
```

### GraphQL Middleware

```go
// pkg/middleware/ratelimit_graphql.go
package middleware

func CreateGraphQLRateLimitMiddleware(config *Config) graphql.FieldMiddleware {
    complexityLimiter := NewComplexityLimiter(config)
    depthLimiter := NewDepthLimiter(config)
    fieldLimiter := NewFieldRateLimiter(config)
    
    return func(ctx context.Context, next graphql.Resolver) graphql.Resolver {
        return func(ctx context.Context, args interface{}) (interface{}, error) {
            // Get field context
            fc := graphql.GetFieldContext(ctx)
            
            // Check field-level rate limit
            if rl := getFieldRateLimit(fc); rl != nil {
                key := getRateLimitKeyFromContext(ctx)
                allowed, err := fieldLimiter.Check(ctx, key, fc.Field.Name, rl)
                if err != nil {
                    return nil, err
                }
                if !allowed {
                    return nil, &RateLimitError{
                        Field: fc.Field.Name,
                        Limit: rl.Max,
                        Window: rl.Window,
                    }
                }
            }
            
            // Execute resolver
            result, err := next(ctx, args)
            
            // Track metrics
            graphqlMetrics.WithLabelValues(
                fc.Object,
                fc.Field.Name,
                "success",
            ).Inc()
            
            return result, err
        }
    }
}
```

## Client Libraries

### JavaScript/TypeScript Client

```typescript
// sdk/js/src/ratelimit.ts

export interface RateLimitInfo {
  limit: number;
  remaining: number;
  reset: Date;
}

export class RateLimitManager {
  private limits: Map<string, RateLimitInfo> = new Map();
  private backoffMultiplier = 1.5;
  private maxBackoff = 60000; // 60 seconds
  
  updateFromHeaders(headers: Headers): void {
    const limit = parseInt(headers.get('X-RateLimit-Limit') || '0');
    const remaining = parseInt(headers.get('X-RateLimit-Remaining') || '0');
    const reset = parseInt(headers.get('X-RateLimit-Reset') || '0');
    
    if (limit > 0) {
      this.limits.set('global', {
        limit,
        remaining,
        reset: new Date(reset * 1000),
      });
    }
  }
  
  shouldThrottle(): boolean {
    const info = this.limits.get('global');
    if (!info) return false;
    
    // Proactive throttling when < 10% remaining
    return info.remaining < info.limit * 0.1;
  }
  
  async handleRateLimit(response: Response): Promise<void> {
    if (response.status !== 429) return;
    
    const retryAfter = response.headers.get('Retry-After');
    const delay = retryAfter 
      ? parseInt(retryAfter) * 1000
      : this.calculateBackoff();
      
    await this.sleep(delay);
  }
  
  private calculateBackoff(): number {
    const jitter = Math.random() * 1000; // 0-1s jitter
    const backoff = Math.min(
      this.backoffMultiplier * 1000,
      this.maxBackoff
    );
    
    this.backoffMultiplier *= 2;
    
    return backoff + jitter;
  }
  
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
```

### Go Client

```go
// sdk/go/ratelimit/manager.go
package ratelimit

import (
    "context"
    "math"
    "net/http"
    "strconv"
    "sync"
    "time"
)

type Manager struct {
    mu              sync.RWMutex
    limits          map[string]*LimitInfo
    backoffStrategy BackoffStrategy
}

type LimitInfo struct {
    Limit     int
    Remaining int
    Reset     time.Time
    UpdatedAt time.Time
}

func (m *Manager) PreCheck(ctx context.Context, key string) error {
    m.mu.RLock()
    info, ok := m.limits[key]
    m.mu.RUnlock()
    
    if !ok {
        return nil // No limit info, proceed
    }
    
    // Check if we should proactively wait
    if info.Remaining == 0 && time.Now().Before(info.Reset) {
        waitTime := time.Until(info.Reset)
        
        select {
        case <-time.After(waitTime):
            return nil
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    // Warn if approaching limit
    if float64(info.Remaining) < float64(info.Limit)*0.1 {
        log.Warn("Approaching rate limit", 
            "remaining", info.Remaining,
            "limit", info.Limit,
            "reset", info.Reset,
        )
    }
    
    return nil
}

func (m *Manager) UpdateFromResponse(resp *http.Response) {
    limit := resp.Header.Get("X-RateLimit-Limit")
    remaining := resp.Header.Get("X-RateLimit-Remaining")
    reset := resp.Header.Get("X-RateLimit-Reset")
    
    if limit == "" || remaining == "" || reset == "" {
        return
    }
    
    limitInt, _ := strconv.Atoi(limit)
    remainingInt, _ := strconv.Atoi(remaining)
    resetInt, _ := strconv.ParseInt(reset, 10, 64)
    
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.limits["global"] = &LimitInfo{
        Limit:     limitInt,
        Remaining: remainingInt,
        Reset:     time.Unix(resetInt, 0),
        UpdatedAt: time.Now(),
    }
}
```

## Testing Rate Limits

### Load Testing Script

```go
// test/loadtest/ratelimit_test.go
package loadtest

import (
    "context"
    "sync"
    "sync/atomic"
    "testing"
    "time"
)

func TestRateLimitUnderLoad(t *testing.T) {
    client := NewTestClient()
    
    // Configuration
    concurrency := 100
    duration := 1 * time.Minute
    expectedRPS := 1000
    
    var (
        totalRequests   int64
        successRequests int64
        rateLimited     int64
        errors          int64
    )
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for {
                select {
                case <-ctx.Done():
                    return
                default:
                    atomic.AddInt64(&totalRequests, 1)
                    
                    resp, err := client.Get("/api/v1/platforms")
                    if err != nil {
                        atomic.AddInt64(&errors, 1)
                        continue
                    }
                    
                    switch resp.StatusCode {
                    case 200:
                        atomic.AddInt64(&successRequests, 1)
                    case 429:
                        atomic.AddInt64(&rateLimited, 1)
                    default:
                        atomic.AddInt64(&errors, 1)
                    }
                    
                    resp.Body.Close()
                }
            }
        }(i)
    }
    
    // Wait for completion
    wg.Wait()
    
    // Analyze results
    actualRPS := float64(successRequests) / duration.Seconds()
    rateLimitedPercent := float64(rateLimited) / float64(totalRequests) * 100
    
    t.Logf("Load test results:")
    t.Logf("- Total requests: %d", totalRequests)
    t.Logf("- Successful: %d", successRequests)
    t.Logf("- Rate limited: %d (%.2f%%)", rateLimited, rateLimitedPercent)
    t.Logf("- Errors: %d", errors)
    t.Logf("- Actual RPS: %.2f", actualRPS)
    
    // Assertions
    assert.InDelta(t, expectedRPS, actualRPS, expectedRPS*0.1, 
        "RPS should be within 10% of expected")
    
    assert.Less(t, errors, totalRequests/100, 
        "Error rate should be less than 1%")
}
```

### Unit Tests

```go
// pkg/ratelimit/tokenbucket/bucket_test.go
package tokenbucket

import (
    "testing"
    "time"
)

func TestTokenBucketRefill(t *testing.T) {
    // Create bucket with 10 tokens/sec, capacity 100
    bucket := NewBucket(100, 10)
    
    // Drain all tokens
    for i := 0; i < 100; i++ {
        allowed := bucket.Allow(1)
        assert.True(t, allowed, "Should allow request %d", i)
    }
    
    // Should be empty
    assert.False(t, bucket.Allow(1), "Bucket should be empty")
    
    // Wait for refill
    time.Sleep(100 * time.Millisecond)
    
    // Should have ~1 token (10 tokens/sec * 0.1 sec)
    assert.True(t, bucket.Allow(1), "Should have refilled 1 token")
    assert.False(t, bucket.Allow(1), "Should not have 2 tokens yet")
}

func TestTokenBucketBurst(t *testing.T) {
    // Create bucket with small refill rate but large capacity
    bucket := NewBucket(1000, 1) // 1 token/sec, 1000 capacity
    
    // Should handle burst
    start := time.Now()
    for i := 0; i < 1000; i++ {
        allowed := bucket.Allow(1)
        assert.True(t, allowed, "Should allow burst request %d", i)
    }
    elapsed := time.Since(start)
    
    // Burst should be fast (< 100ms)
    assert.Less(t, elapsed, 100*time.Millisecond, 
        "Burst should complete quickly")
}
```

This implementation guide provides practical code examples for implementing the rate limiting and quota management system in the Gunj Operator.