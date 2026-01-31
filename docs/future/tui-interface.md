# TUI Interface

## Overview

Add an interactive terminal user interface (TUI) for browsing conversations, viewing details, and managing status without remembering CLI commands.

## Motivation

- More intuitive than memorizing CLI commands
- Quick navigation between conversations
- Real-time status updates
- Better for users who prefer visual interfaces

## Proposed Technology

**[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Go framework for terminal apps

Supporting libraries:
- `bubbles` - Pre-built components (lists, text inputs, spinners)
- `lipgloss` - Styling and layout

## Proposed Features

### Main Dashboard
```
┌─ JobSearch TUI ──────────────────────────────────────┐
│                                                       │
│  Stats: 23 total │ 3 waiting │ 8 pending │ 2 stale  │
│                                                       │
│  ┌─ Conversations ─────────────────────────────────┐ │
│  │ > Stripe        Sarah Chen      waiting_on_me   │ │
│  │   Datadog       Mike Johnson    waiting_on_them │ │
│  │   Okta          Lisa Park       waiting_on_me   │ │
│  │   Google        James Wu        stale           │ │
│  └─────────────────────────────────────────────────┘ │
│                                                       │
│  [Enter] View  [s] Sync  [f] Filter  [/] Search  [q] Quit │
└───────────────────────────────────────────────────────┘
```

### Conversation Detail View
```
┌─ Stripe ─────────────────────────────────────────────┐
│  Recruiter: Sarah Chen <sarah@stripe.com>            │
│  Position:  Senior Backend Engineer                  │
│  Status:    waiting_on_me (5 days)                   │
│                                                       │
│  ┌─ Email Timeline ────────────────────────────────┐ │
│  │ Jan 20  [IN]  Initial outreach                  │ │
│  │ Jan 21  [OUT] My reply                          │ │
│  │ Jan 23  [IN]  Interview scheduling              │ │
│  │ Jan 25  [IN]  Follow-up ← needs response        │ │
│  └─────────────────────────────────────────────────┘ │
│                                                       │
│  [c] Close  [m] Mark status  [Esc] Back              │
└───────────────────────────────────────────────────────┘
```

### Key Bindings
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate list |
| `Enter` | View conversation |
| `s` | Trigger sync |
| `f` | Filter by status |
| `/` | Search |
| `m` | Change status |
| `c` | Close conversation |
| `?` | Help |
| `q` | Quit |

## Implementation Plan

### Phase 1: Basic Dashboard
1. Create main model with list of conversations
2. Implement navigation
3. Add status filtering
4. Style with lipgloss

### Phase 2: Detail View
1. Conversation detail model
2. Email timeline display
3. Status management

### Phase 3: Actions
1. Sync trigger with progress
2. Search functionality
3. Keyboard shortcuts help

## File Structure

```
internal/tui/
├── tui.go           # Main entry point
├── model.go         # Bubble Tea model
├── views/
│   ├── dashboard.go # Main dashboard view
│   ├── detail.go    # Conversation detail
│   └── help.go      # Help overlay
├── components/
│   ├── list.go      # Conversation list
│   ├── timeline.go  # Email timeline
│   └── statusbar.go # Status bar
└── styles/
    └── styles.go    # Lipgloss styles
```

## CLI Integration

```bash
# Launch TUI
jobsearch tui

# Or make it the default for interactive terminals
jobsearch  # Launches TUI if no subcommand
```

## Dependencies to Add

```go
require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/charmbracelet/bubbles v0.18.0
    github.com/charmbracelet/lipgloss v0.9.1
)
```

## Effort Estimate

- Basic dashboard: 2-3 days
- Detail view: 1-2 days
- Polish and testing: 1-2 days
- **Total: ~1 week**
