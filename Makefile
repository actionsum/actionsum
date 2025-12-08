.PHONY: build install clean test test-verbose test-coverage bench run help

BINARY_NAME=actionsum
INSTALL_PATH=/usr/local/bin

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/actionsum
	@echo "Build complete: ./$(BINARY_NAME)"

# Install to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installed successfully"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f /tmp/actionsum.pid
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...
	@echo ""
	@echo "Detailed coverage:"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. ./pkg/...

# Run the application
run: build
	./$(BINARY_NAME)

# Show help
help:
	@echo "actionsum Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build           Build the application"
	@echo "  make install         Install to $(INSTALL_PATH)"
	@echo "  make clean           Remove build artifacts"
	@echo "  make test            Run tests"
	@echo "  make test-verbose    Run tests with verbose output"
	@echo "  make test-coverage   Run tests with coverage report"
	@echo "  make bench           Run benchmarks"
	@echo "  make run             Build and run"
	@echo "  make help            Show this help message"
