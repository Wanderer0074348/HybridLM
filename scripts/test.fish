#!/usr/bin/env fish

echo "🧪 Running HybridLM Test Suite"
echo "================================"

# Run unit tests
echo ""
echo (set_color green)"Running unit tests..."(set_color normal)
if not go test -v -race ./internal/...
    echo (set_color red)"Unit tests failed"(set_color normal)
    exit 1
end

# Run handler tests
echo ""
echo (set_color green)"Running handler tests..."(set_color normal)
if not go test -v -race ./internal/handlers/...
    echo (set_color red)"Handler tests failed"(set_color normal)
    exit 1
end

# Generate coverage
echo ""
echo (set_color green)"Generating coverage report..."(set_color normal)
go test -coverprofile=coverage.out ./... > /dev/null
go tool cover -html=coverage.out -o coverage.html

# Show coverage summary
echo ""
echo (set_color green)"Coverage Summary:"(set_color normal)
go tool cover -func=coverage.out | grep total

echo ""
echo (set_color green)"✅ All tests passed!"(set_color normal)
echo "📄 Coverage report: "(set_color green)"coverage.html"(set_color normal)
