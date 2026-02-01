# ADR-005: MCP Server for IDE Integration

## Status
Accepted

## Context
The tool should integrate with Claude Desktop and other MCP-compatible clients to enable:
- Natural language queries about job search status
- AI-assisted follow-up drafting
- Dashboard views within the IDE

Options for integration:
1. REST API
2. GraphQL API
3. Model Context Protocol (MCP)
4. Language Server Protocol (LSP)

## Decision
Implement an MCP server using JSON-RPC over stdio transport.

### MCP Tools Exposed

1. **list_conversations**
   - List all conversations with optional status filter
   - Supports `include_archived` parameter
   - Returns: array of conversation summaries

2. **get_conversation**
   - Get detailed conversation with email history
   - Input: company name or conversation ID
   - Returns: conversation details + emails

3. **get_pending_actions**
   - Get conversations needing attention
   - Returns: waiting_on_me and stale conversations

4. **search_conversations**
   - Full-text search across conversations
   - Input: query string
   - Returns: matching conversations

5. **get_stats**
   - Dashboard statistics
   - Supports `detailed` flag for extended breakdown
   - Returns: counts by status, response rates

6. **merge_conversations**
   - Merge two conversations into one
   - Input: target and source identifiers
   - Returns: merge result with email count

7. **archive_conversation**
   - Archive or unarchive a conversation
   - Input: identifier, unarchive flag
   - Returns: archive result

## Consequences

### Positive
- Native Claude Desktop integration
- Natural language interface to job search data
- Tools composable by AI for complex queries
- Standard protocol with growing ecosystem

### Negative
- MCP is relatively new protocol
- Limited to MCP-compatible clients
- Debugging can be challenging (stdio transport)

### Configuration
```toml
[mcp]
transport = "stdio"  # stdio or sse
```

### Usage
Add to Claude Desktop config (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "jobsearch": {
      "command": "/path/to/jobsearch",
      "args": ["mcp"]
    }
  }
}
```

## Alternatives Considered

### REST API
- Well understood, easy to implement
- No native IDE integration
- Rejected for lack of AI integration

### GraphQL
- Flexible querying
- Complex to implement
- Overkill for this use case
- Rejected for complexity

### Language Server Protocol
- Good for code intelligence
- Not designed for data tools
- Rejected as wrong protocol
