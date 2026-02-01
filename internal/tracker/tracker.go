package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/classifier"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email/gmail"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/filter"
)

// processedEmail holds a filtered email with optional LLM classification
type processedEmail struct {
	filter.FilteredEmail
	Classification *classifier.ClassifyResponse
}

// Tracker orchestrates the email sync and tracking pipeline
type Tracker struct {
	db         *database.DB
	provider   email.Provider
	filter     *filter.Filter
	classifier *classifier.Client
	config     *config.Config
	learner    *Learner
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
		learner:    NewLearner(db),
	}
}

// SyncOptions configures the sync behavior
type SyncOptions struct {
	Days     int              // Number of days to fetch (0 = use default or last sync)
	FullSync bool             // Ignore last sync time
	Progress ProgressCallback // Optional progress callback
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

// Sync fetches new emails and processes them with default options
func (t *Tracker) Sync(ctx context.Context) (*SyncResult, error) {
	return t.SyncWithOptions(ctx, SyncOptions{})
}

// SyncWithOptions fetches new emails with custom options
func (t *Tracker) SyncWithOptions(ctx context.Context, syncOpts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Helper to report progress
	report := func(phase ProgressPhase, current, total int, desc string) {
		if syncOpts.Progress != nil {
			syncOpts.Progress(Progress{
				Phase:       phase,
				Current:     current,
				Total:       total,
				Description: desc,
			})
		}
	}

	// Get user email for direction detection
	userEmail, err := t.provider.GetUserEmail(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}
	t.userEmail = userEmail

	// Set user email on filter so it can handle outbound emails correctly
	t.filter.SetUserEmail(userEmail)

	// Load learned blacklist from database and add to filter
	learnedBlacklist, err := t.db.GetLearnedBlacklist(ctx)
	if err != nil {
		// Non-fatal: log and continue
		result.Errors = append(result.Errors, fmt.Errorf("failed to load learned blacklist: %w", err))
	} else if len(learnedBlacklist) > 0 {
		t.filter.AddLearnedFilters("domain_blacklist", learnedBlacklist)
	}

	// Get sync state
	syncState, err := t.db.GetSyncState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync state: %w", err)
	}

	// Determine date range
	opts := email.DefaultFetchOptions()
	opts.MaxResults = t.config.Gmail.MaxResults

	// Apply sync options
	if syncOpts.Days > 0 {
		// Use custom days range
		after := time.Now().AddDate(0, 0, -syncOpts.Days)
		opts.After = &after
	} else if syncOpts.FullSync {
		// Full sync - use default 30 days, ignore last sync
		// opts.After is already set by DefaultFetchOptions
	} else if syncState.LastSyncAt != nil {
		// Incremental sync - fetch since last sync
		opts.After = syncState.LastSyncAt
	}

	// Set up progress callback for email provider
	if gmailProvider, ok := t.provider.(*gmail.Provider); ok {
		gmailProvider.SetProgressCallback(func(phase string, current, total int) {
			switch phase {
			case "listing":
				report(PhaseListingEmails, current, total, "Listing emails from Gmail")
			case "fetching":
				report(PhaseFetchingEmails, current, total, "Downloading email content")
			}
		})
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
		_ = t.db.UpdateSyncState(ctx, syncState)
		return result, nil
	}

	// Apply filtering
	report(PhaseFiltering, 0, len(emails), "Applying filters")
	filtered := t.filter.ApplyBatch(emails)
	included := filter.FilterIncluded(filtered)
	uncertain := filter.FilterUncertain(filtered)
	report(PhaseFiltering, len(emails), len(emails), "Filtering complete")

	result.EmailsFiltered = len(included)

	// Build list of emails to process with their classifications
	var toProcess []processedEmail

	// Add already-included emails (from whitelist/keywords)
	for _, fe := range included {
		toProcess = append(toProcess, processedEmail{FilteredEmail: fe})
	}

	// Classify uncertain emails with LLM (in parallel batches)
	if len(uncertain) > 0 && t.classifier != nil && t.classifier.IsRunning(ctx) {
		// Build batch request
		requests := make([]classifier.ClassifyRequest, len(uncertain))
		for i, e := range uncertain {
			requests[i] = classifier.ClassifyRequest{
				EmailSubject: e.Email.Subject,
				EmailBody:    e.Email.Body,
				EmailFrom:    e.Email.From.Email,
			}
		}

		// Classify in parallel with progress reporting
		classifyProgress := func(current, total int) {
			report(PhaseClassifying, current, total, "Classifying with LLM")
		}
		batchResults := t.classifier.ClassifyBatchWithProgress(ctx, requests, t.config.LLM.Primary, t.config.LLM.Fallback, classifyProgress)

		// Process results
		for i, br := range batchResults {
			if br.Error != nil {
				result.Errors = append(result.Errors, fmt.Errorf("classification failed: %w", br.Error))
				continue
			}

			result.EmailsClassified++
			classification := br.Response

			if classification.IsJobRelated {
				e := &uncertain[i]
				e.Result.Include = true
				e.Result.Layer = filter.LayerLLM
				e.Result.Confidence = classification.Confidence
				toProcess = append(toProcess, processedEmail{
					FilteredEmail:  *e,
					Classification: classification,
				})

				// Learn from this classification
				if t.learner != nil {
					_ = t.learner.LearnFromEmail(ctx, &e.Email, classification.Confidence)
				}
			}
		}
	}

	// Process all included emails
	totalToProcess := len(toProcess)
	for i, pe := range toProcess {
		report(PhaseProcessing, i+1, totalToProcess, "Processing emails into conversations")
		newConv, err := t.processEmail(ctx, &pe)
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
	report(PhaseUpdatingStatus, 0, 0, "Updating conversation statuses")
	if err := t.updateAllStatuses(ctx); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to update statuses: %w", err))
	}

	return result, nil
}

