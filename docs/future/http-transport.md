# HTTP Transport for MCP

## Overview

Add HTTP/SSE transport option for MCP server, enabling remote access from web applications or non-local Claude instances.

## Motivation

- Access job search data from web interfaces
- Support remote Claude Desktop connections
- Enable multi-device access
- Foundation for future web dashboard

## Current State

Currently, MCP server only supports stdio transport:
```bash
jobsearch mcp  # Runs on stdin/stdout
```

## Proposed Design

### Server Modes

```toml
[mcp]
transport = "http"  # or "stdio" (default)
port = 8080
host = "127.0.0.1"  # localhost only by default
auth_token = ""     # optional bearer token
```

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mcp` | POST | JSON-RPC endpoint |
| `/mcp/sse` | GET | Server-Sent Events for streaming |
| `/health` | GET | Health check |

### Authentication

Optional bearer token authentication:
```bash
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

## Implementation Plan

### Phase 1: Basic HTTP Server
```go
// internal/mcp/http.go
func (s *Server) StartHTTP(ctx context.Context, addr string) error {
    mux := http.NewServeMux()
    mux.HandleFunc("/mcp", s.handleHTTP)
    mux.HandleFunc("/health", s.handleHealth)

    server := &http.Server{Addr: addr, Handler: mux}
    return server.ListenAndServe()
}
```

### Phase 2: SSE Support
For streaming responses (long-running operations):
```go
mux.HandleFunc("/mcp/sse", s.handleSSE)
```

### Phase 3: Authentication
```go
func (s *Server) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if s.authToken != "" {
            token := r.Header.Get("Authorization")
            if token != "Bearer "+s.authToken {
                http.Error(w, "Unauthorized", 401)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}
```

## CLI Changes

```bash
# Start HTTP server
jobsearch mcp --transport=http --port=8080

# With authentication
jobsearch mcp --transport=http --auth-token=mysecret
```

## Security Considerations

1. **Default to localhost** - Don't expose to network by default
2. **Authentication required for remote** - If binding to 0.0.0.0, require auth
3. **HTTPS support** - Add TLS option for production use
4. **Rate limiting** - Prevent abuse

## Claude Desktop Configuration

```json
{
  "mcpServers": {
    "jobsearch-remote": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer <token>"
      }
    }
  }
}
```

## Effort Estimate

- Basic HTTP endpoint: 1 day
- SSE support: 1 day
- Authentication: 0.5 day
- TLS support: 0.5 day
- **Total: ~3 days**
