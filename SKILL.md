---
name: jobsearch
description: Track your job search conversations from email. Query status, find conversations needing attention, view statistics, and manage your job search pipeline.
allowed-tools: Bash
---

# Job Search Tracker

Track recruiter conversations from your email with smart filtering and status tracking.

> **Note**: This skill maps natural language to CLI commands. It works with Claude Code, Cursor, and any AI agent that can execute shell commands. The frontmatter above is for Claude Code; other agents can ignore it and use the command mappings below.

## Prerequisites

The `jobsearch` CLI must be installed and available in PATH. From the project directory:

```bash
make install-local  # Installs to ~/bin (no sudo)

# Add ~/bin to PATH in your shell profile:
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
```

## Commands

| Query Type | Command |
|------------|---------|
| Overview | `jobsearch stats -o json` |
| Pending actions | `jobsearch list --status=waiting_on_me -o json` |
| Waiting on them | `jobsearch list --status=waiting_on_them -o json` |
| Stale conversations | `jobsearch list --status=stale -o json` |
| Recent activity | `jobsearch list --since=7d -o json` |
| Specific company | `jobsearch show <company> -o json` |
| **Full email thread** | `jobsearch thread <company> -o json` |
| Search | `jobsearch search "<query>" -o json` |
| Sync new emails | `jobsearch sync` |

## Instructions

When user asks about their job search:

### Status queries
- "how's my job search going?" → `jobsearch stats -o json`
- "give me an overview" → `jobsearch stats -o json`
- "what are my numbers?" → `jobsearch stats -o json`

### Action queries
- "what needs my attention?" → `jobsearch list --status=waiting_on_me -o json`
- "what should I respond to?" → `jobsearch list --status=waiting_on_me -o json`
- "pending conversations" → `jobsearch list --status=waiting_on_me -o json`

### Waiting queries
- "what am I waiting on?" → `jobsearch list --status=waiting_on_them -o json`
- "who hasn't responded?" → `jobsearch list --status=waiting_on_them -o json`
- "balls in their court" → `jobsearch list --status=waiting_on_them -o json`

### Follow-up queries
- "stale conversations" → `jobsearch list --status=stale -o json`
- "what should I follow up on?" → `jobsearch list --status=stale -o json`
- "who do I need to ping?" → `jobsearch list --status=stale -o json`

### Company queries
- "show me Stripe" → `jobsearch show stripe -o json`
- "what's happening with Google?" → `jobsearch show google -o json`
- "Anthropic conversation" → `jobsearch show anthropic -o json`

### Full thread queries (fetches complete email content)
- "show me the full thread with Stripe" → `jobsearch thread stripe -o json`
- "read the Anthropic conversation" → `jobsearch thread anthropic -o json`
- "what did the Google recruiter say?" → `jobsearch thread google -o json`
- "pull up the emails from Discord" → `jobsearch thread discord -o json`
- "let me see the full conversation with Meta" → `jobsearch thread meta -o json`

### Time-based queries
- "recent conversations" → `jobsearch list --since=7d -o json`
- "this week's activity" → `jobsearch list --since=7d -o json`
- "last month" → `jobsearch list --since=1m -o json`

### Search queries
- "find interviews" → `jobsearch search "interview" -o json`
- "search for onsite" → `jobsearch search "onsite" -o json`

### Sync
- "check for new emails" → `jobsearch sync`
- "sync my inbox" → `jobsearch sync`

## Output Formatting

Format the JSON output in a user-friendly way:

### For stats:
```
Job Search Overview
━━━━━━━━━━━━━━━━━━━
Total Conversations: 24
├── Waiting on me: 3
├── Waiting on them: 8
├── Stale (need follow-up): 5
└── Closed: 8
```

### For conversations:
```
Conversations Needing Action
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Stripe** - Sarah Chen (sarah@stripe.com)
  Last contact: 2 days ago
  Status: Waiting on me

**Anthropic** - Mike Johnson (mike@anthropic.com)
  Last contact: 1 day ago
  Status: Waiting on me
```

### For company details:
```
Stripe
━━━━━━
Recruiter: Sarah Chen (sarah@stripe.com)
Position: Senior Software Engineer
Status: Waiting on me
Emails: 5
Last Activity: Jan 28, 2025

Timeline:
• Jan 20 - Initial outreach from recruiter
• Jan 22 - You responded with interest
• Jan 25 - Phone screen scheduled
• Jan 28 - They sent interview prep materials
```

### For full email thread:
```
Thread: Stripe
━━━━━━━━━━━━━━
Recruiter: Sarah Chen (sarah@stripe.com)
Position: Senior Software Engineer
Status: Waiting on me

[1/3] RECEIVED - Mon, Jan 20 2025 10:30 AM
From: sarah@stripe.com (Sarah Chen)
Subject: Exciting opportunity at Stripe

Hi! I came across your profile and was impressed by your background
in distributed systems. We have an exciting Senior Engineer role...

---

[2/3] SENT - Wed, Jan 22 2025 2:15 PM
From: you@email.com
Subject: Re: Exciting opportunity at Stripe

Hi Sarah, thank you for reaching out! I'd love to learn more
about the role...

---

[3/3] RECEIVED - Sat, Jan 25 2025 9:00 AM
From: sarah@stripe.com (Sarah Chen)
Subject: Re: Exciting opportunity at Stripe

Great to hear back from you! Let's schedule a call...
```
