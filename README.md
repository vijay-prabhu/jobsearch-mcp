# jobsearch-mcp

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://go.dev)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB.svg)](https://python.org)
[![MCP](https://img.shields.io/badge/MCP-compatible-green.svg)](https://modelcontextprotocol.io)

Track your job search conversations from email. Automatically syncs, classifies, and organizes recruiter emails so you never miss a follow-up.

## Features

- **Smart Email Classification** - AI-powered filtering distinguishes real recruiter conversations from job alerts and spam
- **Self-Learning Filters** - Mark false positives with `mark-spam`, system learns and auto-blacklists repeat offenders
- **Conversation Tracking** - Groups email threads by recruiter, tracks who's waiting on whom
- **Sent Email Tracking** - Captures your replies to recruiters, correctly computes "waiting on them" status
- **Smart Auto-Grouping** - Automatically groups emails from the same recruiter into conversations
- **Stale Detection** - Highlights conversations that need follow-up
- **Natural Language Queries** - Ask about your job search in plain English (works with any AI agent/CLI)
- **MCP Integration** - Works with Claude Desktop, Cursor, and other MCP clients
- **Privacy-First** - Stores metadata only, runs locally, your data stays yours
- **Fast Parallel Processing** - Fetches emails (10 concurrent) and classifies (5 concurrent) in parallel
- **Progress Tracking** - Real-time progress output during sync with emoji indicators
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

### Auto-rebuild Options

To avoid manually rebuilding after `git pull`:

```bash
# Option 1: Wrapper script (rebuilds automatically when source changes)
make install-wrapper

# Option 2: Git hook (rebuilds after every git pull)
make install-hooks
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
jobsearch stats --detailed             # Detailed breakdown with charts

# Sync emails
jobsearch sync                 # Incremental (since last sync, or 30 days)
jobsearch sync --days=60       # Fetch last 60 days
jobsearch sync --full          # Full sync (ignore last sync time)

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

# Merge duplicate conversations
jobsearch merge "Stripe" "stripe-2"    # By company name
jobsearch merge abc123 def456          # By conversation ID

# Archive/unarchive conversations
jobsearch archive "Stripe"             # Hide from default list
jobsearch unarchive "Stripe"           # Show in default list again
jobsearch list --include-archived      # Show all including archived

# Mark false positives (not job-related)
jobsearch mark-spam "Walmart"          # Marks as spam, learns from feedback
                                       # Auto-blacklists domain after 3 reports

# Export data
jobsearch export --format=csv > jobs.csv    # Export to CSV
jobsearch export --format=json > jobs.json  # Export to JSON

# Recent activity
jobsearch list --since=7d

# Search across everything
jobsearch search "onsite interview"
```

### Example output

**Sync with progress:**
```
Syncing emails (last 60 days)...
📋 Listing emails: 158 found
📥 Downloading: 158/158 emails (100%)
🔍 Filtering: 158 emails
🤖 Classifying with LLM: 25/25 (100%)
💾 Processing: 21/21 emails (100%)
🔄 Updating conversation statuses...

Sync complete:
  Emails fetched:        158
  Job-related:           3
  Classified by LLM:     25
  New conversations:     0
  Updated conversations: 21
```

**Stats overview:**
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

**Merge conversations:**
```
Merged conversations:
  From: stripe-2 (abc123)
  Into: Stripe (def456)
  Emails moved: 3
  New total emails: 8
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
| `list_conversations` | List conversations with filters (supports `include_archived`) |
| `get_conversation` | Get details of a specific conversation |
| `get_pending_actions` | Conversations needing your response |
| `search_conversations` | Search across all conversations |
| `get_stats` | Job search statistics (supports `detailed` flag) |
| `merge_conversations` | Merge two conversations into one |
| `archive_conversation` | Archive/unarchive a conversation |

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
[gmail]
credentials_path = "~/.config/jobsearch/credentials.json"
token_path = "~/.config/jobsearch/token.json"
max_results = 100  # Emails per sync (max 5000)

[database]
path = "~/.local/share/jobsearch/jobsearch.db"

[classifier]
host = "http://localhost"
port = 8642
cache_enabled = true     # Cache classifications to reduce LLM calls
min_confidence = 0.5     # Minimum confidence threshold

[llm]
primary = "ollama"      # Local-first
fallback = "openai"     # Cloud fallback

[llm.ollama]
model = "llama3.2:1b"
host = "http://localhost:11434"

[llm.openai]
model = "gpt-4o-mini"
# API key read from OPENAI_API_KEY env var

[filters]
# Layer 1: Always include emails from these domains
domain_whitelist = ["greenhouse.io", "lever.co", "ashbyhq.com", "smartrecruiters.com", "workday.com"]
# Layer 2: Always exclude
domain_blacklist = ["noreply@linkedin.com", "mailchimp.com", "sendgrid.net"]
subject_blacklist = ["job alert", "weekly digest", "new jobs for you"]
# Layer 3: Keyword scoring
subject_keywords = ["opportunity", "role", "position", "interview", "application"]
body_keywords = ["your background", "schedule a call", "interested in", "reaching out"]

[tracking]
stale_after_days = 7

[privacy]
store_email_body = false  # Metadata only by default (set true to cache email bodies)
encryption_key_path = "~/.config/jobsearch/encryption.key"

[mcp]
enabled = true
transport = "stdio"
```

### Configuration Notes

- **Filters are fully customizable** - Add your own domains, keywords, and patterns
- **Email body caching** - Set `store_email_body = true` for faster thread viewing (bodies cached locally)
- **Parallel processing** - Fetches 10 emails concurrently, classifies 5 at a time
- **Optimized queries** - Database indexes for fast filtering and searching
- **Privacy-first** - By default, only metadata is stored; email bodies are fetched on demand

## How It Works

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    Email     │────▶│   Filter &   │────▶│   SQLite     │
│   Provider   │     │   Classify   │     │   Database   │
│ (10 parallel)│     │ (5 parallel) │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                     ┌──────┴──────┐
                     │  LLM Layer  │
                     │ Ollama/API  │
                     └─────────────┘
```

1. **Sync** - Fetches new emails from inbox AND sent folder (10 concurrent connections)
2. **Filter** - Multi-layer filtering (domains, keywords, patterns) for both inbound and outbound
3. **Classify** - LLM determines if email is job-related (5 concurrent)
4. **Group** - Smart auto-grouping by recruiter email (your replies are grouped with recruiter threads)
5. **Track** - Computes conversation status (waiting on me/them, stale) based on who sent last
6. **Query** - CLI, MCP, or natural language access

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
