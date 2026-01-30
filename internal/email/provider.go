package email

import (
	"context"
	"time"
)

// Provider defines the interface for email providers
type Provider interface {
	// Name returns the provider identifier
	Name() string

	// Authenticate performs OAuth or credential validation
	Authenticate(ctx context.Context) error

	// IsAuthenticated checks if valid credentials exist
	IsAuthenticated() bool

	// FetchEmails retrieves emails matching criteria
	FetchEmails(ctx context.Context, opts FetchOptions) ([]Email, error)

	// GetEmail retrieves a single email by ID
	GetEmail(ctx context.Context, id string) (*Email, error)

	// GetUserEmail returns the authenticated user's email address
	GetUserEmail(ctx context.Context) (string, error)
}

// FetchOptions configures email fetching
type FetchOptions struct {
	MaxResults int        // Maximum number of emails to fetch
	After      *time.Time // Fetch emails after this date
	Query      string     // Provider-specific query string
}

// DefaultFetchOptions returns sensible defaults
func DefaultFetchOptions() FetchOptions {
	after := time.Now().AddDate(0, -1, 0) // Last 30 days
	return FetchOptions{
		MaxResults: 100,
		After:      &after,
	}
}
