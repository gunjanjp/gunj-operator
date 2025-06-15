/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConversionOptimizer optimizes conversion operations for performance
type ConversionOptimizer struct {
	logger logr.Logger
	
	// Caching
	cache           *ConversionCache
	cacheEnabled    bool
	cacheHitRate    float64
	
	// Optimization strategies
	strategies      []OptimizationStrategy
	
	// Performance metrics
	metrics         *PerformanceMetrics
	
	// Configuration
	config          OptimizerConfig
}

// OptimizerConfig defines configuration for the conversion optimizer
type OptimizerConfig struct {
	// EnableCaching enables conversion result caching
	EnableCaching bool
	
	// CacheSize maximum number of cached conversions
	CacheSize int
	
	// CacheTTL time-to-live for cached entries
	CacheTTL time.Duration
	
	// EnableParallelization enables parallel conversions
	EnableParallelization bool
	
	// MaxParallelConversions maximum concurrent conversions
	MaxParallelConversions int
	
	// EnableSmartBatching groups similar conversions
	EnableSmartBatching bool
	
	// BatchTimeout maximum time to wait for batch formation
	BatchTimeout time.Duration
}

// ConversionCache caches conversion results
type ConversionCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxSize  int
	ttl      time.Duration
	hits     int64
	misses   int64
}

// CacheEntry represents a cached conversion result
type CacheEntry struct {
	Result     runtime.Object
	Hash       string
	Timestamp  time.Time
	AccessCount int64
	Size       int64
}

// OptimizationStrategy defines a conversion optimization strategy
type OptimizationStrategy interface {
	Name() string
	CanOptimize(source *unstructured.Unstructured, targetVersion string) bool
	Optimize(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error)
}

// PerformanceMetrics tracks conversion performance metrics
type PerformanceMetrics struct {
	mu                    sync.RWMutex
	TotalConversions      int64
	CachedConversions     int64
	OptimizedConversions  int64
	AverageConversionTime time.Duration
	FastestConversion     time.Duration
	SlowestConversion     time.Duration
	ConversionsByVersion  map[string]int64
	OptimizationsByType   map[string]int64
}

// NewConversionOptimizer creates a new conversion optimizer
func NewConversionOptimizer(logger logr.Logger) *ConversionOptimizer {
	config := OptimizerConfig{
		EnableCaching:          true,
		CacheSize:              1000,
		CacheTTL:               5 * time.Minute,
		EnableParallelization:  true,
		MaxParallelConversions: 10,
		EnableSmartBatching:    true,
		BatchTimeout:           100 * time.Millisecond,
	}
	
	optimizer := &ConversionOptimizer{
		logger:       logger.WithName("conversion-optimizer"),
		config:       config,
		cacheEnabled: config.EnableCaching,
		metrics: &PerformanceMetrics{
			ConversionsByVersion: make(map[string]int64),
			OptimizationsByType:  make(map[string]int64),
		},
	}
	
	// Initialize cache
	if config.EnableCaching {
		optimizer.cache = &ConversionCache{
			entries: make(map[string]*CacheEntry),
			maxSize: config.CacheSize,
			ttl:     config.CacheTTL,
		}
	}
	
	// Register optimization strategies
	optimizer.registerStrategies()
	
	// Start cache cleanup routine
	if optimizer.cache != nil {
		go optimizer.cleanupCache()
	}
	
	return optimizer
}

// registerStrategies registers all optimization strategies
func (o *ConversionOptimizer) registerStrategies() {
	o.strategies = []OptimizationStrategy{
		&FieldSkippingStrategy{logger: o.logger},
		&LazyLoadingStrategy{logger: o.logger},
		&BatchingStrategy{logger: o.logger},
		&CompressionStrategy{logger: o.logger},
		&ParallelizationStrategy{logger: o.logger, maxWorkers: o.config.MaxParallelConversions},
	}
}

// OptimizeConversion optimizes a conversion operation
func (o *ConversionOptimizer) OptimizeConversion(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error) {
	start := time.Now()
	defer func() {
		o.recordMetrics(time.Since(start), targetVersion)
	}()
	
	// Check cache first
	if o.cacheEnabled {
		if cached := o.getFromCache(source, targetVersion); cached != nil {
			o.metrics.mu.Lock()
			o.metrics.CachedConversions++
			o.metrics.mu.Unlock()
			return cached, nil
		}
	}
	
	// Apply optimization strategies
	optimized := source.DeepCopy()
	for _, strategy := range o.strategies {
		if strategy.CanOptimize(optimized, targetVersion) {
			result, err := strategy.Optimize(optimized, targetVersion)
			if err != nil {
				o.logger.Error(err, "Optimization strategy failed",
					"strategy", strategy.Name())
				continue
			}
			optimized = result
			
			o.metrics.mu.Lock()
			o.metrics.OptimizedConversions++
			o.metrics.OptimizationsByType[strategy.Name()]++
			o.metrics.mu.Unlock()
		}
	}
	
	// Cache the result
	if o.cacheEnabled {
		o.addToCache(source, targetVersion, optimized)
	}
	
	return optimized, nil
}

// getFromCache retrieves a conversion result from cache
func (o *ConversionOptimizer) getFromCache(source *unstructured.Unstructured, targetVersion string) *unstructured.Unstructured {
	if o.cache == nil {
		return nil
	}
	
	key := o.generateCacheKey(source, targetVersion)
	
	o.cache.mu.RLock()
	defer o.cache.mu.RUnlock()
	
	entry, exists := o.cache.entries[key]
	if !exists {
		o.cache.misses++
		return nil
	}
	
	// Check if entry is expired
	if time.Since(entry.Timestamp) > o.cache.ttl {
		o.cache.misses++
		return nil
	}
	
	o.cache.hits++
	entry.AccessCount++
	
	// Update cache hit rate
	total := float64(o.cache.hits + o.cache.misses)
	if total > 0 {
		o.cacheHitRate = float64(o.cache.hits) / total
	}
	
	return entry.Result.(*unstructured.Unstructured).DeepCopy()
}

