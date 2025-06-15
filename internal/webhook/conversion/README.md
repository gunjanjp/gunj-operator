# Conversion Webhook Tests

This directory contains comprehensive tests for the Gunj Operator's conversion webhooks, which handle API version migrations between v1alpha1 and v1beta1.

## Test Structure

### Core Test Files

1. **conversion_webhook_test.go**
   - Basic conversion functionality
   - v1alpha1 to v1beta1 conversion
   - v1beta1 to v1alpha1 conversion
   - Round-trip conversion tests
   - Edge cases and error scenarios

2. **conversion_performance_test.go**
   - Bulk conversion performance (100+ resources)
   - Concurrent conversion handling
   - Large payload conversion
   - Memory efficiency tests
   - Stress testing with rapid operations

3. **conversion_integration_test.go**
   - CRD conversion configuration validation
   - Unstructured object conversion
   - Status subresource preservation
   - Patch operations across versions
   - Server-side apply compatibility

4. **migration_helpers_test.go**
   - MigrationManager functionality
   - SchemaEvolutionTracker tests
   - ConversionOptimizer strategies
   - BatchConversionProcessor tests
   - Data preservation mechanisms

5. **data_preservation_test.go**
   - Annotation preservation rules
   - Label preservation policies
   - Status field preservation
   - Unknown fields detection
   - Complex preservation scenarios

6. **validation_test.go**
   - Conversion validation rules
   - Field compatibility checks
   - Schema validation
   - Error handling validation

## Running the Tests

### Prerequisites

1. Go 1.21 or later
2. Kubebuilder test dependencies
3. etcd binary (downloaded automatically by envtest)

### Run All Tests

```bash
# From the project root
make test-conversion

# Or directly with go test
go test -v ./internal/webhook/conversion/...
```

### Run Specific Test Suites

```bash
# Run only performance tests
go test -v ./internal/webhook/conversion/... -ginkgo.focus="Performance"

# Run only integration tests
go test -v ./internal/webhook/conversion/... -ginkgo.focus="Integration"

# Run only data preservation tests
go test -v ./internal/webhook/conversion/... -ginkgo.focus="Data Preservation"
```

### Run with Coverage

```bash
# Generate coverage report
go test -v -coverprofile=coverage.out ./internal/webhook/conversion/...

# View coverage in browser
go tool cover -html=coverage.out
```

### Benchmarks

```bash
# Run conversion benchmarks
go test -bench=. -benchmem ./internal/webhook/conversion/...

# Run specific benchmark
go test -bench=BenchmarkConversion -benchmem ./internal/webhook/conversion/...
```

## Test Categories

### Unit Tests
- Individual function testing
- Mock-based testing
- No external dependencies

### Integration Tests
- Real Kubernetes API server interaction
- CRD and webhook configuration
- End-to-end conversion flows

### Performance Tests
- Bulk operations (100+ resources)
- Concurrent access patterns
- Memory usage profiling
- Latency measurements

### Edge Case Tests
- Nil values handling
- Empty collections
- Large payloads
- Invalid data

## Test Coverage Goals

The test suite aims for:
- **>80% code coverage** for all conversion code
- **100% coverage** for critical paths
- **Performance benchmarks** for all major operations
- **Integration tests** for all API interactions

## Key Test Scenarios

### Basic Conversion
1. Simple v1alpha1 to v1beta1 conversion
2. Simple v1beta1 to v1alpha1 conversion
3. Round-trip conversions maintaining data integrity

### Complex Conversion
1. Resources with all fields populated
2. Resources with custom annotations/labels
3. Resources with status information
4. Resources with finalizers

### Data Preservation
1. Custom annotation preservation
2. Label transformation rules
3. Status field migration
4. Unknown field detection and preservation

### Performance
1. 100+ resource bulk conversion
2. 50+ concurrent conversion requests
3. Large resource payloads (>100KB)
4. Rapid create/update/delete cycles

### Error Handling
1. Invalid resource data
2. Conversion failures
3. Timeout scenarios
4. Resource conflicts

## Debugging Tests

### Enable Verbose Logging

```bash
# Set log level
export TEST_LOG_LEVEL=debug

# Run with Ginkgo verbose flag
go test -v ./internal/webhook/conversion/... -ginkgo.v
```

### Run Single Test

```bash
# Run specific test by name
go test -v ./internal/webhook/conversion/... -ginkgo.focus="should convert basic ObservabilityPlatform successfully"
```

### Debug with Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./internal/webhook/conversion/... -- -test.run TestConversionWebhook
```

## Common Issues and Solutions

### Issue: CRD Not Found
**Solution**: Ensure CRDs are generated before running tests
```bash
make generate manifests
```

### Issue: Webhook Certificate Errors
**Solution**: The test environment handles certificates automatically. If issues persist, check envtest setup.

### Issue: Timeout in Tests
**Solution**: Increase timeout values in test constants
```go
const (
    timeout  = time.Minute * 2  // Increase from default
    interval = time.Second
)
```

### Issue: Port Already in Use
**Solution**: The test suite uses random ports. If conflicts occur, restart the test.

## Contributing New Tests

When adding new conversion tests:

1. **Follow the existing pattern**: Use Ginkgo BDD style
2. **Group related tests**: Use `Context` and `Describe` blocks
3. **Use meaningful descriptions**: Make test intent clear
4. **Clean up resources**: Use `BeforeEach` and `AfterEach`
5. **Document complex scenarios**: Add comments for non-obvious test logic
6. **Consider performance**: Add benchmarks for new features
7. **Test error paths**: Don't just test happy paths

## Performance Metrics

Expected performance benchmarks:
- Single resource conversion: <10ms
- Bulk conversion (100 resources): <50ms average per resource
- Concurrent conversions: Handle 50+ simultaneous requests
- Memory usage: <100MB for 1000 resource conversions
