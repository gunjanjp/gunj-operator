# Conversion Tests Makefile
# This file contains targets for running conversion webhook tests

.PHONY: test-conversion test-conversion-unit test-conversion-integration test-conversion-performance test-conversion-coverage

# Run all conversion tests
test-conversion: generate manifests
	@echo "Running all conversion tests..."
	go test -v ./internal/webhook/conversion/... -timeout 30m

# Run only unit tests
test-conversion-unit: generate manifests
	@echo "Running conversion unit tests..."
	go test -v ./internal/webhook/conversion/... -short

# Run only integration tests
test-conversion-integration: generate manifests envtest
	@echo "Running conversion integration tests..."
	go test -v ./internal/webhook/conversion/... -ginkgo.focus="Integration"

# Run only performance tests
test-conversion-performance: generate manifests
	@echo "Running conversion performance tests..."
	go test -v ./internal/webhook/conversion/... -ginkgo.focus="Performance"

# Run with coverage
test-conversion-coverage: generate manifests
	@echo "Running conversion tests with coverage..."
	go test -v -coverprofile=coverage-conversion.out -covermode=atomic ./internal/webhook/conversion/...
	go tool cover -html=coverage-conversion.out -o coverage-conversion.html
	@echo "Coverage report generated: coverage-conversion.html"

# Run benchmarks
bench-conversion: generate manifests
	@echo "Running conversion benchmarks..."
	go test -bench=. -benchmem ./internal/webhook/conversion/... -run=^$$

# Run specific test focus
test-conversion-focus: generate manifests
	@echo "Running focused conversion tests..."
	@echo "Usage: make test-conversion-focus FOCUS=\"pattern\""
	go test -v ./internal/webhook/conversion/... -ginkgo.focus="$(FOCUS)"

# Lint conversion code
lint-conversion:
	@echo "Linting conversion code..."
	golangci-lint run ./internal/webhook/conversion/...

# Check test coverage threshold
check-conversion-coverage: test-conversion-coverage
	@echo "Checking conversion test coverage..."
	@coverage=$$(go tool cover -func=coverage-conversion.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage < 80" | bc) -eq 1 ]; then \
		echo "❌ Coverage $$coverage% is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$coverage% meets 80% threshold"; \
	fi

# Clean test artifacts
clean-conversion-tests:
	@echo "Cleaning conversion test artifacts..."
	rm -f coverage-conversion.out coverage-conversion.html
	rm -rf test-results/conversion

# Generate test report
test-conversion-report: test-conversion-coverage
	@echo "Generating conversion test report..."
	@mkdir -p test-results/conversion
	go test -v -json ./internal/webhook/conversion/... > test-results/conversion/results.json
	@echo "Test results saved to test-results/conversion/results.json"

# Debug a specific test
debug-conversion-test:
	@echo "Debugging conversion test..."
	@echo "Usage: make debug-conversion-test TEST=\"TestName\""
	dlv test ./internal/webhook/conversion/... -- -test.run="$(TEST)"

# Environment setup check
check-conversion-test-env:
	@echo "Checking conversion test environment..."
	@which ginkgo > /dev/null || (echo "❌ ginkgo not installed. Run: go install github.com/onsi/ginkgo/v2/ginkgo@latest" && exit 1)
	@which controller-gen > /dev/null || (echo "❌ controller-gen not installed. Run: make controller-gen" && exit 1)
	@echo "✅ Test environment ready"

# Help for conversion tests
help-conversion-tests:
	@echo "Conversion Test Targets:"
	@echo "  test-conversion                 - Run all conversion tests"
	@echo "  test-conversion-unit            - Run unit tests only"
	@echo "  test-conversion-integration     - Run integration tests only"
	@echo "  test-conversion-performance     - Run performance tests only"
	@echo "  test-conversion-coverage        - Run tests with coverage report"
	@echo "  bench-conversion                - Run benchmarks"
	@echo "  test-conversion-focus FOCUS=X   - Run tests matching pattern X"
	@echo "  lint-conversion                 - Lint conversion code"
	@echo "  check-conversion-coverage       - Check if coverage meets 80% threshold"
	@echo "  clean-conversion-tests          - Clean test artifacts"
	@echo "  test-conversion-report          - Generate JSON test report"
	@echo "  debug-conversion-test TEST=X    - Debug specific test X"
	@echo "  check-conversion-test-env       - Check test environment setup"
