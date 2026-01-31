# jobsearch-mcp

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://go.dev)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB.svg)](https://python.org)
[![MCP](https://img.shields.io/badge/MCP-compatible-green.svg)](https://modelcontextprotocol.io)

Track your job search conversations from email. Automatically syncs, classifies, and organizes recruiter emails so you never miss a follow-up.

## Features

- **Smart Email Classification** - AI-powered filtering distinguishes real recruiter conversations from job alerts and spam
- **Conversation Tracking** - Groups email threads, tracks who's waiting on whom
- **Stale Detection** - Highlights conversations that need follow-up
- **Natural Language Queries** - Ask about your job search in plain English (works with any AI agent/CLI)
- **MCP Integration** - Works with Claude Desktop, Cursor, and other MCP clients
- **Privacy-First** - Stores metadata only, runs locally, your data stays yours
- **Extensible** - Pluggable email providers (Gmail included, add your own)

## Install

```bash
git clone https://github.com/vijay-prabhu/jobsearch-mcp.git
cd jobsearch-mcp
make build
make setup-python
make install-local  # Installs to ~/bin
```

Add `~/bin` to your PATH if not already:
```bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
```

### As a Skill (Claude Code, Cursor, etc.)

The included `SKILL.md` provides natural language command mappings that AI agents can use. For Claude Code:

```bash
git clone https://github.com/vijay-prabhu/jobsearch-mcp.git ~/.claude/skills/jobsearch
cd ~/.claude/skills/jobsearch && make build && make setup-python && make install-local
```

Other AI agents can use `SKILL.md` directly - it maps natural language queries to CLI commands.

## Setup

```bash
# Initialize configuration
jobsearch config init

# Add your email provider credentials (see Email Providers below)
# Then sync your inbox
jobsearch sync
```

## Examples

### Natural language queries

Works with any AI agent that can execute shell commands:

```
how's my job search going?
what conversations need my attention?
show me the Stripe conversation
read the full email thread with Google
who haven't I heard back from?
what should I follow up on?
what did the Anthropic recruiter say?
```

### CLI commands

```bash
# Get overview
jobsearch stats

# Conversations needing your response
jobsearch list --status=waiting_on_me

# Conversations you're waiting on
jobsearch list --status=waiting_on_them

# Stale conversations (need follow-up)
jobsearch list --status=stale

# View specific company
jobsearch show stripe

# Read full email thread (fetches content on demand)
jobsearch thread stripe

# Recent activity
jobsearch list --since=7d

# Search across everything
jobsearch search "onsite interview"
```

### Example output

```
Job Search Overview
━━━━━━━━━━━━━━━━━━━
Total Conversations: 24
├── Waiting on me: 3
├── Waiting on them: 8
├── Stale (need follow-up): 5
└── Closed: 8

Conversations Needing Action
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Stripe - Sarah Chen (sarah@stripe.com)
  Last contact: 2 days ago
  Status: Waiting on me

Anthropic - Mike Johnson (mike@anthropic.com)
  Last contact: 1 day ago
  Status: Waiting on me
```

## Email Providers

jobsearch-mcp uses a pluggable provider architecture. Configure your provider in `~/.config/jobsearch/config.toml`:

### Gmail (included)

```toml
[email]
provider = "gmail"

[gmail]
credentials_path = "~/.config/jobsearch/credentials.json"
token_path = "~/.config/jobsearch/token.json"
```

**Setup**: Create OAuth credentials in [Google Cloud Console](https://console.cloud.google.com/), enable Gmail API, download credentials JSON.

### Adding Other Providers

The codebase supports adding new email providers by implementing the `Provider` interface:

```go
type Provider interface {
    Name() string
    Authenticate(ctx context.Context) error
    FetchEmails(ctx context.Context, opts FetchOptions) ([]Email, error)
    GetEmail(ctx context.Context, id string) (*Email, error)
    GetUserEmail(ctx context.Context) (string, error)
}
```

See `internal/email/gmail/` for reference implementation. Contributions welcome for:
- Outlook (Microsoft Graph API)
- IMAP (generic)
- ProtonMail Bridge

## MCP Integration

Add to your MCP client config:

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):

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

### MCP Tools

| Tool | Description |
|------|-------------|
| `list_conversations` | List conversations with filters |
| `get_conversation` | Get details of a specific conversation |
| `get_pending_actions` | Conversations needing your response |
| `search_conversations` | Search across all conversations |
| `get_stats` | Job search statistics |

### MCP Resources

| Resource | Description |
|----------|-------------|
| `jobsearch://summary` | Overview with key metrics |
| `jobsearch://pending` | Conversations needing action |
| `jobsearch://recent` | Recent activity |
| `jobsearch://companies` | All companies you're talking to |

## Configuration

Full config at `~/.config/jobsearch/config.toml`:

```toml
[email]
provider = "gmail"  # or your provider

[database]
path = "~/.local/share/jobsearch/jobsearch.db"

[classifier]
port = 8642

[llm]
primary = "ollama"      # Local-first
fallback = "openai"     # Cloud fallback

[llm.ollama]
model = "llama3.2:1b"
host = "http://localhost:11434"

[filters]
domain_whitelist = ["greenhouse.io", "lever.co", "ashbyhq.com"]
domain_blacklist = ["noreply@linkedin.com"]
subject_blacklist = ["job alert", "weekly digest"]

[tracking]
stale_after_days = 7

[privacy]
store_email_body = false  # Metadata only by default
```

## How It Works

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    Email     │────▶│   Filter &   │────▶│   SQLite     │
│   Provider   │     │   Classify   │     │   Database   │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                     ┌──────┴──────┐
                     │  LLM Layer  │
                     │ Ollama/API  │
                     └─────────────┘
```

1. **Sync** - Fetches new emails from your provider
2. **Filter** - Multi-layer filtering (domains, keywords, patterns)
3. **Classify** - LLM determines if email is job-related
4. **Track** - Groups into conversations, computes status
5. **Query** - CLI, MCP, or natural language access

## Development

```bash
make build            # Build Go binary
make test             # Run all tests
make lint             # Run linters
make install-local    # Install to ~/bin (no sudo)
make install-system   # Install to /usr/local/bin (sudo)
make serve-classifier # Start classification service
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Want to add an email provider?** Implement the `Provider` interface and submit a PR.

## License

MIT - see [LICENSE](LICENSE)
