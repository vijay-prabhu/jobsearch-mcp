package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/classifier"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/filter"
)

// Tracker orchestrates the email sync and tracking pipeline
type Tracker struct {
	db         *database.DB
	provider   email.Provider
	filter     *filter.Filter
	classifier *classifier.Client
	config     *config.Config
	userEmail  string
}

// New creates a new Tracker
func New(db *database.DB, provider email.Provider, f *filter.Filter, c *classifier.Client, cfg *config.Config) *Tracker {
	return &Tracker{
		db:         db,
		provider:   provider,
		filter:     f,
		classifier: c,
		config:     cfg,
	}
}

// SyncResult contains the results of a sync operation
type SyncResult struct {
	EmailsFetched        int
	EmailsFiltered       int
	EmailsClassified     int
	ConversationsNew     int
	ConversationsUpdated int
	Errors               []error
}

// Sync fetches new emails and processes them
func (t *Tracker) Sync(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{}

	// Get user email for direction detection
	userEmail, err := t.provider.GetUserEmail(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}
	t.userEmail = userEmail

	// Get sync state
	syncState, err := t.db.GetSyncState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync state: %w", err)
	}

	// Determine date range
	opts := email.DefaultFetchOptions()
	opts.MaxResults = t.config.Gmail.MaxResults

	if syncState.LastSyncAt != nil {
		// Incremental sync - fetch since last sync
		opts.After = syncState.LastSyncAt
	}

	// Fetch emails
	emails, err := t.provider.FetchEmails(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}
	result.EmailsFetched = len(emails)

	if len(emails) == 0 {
		// Update sync state even if no new emails
		now := time.Now()
		syncState.LastSyncAt = &now
		t.db.UpdateSyncState(ctx, syncState)
		return result, nil
	}

	// Apply filtering
	filtered := t.filter.ApplyBatch(emails)
	included := filter.FilterIncluded(filtered)
	uncertain := filter.FilterUncertain(filtered)

	result.EmailsFiltered = len(included)

	// Classify uncertain emails with LLM
	if len(uncertain) > 0 && t.classifier != nil && t.classifier.IsRunning(ctx) {
		for i := range uncertain {
			e := &uncertain[i]
			classification, err := t.classifier.ClassifyWithFallback(
				ctx,
				classifier.ClassifyRequest{
					EmailSubject: e.Email.Subject,
					EmailBody:    e.Email.Body,
					EmailFrom:    e.Email.From.Email,
				},
				t.config.LLM.Primary,
				t.config.LLM.Fallback,
			)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("classification failed: %w", err))
				continue
			}

			result.EmailsClassified++

			if classification.IsJobRelated {
				e.Result.Include = true
				e.Result.Layer = filter.LayerLLM
				e.Result.Confidence = classification.Confidence
				included = append(included, *e)
			}
		}
	}

	// Process included emails
	for _, fe := range included {
		newConv, err := t.processEmail(ctx, &fe)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		if newConv {
			result.ConversationsNew++
		} else {
			result.ConversationsUpdated++
		}
	}

	// Update sync state
	now := time.Now()
	syncState.LastSyncAt = &now
	syncState.EmailsProcessed += len(emails)
	if err := t.db.UpdateSyncState(ctx, syncState); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to update sync state: %w", err))
	}

	// Update conversation statuses
	if err := t.updateAllStatuses(ctx); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to update statuses: %w", err))
	}

	return result, nil
}

// processEmail processes a single filtered email
func (t *Tracker) processEmail(ctx context.Context, fe *filter.FilteredEmail) (bool, error) {
	// Check if email already exists
	existing, err := t.db.GetEmailByGmailID(ctx, fe.Email.ID)
	if err != nil {
		return false, err
	}
	if existing != nil {
		return false, nil // Already processed
	}

	// Find or create conversation
	conv, isNew, err := t.findOrCreateConversation(ctx, &fe.Email)
	if err != nil {
		return false, err
	}

	// Determine direction
	direction := database.DirectionInbound
	if fe.Email.IsFromMe(t.userEmail) {
		direction = database.DirectionOutbound
	}

	// Create email record
	subject := fe.Email.Subject
	fromName := fe.Email.From.Name
	snippet := fe.Email.Snippet
	layer := string(fe.Result.Layer)
	confidence := fe.Result.Confidence

	dbEmail := &database.Email{
		ConversationID: conv.ID,
		GmailID:        fe.Email.ID,
		ThreadID:       fe.Email.ThreadID,
		Subject:        &subject,
		FromAddress:    fe.Email.From.Email,
		FromName:       &fromName,
		Date:           fe.Email.Date,
		Direction:      direction,
		Snippet:        &snippet,
		Classification: &layer,
		Confidence:     &confidence,
	}

	if err := t.db.CreateEmail(ctx, dbEmail); err != nil {
		return false, err
	}

	// Update conversation
	if err := t.db.IncrementEmailCount(ctx, conv.ID); err != nil {
		return false, err
	}

	// Update last activity
	if fe.Email.Date.After(conv.LastActivityAt) {
		conv.LastActivityAt = fe.Email.Date
		if err := t.db.UpdateConversation(ctx, conv); err != nil {
			return false, err
		}
	}

	return isNew, nil
}

// findOrCreateConversation finds an existing conversation or creates a new one
func (t *Tracker) findOrCreateConversation(ctx context.Context, e *email.Email) (*database.Conversation, bool, error) {
	// First, try to find by thread ID
	conv, err := t.db.GetConversationByThreadID(ctx, e.ThreadID)
	if err != nil {
		return nil, false, err
	}
	if conv != nil {
		return conv, false, nil
	}

	// Create new conversation
	direction := database.DirectionInbound
	if e.IsFromMe(t.userEmail) {
		direction = database.DirectionOutbound
	}

	// Extract company from domain
	company := filter.ExtractCompanyFromDomain(e.Domain())
	if company == "" {
		company = e.Domain() // Use domain as fallback
	}

	recruiterEmail := e.From.Email
	recruiterName := e.From.Name

	conv = &database.Conversation{
		Company:        company,
		RecruiterEmail: &recruiterEmail,
		RecruiterName:  &recruiterName,
		Direction:      direction,
		Status:         database.StatusActive,
		LastActivityAt: e.Date,
		EmailCount:     0,
	}

	if err := t.db.CreateConversation(ctx, conv); err != nil {
		return nil, false, err
	}

	return conv, true, nil
}

// updateAllStatuses updates the status of all active conversations
func (t *Tracker) updateAllStatuses(ctx context.Context) error {
	// Get all non-closed conversations
	convs, err := t.db.ListConversations(ctx, database.ListOptions{})
	if err != nil {
		return err
	}

	for _, conv := range convs {
		if conv.Status == database.StatusClosed {
			continue
		}

		emails, err := t.db.ListEmailsForConversation(ctx, conv.ID)
		if err != nil {
			continue
		}

		newStatus := ComputeStatus(emails, t.userEmail, t.config.Tracking.StaleAfterDays)
		if newStatus != conv.Status {
			conv.Status = newStatus
			t.db.UpdateConversation(ctx, &conv)
		}
	}

	return nil
}
