# Webhooks

## Overview

Add outbound webhook notifications for job search events, enabling integration with external services like Slack, Discord, or custom automation.

## Motivation

- Real-time notifications without polling
- Integration with team communication tools
- Trigger external workflows
- Mobile notifications via services like Pushover

## Proposed Events

| Event | Trigger | Payload |
|-------|---------|---------|
| `conversation.new` | New conversation created | Conversation details |
| `conversation.stale` | Conversation became stale | Conversation + days |
| `conversation.waiting` | Status changed to waiting_on_me | Conversation |
| `sync.complete` | Sync finished | Stats summary |
| `email.new` | New job-related email | Email summary |

## Configuration

```toml
[webhooks]
enabled = true

[[webhooks.endpoints]]
url = "https://hooks.slack.com/services/xxx"
events = ["conversation.new", "conversation.stale"]
secret = "webhook-secret-for-signing"

[[webhooks.endpoints]]
url = "https://discord.com/api/webhooks/xxx"
events = ["*"]  # All events
format = "discord"  # Discord-specific formatting
```

## Payload Format

### Standard JSON

```json
{
  "event": "conversation.new",
  "timestamp": "2025-01-30T10:00:00Z",
  "data": {
    "id": "abc-123",
    "company": "Stripe",
    "recruiter_name": "Sarah Chen",
    "recruiter_email": "sarah@stripe.com",
    "status": "waiting_on_me"
  }
}
```

### Slack Format

```json
{
  "text": "New conversation with Stripe",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*New Job Conversation*\n*Company:* Stripe\n*Recruiter:* Sarah Chen"
      }
    }
  ]
}
```

### Discord Format

```json
{
  "embeds": [{
    "title": "New Job Conversation",
    "color": 5814783,
    "fields": [
      {"name": "Company", "value": "Stripe", "inline": true},
      {"name": "Recruiter", "value": "Sarah Chen", "inline": true}
    ]
  }]
}
```

## Implementation

### Webhook Manager

```go
// internal/webhook/manager.go
type Manager struct {
    endpoints []Endpoint
    client    *http.Client
}

func (m *Manager) Send(ctx context.Context, event Event) error {
    for _, ep := range m.endpoints {
        if ep.Matches(event.Type) {
            payload := ep.Format(event)
            go m.deliver(ctx, ep, payload)
        }
    }
    return nil
}
```

### Signature Verification

For security, sign payloads with HMAC:

```go
func (m *Manager) sign(payload []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    return hex.EncodeToString(mac.Sum(nil))
}

// Header: X-Webhook-Signature: sha256=<signature>
```

### Retry Logic

```go
func (m *Manager) deliver(ctx context.Context, ep Endpoint, payload []byte) {
    for attempt := 0; attempt < 3; attempt++ {
        resp, err := m.client.Post(ep.URL, "application/json", bytes.NewReader(payload))
        if err == nil && resp.StatusCode < 300 {
            return
        }
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }
    log.Printf("webhook delivery failed: %s", ep.URL)
}
```

## Integration Points

### In Tracker

```go
func (t *Tracker) processEmail(ctx context.Context, pe *processedEmail) (bool, error) {
    // ... existing logic ...

    if isNew {
        t.webhooks.Send(ctx, Event{
            Type: "conversation.new",
            Data: conv,
        })
    }

    return isNew, nil
}
```

### In Status Updater

```go
if newStatus == StatusStale && oldStatus != StatusStale {
    t.webhooks.Send(ctx, Event{
        Type: "conversation.stale",
        Data: conv,
    })
}
```

## CLI Commands

```bash
# Test webhook endpoint
jobsearch webhooks test https://hooks.slack.com/xxx

# List configured webhooks
jobsearch webhooks list

# View delivery history
jobsearch webhooks history
```

## File Structure

```
internal/webhook/
├── manager.go     # Webhook manager
├── events.go      # Event types
├── formatters.go  # Slack, Discord formatters
└── delivery.go    # HTTP delivery with retry
```

## Effort Estimate

- Core webhook delivery: 1 day
- Formatters (Slack, Discord): 1 day
- Retry logic: 0.5 day
- CLI commands: 0.5 day
- Integration: 1 day
- **Total: ~4 days**