// addToCache adds a conversion result to cache
func (o *ConversionOptimizer) addToCache(source *unstructured.Unstructured, targetVersion string, result *unstructured.Unstructured) {
	if o.cache == nil {
		return
	}
	
	key := o.generateCacheKey(source, targetVersion)
	
	o.cache.mu.Lock()
	defer o.cache.mu.Unlock()
	
	// Check cache size and evict if necessary
	if len(o.cache.entries) >= o.cache.maxSize {
		o.evictLRU()
	}
	
	o.cache.entries[key] = &CacheEntry{
		Result:      result.DeepCopy(),
		Hash:        key,
		Timestamp:   time.Now(),
		AccessCount: 0,
		Size:        int64(len(fmt.Sprint(result.Object))),
	}
}

// generateCacheKey generates a cache key for a conversion
func (o *ConversionOptimizer) generateCacheKey(source *unstructured.Unstructured, targetVersion string) string {
	h := sha256.New()
	h.Write([]byte(source.GetAPIVersion()))
	h.Write([]byte(source.GetKind()))
	h.Write([]byte(source.GetNamespace()))
	h.Write([]byte(source.GetName()))
	h.Write([]byte(source.GetResourceVersion()))
	h.Write([]byte(targetVersion))
	h.Write([]byte(fmt.Sprint(source.Object["spec"])))
	return hex.EncodeToString(h.Sum(nil))
}

// evictLRU evicts the least recently used cache entry
func (o *ConversionOptimizer) evictLRU() {
	var lruKey string
	var lruEntry *CacheEntry
	
	for key, entry := range o.cache.entries {
		if lruEntry == nil || entry.AccessCount < lruEntry.AccessCount {
			lruKey = key
			lruEntry = entry
		}
	}
	
	if lruKey != "" {
		delete(o.cache.entries, lruKey)
		o.logger.V(2).Info("Evicted cache entry", "key", lruKey)
	}
}

// cleanupCache periodically cleans up expired cache entries
func (o *ConversionOptimizer) cleanupCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		o.cache.mu.Lock()
		
		expiredKeys := []string{}
		for key, entry := range o.cache.entries {
			if time.Since(entry.Timestamp) > o.cache.ttl {
				expiredKeys = append(expiredKeys, key)
			}
		}
		
		for _, key := range expiredKeys {
			delete(o.cache.entries, key)
		}
		
		o.cache.mu.Unlock()
		
		if len(expiredKeys) > 0 {
			o.logger.V(2).Info("Cleaned up expired cache entries", "count", len(expiredKeys))
		}
	}
}

// recordMetrics records conversion performance metrics
func (o *ConversionOptimizer) recordMetrics(duration time.Duration, targetVersion string) {
	o.metrics.mu.Lock()
	defer o.metrics.mu.Unlock()
	
	o.metrics.TotalConversions++
	o.metrics.ConversionsByVersion[targetVersion]++
	
	// Update average conversion time
	if o.metrics.AverageConversionTime == 0 {
		o.metrics.AverageConversionTime = duration
	} else {
		// Calculate running average
		o.metrics.AverageConversionTime = (o.metrics.AverageConversionTime*time.Duration(o.metrics.TotalConversions-1) + duration) / time.Duration(o.metrics.TotalConversions)
	}
	
	// Update fastest/slowest
	if o.metrics.FastestConversion == 0 || duration < o.metrics.FastestConversion {
		o.metrics.FastestConversion = duration
	}
	if duration > o.metrics.SlowestConversion {
		o.metrics.SlowestConversion = duration
	}
}

// GetMetrics returns current performance metrics
func (o *ConversionOptimizer) GetMetrics() PerformanceMetrics {
	o.metrics.mu.RLock()
	defer o.metrics.mu.RUnlock()
	
	// Return a copy
	return PerformanceMetrics{
		TotalConversions:      o.metrics.TotalConversions,
		CachedConversions:     o.metrics.CachedConversions,
		OptimizedConversions:  o.metrics.OptimizedConversions,
		AverageConversionTime: o.metrics.AverageConversionTime,
		FastestConversion:     o.metrics.FastestConversion,
		SlowestConversion:     o.metrics.SlowestConversion,
		ConversionsByVersion:  copyMap(o.metrics.ConversionsByVersion),
		OptimizationsByType:   copyMap(o.metrics.OptimizationsByType),
	}
}

// GetCacheStats returns cache statistics
func (o *ConversionOptimizer) GetCacheStats() map[string]interface{} {
	if o.cache == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	
	o.cache.mu.RLock()
	defer o.cache.mu.RUnlock()
	
	return map[string]interface{}{
		"enabled":     true,
		"entries":     len(o.cache.entries),
		"maxSize":     o.cache.maxSize,
		"hits":        o.cache.hits,
		"misses":      o.cache.misses,
		"hitRate":     o.cacheHitRate,
		"ttl":         o.cache.ttl.String(),
	}
}

// ClearCache clears the conversion cache
func (o *ConversionOptimizer) ClearCache() {
	if o.cache == nil {
		return
	}
	
	o.cache.mu.Lock()
	defer o.cache.mu.Unlock()
	
	o.cache.entries = make(map[string]*CacheEntry)
	o.cache.hits = 0
	o.cache.misses = 0
	o.cacheHitRate = 0
	
	o.logger.Info("Conversion cache cleared")
}
