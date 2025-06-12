# Gunj Operator Performance Testing Framework

**Version**: 1.0  
**Date**: June 12, 2025  
**Related**: MT-1.4.2.6 - Performance Benchmarks  

---

## 🧪 Performance Test Implementation

### Test Structure

```go
// test/performance/framework.go
package performance

import (
    "context"
    "time"
    "github.com/prometheus/client_golang/prometheus"
)

type PerformanceTest struct {
    Name        string
    Description string
    Duration    time.Duration
    Setup       func(context.Context) error
    Execute     func(context.Context) error
    Teardown    func(context.Context) error
    Validators  []Validator
}

type Validator interface {
    Validate(metrics TestMetrics) ValidationResult
}

type TestMetrics struct {
    StartTime       time.Time
    EndTime         time.Time
    TotalRequests   int64
    SuccessRequests int64
    FailedRequests  int64
    Latencies       []time.Duration
    ResourceUsage   ResourceMetrics
}

type ResourceMetrics struct {
    CPUUsage    []float64
    MemoryUsage []int64
    NetworkIO   []NetworkStat
    DiskIO      []DiskStat
}
```

### Operator Performance Tests

```go
// test/performance/operator_test.go
package performance

import (
    "testing"
    "time"
)

func TestReconciliationPerformance(t *testing.T) {
    tests := []struct {
        name           string
        platformCount  int
        changeRate     int // changes per minute
        expectedP95    time.Duration
        expectedP99    time.Duration
    }{
        {
            name:          "small_scale",
            platformCount: 10,
            changeRate:    60,
            expectedP95:   100 * time.Millisecond,
            expectedP99:   200 * time.Millisecond,
        },
        {
            name:          "medium_scale",
            platformCount: 100,
            changeRate:    300,
            expectedP95:   200 * time.Millisecond,
            expectedP99:   500 * time.Millisecond,
        },
        {
            name:          "large_scale",
            platformCount: 1000,
            changeRate:    600,
            expectedP95:   500 * time.Millisecond,
            expectedP99:   1 * time.Second,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := runReconciliationTest(tt.platformCount, tt.changeRate)
            
            if result.P95Latency > tt.expectedP95 {
                t.Errorf("P95 latency %v exceeds target %v", 
                    result.P95Latency, tt.expectedP95)
            }
            
            if result.P99Latency > tt.expectedP99 {
                t.Errorf("P99 latency %v exceeds target %v", 
                    result.P99Latency, tt.expectedP99)
            }
        })
    }
}

func BenchmarkReconciliation(b *testing.B) {
    platform := createTestPlatform()
    reconciler := createTestReconciler()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = reconciler.Reconcile(context.Background(), platform)
    }
}
```

### API Performance Tests

```typescript
// test/performance/api.k6.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const apiErrors = new Rate('api_errors');
const apiLatency = new Trend('api_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp up
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 500 },  // Spike to 500 users
    { duration: '5m', target: 500 },  // Stay at 500 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<200', 'p(99)<500'],
    'api_errors': ['rate<0.01'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080';

export default function() {
  // Test different API endpoints
  const endpoints = [
    { method: 'GET', path: '/api/v1/platforms', weight: 40 },
    { method: 'GET', path: '/api/v1/platforms/test-platform', weight: 30 },
    { method: 'POST', path: '/api/v1/platforms', weight: 10, body: createPlatformPayload() },
    { method: 'PUT', path: '/api/v1/platforms/test-platform', weight: 15, body: updatePlatformPayload() },
    { method: 'DELETE', path: '/api/v1/platforms/test-platform', weight: 5 },
  ];
  
  // Select endpoint based on weight
  const endpoint = selectByWeight(endpoints);
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${__ENV.API_TOKEN}`,
    },
  };
  
  const start = Date.now();
  const res = http.request(endpoint.method, `${BASE_URL}${endpoint.path}`, 
    endpoint.body ? JSON.stringify(endpoint.body) : null, params);
  const latency = Date.now() - start;
  
  // Record metrics
  apiLatency.add(latency);
  apiErrors.add(res.status >= 400);
  
  // Validate response
  check(res, {
    'status is 200-299': (r) => r.status >= 200 && r.status < 300,
    'response time < 200ms': (r) => latency < 200,
    'has content-type': (r) => r.headers['Content-Type'] !== undefined,
  });
  
  sleep(1);
}
```

### UI Performance Tests

```typescript
// test/performance/ui.lighthouse.ts
import lighthouse from 'lighthouse';
import * as chromeLauncher from 'chrome-launcher';

