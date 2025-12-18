.PHONY: build install clean test test-verbose test-coverage bench run help release bump-version

BINARY_NAME=actionsum
INSTALL_PATH=/usr/local/bin
VERSION_FILE=version.json
VERSION=$(shell jq -r '.version' $(VERSION_FILE) 2>/dev/null || echo "v0.0.0")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
NEW_VERSION=

# Build the application
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Running tests..."
	go test ./...
	@echo "Build complete: ./$(BINARY_NAME)"

# Push changes and tags to Git
push-git:
	git push origin $(NEW_VERSION)
	git push origin main
	echo "Changes and tags pushed to Git."

# Increment version and update version.json
bump-version:
	@echo "Running tests..."; \
	if ! make test; then \
		echo "Tests failed. Aborting version bump."; \
		exit 1; \
	fi
	@if [ -z "$(TYPE)" ]; then \
		echo "Error: TYPE argument (major, feat, fix) is required."; \
		exit 1; \
	fi
	@echo "Current version: $(VERSION)"
	@MAJOR=$$(echo $(VERSION) | cut -d. -f1 | tr -d 'v'); \
	MINOR=$$(echo $(VERSION) | cut -d. -f2); \
	PATCH=$$(echo $(VERSION) | cut -d. -f3); \
	if [ -z "$$PATCH" ]; then PATCH=0; fi; \
	case "$(TYPE)" in \
		major) MAJOR=$$((MAJOR + 1)); MINOR=0; PATCH=0 ;; \
		feat) MINOR=$$((MINOR + 1)); PATCH=0 ;; \
		fix) PATCH=$$((PATCH + 1)) ;; \
		*) echo "Invalid TYPE: $(TYPE). Use 'major', 'feat', or 'fix'."; exit 1 ;; \
	esac; \
	NEW_VERSION="v$$MAJOR.$$MINOR.$$PATCH"; \
	echo "New version: $$NEW_VERSION"; \
	echo '{' > $(VERSION_FILE); \
	echo '  "version": "'$$NEW_VERSION'",' >> $(VERSION_FILE); \
	echo '  "date": "$(DATE)"' >> $(VERSION_FILE); \
	echo '}' >> $(VERSION_FILE); \
	git add .; \
	git commit -m "Bump version to $$NEW_VERSION"; \
	git tag $$NEW_VERSION; \
	NEW_VERSION=$$NEW_VERSION make push-git; \
	echo "Version bumped to $$NEW_VERSION, tagged, and pushed."

# Install to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installed successfully"

# Uninstall from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstalled successfully"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f /tmp/actionsum.pid
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
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
	@echo "  make bump-version TYPE=fix|feat|major  Increment version"
	@echo "  make help            Show this help message"
