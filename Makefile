.PHONY: test test-unit test-integration test-coverage test-verbose bench clean help

help:
	@echo "Available targets:"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  bench          - Run benchmarks"
	@echo "  clean          - Clean test artifacts"

test:
	@echo "🧪 Running all tests..."
	@go test -race ./...

test-unit:
	@echo "📦 Running unit tests..."
	@go test -v -race ./internal/...

test-verbose:
	@echo "📋 Running tests with verbose output..."
	@go test -v -race ./...

test-coverage:
	@echo "📊 Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "✅ Coverage report: coverage.html"

bench:
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./...

clean:
	@echo "🧹 Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@go clean -testcache
