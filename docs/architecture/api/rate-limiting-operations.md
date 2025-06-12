# Rate Limiting Monitoring & Operations Guide

This guide covers monitoring, alerting, and operational aspects of the Gunj Operator rate limiting system.

## Monitoring Dashboard

### Grafana Dashboard JSON

```json
{
  "dashboard": {
    "title": "Gunj Operator - Rate Limiting",
    "panels": [
      {
        "title": "Request Rate by User Tier",
        "targets": [
          {
            "expr": "sum(rate(gunj_ratelimit_requests_total[5m])) by (user_tier)"
          }
        ],
        "type": "graph",
        "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 }
      },
      {
        "title": "Rate Limit Violations",
        "targets": [
          {
            "expr": "sum(rate(gunj_ratelimit_violations_total[5m])) by (endpoint)"
          }
        ],
        "type": "graph",
        "gridPos": { "h": 8, "w": 12, "x": 12, "y": 0 }
      },
      {
        "title": "Current Token Usage",
        "targets": [
          {
            "expr": "(gunj_ratelimit_limit - gunj_ratelimit_remaining) / gunj_ratelimit_limit * 100"
          }
        ],
        "type": "gauge",
        "gridPos": { "h": 8, "w": 6, "x": 0, "y": 8 }
      },
      {
        "title": "Top API Consumers",
        "targets": [
          {
            "expr": "topk(10, sum(rate(gunj_ratelimit_requests_total[1h])) by (user_id))"
          }
        ],
        "type": "table",
        "gridPos": { "h": 8, "w": 12, "x": 6, "y": 8 }
      },
      {
        "title": "GraphQL Query Complexity",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, gunj_graphql_query_complexity_bucket)"
          }
        ],
        "type": "graph",
        "gridPos": { "h": 8, "w": 12, "x": 0, "y": 16 }
      },
      {
        "title": "Rate Limiter Performance",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, gunj_ratelimit_check_duration_seconds_bucket)"
          }
        ],
        "type": "graph",
        "gridPos": { "h": 8, "w": 12, "x": 12, "y": 16 }
      }
    ]
  }
}
```

### Key Metrics to Monitor

```yaml
# Essential Rate Limiting Metrics
metrics:
  # Request patterns
  - metric: gunj_ratelimit_requests_total
    description: Total API requests
    alert_on: Sudden spike or drop
    
  - metric: gunj_ratelimit_violations_total
    description: Rate limit violations
    alert_on: High violation rate
    
  # Resource usage
  - metric: gunj_quota_usage_ratio
    description: Quota usage percentage
    alert_on: > 90% usage
    
  # Performance
  - metric: gunj_ratelimit_check_duration_seconds
    description: Rate limit check latency
    alert_on: p99 > 10ms
    
  # System health
  - metric: gunj_ratelimit_redis_errors_total
    description: Redis connection errors
    alert_on: Any errors
```

## Alerting Rules

### Prometheus Alert Rules