// processEmail processes a single filtered email with optional classification
func (t *Tracker) processEmail(ctx context.Context, pe *processedEmail) (bool, error) {
	fe := &pe.FilteredEmail

	// Check if email already exists
	existing, err := t.db.GetEmailByGmailID(ctx, fe.Email.ID)
	if err != nil {
		return false, err
	}
	if existing != nil {
		return false, nil // Already processed
	}

	// Find or create conversation
	conv, isNew, err := t.findOrCreateConversation(ctx, &fe.Email, pe.Classification)
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

	// Store extracted data from LLM if available
	if pe.Classification != nil {
		extractedData := map[string]interface{}{
			"company":        pe.Classification.Company,
			"position":       pe.Classification.Position,
			"recruiter_name": pe.Classification.RecruiterName,
			"classification": pe.Classification.Classification,
			"reasoning":      pe.Classification.Reasoning,
		}
		if jsonData, err := json.Marshal(extractedData); err == nil {
			jsonStr := string(jsonData)
			dbEmail.ExtractedData = &jsonStr
		}
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
func (t *Tracker) findOrCreateConversation(ctx context.Context, e *email.Email, classification *classifier.ClassifyResponse) (*database.Conversation, bool, error) {
	// First, try to find by thread ID (exact thread match)
	conv, err := t.db.GetConversationByThreadID(ctx, e.ThreadID)
	if err != nil {
		return nil, false, err
	}
	if conv != nil {
		return conv, false, nil
	}

	// Determine recruiter email for smart grouping
	groupByEmail := e.From.Email
	if e.IsFromMe(t.userEmail) {
		// For outbound emails, try to find recruiter from To address
		if len(e.To) > 0 {
			groupByEmail = e.To[0].Email
		}
	}

	// Smart grouping: try to find existing conversation with same recruiter email
	conv, err = t.db.GetConversationByRecruiterEmail(ctx, groupByEmail)
	if err != nil {
		return nil, false, err
	}
	if conv != nil {
		// Found existing conversation with same recruiter - add email to it
		return conv, false, nil
	}

	// Create new conversation
	direction := database.DirectionInbound
	isOutbound := e.IsFromMe(t.userEmail)
	if isOutbound {
		direction = database.DirectionOutbound
	}

	// Determine company name
	company := t.extractCompanyName(e, classification)

	// For outbound emails, use the recipient as the recruiter
	var recruiterEmail, recruiterName string
	if isOutbound && len(e.To) > 0 {
		recruiterEmail = e.To[0].Email
		recruiterName = e.To[0].Name
	} else {
		recruiterEmail = e.From.Email
		recruiterName = e.From.Name
	}

	// If LLM extracted a recruiter name, use it
	if classification != nil && classification.RecruiterName != nil && *classification.RecruiterName != "" {
		recruiterName = *classification.RecruiterName
	}

	var position *string
	if classification != nil && classification.Position != nil && *classification.Position != "" {
		position = classification.Position
	}

	conv = &database.Conversation{
		Company:        company,
		Position:       position,
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

// extractCompanyName determines the company name from email and classification
func (t *Tracker) extractCompanyName(e *email.Email, classification *classifier.ClassifyResponse) string {
	// Get the relevant address (for outbound emails, use To address)
	var relevantEmail, relevantName, relevantDomain string
	if e.IsFromMe(t.userEmail) && len(e.To) > 0 {
		relevantEmail = e.To[0].Email
		relevantName = e.To[0].Name
		relevantDomain = e.To[0].Domain()
	} else {
		relevantEmail = e.From.Email
		relevantName = e.From.Name
		relevantDomain = e.Domain()
	}

	// Check if this is a LinkedIn InMail
	isLinkedInInMail := strings.Contains(strings.ToLower(relevantEmail), "linkedin.com")

	// If LLM extracted a company name, prefer that
	if classification != nil && classification.Company != nil && *classification.Company != "" {
		return *classification.Company
	}

	// For LinkedIn InMails without LLM company, use recruiter name as identifier
	if isLinkedInInMail {
		if relevantName != "" {
			return relevantName + " (via LinkedIn)"
		}
		return "LinkedIn InMail"
	}

	// Extract company from domain
	company := filter.ExtractCompanyFromDomain(relevantDomain)
	if company == "" {
		company = relevantDomain
	}

	return company
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
			_ = t.db.UpdateConversation(ctx, &conv)
		}
	}

	return nil
}

// MarkFalsePositive marks a conversation as incorrectly included (learns from mistake)
func (t *Tracker) MarkFalsePositive(ctx context.Context, convID string) error {
	conv, err := t.db.GetConversation(ctx, convID)
	if err != nil || conv == nil {
		return fmt.Errorf("conversation not found: %s", convID)
	}

	// Get first email to learn from its domain
	emails, err := t.db.ListEmailsForConversation(ctx, conv.ID)
	if err != nil || len(emails) == 0 {
		return fmt.Errorf("no emails found for conversation")
	}

	// Create email.Email from database.Email for the learner
	e := &email.Email{
		From: email.Address{Email: emails[0].FromAddress},
	}
	if emails[0].Subject != nil {
		e.Subject = *emails[0].Subject
	}

	// Learn from feedback
	if t.learner != nil {
		if err := t.learner.LearnFromFeedback(ctx, e, true); err != nil {
			return err
		}
	}

	// Mark conversation as closed
	conv.Status = database.StatusClosed
	return t.db.UpdateConversation(ctx, conv)
}

// MarkFalseNegative records that an email was incorrectly excluded (for learning)
func (t *Tracker) MarkFalseNegative(ctx context.Context, fromEmail, subject string) error {
	e := &email.Email{
		From:    email.Address{Email: fromEmail},
		Subject: subject,
	}

	if t.learner != nil {
		return t.learner.LearnFromFeedback(ctx, e, false)
	}

	return nil
}
