# Architecture Decision Record: Redis 8 as Caching Solution

**Date**: June 12, 2025  
**Status**: Accepted  
**Phase**: 1.2 - Technology Stack Finalization  
**Micro-task**: 1.2.1.4 - Select caching solution  

## Context

The Gunj Operator requires a high-performance caching solution to handle:
- Session management (if not using PostgreSQL)
- API response caching
- Temporary data for async operations
- Rate limiting counters
- Real-time metrics aggregation
- WebSocket connection state

## Decision

We have selected **Redis 8.0+** as the caching solution for the Gunj Operator.

## Rationale

### Why Redis 8?

1. **Performance Excellence**
   - In-memory data store with sub-millisecond latency
   - Optimized for high-throughput scenarios
   - Latest version with improved memory efficiency

2. **Feature Completeness**
   - Native support for all required data structures
   - Built-in pub/sub for real-time features
   - Redis Streams for event streaming
   - Lua scripting for complex atomic operations

3. **Production Readiness**
   - Battle-tested in enterprise environments
   - Excellent Kubernetes integration
   - Strong community and ecosystem

4. **Specific Feature Mapping**:
   | Use Case | Redis Feature | Implementation |
   |----------|--------------|----------------|
   | Session Management | Strings/Hashes with TTL | JWT tokens with metadata |
   | API Response Cache | Strings with expiration | JSON responses with cache keys |
   | Async Operations | Lists/Streams | Job queues and task processing |
   | Rate Limiting | Sorted Sets/Lua | Sliding window algorithm |
   | Metrics Aggregation | Streams/Time Series | Real-time data pipeline |
   | WebSocket State | Pub/Sub/Hashes | Connection registry and broadcasts |

## Implementation Details

### Deployment Architecture

```yaml
# Redis Sentinel Configuration for HA
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: gunj-system
data:
  redis.conf: |
    # Basic Configuration
    bind 0.0.0.0
    protected-mode yes
    port 6379
    tcp-backlog 511
    timeout 300
    tcp-keepalive 300
    
    # Memory Management
    maxmemory 4gb
    maxmemory-policy allkeys-lru
    maxmemory-samples 5
    
    # Persistence
    save 900 1
    save 300 10
    save 60 10000
    stop-writes-on-bgsave-error yes
    rdbcompression yes
    rdbchecksum yes
    dbfilename dump.rdb
    
    # AOF
    appendonly yes
    appendfilename "appendonly.aof"
    appendfsync everysec
    no-appendfsync-on-rewrite no
    
    # Security
    requirepass ${REDIS_PASSWORD}
    
    # ACL Configuration
    aclfile /etc/redis/users.acl
    
    # TLS/SSL
    tls-port 6380
    tls-cert-file /tls/tls.crt
    tls-key-file /tls/tls.key
    tls-ca-cert-file /tls/ca.crt
    tls-dh-params-file /tls/redis.dh
```

### High Availability Setup

1. **Redis Sentinel** (3 nodes minimum)
   - Automatic failover
   - Configuration management
   - Monitoring and notification

2. **Persistence Strategy**
   - RDB snapshots every 5 minutes
   - AOF with per-second fsync
   - Backup to object storage

3. **Security**
   - TLS for all connections
   - ACL for user management
   - Network policies for access control

### Go Client Configuration

```go
package cache

import (
    "context"
    "crypto/tls"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type RedisConfig struct {
    Addresses  []string
    Password   string
    MasterName string
    TLSConfig  *tls.Config
    PoolSize   int
    MaxRetries int
}

func NewRedisClient(cfg RedisConfig) *redis.Client {
    return redis.NewFailoverClient(&redis.FailoverOptions{
        MasterName:    cfg.MasterName,
        SentinelAddrs: cfg.Addresses,
        Password:      cfg.Password,
        DB:            0,
        
        // Connection pool
        PoolSize:     cfg.PoolSize,
        MinIdleConns: 10,
        MaxRetries:   cfg.MaxRetries,
        
        // Timeouts
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        
        // TLS
        TLSConfig: cfg.TLSConfig,
    })
}
```

### Use Case Implementations

#### 1. Rate Limiting
```go
func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
    script := `
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])
        
        redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
        local count = redis.call('ZCARD', key)
        
        if count < limit then
            redis.call('ZADD', key, now, now)
            redis.call('EXPIRE', key, window)
            return 1
        end
        return 0
    `
    
    allowed, err := r.client.Eval(ctx, script, []string{key}, 
        limit, int(window.Seconds()), time.Now().Unix()).Bool()
    return allowed, err
}
```

#### 2. Session Management
```go
type SessionStore struct {
    client *redis.Client
    prefix string
    ttl    time.Duration
}

func (s *SessionStore) Set(ctx context.Context, sessionID string, data map[string]interface{}) error {
    key := s.prefix + sessionID
    return s.client.HSet(ctx, key, data).Err()
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (map[string]string, error) {
    key := s.prefix + sessionID
    return s.client.HGetAll(ctx, key).Result()
}
```

#### 3. Real-time Metrics
```go
func (m *MetricsAggregator) PublishMetric(ctx context.Context, metric Metric) error {
    return m.client.XAdd(ctx, &redis.XAddArgs{
        Stream: "metrics:" + metric.Type,
        Values: map[string]interface{}{
            "timestamp": metric.Timestamp,
            "value":     metric.Value,
            "labels":    metric.Labels,
        },
    }).Err()
}
```

## Monitoring and Observability

### Key Metrics to Monitor
- Memory usage percentage
- Hit/miss ratio
- Command latency (p50, p95, p99)
- Connection count
- Replication lag
- Persistence status

### Prometheus Metrics
```yaml
apiVersion: v1
kind: ServiceMonitor
metadata:
  name: redis-metrics
spec:
  selector:
    matchLabels:
      app: redis
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

## Alternatives Considered

1. **Memcached**
   - Pros: Simpler, lower memory overhead
   - Cons: Limited data structures, no persistence, no pub/sub
   - Rejected: Lacks features needed for our use cases

2. **Hazelcast**
   - Pros: Java-native, distributed computing
   - Cons: Heavier footprint, complex for our needs
   - Rejected: Overkill for caching requirements

3. **PostgreSQL Only**
   - Pros: Single database, ACID compliance
   - Cons: Not optimized for caching patterns
   - Rejected: Performance limitations for real-time features

## Consequences

### Positive
- Unified caching solution for all use cases
- High performance with sub-millisecond latency
- Rich feature set enables advanced patterns
- Strong ecosystem and tooling
- Easy horizontal scaling

### Negative
- Additional infrastructure component
- Memory usage requires careful management
- Persistence adds complexity
- Requires expertise for optimization

## References

- [Redis 8 Documentation](https://redis.io/docs/)
- [Redis Best Practices](https://redis.io/docs/manual/patterns/)
- [Redis on Kubernetes](https://redis.io/docs/manual/kubernetes/)
- [go-redis Client](https://github.com/redis/go-redis)
