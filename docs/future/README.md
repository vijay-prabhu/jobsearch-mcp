# Future Enhancements

This directory contains design documents for planned future features. These are not yet implemented but are documented to guide future development.

## Feature Roadmap

| Feature | Effort | Priority | Document |
|---------|--------|----------|----------|
| TUI Interface | Medium | Medium | [tui-interface.md](tui-interface.md) |
| HTTP Transport | Medium | Low | [http-transport.md](http-transport.md) |
| Outlook Provider | High | Low | [outlook-provider.md](outlook-provider.md) |
| IMAP Provider | High | Low | [imap-provider.md](imap-provider.md) |
| Webhooks | Medium | Low | [webhooks.md](webhooks.md) |
| Calendar Integration | Medium | Medium | [calendar-integration.md](calendar-integration.md) |
| Notion Export | Medium | Low | [notion-export.md](notion-export.md) |
| Integrations | Various | Various | [integrations.md](integrations.md) |

## Already Implemented

These features from the original plan have been implemented:

- **MCP Resources** - Read-only context data for Claude (v0.1.0)
- **n8n Workflows** - Automation templates (v0.1.0)
- **AI Keyword Learning** - Auto-suggest filters (v0.1.0)

## Contributing

If you'd like to implement one of these features:

1. Read the design document thoroughly
2. Open an issue to discuss your approach
3. Reference the ADRs in `docs/adr/` for architectural context
4. Submit a PR with tests and documentation updates
