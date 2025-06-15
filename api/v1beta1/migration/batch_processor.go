/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// BatchConversionProcessor handles batch conversion of resources
type BatchConversionProcessor struct {
	client      client.Client
	scheme      *runtime.Scheme
	logger      logr.Logger
	
	// Configuration
	batchSize   int
	maxWorkers  int
	timeout     time.Duration
	
	// Runtime state
	queue       workqueue.RateLimitingInterface
	workers     int
	mu          sync.RWMutex
	
	// Metrics
	metrics     *BatchMetrics
}

// BatchConversionResult represents the result of a batch conversion
type BatchConversionResult struct {
	Resource   types.NamespacedName
	Status     BatchResultStatus
	Error      error
	Duration   time.Duration
	RetryCount int
}

// BatchResultStatus represents the status of a batch conversion result
type BatchResultStatus string

const (
	BatchResultStatusSuccess  BatchResultStatus = "success"
	BatchResultStatusFailed   BatchResultStatus = "failed"
	BatchResultStatusSkipped  BatchResultStatus = "skipped"
	BatchResultStatusRetrying BatchResultStatus = "retrying"
)

// BatchMetrics tracks batch processing metrics
type BatchMetrics struct {
	mu                  sync.RWMutex
	TotalBatches        int64
	TotalResources      int64
	SuccessfulResources int64
	FailedResources     int64
	SkippedResources    int64
	AverageBatchTime    time.Duration
	AverageResourceTime time.Duration
	LargestBatch        int
	CurrentQueueSize    int
}

// BatchWorkItem represents a work item in the batch queue
type BatchWorkItem struct {
	Resource      types.NamespacedName
	TargetVersion string
	RetryCount    int
	EnqueueTime   time.Time
}

// NewBatchConversionProcessor creates a new batch conversion processor
func NewBatchConversionProcessor(client client.Client, scheme *runtime.Scheme, logger logr.Logger, batchSize int) *BatchConversionProcessor {
	return &BatchConversionProcessor{
		client:     client,
		scheme:     scheme,
		logger:     logger.WithName("batch-processor"),
		batchSize:  batchSize,
		maxWorkers: 5,
		timeout:    5 * time.Minute,
		queue: workqueue.NewRateLimitingQueue(
			workqueue.NewItemExponentialFailureRateLimiter(time.Second, 30*time.Second),
		),
		metrics: &BatchMetrics{},
	}
}

// ProcessBatch processes a batch of resources for conversion
func (b *BatchConversionProcessor) ProcessBatch(ctx context.Context, resources []types.NamespacedName, targetVersion string) ([]BatchConversionResult, error) {
	b.logger.Info("Starting batch conversion",
		"resourceCount", len(resources),
		"targetVersion", targetVersion,
		"batchSize", b.batchSize)
	
	// Update metrics
	b.metrics.mu.Lock()
	b.metrics.TotalBatches++
	b.metrics.TotalResources += int64(len(resources))
	if len(resources) > b.metrics.LargestBatch {
		b.metrics.LargestBatch = len(resources)
	}
	b.metrics.mu.Unlock()
	
	startTime := time.Now()
	
	// Initialize results
	results := make([]BatchConversionResult, 0, len(resources))
	resultsChan := make(chan BatchConversionResult, len(resources))
	
	// Start workers
	workerCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// Determine number of workers
	numWorkers := b.maxWorkers
	if len(resources) < numWorkers {
		numWorkers = len(resources)
	}
	
	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go b.worker(workerCtx, &wg, resultsChan)
	}
	
	// Queue all resources
	for _, resource := range resources {
		item := &BatchWorkItem{
			Resource:      resource,
			TargetVersion: targetVersion,
			EnqueueTime:   time.Now(),
		}
		b.queue.Add(item)
	}
	
	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
	
	// Collect results
	for result := range resultsChan {
		results = append(results, result)
		
		// Update metrics
		b.updateMetrics(result)
	}
	
	// Calculate batch metrics
	batchDuration := time.Since(startTime)
	b.updateBatchMetrics(batchDuration, len(resources))
	
	// Check for failures
	failedCount := 0
	for _, result := range results {
		if result.Status == BatchResultStatusFailed {
			failedCount++
		}
	}
	
	if failedCount > 0 {
		return results, fmt.Errorf("batch conversion completed with %d failures out of %d resources", 
			failedCount, len(resources))
	}
	
	b.logger.Info("Batch conversion completed successfully",
		"duration", batchDuration,
		"resourceCount", len(resources))
	
	return results, nil
}

