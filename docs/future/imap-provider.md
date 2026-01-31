# IMAP Email Provider

## Overview

Add generic IMAP support for any email provider that supports IMAP, as a fallback for users who can't use Gmail or Outlook APIs.

## Motivation

- Support any email provider (ProtonMail, Fastmail, self-hosted)
- No API registration required
- Privacy-focused users may prefer IMAP
- Works with email forwarding setups

## IMAP vs API Trade-offs

| Aspect | IMAP | API (Gmail/Outlook) |
|--------|------|---------------------|
| Setup | Username/password or app password | OAuth registration |
| Threading | Manual (References header) | Native support |
| Performance | Slower (sync entire folders) | Faster (incremental) |
| Features | Basic | Rich (labels, categories) |
| Security | App passwords | OAuth tokens |

## Proposed Implementation

### Configuration

```toml
[imap]
server = "imap.example.com"
port = 993
username = "user@example.com"
# Password from environment: JOBSEARCH_IMAP_PASSWORD
use_tls = true
folder = "INBOX"  # Folder to monitor

[email]
provider = "imap"
```

### Provider Interface

```go
// internal/email/imap/provider.go
type IMAPProvider struct {
    config   IMAPConfig
    client   *client.Client
    userEmail string
}

func (i *IMAPProvider) Name() string { return "imap" }
func (i *IMAPProvider) Authenticate(ctx context.Context) error
func (i *IMAPProvider) FetchEmails(ctx context.Context, opts FetchOptions) ([]Email, error)
func (i *IMAPProvider) GetEmail(ctx context.Context, id string) (*Email, error)
func (i *IMAPProvider) GetUserEmail(ctx context.Context) (string, error)
```

### Threading via References Header

IMAP doesn't have native threading. Use email headers:

```go
func (i *IMAPProvider) extractThreadID(msg *imap.Message) string {
    // Try References header first
    refs := msg.Envelope.InReplyTo
    if refs != "" {
        return hashReferences(refs)
    }

    // Fall back to Message-ID
    return msg.Envelope.MessageId
}
```

### Incremental Sync

Track sync state by:
1. Store last seen UID per folder
2. Use IMAP SEARCH to find new messages
3. Only fetch messages with UID > last_seen

```go
func (i *IMAPProvider) fetchNewEmails(ctx context.Context, lastUID uint32) ([]Email, error) {
    criteria := imap.NewSearchCriteria()
    criteria.Uid = new(imap.SeqSet)
    criteria.Uid.AddRange(lastUID+1, 0) // 0 = * (all)

    uids, _ := i.client.Search(criteria)
    // Fetch messages by UID...
}
```

## File Structure

```
internal/email/imap/
├── provider.go    # Provider implementation
├── auth.go        # Connection/auth
├── fetch.go       # Email fetching
├── threading.go   # Thread ID extraction
└── convert.go     # Convert to common types
```

## Dependencies

```go
require (
    github.com/emersion/go-imap v1.2.1
    github.com/emersion/go-message v0.16.0
)
```

## Challenges

1. **Threading accuracy** - References header not always reliable
2. **Performance** - Full message fetch is slow
3. **TLS variations** - Different servers have different requirements
4. **Password security** - Storing credentials safely

## Security Recommendations

1. Use app-specific passwords when available
2. Store password in system keychain if possible
3. Support environment variable for CI/automation
4. Never store plaintext in config file

## Provider-Specific Notes

### ProtonMail
- Requires ProtonMail Bridge running locally
- IMAP on localhost:1143

### Fastmail
- App passwords: Settings → Password & Security
- IMAP: imap.fastmail.com:993

### Self-hosted (Dovecot)
- Standard IMAP, varies by setup

## Effort Estimate

- IMAP connection: 1 day
- Email fetching: 2 days
- Threading logic: 2 days
- Provider variations: 1 day
- Testing: 2 days
- **Total: ~8 days**
