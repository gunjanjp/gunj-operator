/*
Copyright 2025 The Gunj Operator Authors.

Licensed under the MIT License.
*/

package conversion

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ConversionOptimizer optimizes conversion operations for performance
type ConversionOptimizer struct {
	log         logr.Logger
	cache       *ConversionCache
	metrics     *OptimizationMetrics
	strategies  []OptimizationStrategy
	mu          sync.RWMutex
}

// OptimizationStrategy represents a conversion optimization strategy
type OptimizationStrategy interface {
	Name() string
	Apply(resources []types.NamespacedName) []types.NamespacedName
}

// ConversionCache caches conversion results for reuse
type ConversionCache struct {
	mu        sync.RWMutex
	cache     map[string]*CachedConversion
	maxSize   int
	evictFunc func(key string, value *CachedConversion)
}

// CachedConversion represents a cached conversion result
type CachedConversion struct {
	Source       runtime.Object
	Target       runtime.Object
	LastAccessed time.Time
	AccessCount  int
	Size         int64
}

// OptimizationMetrics tracks optimization performance
type OptimizationMetrics struct {
	mu                   sync.RWMutex
	CacheHits            int64
	CacheMisses          int64
	ConversionsOptimized int64
	TimeSaved            time.Duration
	MemorySaved          int64
}

// NewConversionOptimizer creates a new conversion optimizer
func NewConversionOptimizer(log logr.Logger) *ConversionOptimizer {
	o := &ConversionOptimizer{
		log:     log.WithName("conversion-optimizer"),
		cache:   NewConversionCache(1000), // Cache up to 1000 conversions
		metrics: &OptimizationMetrics{},
	}

	// Register default optimization strategies
	o.registerDefaultStrategies()

	return o
}

// registerDefaultStrategies registers default optimization strategies
func (o *ConversionOptimizer) registerDefaultStrategies() {
	o.strategies = []OptimizationStrategy{
		&DependencyOrderStrategy{},
		&NamespaceGroupingStrategy{},
		&ResourceSizeStrategy{},
		&PriorityStrategy{},
	}
}

// OptimizeBatch optimizes a batch of resources for conversion
func (o *ConversionOptimizer) OptimizeBatch(resources []types.NamespacedName) []types.NamespacedName {
	o.log.V(1).Info("Optimizing batch", "size", len(resources))

	// Apply each optimization strategy
	optimized := resources
	for _, strategy := range o.strategies {
		optimized = strategy.Apply(optimized)
		o.log.V(2).Info("Applied optimization strategy", "strategy", strategy.Name())
	}

	o.mu.Lock()
	o.metrics.ConversionsOptimized += int64(len(optimized))
	o.mu.Unlock()

	return optimized
}

// OptimizeConversion optimizes a single conversion operation
func (o *ConversionOptimizer) OptimizeConversion(ctx context.Context, source runtime.Object, targetGVK runtime.GroupVersionKind) (runtime.Object, bool, error) {
	// Generate cache key
	key := o.generateCacheKey(source, targetGVK)

	// Check cache
	if cached, found := o.cache.Get(key); found {
		o.mu.Lock()
		o.metrics.CacheHits++
		o.mu.Unlock()
		return cached.Target, true, nil
	}

	o.mu.Lock()
	o.metrics.CacheMisses++
	o.mu.Unlock()

	return nil, false, nil
}

// CacheConversion caches a conversion result
func (o *ConversionOptimizer) CacheConversion(source runtime.Object, target runtime.Object) error {
	key := o.generateCacheKey(source, target.GetObjectKind().GroupVersionKind())
	
	cached := &CachedConversion{
		Source:       source,
		Target:       target,
		LastAccessed: time.Now(),
		AccessCount:  1,
		Size:         o.estimateObjectSize(target),
	}

	o.cache.Put(key, cached)
	return nil
}

// generateCacheKey generates a cache key for a conversion
func (o *ConversionOptimizer) generateCacheKey(source runtime.Object, targetGVK runtime.GroupVersionKind) string {
	meta, err := meta.Accessor(source)
	if err != nil {
		return ""
	}

	sourceGVK := source.GetObjectKind().GroupVersionKind()
	return fmt.Sprintf("%s/%s/%s:%s->%s", 
		meta.GetNamespace(), 
		meta.GetName(), 
		sourceGVK.String(),
		meta.GetResourceVersion(),
		targetGVK.String())
}