// worker processes items from the queue
func (b *BatchConversionProcessor) worker(ctx context.Context, wg *sync.WaitGroup, results chan<- BatchConversionResult) {
	defer wg.Done()
	
	b.mu.Lock()
	b.workers++
	b.mu.Unlock()
	
	defer func() {
		b.mu.Lock()
		b.workers--
		b.mu.Unlock()
	}()
	
	for {
		// Check context
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		// Get item from queue
		item, shutdown := b.queue.Get()
		if shutdown {
			return
		}
		
		// Process item
		workItem, ok := item.(*BatchWorkItem)
		if !ok {
			b.queue.Done(item)
			continue
		}
		
		// Convert resource
		result := b.convertResource(ctx, workItem)
		
		// Handle result
		switch result.Status {
		case BatchResultStatusSuccess:
			b.queue.Forget(item)
			results <- result
			
		case BatchResultStatusFailed:
			if workItem.RetryCount < 3 {
				workItem.RetryCount++
				b.queue.AddRateLimited(workItem)
				result.Status = BatchResultStatusRetrying
			} else {
				b.queue.Forget(item)
				results <- result
			}
			
		case BatchResultStatusSkipped:
			b.queue.Forget(item)
			results <- result
		}
		
		b.queue.Done(item)
	}
}

