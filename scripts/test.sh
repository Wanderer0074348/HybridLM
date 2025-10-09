#!/bin/bash

set -e

echo "🧪 Running HybridLM Test Suite"
echo "================================"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Run unit tests
echo -e "\n${GREEN}Running unit tests...${NC}"
go test -v -race ./internal/... || { echo -e "${RED}Unit tests failed${NC}"; exit 1; }

# Run handler tests
echo -e "\n${GREEN}Running handler tests...${NC}"
go test -v -race ./internal/handlers/... || { echo -e "${RED}Handler tests failed${NC}"; exit 1; }

# Generate coverage
echo -e "\n${GREEN}Generating coverage report...${NC}"
go test -coverprofile=coverage.out ./... > /dev/null
go tool cover -html=coverage.out -o coverage.html

# Show coverage summary
echo -e "\n${GREEN}Coverage Summary:${NC}"
go tool cover -func=coverage.out | grep total

echo -e "\n${GREEN}✅ All tests passed!${NC}"
echo -e "📄 Coverage report: ${GREEN}coverage.html${NC}"