// estimateObjectSize estimates the size of an object in bytes
func (o *ConversionOptimizer) estimateObjectSize(obj runtime.Object) int64 {
	// Simple estimation based on unstructured representation
	if unstr, ok := obj.(*unstructured.Unstructured); ok {
		// Rough estimate: count fields and estimate bytes
		return int64(len(fmt.Sprintf("%v", unstr.Object)) * 2)
	}
	return 1024 // Default estimate
}

// GetMetrics returns optimization metrics
func (o *ConversionOptimizer) GetMetrics() OptimizationMetrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return *o.metrics
}

// ResetMetrics resets optimization metrics
func (o *ConversionOptimizer) ResetMetrics() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.metrics = &OptimizationMetrics{}
}

// PrewarmCache prewarms the cache with common conversions
func (o *ConversionOptimizer) PrewarmCache(ctx context.Context, resources []runtime.Object) error {
	o.log.Info("Prewarming conversion cache", "resources", len(resources))
	
	// TODO: Implement cache prewarming logic
	// This would convert common resources and cache them
	
	return nil
}

// NewConversionCache creates a new conversion cache
func NewConversionCache(maxSize int) *ConversionCache {
	return &ConversionCache{
		cache:   make(map[string]*CachedConversion),
		maxSize: maxSize,
	}
}

// Get retrieves a cached conversion
func (c *ConversionCache) Get(key string) (*CachedConversion, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cached, found := c.cache[key]; found {
		cached.LastAccessed = time.Now()
		cached.AccessCount++
		return cached, true
	}

	return nil, false
}

// Put stores a conversion in the cache
func (c *ConversionCache) Put(key string, value *CachedConversion) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.cache) >= c.maxSize {
		c.evictLRU()
	}

	c.cache[key] = value
}

// evictLRU evicts the least recently used entry
func (c *ConversionCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range c.cache {
		if oldestTime.IsZero() || cached.LastAccessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.LastAccessed
		}
	}

	if oldestKey != "" {
		if c.evictFunc != nil {
			c.evictFunc(oldestKey, c.cache[oldestKey])
		}
		delete(c.cache, oldestKey)
	}
}

// Clear clears the cache
func (c *ConversionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CachedConversion)
}

