# Notion Export

## Overview

Export job search data to a Notion database for visual tracking, sharing with career coaches, or integration with existing Notion workflows.

## Motivation

- Visual kanban/table view of job search
- Share progress with mentors/coaches
- Integrate with existing Notion workflow
- Rich formatting and notes

## Notion Database Schema

### Job Applications Database

| Property | Type | Mapping |
|----------|------|---------|
| Company | Title | `conversation.company` |
| Position | Text | `conversation.position` |
| Recruiter | Text | `conversation.recruiter_name` |
| Status | Select | `conversation.status` |
| Last Contact | Date | `conversation.last_activity_at` |
| Email Count | Number | `conversation.email_count` |
| Direction | Select | `conversation.direction` |
| Notes | Text | User-added |
| Link | URL | Link to email thread |

### Status Mapping

| Our Status | Notion Status | Color |
|------------|---------------|-------|
| waiting_on_me | Action Needed | Red |
| waiting_on_them | Waiting | Yellow |
| active | In Progress | Blue |
| stale | Follow Up | Orange |
| closed | Closed | Gray |

## Implementation

### Notion API Client

```go
// internal/notion/client.go
type Client struct {
    apiKey     string
    databaseID string
    httpClient *http.Client
}

func (c *Client) CreatePage(ctx context.Context, conv *database.Conversation) error {
    page := NotionPage{
        Parent:     DatabaseParent{DatabaseID: c.databaseID},
        Properties: c.mapProperties(conv),
    }

    return c.post("/pages", page)
}

func (c *Client) UpdatePage(ctx context.Context, pageID string, conv *database.Conversation) error {
    return c.patch("/pages/"+pageID, c.mapProperties(conv))
}
```

### Property Mapping

```go
func (c *Client) mapProperties(conv *database.Conversation) map[string]Property {
    return map[string]Property{
        "Company": TitleProperty{
            Title: []RichText{{Text: TextContent{Content: conv.Company}}},
        },
        "Position": RichTextProperty{
            RichText: []RichText{{Text: TextContent{Content: ptrString(conv.Position)}}},
        },
        "Status": SelectProperty{
            Select: SelectOption{Name: mapStatus(conv.Status)},
        },
        "Last Contact": DateProperty{
            Date: DateValue{Start: conv.LastActivityAt.Format("2006-01-02")},
        },
        // ...
    }
}
```

### Sync Strategy

1. **Initial Export**: Create pages for all conversations
2. **Incremental Sync**: Track Notion page IDs in local DB
3. **Two-way Sync** (optional): Poll Notion for status changes

```sql
-- Add to schema
ALTER TABLE conversations ADD COLUMN notion_page_id TEXT;
```

## Configuration

```toml
[notion]
enabled = true
api_key = ""  # From env: NOTION_API_KEY
database_id = "abc123..."  # Target database ID

[notion.sync]
auto_sync = false  # Sync after each jobsearch sync
create_new = true  # Create pages for new conversations
update_existing = true  # Update existing pages
```

## CLI Commands

```bash
# Export all conversations to Notion
jobsearch notion export

# Sync changes (update existing, add new)
jobsearch notion sync

# Setup wizard (creates database template)
jobsearch notion setup
```

## Setup Wizard

Interactive setup to create Notion database:

```
$ jobsearch notion setup

Notion Export Setup
===================

1. Create a Notion integration at https://www.notion.so/my-integrations
2. Copy the API key and paste below:

API Key: secret_xxx...

3. Create a new database in Notion (or use existing)
4. Share the database with your integration
5. Copy the database URL and paste below:

Database URL: https://notion.so/xxx?v=yyy

Extracting database ID: abc123...

Testing connection... ✓
Verifying database schema...

Missing properties detected. Create them? [Y/n] y

Creating properties:
  ✓ Company (title)
  ✓ Position (rich_text)
  ✓ Recruiter (rich_text)
  ✓ Status (select)
  ✓ Last Contact (date)
  ✓ Email Count (number)

Setup complete! Run 'jobsearch notion export' to sync your data.
```

## Database Template

Provide a Notion template users can duplicate:

```
[Duplicate this template](https://notion.so/templates/job-search-tracker)
```

## API Limits

Notion API has rate limits:
- 3 requests per second
- Batch operations where possible

```go
func (c *Client) ExportAll(ctx context.Context, convs []Conversation) error {
    limiter := rate.NewLimiter(rate.Every(350*time.Millisecond), 1)

    for _, conv := range convs {
        limiter.Wait(ctx)
        if err := c.CreatePage(ctx, &conv); err != nil {
            return err
        }
    }
    return nil
}
```

## Dependencies

```go
require (
    github.com/jomei/notionapi v1.12.0  // Or implement directly
    golang.org/x/time/rate              // Rate limiting
)
```

## Effort Estimate

- Notion API client: 2 days
- Property mapping: 1 day
- Sync logic: 1 day
- Setup wizard: 1 day
- Testing: 1 day
- **Total: ~6 days**
