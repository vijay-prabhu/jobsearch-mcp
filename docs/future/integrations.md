# Integrations Overview

## Overview

This document outlines potential integrations with external services and tools that could enhance the job search tracking experience.

## Communication Integrations

### Slack

**Use Case**: Team notifications, daily digests in a channel

```toml
[integrations.slack]
webhook_url = "https://hooks.slack.com/services/xxx"
channel = "#job-search"
notify_on = ["new_conversation", "stale", "interview"]
```

**Features**:
- Daily digest to channel
- Slash command: `/jobsearch status`
- Interactive buttons for quick actions

### Discord

**Use Case**: Personal server notifications

```toml
[integrations.discord]
webhook_url = "https://discord.com/api/webhooks/xxx"
```

### Telegram

**Use Case**: Mobile notifications

```toml
[integrations.telegram]
bot_token = "xxx"
chat_id = "123456"
```

## Productivity Integrations

### Notion

See [notion-export.md](notion-export.md)

### Airtable

**Use Case**: Spreadsheet-like tracking with views

```toml
[integrations.airtable]
api_key = ""  # From env
base_id = "appXXX"
table_name = "Job Applications"
```

### Trello

**Use Case**: Kanban board for visual pipeline

```toml
[integrations.trello]
api_key = ""
board_id = "xxx"
lists = {
    waiting_on_me = "To Do",
    waiting_on_them = "Waiting",
    stale = "Follow Up",
    closed = "Done"
}
```

### Todoist / Things

**Use Case**: Task management integration

Create tasks for:
- Follow-up reminders
- Interview preparation
- Application deadlines

## Calendar Integrations

See [calendar-integration.md](calendar-integration.md)

### Supported Calendars

- Google Calendar
- Apple Calendar (CalDAV)
- Outlook Calendar
- ICS file export

## Analytics Integrations

### Google Sheets

**Use Case**: Custom analytics and reporting

```toml
[integrations.sheets]
spreadsheet_id = "xxx"
sheet_name = "Job Search Data"
```

Export data for:
- Response rate tracking
- Time-to-response analysis
- Company comparison

### CSV Export

**Use Case**: Import into any tool

```bash
jobsearch export --format=csv --output=jobsearch.csv
```

## Automation Platforms

### n8n

See `workflows/` directory for templates.

### Zapier

**Trigger**: Webhook on new conversation
**Actions**:
- Add to Notion
- Send Slack message
- Create calendar event

### IFTTT

**Applets**:
- New conversation → Add to iOS Reminders
- Stale conversation → Send push notification

## Developer Integrations

### REST API

For custom integrations:

```bash
# Start API server
jobsearch api --port=8080
```

Endpoints:
- `GET /api/conversations`
- `GET /api/conversations/:id`
- `GET /api/stats`
- `POST /api/sync`

### GraphQL

Future consideration for flexible querying:

```graphql
query {
  conversations(status: WAITING_ON_ME) {
    company
    recruiter { name email }
    emails { subject date }
  }
}
```

## AI Integrations

### OpenAI / Claude

Already integrated for classification. Future:
- Draft follow-up emails
- Summarize conversation history
- Suggest talking points for interviews

### Perplexity / Search

Research companies before interviews:

```bash
jobsearch research stripe
# Fetches company info, recent news, interview tips
```

## CRM Integrations

### HubSpot / Salesforce

For users tracking job search professionally:
- Sync conversations as deals
- Track pipeline stages
- Generate reports

## Implementation Priority

| Integration | Effort | Value | Priority |
|-------------|--------|-------|----------|
| Slack webhook | Low | High | P1 |
| CSV export | Low | Medium | P1 |
| Google Calendar | Medium | High | P1 |
| Notion | Medium | High | P2 |
| REST API | Medium | High | P2 |
| Trello | Medium | Medium | P3 |
| Zapier | Low | Medium | P3 |

## Generic Integration Framework

Design for extensibility:

```go
// internal/integrations/integration.go
type Integration interface {
    Name() string
    Configure(config map[string]string) error
    OnConversationNew(ctx context.Context, conv *Conversation) error
    OnConversationUpdate(ctx context.Context, conv *Conversation) error
    OnSyncComplete(ctx context.Context, result *SyncResult) error
}

// Registry
type Registry struct {
    integrations map[string]Integration
}

func (r *Registry) Register(i Integration) {
    r.integrations[i.Name()] = i
}

func (r *Registry) NotifyAll(ctx context.Context, event Event) {
    for _, i := range r.integrations {
        switch event.Type {
        case "conversation.new":
            i.OnConversationNew(ctx, event.Data.(*Conversation))
        // ...
        }
    }
}
```

## Contributing an Integration

1. Implement the `Integration` interface
2. Add configuration to `config.toml` spec
3. Register in the integration registry
4. Add documentation
5. Submit PR with tests