interface PerformanceTarget {
  metric: string;
  target: number;
  critical: number;
}

const performanceTargets: PerformanceTarget[] = [
  { metric: 'first-contentful-paint', target: 1000, critical: 2500 },
  { metric: 'largest-contentful-paint', target: 1500, critical: 4000 },
  { metric: 'total-blocking-time', target: 200, critical: 600 },
  { metric: 'cumulative-layout-shift', target: 0.05, critical: 0.1 },
  { metric: 'speed-index', target: 2000, critical: 4000 },
];

async function runLighthouseTest(url: string): Promise<void> {
  const chrome = await chromeLauncher.launch({ chromeFlags: ['--headless'] });
  
  const options = {
    logLevel: 'info',
    output: 'json',
    port: chrome.port,
    onlyCategories: ['performance'],
  };
  
  const runnerResult = await lighthouse(url, options);
  const report = runnerResult.lhr;
  
  // Validate against targets
  const failures: string[] = [];
  
  for (const target of performanceTargets) {
    const audit = report.audits[target.metric];
    if (audit) {
      const value = audit.numericValue;
      
      if (value > target.critical) {
        failures.push(
          `CRITICAL: ${target.metric} = ${value}ms (critical threshold: ${target.critical}ms)`
        );
      } else if (value > target.target) {
        console.warn(
          `WARNING: ${target.metric} = ${value}ms (target: ${target.target}ms)`
        );
      }
    }
  }
  
  await chrome.kill();
  
  if (failures.length > 0) {
    throw new Error(`Performance test failed:\n${failures.join('\n')}`);
  }
}

// Test different pages
const pagesToTest = [
  { name: 'Dashboard', path: '/' },
  { name: 'Platform List', path: '/platforms' },
  { name: 'Platform Detail', path: '/platforms/test-platform' },
  { name: 'Metrics View', path: '/platforms/test-platform/metrics' },
];

describe('UI Performance Tests', () => {
  for (const page of pagesToTest) {
    test(`${page.name} performance`, async () => {
      await runLighthouseTest(`http://localhost:3000${page.path}`);
    });
  }
});
```

### Memory Leak Detection

```go
// test/performance/memory_test.go
package performance

import (
    "runtime"
    "testing"
    "time"
)

func TestMemoryLeaks(t *testing.T) {
    // Force GC and get baseline
    runtime.GC()
    var baseline runtime.MemStats
    runtime.ReadMemStats(&baseline)
    
    // Run operations that should not leak
    for i := 0; i < 1000; i++ {
        platform := createTestPlatform()
        _ = reconciler.Reconcile(ctx, platform)
        
        // Periodic GC to simulate real conditions
        if i%100 == 0 {
            runtime.GC()
        }
    }
    
    // Force GC and measure
    runtime.GC()
    time.Sleep(100 * time.Millisecond)
    runtime.GC()
    
    var after runtime.MemStats
    runtime.ReadMemStats(&after)
    
    // Check for leaks
    heapGrowth := after.HeapAlloc - baseline.HeapAlloc
    maxGrowth := uint64(10 * 1024 * 1024) // 10MB tolerance
    
    if heapGrowth > maxGrowth {
        t.Errorf("Potential memory leak detected: heap grew by %d bytes", heapGrowth)
    }
}
```

### Database Performance Tests

```sql
-- test/performance/database_queries.sql
-- Test query performance

-- Platform listing query
EXPLAIN ANALYZE
SELECT p.*, 
       COUNT(c.id) as component_count,
       MAX(c.updated_at) as last_component_update
FROM platforms p
LEFT JOIN components c ON c.platform_id = p.id
WHERE p.namespace = 'default'
  AND p.deleted_at IS NULL
GROUP BY p.id
ORDER BY p.created_at DESC
LIMIT 20 OFFSET 0;

-- Metrics aggregation query
EXPLAIN ANALYZE
SELECT 
    date_trunc('minute', timestamp) as minute,
    component_type,
    AVG(cpu_usage) as avg_cpu,
    AVG(memory_usage) as avg_memory,
    MAX(cpu_usage) as max_cpu,
    MAX(memory_usage) as max_memory
