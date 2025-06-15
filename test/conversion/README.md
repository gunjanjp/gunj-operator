# Conversion Tests for Gunj Operator

This directory contains comprehensive tests for the conversion functionality between different API versions of the ObservabilityPlatform CRD.

## Test Structure

### Unit Tests (`unit_test.go`)
- **Field Mapping Tests**: Verifies all fields are properly mapped during conversion
- **Round-trip Tests**: Ensures data integrity through v1alpha1 → v1beta1 → v1alpha1 conversion
- **Status Conversion**: Tests status field conversion between versions
- **Annotation Handling**: Verifies conversion annotations are properly added

### Integration Tests (`integration_test.go`)
- **Webhook Integration**: Tests conversion webhook in a realistic environment
- **HTTP Handling**: Tests webhook HTTP request/response handling
- **Concurrent Conversions**: Verifies thread-safety with concurrent requests
- **Large Object Handling**: Tests conversion of objects with many fields
- **Batch Conversions**: Tests converting multiple objects in one request

### Benchmark Tests (`benchmark_test.go`)
- **Performance Benchmarks**: Measures conversion speed and allocations
- **Resource Conversion**: Benchmarks resource quantity parsing
- **Memory Usage**: Tracks memory allocations during conversion
- **Throughput Testing**: Measures conversions per second

### Edge Case Tests (`edge_case_test.go`)
- **Empty/Nil Fields**: Tests handling of empty strings, nil maps, etc.
- **Special Characters**: Tests Unicode, newlines, special characters
- **Extreme Values**: Tests zero values, maximum values
- **Invalid Data**: Tests graceful handling of invalid inputs
- **Malformed Data**: Tests conversion with corrupted data

### Fuzzing Tests (`fuzz_test.go`)
- **Random Input Testing**: Fuzzes conversion with random data
- **Resource Quantity Fuzzing**: Tests resource parsing with random inputs
- **Complex Object Fuzzing**: Tests conversion stability with random complex objects
- **JSON Round-trip Fuzzing**: Fuzzes JSON serialization/deserialization

### Error Tests (`error_test.go`)
- **Error Scenarios**: Tests various error conditions
- **Error Propagation**: Verifies errors are properly reported
- **Validation Testing**: Tests conversion validation logic
- **Custom Error Types**: Tests error handling and wrapping

### Multi-version Tests (`multiversion_test.go`)
- **Version Chain Testing**: Tests conversion through multiple versions
- **Cross-version Compatibility**: Tests compatibility between versions
- **Migration Scenarios**: Tests real-world migration use cases
- **Parallel Version Support**: Tests supporting multiple versions simultaneously

## Running the Tests

### Run All Tests
```bash
cd test/conversion
go test -v ./...
```

### Run Specific Test Categories
```bash
# Unit tests only
go test -v -run TestConversion unit_test.go

# Integration tests
go test -v -run TestWebhook integration_test.go

# Benchmarks
go test -bench=. -benchmem benchmark_test.go

# Fuzzing (Go 1.18+)
go test -fuzz=FuzzV1Alpha1ToV1Beta1Conversion -fuzztime=30s
```

### Run with Coverage
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Run with Race Detector
```bash
go test -race -v ./...
```

## Test Data

The tests use various helper functions to create test data:

- `createMinimalV1Alpha1Platform()`: Minimal valid platform
- `createTypicalV1Alpha1Platform()`: Typical production-like platform
- `createComplexV1Alpha1Platform()`: Platform with all fields populated
- `createLargeV1Alpha1Platform()`: Platform with many labels, annotations, etc.

## Key Test Scenarios

### 1. Basic Conversion
- Converting simple platforms between versions
- Verifying all fields are properly mapped
- Ensuring no data loss for supported fields

### 2. Field Loss Handling
- Converting from v1beta1 to v1alpha1 with fields that don't exist in v1alpha1
- Verifying lost fields are documented in annotations
- Testing graceful degradation

### 3. Resource Conversion
- Converting Kubernetes resource quantities between string and resource.Quantity
- Handling invalid resource values
- Testing extreme values

### 4. Error Handling
- Invalid resource quantities
- Nil/wrong type conversions
- Malformed data handling

### 5. Performance
- Conversion speed for different object sizes
- Memory allocation tracking
- Concurrent conversion handling

## Expected Behavior

### Successful Conversions
- All v1alpha1 fields are preserved when converting to v1beta1
- Conversion annotations are added to track version history
- Round-trip conversions maintain data integrity

### Field Loss
- When converting v1beta1 → v1alpha1, fields not in v1alpha1 are lost
- Lost fields are documented in `observability.io/conversion-lost-fields` annotation
- Basic functionality is preserved despite field loss

### Error Handling
- Invalid resource quantities cause conversion errors
- Nil hub objects are rejected
- Wrong type conversions are rejected
- Errors include helpful context about what failed

## Contributing

When adding new tests:

1. Follow the existing test patterns
2. Use table-driven tests where appropriate
3. Include both positive and negative test cases
4. Document any special test requirements
5. Ensure tests are deterministic and repeatable

## Maintenance

Regular maintenance tasks:

1. Update tests when adding new fields to CRDs
2. Add benchmarks for new conversion logic
3. Expand fuzzing corpus with real-world examples
4. Review and update error messages for clarity
5. Keep test data generators up to date with API changes
