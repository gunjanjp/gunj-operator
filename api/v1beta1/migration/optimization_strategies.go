/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FieldSkippingStrategy skips unnecessary fields during conversion
type FieldSkippingStrategy struct {
	logger logr.Logger
}

// Name returns the strategy name
func (s *FieldSkippingStrategy) Name() string {
	return "field-skipping"
}

// CanOptimize checks if this strategy can optimize the conversion
func (s *FieldSkippingStrategy) CanOptimize(source *unstructured.Unstructured, targetVersion string) bool {
	// Can optimize if source has metadata fields that don't need conversion
	metadata, found := source.Object["metadata"].(map[string]interface{})
	if !found {
		return false
	}
	
	// Check for fields that can be skipped
	skipFields := []string{"managedFields", "selfLink", "initializers"}
	for _, field := range skipFields {
		if _, exists := metadata[field]; exists {
			return true
		}
	}
	
	return false
}

// Optimize applies the field skipping optimization
func (s *FieldSkippingStrategy) Optimize(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error) {
	optimized := source.DeepCopy()
	
	// Remove unnecessary metadata fields
	metadata, _ := optimized.Object["metadata"].(map[string]interface{})
	skipFields := []string{"managedFields", "selfLink", "initializers", "clusterName"}
	
	for _, field := range skipFields {
		delete(metadata, field)
	}
	
	// Remove empty status if converting to a new resource
	if status, exists := optimized.Object["status"]; exists {
		if statusMap, ok := status.(map[string]interface{}); ok && len(statusMap) == 0 {
			delete(optimized.Object, "status")
		}
	}
	
	s.logger.V(2).Info("Applied field skipping optimization", 
		"resource", optimized.GetName())
	
	return optimized, nil
}

// LazyLoadingStrategy defers loading of large fields until needed
type LazyLoadingStrategy struct {
	logger logr.Logger
}

// Name returns the strategy name
func (s *LazyLoadingStrategy) Name() string {
	return "lazy-loading"
}

// CanOptimize checks if this strategy can optimize the conversion
func (s *LazyLoadingStrategy) CanOptimize(source *unstructured.Unstructured, targetVersion string) bool {
	spec, found := source.Object["spec"].(map[string]interface{})
	if !found {
		return false
	}
	
	// Check for large configuration fields
	largeFields := []string{"rawConfig", "customConfig", "advancedSettings"}
	for _, field := range largeFields {
		if val, exists := spec[field]; exists {
			// Check if field is large (more than 1KB when serialized)
			if str, ok := val.(string); ok && len(str) > 1024 {
				return true
			}
		}
	}
	
	return false
}

// Optimize applies the lazy loading optimization
func (s *LazyLoadingStrategy) Optimize(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error) {
	optimized := source.DeepCopy()
	spec, _ := optimized.Object["spec"].(map[string]interface{})
	
	// Replace large fields with references
	largeFields := []string{"rawConfig", "customConfig", "advancedSettings"}
	lazyRefs := make(map[string]interface{})
	
	for _, field := range largeFields {
		if val, exists := spec[field]; exists {
			if str, ok := val.(string); ok && len(str) > 1024 {
				// Store reference instead of actual value
				refKey := fmt.Sprintf("lazy:%s:%s", optimized.GetName(), field)
				lazyRefs[field] = map[string]interface{}{
					"$ref":  refKey,
					"size":  len(str),
					"type":  "lazy-loaded",
				}
				spec[field] = lazyRefs[field]
			}
		}
	}
	
	// Add lazy loading metadata
	if len(lazyRefs) > 0 {
		if annotations := optimized.GetAnnotations(); annotations == nil {
			optimized.SetAnnotations(map[string]string{
				"conversion.observability.io/lazy-loaded": "true",
			})
		} else {
			annotations["conversion.observability.io/lazy-loaded"] = "true"
			optimized.SetAnnotations(annotations)
		}
	}
	
	s.logger.V(2).Info("Applied lazy loading optimization",
		"resource", optimized.GetName(),
		"lazyFields", len(lazyRefs))
	
	return optimized, nil
}

// BatchingStrategy groups similar conversions for efficiency
type BatchingStrategy struct {
	logger logr.Logger
	mu     sync.Mutex
	batch  map[string][]*unstructured.Unstructured
}

// Name returns the strategy name
func (s *BatchingStrategy) Name() string {
	return "batching"
}

// CanOptimize checks if this strategy can optimize the conversion
func (s *BatchingStrategy) CanOptimize(source *unstructured.Unstructured, targetVersion string) bool {
	// Batching is beneficial for multiple similar resources
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.batch == nil {
		s.batch = make(map[string][]*unstructured.Unstructured)
	}
	
	key := fmt.Sprintf("%s:%s", source.GetKind(), targetVersion)
	s.batch[key] = append(s.batch[key], source)
	
	// Can optimize if we have multiple similar resources
	return len(s.batch[key]) > 1
}

// Optimize applies the batching optimization
func (s *BatchingStrategy) Optimize(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error) {
	// For individual resources, just mark them as batch-eligible
	optimized := source.DeepCopy()
	
	if annotations := optimized.GetAnnotations(); annotations == nil {
		optimized.SetAnnotations(map[string]string{
			"conversion.observability.io/batch-eligible": "true",
		})
	} else {
		annotations["conversion.observability.io/batch-eligible"] = "true"
		optimized.SetAnnotations(annotations)
	}
	
	s.logger.V(2).Info("Marked resource as batch-eligible",
		"resource", optimized.GetName())
	
	return optimized, nil
}