FROM metrics
WHERE platform_id = 'test-platform-uuid'
  AND timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY minute, component_type
ORDER BY minute DESC;
```

### Continuous Performance Testing

```yaml
# .github/workflows/performance-tests.yml
name: Performance Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  operator-performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup test cluster
        run: |
          kind create cluster --config test/performance/kind-config.yaml
          kubectl apply -f config/crd/
      
      - name: Run operator benchmarks
        run: |
          go test -bench=. -benchmem -benchtime=10x \
            -run=^$ ./test/performance/... \
            -o operator-performance.test
      
      - name: Generate performance report
        run: |
          go tool pprof -http=:8080 -no_browser operator-performance.test &
          sleep 5
          curl http://localhost:8080/ui/flamegraph > flamegraph.svg
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: operator-performance
          path: |
            *.test
            *.svg
            benchmark-results.json

  api-load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Start API server
        run: |
          docker-compose up -d api
          ./scripts/wait-for-api.sh
      
      - name: Run k6 load tests
        run: |
          docker run --rm \
            -v $PWD/test/performance:/scripts \
            -e API_URL=http://host.docker.internal:8080 \
            loadimpact/k6 run /scripts/api.k6.js \
            --out json=api-performance.json
      
      - name: Analyze results
        run: |
          python3 scripts/analyze-performance.py api-performance.json
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: api-performance
          path: |
            api-performance.json
            performance-report.html

  ui-performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Build and start UI
        run: |
          cd ui
          npm ci
          npm run build
          npm run start &
          npx wait-on http://localhost:3000
      
      - name: Run Lighthouse tests
        run: |
          npm run test:lighthouse
      
      - name: Run bundle analysis
        run: |
          npm run build:analyze
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: ui-performance
          path: |
            lighthouse-reports/
            bundle-stats.html
```

---

## 📊 Performance Monitoring Setup

### Prometheus Configuration

```yaml
# config/monitoring/prometheus-performance.yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'gunj-operator'
    static_configs:
      - targets: ['operator:8080']
    metric_relabel_configs:
      # Keep only performance-related metrics
      - source_labels: [__name__]
        regex: '(gunj_operator_reconcile_duration_seconds|gunj_operator_api_request_duration_seconds|gunj_operator_resource_usage_.*|go_gc_duration_seconds|process_.*)'
        action: keep

  - job_name: 'api-server'
    static_configs:
      - targets: ['api:9090']
    
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

rule_files:
  - '/etc/prometheus/performance-rules.yaml'

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Gunj Operator Performance",
    "panels": [
      {
        "title": "Reconciliation Performance",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, gunj_operator_reconcile_duration_seconds_bucket)",
            "legendFormat": "p95"
          },
          {
            "expr": "histogram_quantile(0.99, gunj_operator_reconcile_duration_seconds_bucket)",
            "legendFormat": "p99"
          }
        ]
      },
      {
        "title": "API Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, gunj_operator_api_request_duration_seconds_bucket)",
            "legendFormat": "{{method}} {{endpoint}} p95"
          }
        ]
      },
      {
        "title": "Resource Usage",
        "targets": [
          {
            "expr": "gunj_operator_resource_usage_cpu_cores",
            "legendFormat": "CPU Cores"
          },
          {
            "expr": "gunj_operator_resource_usage_memory_bytes / 1024 / 1024",
            "legendFormat": "Memory (MB)"
          }
        ]
      }
    ]
  }
}
```

---

## 📋 Performance Test Checklist

### Before Release
- [ ] All performance tests passing
- [ ] Benchmarks meet defined targets
- [ ] No memory leaks detected
- [ ] Database queries optimized
- [ ] UI performance validated
- [ ] Load tests successful
- [ ] Performance dashboard configured
- [ ] Alerts configured
- [ ] Performance runbook updated
- [ ] Baseline metrics captured

### Continuous Monitoring
- [ ] Daily performance tests scheduled
- [ ] Weekly performance review meetings
- [ ] Monthly trend analysis
- [ ] Quarterly capacity planning
- [ ] Performance regression detection
- [ ] User experience monitoring
- [ ] Cost analysis reports
- [ ] Optimization backlog maintained
- [ ] Team performance training
- [ ] Tool evaluation and updates

---

**Last Updated**: June 12, 2025