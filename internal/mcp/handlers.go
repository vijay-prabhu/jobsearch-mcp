package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
)

func (s *Server) registerHandlers() {
	s.handlers["list_conversations"] = s.handleListConversations
	s.handlers["get_conversation"] = s.handleGetConversation
	s.handlers["get_pending_actions"] = s.handleGetPendingActions
	s.handlers["search_conversations"] = s.handleSearchConversations
	s.handlers["get_stats"] = s.handleGetStats
}

type listConversationsParams struct {
	Status    string `json:"status"`
	Company   string `json:"company"`
	SinceDays int    `json:"since_days"`
	Limit     int    `json:"limit"`
}

func (s *Server) handleListConversations(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p listConversationsParams
	if params != nil {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	opts := database.ListOptions{}

	if p.Status != "" && p.Status != "all" {
		status := database.ConversationStatus(p.Status)
		opts.Status = &status
	}

	if p.Company != "" {
		opts.Company = &p.Company
	}

	if p.SinceDays > 0 {
		since := time.Now().AddDate(0, 0, -p.SinceDays)
		opts.Since = &since
	}

	if p.Limit > 0 {
		opts.Limit = p.Limit
	} else {
		opts.Limit = 20
	}

	convs, err := s.db.ListConversations(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return convs, nil
}

type getConversationParams struct {
	Identifier string `json:"identifier"`
}

type conversationWithEmails struct {
	Conversation *database.Conversation `json:"conversation"`
	Emails       []database.Email       `json:"emails"`
}

func (s *Server) handleGetConversation(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p getConversationParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.Identifier == "" {
		return nil, fmt.Errorf("identifier is required")
	}

	// Try to find by company first
	conv, err := s.db.GetConversationByCompany(ctx, p.Identifier)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if conv == nil {
		// Try by ID
		conv, err = s.db.GetConversation(ctx, p.Identifier)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
	}

	if conv == nil {
		return nil, fmt.Errorf("conversation not found: %s", p.Identifier)
	}

	// Get emails
	emails, err := s.db.ListEmailsForConversation(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}

	return conversationWithEmails{
		Conversation: conv,
		Emails:       emails,
	}, nil
}

type getPendingActionsParams struct {
	IncludeStale *bool `json:"include_stale"`
}

type pendingActionsResult struct {
	WaitingOnMe []database.Conversation `json:"waiting_on_me"`
	Stale       []database.Conversation `json:"stale,omitempty"`
	Summary     string                  `json:"summary"`
}

func (s *Server) handleGetPendingActions(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p getPendingActionsParams
	if params != nil {
		json.Unmarshal(params, &p)
	}

	includeStale := true
	if p.IncludeStale != nil {
		includeStale = *p.IncludeStale
	}

	result := pendingActionsResult{}

	// Get waiting_on_me
	statusWaiting := database.StatusWaitingOnMe
	waitingOnMe, err := s.db.ListConversations(ctx, database.ListOptions{
		Status: &statusWaiting,
	})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	result.WaitingOnMe = waitingOnMe

	// Get stale if requested
	if includeStale {
		statusStale := database.StatusStale
		stale, err := s.db.ListConversations(ctx, database.ListOptions{
			Status: &statusStale,
		})
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		result.Stale = stale
	}

	// Build summary
	if len(result.WaitingOnMe) == 0 && len(result.Stale) == 0 {
		result.Summary = "No pending actions! All caught up."
	} else {
		result.Summary = fmt.Sprintf("%d conversation(s) waiting for your response", len(result.WaitingOnMe))
		if len(result.Stale) > 0 {
			result.Summary += fmt.Sprintf(", %d stale conversation(s) may need follow-up", len(result.Stale))
		}
	}

	return result, nil
}

type searchParams struct {
	Query string `json:"query"`
}

func (s *Server) handleSearchConversations(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p searchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	results, err := s.db.Search(ctx, p.Query)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	return results, nil
}

type getStatsParams struct {
	SinceDays int `json:"since_days"`
}

func (s *Server) handleGetStats(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p getStatsParams
	if params != nil {
		json.Unmarshal(params, &p)
	}

	var since *time.Time
	if p.SinceDays > 0 {
		t := time.Now().AddDate(0, 0, -p.SinceDays)
		since = &t
	}

	stats, err := s.db.GetStats(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return stats, nil
}