// convertResource converts a single resource
func (b *BatchConversionProcessor) convertResource(ctx context.Context, item *BatchWorkItem) BatchConversionResult {
	startTime := time.Now()
	result := BatchConversionResult{
		Resource:   item.Resource,
		RetryCount: item.RetryCount,
	}
	
	// Get the resource
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "observability.io",
		Version: "v1alpha1", // Assuming we're converting from v1alpha1
		Kind:    "ObservabilityPlatform",
	})
	
	if err := b.client.Get(ctx, item.Resource, u); err != nil {
		if errors.IsNotFound(err) {
			result.Status = BatchResultStatusSkipped
			result.Error = err
		} else {
			result.Status = BatchResultStatusFailed
			result.Error = fmt.Errorf("failed to get resource: %w", err)
		}
		result.Duration = time.Since(startTime)
		return result
	}
	
	// Check if already at target version
	if u.GetAPIVersion() == fmt.Sprintf("observability.io/%s", item.TargetVersion) {
		result.Status = BatchResultStatusSkipped
		result.Duration = time.Since(startTime)
		b.logger.V(1).Info("Resource already at target version",
			"resource", item.Resource,
			"version", item.TargetVersion)
		return result
	}
	
	// Perform conversion
	converted, err := b.convert(u, item.TargetVersion)
	if err != nil {
		result.Status = BatchResultStatusFailed
		result.Error = fmt.Errorf("conversion failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}
	
	// Update the resource
	if err := b.client.Update(ctx, converted); err != nil {
		if errors.IsConflict(err) && item.RetryCount < 3 {
			// Conflict - will retry
			result.Status = BatchResultStatusRetrying
			result.Error = err
		} else {
			result.Status = BatchResultStatusFailed
			result.Error = fmt.Errorf("failed to update resource: %w", err)
		}
		result.Duration = time.Since(startTime)
		return result
	}
	
	result.Status = BatchResultStatusSuccess
	result.Duration = time.Since(startTime)
	
	b.logger.V(1).Info("Successfully converted resource",
		"resource", item.Resource,
		"targetVersion", item.TargetVersion,
		"duration", result.Duration)
	
	return result
}

// convert performs the actual conversion
func (b *BatchConversionProcessor) convert(u *unstructured.Unstructured, targetVersion string) (runtime.Object, error) {
	switch targetVersion {
	case "v1beta1":
		// Convert from v1alpha1 to v1beta1
		v1alpha1Obj := &observabilityv1alpha1.ObservabilityPlatform{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, v1alpha1Obj); err != nil {
			return nil, fmt.Errorf("failed to convert to v1alpha1: %w", err)
		}
		
		v1beta1Obj := &observabilityv1beta1.ObservabilityPlatform{}
		if err := v1alpha1Obj.ConvertTo(v1beta1Obj); err != nil {
			return nil, fmt.Errorf("failed to convert to v1beta1: %w", err)
		}
		
		return v1beta1Obj, nil
		
	default:
		return nil, fmt.Errorf("unsupported target version: %s", targetVersion)
	}
}

// updateMetrics updates metrics for a single result
func (b *BatchConversionProcessor) updateMetrics(result BatchConversionResult) {
	b.metrics.mu.Lock()
	defer b.metrics.mu.Unlock()
	
	switch result.Status {
	case BatchResultStatusSuccess:
		b.metrics.SuccessfulResources++
	case BatchResultStatusFailed:
		b.metrics.FailedResources++
	case BatchResultStatusSkipped:
		b.metrics.SkippedResources++
	}
	
	// Update average resource time
	if b.metrics.AverageResourceTime == 0 {
		b.metrics.AverageResourceTime = result.Duration
	} else {
		total := b.metrics.SuccessfulResources + b.metrics.FailedResources + b.metrics.SkippedResources
		b.metrics.AverageResourceTime = (b.metrics.AverageResourceTime*time.Duration(total-1) + result.Duration) / time.Duration(total)
	}
	
	b.metrics.CurrentQueueSize = b.queue.Len()
}

// updateBatchMetrics updates batch-level metrics
func (b *BatchConversionProcessor) updateBatchMetrics(duration time.Duration, resourceCount int) {
	b.metrics.mu.Lock()
	defer b.metrics.mu.Unlock()
	
	// Update average batch time
	if b.metrics.AverageBatchTime == 0 {
		b.metrics.AverageBatchTime = duration
	} else {
		b.metrics.AverageBatchTime = (b.metrics.AverageBatchTime*time.Duration(b.metrics.TotalBatches-1) + duration) / time.Duration(b.metrics.TotalBatches)
	}
}

// GetMetrics returns current batch processing metrics
func (b *BatchConversionProcessor) GetMetrics() BatchMetrics {
	b.metrics.mu.RLock()
	defer b.metrics.mu.RUnlock()
	
	return BatchMetrics{
		TotalBatches:        b.metrics.TotalBatches,
		TotalResources:      b.metrics.TotalResources,
		SuccessfulResources: b.metrics.SuccessfulResources,
		FailedResources:     b.metrics.FailedResources,
		SkippedResources:    b.metrics.SkippedResources,
		AverageBatchTime:    b.metrics.AverageBatchTime,
		AverageResourceTime: b.metrics.AverageResourceTime,
		LargestBatch:        b.metrics.LargestBatch,
		CurrentQueueSize:    b.queue.Len(),
	}
}

// Stop stops the batch processor
func (b *BatchConversionProcessor) Stop() {
	b.logger.Info("Stopping batch processor")
	b.queue.ShutDown()
}

// SetBatchSize updates the batch size
func (b *BatchConversionProcessor) SetBatchSize(size int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.batchSize = size
	b.logger.Info("Updated batch size", "size", size)
}

// SetMaxWorkers updates the maximum number of workers
func (b *BatchConversionProcessor) SetMaxWorkers(workers int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.maxWorkers = workers
	b.logger.Info("Updated max workers", "workers", workers)
}
