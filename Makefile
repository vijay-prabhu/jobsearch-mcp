.PHONY: build test lint clean setup-python serve-classifier install help

# Default target
help:
	@echo "jobsearch-mcp - Developer-focused job search automation tool"
	@echo ""
	@echo "Usage:"
	@echo "  make build            Build Go binary"
	@echo "  make test             Run all tests (Go + Python)"
	@echo "  make test-go          Run Go tests only"
	@echo "  make test-python      Run Python tests only"
	@echo "  make lint             Run linters (Go + Python)"
	@echo "  make setup-python     Install Python dependencies"
	@echo "  make serve-classifier Start classification service"
	@echo "  make install          Install jobsearch binary"
	@echo "  make clean            Clean build artifacts"

# Build Go binary
build:
	go build -o bin/jobsearch cmd/jobsearch/main.go

# Install binary to GOPATH/bin
install:
	go install ./cmd/jobsearch

# Run all tests
test: test-go test-python

# Run Go tests
test-go:
	go test -v ./...

# Run Python tests
test-python:
	cd classifier && python -m pytest -v

# Run linters
lint:
	golangci-lint run ./...
	cd classifier && ruff check .

# Install Python dependencies
setup-python:
	cd classifier && pip install -e ".[dev]"

# Start classification service
serve-classifier:
	cd classifier && uvicorn src.classifier.main:app --port 8642 --reload

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf classifier/dist/
	rm -rf classifier/*.egg-info
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .pytest_cache -exec rm -rf {} + 2>/dev/null || true
