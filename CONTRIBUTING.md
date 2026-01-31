# Contributing to JobSearch MCP

Thank you for considering contributing to JobSearch MCP! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites

- Go 1.22 or later
- Python 3.11 or later
- Ollama (optional, for local LLM classification)

### Getting Started

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/YOUR_USERNAME/jobsearch-mcp.git
   cd jobsearch-mcp
   ```

2. Install Go dependencies:
   ```bash
   go mod download
   ```

3. Set up the Python classifier service:
   ```bash
   cd classifier
   python -m venv venv
   source venv/bin/activate  # or `venv\Scripts\activate` on Windows
   pip install -e ".[dev]"
   ```

4. Run tests:
   ```bash
   # Go tests
   go test ./...

   # Python tests
   cd classifier
   pytest tests/
   ```

## Project Structure

```
jobsearch-mcp/
├── cmd/jobsearch/     # Main CLI entry point
├── internal/          # Go packages
│   ├── cli/           # Cobra CLI commands
│   ├── config/        # Configuration loading
│   ├── database/      # SQLite database layer
│   ├── email/         # Email providers (Gmail)
│   ├── filter/        # Multi-layer filtering
│   ├── tracker/       # Conversation tracking
│   ├── classifier/    # Go client for Python service
│   ├── mcp/           # MCP server implementation
│   └── output/        # Output formatting
├── classifier/        # Python classification service
│   ├── src/classifier/
│   └── tests/
└── docs/              # Documentation and ADRs
```

## Code Style

### Go

- Follow standard Go conventions
- Run `gofmt` and `goimports` before committing
- Use meaningful variable names
- Add comments for exported functions

### Python

- Follow PEP 8
- Use type hints
- Run `ruff check` before committing

## Testing

All changes should include appropriate tests:

- Go: Add tests in `*_test.go` files
- Python: Add tests in `classifier/tests/`

Run the full test suite before submitting:

```bash
go test ./...
cd classifier && pytest tests/
```

## Submitting Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes with clear, atomic commits

3. Ensure tests pass and code is formatted

4. Push and create a Pull Request

### Commit Messages

Use clear, descriptive commit messages:

- `feat: add new CLI command for exporting data`
- `fix: handle empty email body in classifier`
- `docs: update installation instructions`
- `test: add tests for filter edge cases`

## Reporting Issues

When reporting bugs, please include:

- Your operating system and version
- Go and Python versions
- Steps to reproduce the issue
- Expected vs actual behavior
- Any relevant logs or error messages

## Architecture Decisions

For significant changes, please read the Architecture Decision Records (ADRs) in `docs/adr/` to understand the design rationale.

If your change involves a significant architectural decision, please propose it as an ADR first.

## Questions?

Feel free to open an issue for questions or discussions about the project.
