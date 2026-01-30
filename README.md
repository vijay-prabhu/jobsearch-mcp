# jobsearch-mcp

A developer-focused job search automation tool that tracks recruiter conversations from your email with CLI and MCP (Model Context Protocol) server support.

## Features

- **Email Integration**: Connects to Gmail via OAuth to fetch job-related emails
- **Smart Filtering**: Multi-layer filtering (domain whitelist/blacklist, keywords, LLM classification)
- **Conversation Tracking**: Groups emails into conversations, tracks status (waiting on me/them, stale)
- **CLI Interface**: Full-featured command-line interface for managing job search
- **MCP Server**: Integrates with Claude Desktop/Cursor for AI-assisted job search management
- **Privacy-First**: Stores metadata only by default, optional encrypted body storage
- **Local LLM Support**: Uses Ollama for classification (with OpenAI fallback)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Go Layer                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│  │   CLI    │  │   MCP    │  │  Email   │  │     SQLite       │ │
│  │  (Cobra) │  │  Server  │  │ Fetcher  │  │    Database      │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘ │
│       │             │             │                  │           │
│       └─────────────┴──────┬──────┴──────────────────┘           │
│                            │                                     │
└────────────────────────────┼─────────────────────────────────────┘
                             │ HTTP
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Python Layer                                │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Classification Service (FastAPI)             │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │   │
│  │  │   Ollama    │  │   OpenAI    │  │    Structured    │  │   │
│  │  │  (Primary)  │  │ (Fallback)  │  │    Extraction    │  │   │
│  │  └─────────────┘  └─────────────┘  └──────────────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Go 1.22+
- Python 3.11+
- [Ollama](https://ollama.ai) with `llama3.2:1b` model
- Google Cloud project with Gmail API enabled

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/vijay-prabhu/jobsearch-mcp.git
cd jobsearch-mcp

# Build Go binary
make build

# Install Python dependencies
make setup-python
```

### Gmail API Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select existing)
3. Enable the Gmail API:
   - Navigate to "APIs & Services" > "Library"
   - Search for "Gmail API" and enable it
4. Create OAuth 2.0 credentials:
   - Go to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth client ID"
   - Select "Desktop app" as application type
   - Download the JSON file
5. Save as `~/.config/jobsearch/credentials.json`

## Quick Start

```bash
# Initialize configuration
jobsearch config init

# Authenticate with Gmail (opens browser)
jobsearch sync

# List conversations needing your response
jobsearch list --status=waiting_on_me

# Show details for a specific company
jobsearch show stripe

# Get job search statistics
jobsearch stats
```

## Configuration

Configuration file location: `~/.config/jobsearch/config.toml`

```toml
[gmail]
credentials_path = "~/.config/jobsearch/credentials.json"
token_path = "~/.config/jobsearch/token.json"
max_results = 100

[database]
path = "~/.local/share/jobsearch/jobsearch.db"

[llm]
primary = "ollama"
fallback = "openai"

[llm.ollama]
model = "llama3.2:1b"
host = "http://localhost:11434"

[llm.openai]
model = "gpt-4o-mini"
# API key read from OPENAI_API_KEY env var

[classifier]
port = 8642

[filters]
domain_whitelist = ["greenhouse.io", "lever.co", "ashbyhq.com"]
domain_blacklist = ["noreply@linkedin.com", "mailchimp.com"]
subject_blacklist = ["job alert", "new jobs for you", "weekly digest"]
subject_keywords = ["opportunity", "role", "position", "interview"]
body_keywords = ["your background", "schedule a call", "reaching out"]

[tracking]
stale_after_days = 7

[privacy]
store_email_body = false

[mcp]
enabled = true
transport = "stdio"
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `jobsearch sync` | Fetch and process new emails |
| `jobsearch list` | List conversations (with filters) |
| `jobsearch show <company>` | Show conversation details |
| `jobsearch stats` | Display job search statistics |
| `jobsearch search <query>` | Search across conversations |
| `jobsearch config init` | Create default config file |
| `jobsearch mcp` | Start MCP server (stdio) |

## MCP Integration

Add to Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

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

### Available MCP Tools

- `list_conversations` - List job search conversations with filters
- `get_conversation` - Get details of a specific conversation
- `get_pending_actions` - Get conversations needing attention
- `search_conversations` - Search across all conversations
- `get_stats` - Get job search statistics

## Development

```bash
# Run tests
make test

# Run linters
make lint

# Start classifier service (for development)
make serve-classifier
```

## License

MIT License - see [LICENSE](LICENSE) for details.
