package gmail

import (
	"context"
	"fmt"
	"os"
	"sync"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// concurrentFetches is the number of parallel Gmail API calls
const concurrentFetches = 10

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

// FetchEmails retrieves emails matching criteria using parallel fetching
func (p *Provider) FetchEmails(ctx context.Context, opts email.FetchOptions) ([]email.Email, error) {
	if p.service == nil {
		return nil, fmt.Errorf("not authenticated - call Authenticate() first")
	}

	// Build query
	query := buildQuery(opts)

	// Step 1: Collect all message IDs
	var messageIDs []string
	pageToken := ""

	for {
		req := p.service.Users.Messages.List("me").
			Q(query).
			MaxResults(int64(min(opts.MaxResults-len(messageIDs), 500)))

		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, err := req.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list messages: %w", err)
		}

		for _, msg := range resp.Messages {
			messageIDs = append(messageIDs, msg.Id)
			if len(messageIDs) >= opts.MaxResults {
				break
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" || len(messageIDs) >= opts.MaxResults {
			break
		}
	}

	if len(messageIDs) == 0 {
		return nil, nil
	}

	// Step 2: Fetch messages in parallel
	return p.fetchMessagesParallel(ctx, messageIDs)
}

// fetchMessagesParallel fetches multiple messages concurrently
func (p *Provider) fetchMessagesParallel(ctx context.Context, messageIDs []string) ([]email.Email, error) {
	// Result channel and slice
	type result struct {
		index int
		email email.Email
		err   error
	}

	results := make(chan result, len(messageIDs))
	var wg sync.WaitGroup

	// Semaphore to limit concurrent requests
	sem := make(chan struct{}, concurrentFetches)

	// Launch workers
	for i, msgID := range messageIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results <- result{index: index, err: ctx.Err()}
				return
			}

			// Fetch message
			fullMsg, err := p.service.Users.Messages.Get("me", id).
				Format("full").
				Context(ctx).
				Do()
			if err != nil {
				results <- result{index: index, err: err}
				return
			}

			results <- result{index: index, email: convertMessage(fullMsg)}
		}(i, msgID)
	}

	// Close results channel when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	emails := make([]email.Email, len(messageIDs))
	var fetchErrors []error

	for r := range results {
		if r.err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("message %d: %w", r.index, r.err))
			continue
		}
		emails[r.index] = r.email
	}

	// Filter out zero-value emails (from errors)
	var validEmails []email.Email
	for _, e := range emails {
		if e.ID != "" {
			validEmails = append(validEmails, e)
		}
	}

	// Log errors if any
	if len(fetchErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: failed to fetch %d messages\n", len(fetchErrors))
	}

	return validEmails, nil
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
