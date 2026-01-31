.PHONY: build test lint clean setup-python serve-classifier install install-local install-system uninstall help

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
	@echo "  make install          Install jobsearch to GOPATH/bin"
	@echo "  make install-local    Install jobsearch to ~/bin (no sudo)"
	@echo "  make install-system   Install jobsearch to /usr/local/bin (sudo)"
	@echo "  make uninstall        Remove jobsearch from ~/bin"
	@echo "  make clean            Clean build artifacts"

# Build Go binary
build:
	go build -o bin/jobsearch cmd/jobsearch/main.go

# Install binary to GOPATH/bin
install:
	go install ./cmd/jobsearch

# Install binary to ~/bin (no sudo required)
install-local: build
	@mkdir -p $(HOME)/bin
	@cp bin/jobsearch $(HOME)/bin/jobsearch
	@echo "Installed jobsearch to ~/bin/jobsearch"
	@echo "Add to PATH if needed: export PATH=\"\$$HOME/bin:\$$PATH\""

# Install binary to /usr/local/bin (requires sudo)
install-system: build
	@echo "Installing jobsearch to /usr/local/bin..."
	sudo cp bin/jobsearch /usr/local/bin/jobsearch
	@echo "Done! Run 'jobsearch --help' to verify."

# Remove binary from ~/bin
uninstall:
	@rm -f $(HOME)/bin/jobsearch
	@echo "Removed jobsearch from ~/bin"

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
