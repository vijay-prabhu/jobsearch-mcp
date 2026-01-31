# Outlook Email Provider

## Overview

Add Microsoft Outlook/Office 365 support via Microsoft Graph API, enabling users with Microsoft accounts to use jobsearch-mcp.

## Motivation

- Many enterprise users have Outlook/O365
- Expand user base beyond Gmail
- Support hybrid environments

## Microsoft Graph API

### Required Permissions

```
Mail.Read        # Read user mail
User.Read        # Get user profile (email address)
offline_access   # Refresh tokens
```

### Authentication Flow

1. Register app in Azure AD
2. OAuth 2.0 authorization code flow
3. Store refresh token locally

## Proposed Implementation

### Provider Interface

```go
// internal/email/outlook/provider.go
type OutlookProvider struct {
    client    *msgraph.Client
    userEmail string
    credPath  string
    tokenPath string
}

func (o *OutlookProvider) Name() string { return "outlook" }
func (o *OutlookProvider) Authenticate(ctx context.Context) error
func (o *OutlookProvider) FetchEmails(ctx context.Context, opts FetchOptions) ([]Email, error)
func (o *OutlookProvider) GetEmail(ctx context.Context, id string) (*Email, error)
func (o *OutlookProvider) GetUserEmail(ctx context.Context) (string, error)
```

### Configuration

```toml
[outlook]
credentials_path = "~/.config/jobsearch/outlook_credentials.json"
token_path = "~/.config/jobsearch/outlook_token.json"
max_results = 100

# Provider selection
[email]
provider = "outlook"  # or "gmail" (default)
```

### API Mapping

| Our Concept | Graph API |
|-------------|-----------|
| Thread ID | `conversationId` |
| Email ID | `id` |
| Labels | `categories` |
| Snippet | `bodyPreview` |

### Key Differences from Gmail

1. **Threading** - Outlook uses `conversationId` (similar concept)
2. **Search** - OData query syntax vs Gmail search syntax
3. **Incremental sync** - Delta queries instead of history ID

## File Structure

```
internal/email/outlook/
├── provider.go    # Provider implementation
├── auth.go        # OAuth flow
├── fetch.go       # Email fetching
└── convert.go     # Convert to common types
```

## Setup Instructions (for users)

1. Go to Azure Portal → App Registrations
2. Create new registration
3. Add redirect URI: `http://localhost:8642/callback`
4. Add Mail.Read, User.Read permissions
5. Create client secret
6. Download credentials JSON
7. Run `jobsearch config init --provider=outlook`

## Dependencies

```go
require (
    github.com/microsoftgraph/msgraph-sdk-go v1.0.0
    github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.0.0
)
```

## Challenges

1. **Azure AD setup** - More complex than Google Cloud
2. **Different threading model** - Need mapping layer
3. **Rate limits** - Graph API has different limits
4. **Testing** - Need O365 test account

## Effort Estimate

- OAuth implementation: 2 days
- Email fetching: 2 days
- Type conversion: 1 day
- Testing & docs: 2 days
- **Total: ~7 days**
