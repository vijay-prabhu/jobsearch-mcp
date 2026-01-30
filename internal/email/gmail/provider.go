package gmail

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// Provider implements the email.Provider interface for Gmail
type Provider struct {
	credPath  string
	tokenPath string
	service   *gmail.Service
	userEmail string
}

// New creates a new Gmail provider
func New(credPath, tokenPath string) *Provider {
	return &Provider{
		credPath:  credPath,
		tokenPath: tokenPath,
	}
}

// Name returns the provider identifier
func (p *Provider) Name() string {
	return "gmail"
}

// IsAuthenticated checks if valid token exists
func (p *Provider) IsAuthenticated() bool {
	_, err := loadToken(p.tokenPath)
	return err == nil
}

// Authenticate performs OAuth authentication
func (p *Provider) Authenticate(ctx context.Context) error {
	config, err := loadCredentials(p.credPath)
	if err != nil {
		return err
	}

	client, err := getClient(ctx, config, p.tokenPath)
	if err != nil {
		return fmt.Errorf("failed to get OAuth client: %w", err)
	}

	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	p.service = service

	// Get and cache user email
	profile, err := service.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	p.userEmail = profile.EmailAddress
	return nil
}

// GetUserEmail returns the authenticated user's email address
func (p *Provider) GetUserEmail(ctx context.Context) (string, error) {
	if p.userEmail == "" {
		return "", fmt.Errorf("not authenticated")
	}
	return p.userEmail, nil
}

// FetchEmails retrieves emails matching criteria
func (p *Provider) FetchEmails(ctx context.Context, opts email.FetchOptions) ([]email.Email, error) {
	if p.service == nil {
		return nil, fmt.Errorf("not authenticated - call Authenticate() first")
	}

	// Build query
	query := buildQuery(opts)

	// Fetch message list
	var emails []email.Email
	pageToken := ""

	for {
		req := p.service.Users.Messages.List("me").
			Q(query).
			MaxResults(int64(min(opts.MaxResults-len(emails), 100)))

		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, err := req.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list messages: %w", err)
		}

		// Fetch full details for each message
		for _, msg := range resp.Messages {
			fullMsg, err := p.service.Users.Messages.Get("me", msg.Id).
				Format("full").
				Context(ctx).
				Do()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch message %s: %v\n", msg.Id, err)
				continue
			}

			emails = append(emails, convertMessage(fullMsg))

			if len(emails) >= opts.MaxResults {
				return emails, nil
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" || len(emails) >= opts.MaxResults {
			break
		}
	}

	return emails, nil
}

// GetEmail retrieves a single email by ID
func (p *Provider) GetEmail(ctx context.Context, id string) (*email.Email, error) {
	if p.service == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	msg, err := p.service.Users.Messages.Get("me", id).
		Format("full").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	result := convertMessage(msg)
	return &result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
