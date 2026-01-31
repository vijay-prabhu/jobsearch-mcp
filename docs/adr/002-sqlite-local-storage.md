# ADR-002: SQLite for Local Storage

## Status
Accepted

## Context
The tool needs to persist:
- Conversation state (company, recruiter, status)
- Email references (Gmail IDs, metadata)
- Sync state (last sync timestamp, history ID)
- User preferences and configuration

Options considered:
1. SQLite
2. PostgreSQL
3. File-based JSON/YAML
4. Embedded key-value store (BoltDB, BadgerDB)

## Decision
Use SQLite as the primary storage backend.

## Consequences

### Positive
- Zero configuration - no server to run
- Single file database, easy to backup/restore
- SQL provides flexible querying
- Widely supported with excellent tooling
- Cross-platform compatibility
- Good performance for expected data volumes (<100k records)

### Negative
- Single-writer limitation (not relevant for single-user tool)
- No built-in replication
- Limited concurrent access

### Design Details
- Database file stored at `~/.local/share/jobsearch/jobsearch.db`
- Schema migrations embedded in Go binary
- Uses `modernc.org/sqlite` (pure Go, no CGO)
- Tables: `conversations`, `emails`, `sync_state`

## Alternatives Considered

### PostgreSQL
- Overkill for single-user local tool
- Requires running a server
- Rejected for complexity

### JSON/YAML Files
- Simple but no querying capability
- Difficult to update partial records
- Rejected for lack of flexibility

### BoltDB/BadgerDB
- Good for key-value but we need relational queries
- Rejected for query limitations
