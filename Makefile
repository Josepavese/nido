.PHONY: fmt lint test build tui-demo clean

# Default target
all: build

# Format code with gofumpt (or gofmt if gofumpt not available)
fmt:
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		gofmt -w .; \
	fi

# Run linters
lint:
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed, skipping"; \
	fi

# Run all tests
test:
	go test -v ./...

# Run only theme tests (quick check during development)
test-theme:
	go test -v ./internal/tui/theme/...

# Build the main binary
build:
	go build -o bin/nido ./cmd/nido

# Run TUI in demo/development mode
tui-demo:
	go run ./cmd/nido gui

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Full CI check: format, lint, test, build
ci: fmt lint test build
	@echo "✅ All checks passed"