```yaml
groups:
  - name: rate_limiting_alerts
    interval: 30s
    rules:
      # High rate limit violation rate
      - alert: HighRateLimitViolationRate
        expr: |
          (
            sum(rate(gunj_ratelimit_violations_total[5m])) 
            / 
            sum(rate(gunj_ratelimit_requests_total[5m]))
          ) > 0.1
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "High rate limit violation rate: {{ $value | humanizePercentage }}"
          description: "More than 10% of requests are being rate limited"
          runbook_url: "https://docs.gunj-operator.com/runbooks/high-rate-limit-violations"
          
      # Specific user consuming too much
      - alert: UserApproachingRateLimit
        expr: |
          (gunj_ratelimit_limit - gunj_ratelimit_remaining) / gunj_ratelimit_limit > 0.9
        for: 2m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "User {{ $labels.user_id }} approaching rate limit"
          description: "User has consumed {{ $value | humanizePercentage }} of their rate limit"
          
      # Quota exhaustion
      - alert: QuotaNearExhaustion
        expr: |
          gunj_quota_usage / gunj_quota_limit > 0.9
        for: 10m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "{{ $labels.resource_type }} quota near exhaustion"
          description: "User {{ $labels.user_id }} at {{ $value | humanizePercentage }} of quota"
          action: "Consider increasing quota or notifying user"
          
      # Redis unavailable
      - alert: RateLimitRedisDown
        expr: |
          up{job="redis-ratelimit"} == 0
        for: 1m
        labels:
          severity: critical
          team: platform
        annotations:
          summary: "Rate limiting Redis cluster is down"
          description: "Rate limiting falling back to local mode with reduced limits"
          impact: "Rate limits reduced by 20% in degraded mode"
          
      # Performance degradation
      - alert: RateLimitLatencyHigh
        expr: |
          histogram_quantile(0.99, gunj_ratelimit_check_duration_seconds_bucket) > 0.01
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "Rate limit check latency high"
          description: "P99 latency is {{ $value }}s (threshold: 10ms)"
          
      # Distributed sync issues
      - alert: RateLimitSyncLag
        expr: |
          gunj_ratelimit_sync_lag_seconds > 5
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "Rate limit synchronization lagging"
          description: "Sync lag is {{ $value }}s between instances"
```

### PagerDuty Integration

```yaml
# PagerDuty alert routing
route:
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
  receiver: 'platform-team'
  
  routes:
    # Critical alerts
    - match:
        severity: critical
      receiver: 'pagerduty-critical'
      
    # Rate limit specific alerts
    - match_re:
        alertname: '^RateLimit.*'
      receiver: 'rate-limit-team'
      group_by: ['user_id']
      
receivers:
  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: '<service-key>'
        severity: 'critical'
        client: 'Gunj Operator'
        client_url: 'https://gunj-operator.com/alerts'
```

## Operational Procedures

### Rate Limit Adjustment Playbook

```bash
#!/bin/bash
# scripts/adjust-rate-limits.sh

# Function to update rate limits
update_rate_limit() {
    local user_id=$1
    local new_limit=$2
    local reason=$3
    
    echo "Updating rate limit for user: $user_id"
    echo "New limit: $new_limit requests/hour"
    echo "Reason: $reason"
    
    # Update in database
    kubectl exec -n gunj-system deployment/gunj-api -- \
        gunj-cli ratelimit update \
        --user-id "$user_id" \
        --limit "$new_limit" \
        --reason "$reason"
    
    # Clear cache
    kubectl exec -n gunj-system deployment/redis-master -- \
        redis-cli DEL "gunj:ratelimit:user:$user_id"
    
    # Log the change
    echo "$(date): Rate limit updated for $user_id to $new_limit - $reason" >> /var/log/gunj/rate-limit-changes.log
}

# Emergency rate limit increase
emergency_increase() {
    local user_id=$1
    
    # Get current limit
    current=$(kubectl exec -n gunj-system deployment/gunj-api -- \
        gunj-cli ratelimit get --user-id "$user_id" | jq -r .limit)
    
    # Increase by 50%
    new_limit=$((current * 3 / 2))
    
    update_rate_limit "$user_id" "$new_limit" "Emergency increase - incident response"
    
    # Set reminder to review
    echo "Review rate limit for $user_id" | at now + 1 day
}
```

### Redis Failover Procedure

```yaml
# Standard Operating Procedure: Redis Failover
procedure:
  name: rate-limit-redis-failover
  severity: high
  estimated_time: 15min
  
steps:
  - name: Verify Redis failure
    commands:
      - kubectl get pods -n gunj-system -l app=redis-ratelimit
      - kubectl logs -n gunj-system -l app=redis-ratelimit --tail=50
      
  - name: Enable local fallback
    description: Rate limiter should automatically fallback
    verify:
      - Check logs for "Falling back to local rate limiting"
      - Verify reduced limits are applied (80% of normal)
      
  - name: Investigate root cause
    checks:
      - Network connectivity
      - Resource exhaustion
      - Redis configuration
      
  - name: Restore Redis cluster
    options:
      - Restart pods: kubectl rollout restart statefulset/redis-ratelimit
      - Scale up: kubectl scale statefulset/redis-ratelimit --replicas=5
      - Restore from backup if data corruption
      
  - name: Verify recovery
    commands:
      - redis-cli -h redis-ratelimit PING
      - Check metrics: gunj_ratelimit_redis_connected
      
  - name: Clear local caches
    description: Force resync with Redis
    command: kubectl rollout restart deployment/gunj-api
```

