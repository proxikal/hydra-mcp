.PHONY: all build test test-race test-integration test-benchmarks lint coverage clean install help

# Default target
all: test lint build

# Build the binary
build:
	@echo "🔨 Building Hydra..."
	@go build -o bin/hydra cmd/hydra/main.go
	@echo "✅ Build complete: bin/hydra"

# Install to /usr/local/bin
install: build
	@echo "📦 Installing to /usr/local/bin..."
	@sudo mv bin/hydra /usr/local/bin/
	@echo "✅ Installed: /usr/local/bin/hydra"

# Run all tests
test:
	@echo "🧪 Running unit tests..."
	@go test -v ./...

# Run tests with race detector
test-race:
	@echo "🏁 Running tests with race detector..."
	@go test -race -v ./...

# Run integration tests only
test-integration:
	@echo "🔗 Running integration tests..."
	@go test -race -v ./test/integration/...

# Run benchmark tests
test-benchmarks:
	@echo "⚡ Running benchmark tests..."
	@go test -bench=. -benchmem ./test/benchmarks/...

# Run linter
lint:
	@echo "🔍 Running golangci-lint..."
	@golangci-lint run --timeout=5m

# Generate coverage report
coverage:
	@echo "📊 Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Show help
help:
	@echo "Hydra Makefile Commands:"
	@echo ""
	@echo "  make build          - Build the binary"
	@echo "  make install        - Install to /usr/local/bin"
	@echo "  make test           - Run unit tests"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-benchmarks - Run benchmark tests"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make coverage       - Generate coverage report"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make all            - Run tests, lint, and build"
	@echo "  make help           - Show this help message"
