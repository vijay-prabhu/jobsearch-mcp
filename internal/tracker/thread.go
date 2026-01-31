package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// ThreadEmail represents an email in a thread with full body content
type ThreadEmail struct {
	ID         string    `json:"id"`
	Subject    string    `json:"subject"`
	From       string    `json:"from"`
	FromName   string    `json:"from_name,omitempty"`
	To         string    `json:"to,omitempty"`
	Date       time.Time `json:"date"`
	Direction  string    `json:"direction"`
	Body       string    `json:"body"`
	Snippet    string    `json:"snippet,omitempty"`
	ProviderID string    `json:"provider_id,omitempty"`
}

// Thread represents a full email thread with conversation metadata
type Thread struct {
	Conversation *database.Conversation `json:"conversation"`
	Emails       []ThreadEmail          `json:"emails"`
	FetchedAt    time.Time              `json:"fetched_at"`
}

// FetchThread retrieves the full email thread for a conversation
func (t *Tracker) FetchThread(ctx context.Context, companyOrID string) (*Thread, error) {
	// Try to find conversation by company name first
	conv, err := t.db.GetConversationByCompany(ctx, companyOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup conversation: %w", err)
	}

	// If not found by company, try by ID
	if conv == nil {
		conv, err = t.db.GetConversation(ctx, companyOrID)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup conversation: %w", err)
		}
	}

	if conv == nil {
		return nil, fmt.Errorf("conversation not found: %s", companyOrID)
	}

	// Get all emails for this conversation from database
	dbEmails, err := t.db.ListEmailsForConversation(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}

	if len(dbEmails) == 0 {
		return nil, fmt.Errorf("no emails found for conversation: %s", conv.Company)
	}

	// Fetch full content for each email from provider
	var threadEmails []ThreadEmail
	for _, dbEmail := range dbEmails {
		te := ThreadEmail{
			ID:         dbEmail.ID,
			From:       dbEmail.FromAddress,
			Date:       dbEmail.Date,
			Direction:  string(dbEmail.Direction),
			ProviderID: dbEmail.GmailID,
		}

		if dbEmail.Subject != nil {
			te.Subject = *dbEmail.Subject
		}
		if dbEmail.FromName != nil {
			te.FromName = *dbEmail.FromName
		}
		if dbEmail.ToAddress != nil {
			te.To = *dbEmail.ToAddress
		}
		if dbEmail.Snippet != nil {
			te.Snippet = *dbEmail.Snippet
		}

		// Fetch full email content from provider
		fullEmail, err := t.provider.GetEmail(ctx, dbEmail.GmailID)
		if err != nil {
			// If fetch fails, use snippet as fallback
			te.Body = te.Snippet
		} else if fullEmail != nil {
			te.Body = fullEmail.Body
			// Update other fields if they were empty
			if te.Subject == "" {
				te.Subject = fullEmail.Subject
			}
			if te.To == "" && len(fullEmail.To) > 0 {
				te.To = fullEmail.To[0].Email
			}
		}

		threadEmails = append(threadEmails, te)
	}

	return &Thread{
		Conversation: conv,
		Emails:       threadEmails,
		FetchedAt:    time.Now(),
	}, nil
}

// FetchThreadByID retrieves the full email thread by conversation ID
func (t *Tracker) FetchThreadByID(ctx context.Context, convID string) (*Thread, error) {
	conv, err := t.db.GetConversation(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("conversation not found: %s", convID)
	}

	return t.FetchThread(ctx, conv.Company)
}

// GetProvider returns the email provider (needed for CLI to check auth)
func (t *Tracker) GetProvider() email.Provider {
	return t.provider
}
