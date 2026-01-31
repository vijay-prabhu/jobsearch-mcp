# ADR-001: Hybrid Go + Python Architecture

## Status
Accepted

## Context
We need to build a job search tracking tool that requires:
1. CLI commands for user interaction
2. MCP server for Claude Desktop integration
3. Email processing with Gmail API
4. LLM-based email classification

The classification component benefits from Python's ecosystem (FastAPI, ollama-python, openai), while the CLI/MCP server benefits from Go's single-binary distribution and robust CLI frameworks.

## Decision
Use a hybrid architecture:
- **Go** for the main CLI binary, MCP server, database layer, and email fetching
- **Python** for the LLM classification service (FastAPI microservice)

Communication between Go and Python happens via HTTP REST API.

## Consequences

### Positive
- Go produces a single static binary, easy to distribute
- Go's Cobra framework provides excellent CLI experience
- Python's LLM libraries are mature and well-maintained
- Classification service can be scaled or replaced independently
- Each language used for its strengths

### Negative
- Requires two runtimes (Go binary + Python service)
- Additional deployment complexity
- HTTP overhead for classification calls
- Users need Python 3.11+ installed for full functionality

### Mitigations
- Classification service is optional (keyword filtering works without it)
- Docker/container deployment can bundle both components
- HTTP latency is minimal for local communication

## Alternatives Considered

### Pure Go
- Ollama has a Go client, but it's less mature
- Would require FFI or subprocess for some LLM features
- Rejected due to LLM library limitations

### Pure Python
- Would work but CLI distribution is harder
- MCP server implementation less clean
- Rejected for deployment complexity
