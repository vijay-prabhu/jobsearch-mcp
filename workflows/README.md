# n8n Workflow Templates

Pre-built automation workflows for jobsearch-mcp that can be imported into [n8n](https://n8n.io/).

## Available Workflows

### 1. Daily Digest (`daily-digest.json`)
Sends a daily email summary of your job search status including:
- Conversations waiting on your response
- Stale conversations needing follow-up
- Overall statistics

**Trigger:** Scheduled (daily at 9 AM)

### 2. Stale Alert (`stale-alert.json`)
Sends an immediate notification when a conversation becomes stale (no activity for 7+ days).

**Trigger:** Scheduled (every 6 hours)

### 3. Weekly Report (`weekly-report.json`)
Generates a comprehensive weekly report with:
- New conversations this week
- Response rate metrics
- Company breakdown

**Trigger:** Scheduled (weekly on Monday)

## Setup Instructions

### Prerequisites
1. Install n8n: `npm install -g n8n` or use Docker
2. Have jobsearch-mcp installed and configured
3. SMTP credentials for email notifications (or use another notification node)

### Import Workflow
1. Open n8n (usually at http://localhost:5678)
2. Click "Add workflow" → "Import from file"
3. Select the desired workflow JSON file
4. Configure the following:
   - Update the jobsearch binary path in Execute Command nodes
   - Configure email credentials (or replace with Slack/Discord nodes)
   - Adjust schedule timing if needed

### Configuration

Each workflow uses the "Execute Command" node to run jobsearch CLI commands. Update the command path:

```
/path/to/jobsearch list --status=waiting_on_me --output=json
```

Replace `/path/to/jobsearch` with your actual binary location.

## Customization

### Change Notification Method
The templates use email by default. You can replace the email node with:
- Slack
- Discord
- Telegram
- Microsoft Teams
- Any other n8n notification node

### Adjust Schedule
Modify the Schedule Trigger node to change when workflows run.

### Add Filters
Add additional Execute Command nodes or Code nodes to filter/transform data.