// Size returns the current cache size
func (c *ConversionCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Optimization Strategies

// DependencyOrderStrategy orders resources by dependencies
type DependencyOrderStrategy struct{}

func (s *DependencyOrderStrategy) Name() string { return "dependency-order" }

func (s *DependencyOrderStrategy) Apply(resources []types.NamespacedName) []types.NamespacedName {
	// Group resources by type to process dependencies first
	groups := make(map[string][]types.NamespacedName)
	
	for _, res := range resources {
		// Extract type from name convention (if follows pattern)
		resourceType := "other"
		if len(res.Name) > 0 {
			// Simple heuristic: ConfigMaps and Secrets should be processed first
			switch {
			case contains(res.Name, "config"):
				resourceType = "configmap"
			case contains(res.Name, "secret"):
				resourceType = "secret"
			case contains(res.Name, "service"):
				resourceType = "service"
			default:
				resourceType = "workload"
			}
		}
		groups[resourceType] = append(groups[resourceType], res)
	}

	// Order: ConfigMaps/Secrets -> Services -> Workloads -> Others
	ordered := []types.NamespacedName{}
	order := []string{"configmap", "secret", "service", "workload", "other"}
	
	for _, resType := range order {
		if group, exists := groups[resType]; exists {
			ordered = append(ordered, group...)
		}
	}

	return ordered
}

// NamespaceGroupingStrategy groups resources by namespace
type NamespaceGroupingStrategy struct{}

func (s *NamespaceGroupingStrategy) Name() string { return "namespace-grouping" }

func (s *NamespaceGroupingStrategy) Apply(resources []types.NamespacedName) []types.NamespacedName {
	// Group by namespace to improve locality
	namespaceGroups := make(map[string][]types.NamespacedName)
	
	for _, res := range resources {
		ns := res.Namespace
		if ns == "" {
			ns = "cluster-scoped"
		}
		namespaceGroups[ns] = append(namespaceGroups[ns], res)
	}

	// Sort namespaces for consistent ordering
	namespaces := make([]string, 0, len(namespaceGroups))
	for ns := range namespaceGroups {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	// Rebuild ordered list
	ordered := []types.NamespacedName{}
	for _, ns := range namespaces {
		ordered = append(ordered, namespaceGroups[ns]...)
	}

	return ordered
}

// ResourceSizeStrategy orders resources by estimated size
type ResourceSizeStrategy struct{}

func (s *ResourceSizeStrategy) Name() string { return "resource-size" }

func (s *ResourceSizeStrategy) Apply(resources []types.NamespacedName) []types.NamespacedName {
	// Sort by name length as a proxy for complexity
	// In a real implementation, this would query actual resource sizes
	sorted := make([]types.NamespacedName, len(resources))
	copy(sorted, resources)
	
	sort.Slice(sorted, func(i, j int) bool {
		// Process smaller resources first for better parallelism
		return len(sorted[i].String()) < len(sorted[j].String())
	})

	return sorted
}

// PriorityStrategy orders resources by priority
type PriorityStrategy struct{}

func (s *PriorityStrategy) Name() string { return "priority" }

func (s *PriorityStrategy) Apply(resources []types.NamespacedName) []types.NamespacedName {
	// Define priority based on namespace
	priorityMap := map[string]int{
		"kube-system":      1,
		"kube-public":      2,
		"default":          3,
		"gunj-system":      4,
		"observability":    5,
		"monitoring":       6,
	}

	sorted := make([]types.NamespacedName, len(resources))
	copy(sorted, resources)
	
	sort.Slice(sorted, func(i, j int) bool {
		iPriority, iExists := priorityMap[sorted[i].Namespace]
		jPriority, jExists := priorityMap[sorted[j].Namespace]
		
		if !iExists {
			iPriority = 999
		}
		if !jExists {
			jPriority = 999
		}
		
		if iPriority != jPriority {
			return iPriority < jPriority
		}
		
		// Same priority, sort by name
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
	       len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

// PerformanceAnalyzer analyzes conversion performance
type PerformanceAnalyzer struct {
	log     logr.Logger
	samples []PerformanceSample
	mu      sync.RWMutex
}

// PerformanceSample represents a performance measurement
type PerformanceSample struct {
	ResourceType string
	Operation    string
	Duration     time.Duration
	Size         int64
	Success      bool
	Timestamp    time.Time
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer(log logr.Logger) *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		log:     log.WithName("performance-analyzer"),
		samples: make([]PerformanceSample, 0),
	}
}

// RecordSample records a performance sample
func (a *PerformanceAnalyzer) RecordSample(sample PerformanceSample) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.samples = append(a.samples, sample)
	
	// Keep only recent samples (last 1000)
	if len(a.samples) > 1000 {
		a.samples = a.samples[len(a.samples)-1000:]
	}
}

// GetPerformanceReport generates a performance report
func (a *PerformanceAnalyzer) GetPerformanceReport() *PerformanceReport {
	a.mu.RLock()
	defer a.mu.RUnlock()

	report := &PerformanceReport{
		TotalSamples: len(a.samples),
		ByResource:   make(map[string]*ResourcePerformance),
	}

	// Analyze samples
	for _, sample := range a.samples {
		if _, exists := report.ByResource[sample.ResourceType]; !exists {
			report.ByResource[sample.ResourceType] = &ResourcePerformance{
				ResourceType: sample.ResourceType,
			}
		}
		
		perf := report.ByResource[sample.ResourceType]
		perf.TotalOperations++
		perf.TotalDuration += sample.Duration
		
		if sample.Success {
			perf.SuccessfulOperations++
		}
		
		if perf.MaxDuration < sample.Duration {
			perf.MaxDuration = sample.Duration
		}
		
		if perf.MinDuration == 0 || perf.MinDuration > sample.Duration {
			perf.MinDuration = sample.Duration
		}
	}

	// Calculate averages
	for _, perf := range report.ByResource {
		if perf.TotalOperations > 0 {
			perf.AverageDuration = perf.TotalDuration / time.Duration(perf.TotalOperations)
			perf.SuccessRate = float64(perf.SuccessfulOperations) / float64(perf.TotalOperations) * 100
		}
	}

	return report
}

// PerformanceReport contains performance analysis results
type PerformanceReport struct {
	TotalSamples int
	ByResource   map[string]*ResourcePerformance
}

// ResourcePerformance contains performance metrics for a resource type
type ResourcePerformance struct {
	ResourceType         string
	TotalOperations      int
	SuccessfulOperations int
	TotalDuration        time.Duration
	AverageDuration      time.Duration
	MinDuration          time.Duration
	MaxDuration          time.Duration
	SuccessRate          float64
}