// CompressionStrategy compresses large fields during conversion
type CompressionStrategy struct {
	logger logr.Logger
}

// Name returns the strategy name
func (s *CompressionStrategy) Name() string {
	return "compression"
}

// CanOptimize checks if this strategy can optimize the conversion
func (s *CompressionStrategy) CanOptimize(source *unstructured.Unstructured, targetVersion string) bool {
	spec, found := source.Object["spec"].(map[string]interface{})
	if !found {
		return false
	}
	
	// Check for compressible fields
	compressibleFields := []string{"configuration", "rules", "dashboards"}
	for _, field := range compressibleFields {
		if val, exists := spec[field]; exists {
			// Check if field is large enough to benefit from compression
			if str, ok := val.(string); ok && len(str) > 512 {
				return true
			}
		}
	}
	
	return false
}

// Optimize applies the compression optimization
func (s *CompressionStrategy) Optimize(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error) {
	optimized := source.DeepCopy()
	spec, _ := optimized.Object["spec"].(map[string]interface{})
	
	compressibleFields := []string{"configuration", "rules", "dashboards"}
	compressed := 0
	
	for _, field := range compressibleFields {
		if val, exists := spec[field]; exists {
			if str, ok := val.(string); ok && len(str) > 512 {
				// Compress the field
				compressedData, err := s.compressString(str)
				if err != nil {
					s.logger.Error(err, "Failed to compress field", "field", field)
					continue
				}
				
				// Store compressed data
				spec[field] = map[string]interface{}{
					"compressed":     true,
					"encoding":       "gzip+base64",
					"data":           compressedData,
					"originalSize":   len(str),
					"compressedSize": len(compressedData),
				}
				compressed++
			}
		}
	}
	
	// Add compression metadata
	if compressed > 0 {
		if annotations := optimized.GetAnnotations(); annotations == nil {
			optimized.SetAnnotations(map[string]string{
				"conversion.observability.io/compressed": fmt.Sprintf("%d", compressed),
			})
		} else {
			annotations["conversion.observability.io/compressed"] = fmt.Sprintf("%d", compressed)
			optimized.SetAnnotations(annotations)
		}
	}
	
	s.logger.V(2).Info("Applied compression optimization",
		"resource", optimized.GetName(),
		"compressedFields", compressed)
	
	return optimized, nil
}

// compressString compresses a string using gzip and encodes it in base64
func (s *CompressionStrategy) compressString(data string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	
	if _, err := gz.Write([]byte(data)); err != nil {
		return "", err
	}
	
	if err := gz.Close(); err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ParallelizationStrategy parallelizes conversion of independent fields
type ParallelizationStrategy struct {
	logger     logr.Logger
	maxWorkers int
}

// Name returns the strategy name
func (s *ParallelizationStrategy) Name() string {
	return "parallelization"
}

// CanOptimize checks if this strategy can optimize the conversion
func (s *ParallelizationStrategy) CanOptimize(source *unstructured.Unstructured, targetVersion string) bool {
	spec, found := source.Object["spec"].(map[string]interface{})
	if !found {
		return false
	}
	
	// Check if there are multiple independent components to convert
	components, found := spec["components"].(map[string]interface{})
	if !found {
		return false
	}
	
	// Beneficial if we have multiple components
	return len(components) > 1
}

// Optimize applies the parallelization optimization
func (s *ParallelizationStrategy) Optimize(source *unstructured.Unstructured, targetVersion string) (*unstructured.Unstructured, error) {
	optimized := source.DeepCopy()
	spec, _ := optimized.Object["spec"].(map[string]interface{})
	components, _ := spec["components"].(map[string]interface{})
	
	// Process components in parallel
	var wg sync.WaitGroup
	results := make(map[string]interface{})
	resultsMu := sync.Mutex{}
	
	// Create a worker pool
	workCh := make(chan struct {
		name string
		data interface{}
	}, len(components))
	
	// Start workers
	for i := 0; i < s.maxWorkers && i < len(components); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workCh {
				// Simulate component conversion
				converted := s.convertComponent(work.name, work.data)
				
				resultsMu.Lock()
				results[work.name] = converted
				resultsMu.Unlock()
			}
		}()
	}
	
	// Queue work
	for name, data := range components {
		workCh <- struct {
			name string
			data interface{}
		}{name: name, data: data}
	}
	close(workCh)
	
	// Wait for completion
	wg.Wait()
	
	// Update with converted components
	spec["components"] = results
	
	// Add parallelization metadata
	if annotations := optimized.GetAnnotations(); annotations == nil {
		optimized.SetAnnotations(map[string]string{
			"conversion.observability.io/parallel-converted": fmt.Sprintf("%d", len(components)),
		})
	} else {
		annotations["conversion.observability.io/parallel-converted"] = fmt.Sprintf("%d", len(components))
		optimized.SetAnnotations(annotations)
	}
	
	s.logger.V(2).Info("Applied parallelization optimization",
		"resource", optimized.GetName(),
		"parallelComponents", len(components))
	
	return optimized, nil
}

// convertComponent simulates component conversion
func (s *ParallelizationStrategy) convertComponent(name string, data interface{}) interface{} {
	// This is a placeholder for actual component conversion logic
	// In real implementation, this would apply version-specific transformations
	converted := make(map[string]interface{})
	
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			// Apply transformations based on component type and target version
			converted[k] = v
		}
		
		// Add conversion metadata
		converted["convertedAt"] = "parallel"
	}
	
	return converted
}