### Debugging Rate Limit Issues

```bash
# Debug script for rate limit issues
#!/bin/bash

debug_user_ratelimit() {
    local user_id=$1
    
    echo "=== Rate Limit Debug for User: $user_id ==="
    
    # Check current limits
    echo -e "\n1. Current Rate Limits:"
    curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
        "https://api.gunj-operator.com/admin/users/$user_id/ratelimits" | jq .
    
    # Check recent requests
    echo -e "\n2. Recent Request History:"
    kubectl exec -n gunj-system deployment/gunj-api -- \
        gunj-cli audit --user-id "$user_id" --limit 20
    
    # Check Redis state
    echo -e "\n3. Redis Token Bucket State:"
    kubectl exec -n gunj-system deployment/redis-master -- \
        redis-cli HGETALL "gunj:ratelimit:user:$user_id"
    
    # Check current usage
    echo -e "\n4. Current Usage Metrics:"
    curl -s "http://prometheus:9090/api/v1/query?query=gunj_ratelimit_remaining{user_id=\"$user_id\"}" | \
        jq '.data.result[0].value[1]'
    
    # Check for violations
    echo -e "\n5. Recent Violations:"
    curl -s "http://prometheus:9090/api/v1/query_range?query=gunj_ratelimit_violations_total{user_id=\"$user_id\"}&start=$(date -d '1 hour ago' +%s)&end=$(date +%s)&step=60" | \
        jq '.data.result'
}
```

## Performance Tuning

### Redis Optimization

```lua
-- Optimized Lua script for token bucket
-- scripts/token_bucket_optimized.lua

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local requested = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local burst_multiplier = tonumber(ARGV[5]) or 1

-- Get current state with single call
local state = redis.call('HMGET', key, 'tokens', 'last_refill', 'burst_tokens', 'burst_start')
local tokens = tonumber(state[1]) or capacity
local last_refill = tonumber(state[2]) or now
local burst_tokens = tonumber(state[3]) or 0
local burst_start = tonumber(state[4]) or 0

-- Calculate refill
local elapsed = math.max(0, now - last_refill)
local tokens_to_add = elapsed * refill_rate
tokens = math.min(tokens + tokens_to_add, capacity)

-- Check burst window (1 minute)
if now - burst_start > 60 then
    burst_tokens = capacity * (burst_multiplier - 1)
    burst_start = now
end

-- Check total available (regular + burst)
local total_available = tokens + burst_tokens

if total_available >= requested then
    -- Deduct from regular tokens first
    if tokens >= requested then
        tokens = tokens - requested
    else
        local from_burst = requested - tokens
        tokens = 0
        burst_tokens = burst_tokens - from_burst
    end
    
    -- Update state atomically
    redis.call('HMSET', key,
        'tokens', tokens,
        'last_refill', now,
        'burst_tokens', burst_tokens,
        'burst_start', burst_start
    )
    redis.call('EXPIRE', key, 3600)
    
    return {1, tokens, burst_tokens}
else
    -- Update refill time even on denial
    redis.call('HSET', key, 'last_refill', now)
    redis.call('EXPIRE', key, 3600)
    
    -- Calculate wait time
    local needed = requested - total_available
    local wait = needed / refill_rate
    
    return {0, tokens, burst_tokens, wait}
end
```

### Local Cache Optimization

