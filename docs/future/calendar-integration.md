# Calendar Integration

## Overview

Automatically create calendar events for interviews and follow-up reminders detected in job search emails.

## Motivation

- Never miss an interview
- Automated follow-up reminders
- Reduce manual calendar management
- Visual timeline of job search progress

## Proposed Features

### 1. Interview Detection

Parse emails for interview scheduling:

```
Subject: "Interview with Stripe - Jan 30th 2pm"
Body: "...looking forward to meeting you on Tuesday at 2:00 PM PST..."
```

Creates:
```
Event: Interview - Stripe
When: Jan 30, 2025 2:00 PM PST
Where: [Zoom link if detected]
Description: With Sarah Chen (sarah@stripe.com)
```

### 2. Follow-up Reminders

Auto-create reminders for stale conversations:

```
Event: Follow up with Stripe
When: [7 days after last email]
Description: Conversation has been idle. Consider sending a follow-up.
```

### 3. Application Deadlines

Detect and track application deadlines:

```
Body: "...applications close on February 15th..."
```

Creates:
```
Event: Deadline - Stripe Application
When: Feb 15, 2025 (all day)
Reminder: 2 days before
```

## Calendar Providers

### Google Calendar

```toml
[calendar]
provider = "google"
credentials_path = "~/.config/jobsearch/calendar_credentials.json"
calendar_id = "primary"  # or specific calendar ID
```

### Apple Calendar (via CalDAV)

```toml
[calendar]
provider = "caldav"
url = "https://caldav.icloud.com"
username = "user@icloud.com"
# Password from keychain or env
calendar_name = "Job Search"
```

### ICS File Export

For manual import:

```toml
[calendar]
provider = "ics"
output_path = "~/jobsearch-events.ics"
```

## Implementation

### Event Extraction

```go
// internal/calendar/extractor.go
type EventExtractor struct {
    patterns []DatePattern
}

func (e *EventExtractor) Extract(email *Email) []CalendarEvent {
    var events []CalendarEvent

    // Check for interview keywords
    if containsInterviewKeywords(email.Subject, email.Body) {
        if date := e.extractDate(email.Body); date != nil {
            events = append(events, CalendarEvent{
                Type:    EventInterview,
                Title:   "Interview - " + email.Company,
                Start:   *date,
                // ...
            })
        }
    }

    return events
}
```

### Date Parsing

Use natural language date parsing:

```go
import "github.com/olebedev/when"

func (e *EventExtractor) extractDate(text string) *time.Time {
    r, _ := when.EN.Parse(text, time.Now())
    if r != nil {
        return &r.Time
    }
    return nil
}
```

### Google Calendar API

```go
// internal/calendar/google/provider.go
func (g *GoogleCalendar) CreateEvent(ctx context.Context, event CalendarEvent) error {
    calEvent := &calendar.Event{
        Summary:     event.Title,
        Description: event.Description,
        Start:       &calendar.EventDateTime{DateTime: event.Start.Format(time.RFC3339)},
        End:         &calendar.EventDateTime{DateTime: event.End.Format(time.RFC3339)},
        Reminders:   &calendar.EventReminders{UseDefault: false, Overrides: [...]},
    }

    _, err := g.service.Events.Insert(g.calendarID, calEvent).Do()
    return err
}
```

## Configuration

```toml
[calendar]
enabled = true
provider = "google"
calendar_id = "primary"

[calendar.events]
create_interviews = true
create_followups = true
create_deadlines = true
followup_days = 7  # Create reminder after N days of inactivity

[calendar.reminders]
interview = ["1d", "1h"]  # 1 day and 1 hour before
followup = ["9am"]        # Same day at 9 AM
deadline = ["2d", "1d"]   # 2 days and 1 day before
```

## CLI Commands

```bash
# Scan emails and create events
jobsearch calendar sync

# Preview events without creating
jobsearch calendar preview

# Export to ICS file
jobsearch calendar export --output=events.ics

# List upcoming job search events
jobsearch calendar list
```

## Privacy Considerations

1. Only create events from confirmed job emails
2. Option to use separate calendar
3. Don't include sensitive email content in event descriptions
4. Support local-only ICS export

## Dependencies

```go
require (
    google.golang.org/api/calendar/v3
    github.com/olebedev/when  // Natural date parsing
    github.com/arran4/golang-ical  // ICS generation
)
```

## Effort Estimate

- Date extraction: 2 days
- Google Calendar integration: 2 days
- ICS export: 1 day
- CLI commands: 1 day
- Testing: 1 day
- **Total: ~7 days**