```go
// pkg/ratelimit/cache/local.go
package cache

import (
    "sync"
    "time"
    
    "github.com/hashicorp/golang-lru/v2"
)

type LocalCache struct {
    lru        *lru.Cache[string, *entry]
    mu         sync.RWMutex
    ttl        time.Duration
    maxEntries int
}

type entry struct {
    value      interface{}
    expiration time.Time
}

func NewLocalCache(maxEntries int, ttl time.Duration) *LocalCache {
    cache, _ := lru.New[string, *entry](maxEntries)
    
    lc := &LocalCache{
        lru:        cache,
        ttl:        ttl,
        maxEntries: maxEntries,
    }
    
    // Start cleanup routine
    go lc.cleanupExpired()
    
    return lc
}

func (c *LocalCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if e, ok := c.lru.Get(key); ok {
        if time.Now().Before(e.expiration) {
            return e.value, true
        }
        
        // Expired, remove it
        c.lru.Remove(key)
    }
    
    return nil, false
}

func (c *LocalCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.lru.Add(key, &entry{
        value:      value,
        expiration: time.Now().Add(c.ttl),
    })
}

func (c *LocalCache) cleanupExpired() {
    ticker := time.NewTicker(c.ttl / 2)
    defer ticker.Stop()
    
    for range ticker.C {
        c.mu.Lock()
        
        // Get all keys
        keys := c.lru.Keys()
        now := time.Now()
        
        for _, key := range keys {
            if e, ok := c.lru.Peek(key); ok {
                if now.After(e.expiration) {
                    c.lru.Remove(key)
                }
            }
        }
        
        c.mu.Unlock()
    }
}
```

## Capacity Planning

### Load Testing Results

```yaml
# Rate Limiter Capacity Benchmarks
benchmarks:
  single_instance:
    setup:
      cpu: 2 cores
      memory: 4GB
      redis: single node
      
    results:
      max_rps: 50000
      p99_latency: 5ms
      memory_usage: 2.1GB
      
  ha_cluster:
    setup:
      instances: 3
      cpu_per_instance: 4 cores
      memory_per_instance: 8GB
      redis: 3 node cluster
      
    results:
      max_rps: 150000
      p99_latency: 3ms
      failover_time: 2s
      
  recommendations:
    - 1000 users: Single instance sufficient
    - 10000 users: HA cluster with 3 instances
    - 100000 users: HA cluster with 5+ instances + Redis cluster
```

### Resource Sizing Guide

```yaml
# Resource Requirements by User Scale
sizing:
  small:
    users: 1000
    api_instances: 1
    redis: standalone
    resources:
      api_cpu: 2
      api_memory: 4Gi
      redis_memory: 2Gi
      
  medium:
    users: 10000
    api_instances: 3
    redis: sentinel (3 nodes)
    resources:
      api_cpu: 4
      api_memory: 8Gi
      redis_memory: 4Gi
      
  large:
    users: 100000
    api_instances: 5
    redis: cluster (6 nodes)
    resources:
      api_cpu: 8
      api_memory: 16Gi
      redis_memory: 8Gi
      
  enterprise:
    users: 1000000+
    api_instances: 10+
    redis: cluster (12+ nodes)
    resources:
      api_cpu: 16
      api_memory: 32Gi
      redis_memory: 16Gi
```

## Troubleshooting Guide

### Common Issues and Solutions

```yaml
issues:
  - symptom: "Legitimate users getting rate limited"
    causes:
      - Rate limits too low
      - Shared IP addresses (NAT)
      - Burst traffic patterns
    solutions:
      - Review and adjust limits
      - Implement user-based limiting instead of IP
      - Increase burst allowance
      
  - symptom: "Rate limits not being enforced"
    causes:
      - Redis connection issues
      - Middleware not applied
      - Cache inconsistency
    solutions:
      - Check Redis connectivity
      - Verify middleware chain
      - Clear local caches
      
  - symptom: "High latency on rate limit checks"
    causes:
      - Redis performance issues
      - Network latency
      - Lock contention
    solutions:
      - Optimize Redis configuration
      - Use local caching
      - Implement read replicas
      
  - symptom: "Inconsistent limits across instances"
    causes:
      - Sync lag
      - Configuration drift
      - Cache inconsistency
    solutions:
      - Reduce sync interval
      - Centralize configuration
      - Implement cache invalidation
```

This operations guide provides comprehensive monitoring, alerting, and troubleshooting information for managing the rate limiting system in production.