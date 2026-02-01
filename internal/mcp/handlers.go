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
	s.handlers["merge_conversations"] = s.handleMergeConversations
	s.handlers["archive_conversation"] = s.handleArchiveConversation
}

type listConversationsParams struct {
	Status          string `json:"status"`
	Company         string `json:"company"`
	SinceDays       int    `json:"since_days"`
	Limit           int    `json:"limit"`
	IncludeArchived bool   `json:"include_archived"`
}

func (s *Server) handleListConversations(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p listConversationsParams
	if params != nil {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}

	opts := database.ListOptions{
		IncludeArchived: p.IncludeArchived,
	}

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
		_ = json.Unmarshal(params, &p) // Ignore error, use defaults
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

type getStatsParamsExtended struct {
	SinceDays int  `json:"since_days"`
	Detailed  bool `json:"detailed"`
}

func (s *Server) handleGetStats(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p getStatsParamsExtended
	if params != nil {
		_ = json.Unmarshal(params, &p) // Ignore error, use defaults
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

	if !p.Detailed {
		return stats, nil
	}

	// Get detailed stats
	convs, err := s.db.ListConversations(ctx, database.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Calculate response rate
	var inbound, withReply int
	for _, c := range convs {
		if c.Direction == database.DirectionInbound {
			inbound++
			if c.EmailCount > 1 {
				withReply++
			}
		}
	}
	responseRate := 0.0
	if inbound > 0 {
		responseRate = float64(withReply) / float64(inbound) * 100
	}

	return map[string]interface{}{
		"basic":                    stats,
		"total_inbound":            inbound,
		"conversations_with_reply": withReply,
		"response_rate_percent":    responseRate,
		"total_active_companies":   len(convs),
	}, nil
}

type mergeConversationsParams struct {
	Target string `json:"target"`
	Source string `json:"source"`
}

func (s *Server) handleMergeConversations(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p mergeConversationsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.Target == "" || p.Source == "" {
		return nil, fmt.Errorf("both target and source are required")
	}

	// Find target conversation
	targetConv, err := s.findConversation(ctx, p.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to find target: %w", err)
	}
	if targetConv == nil {
		return nil, fmt.Errorf("target conversation not found: %s", p.Target)
	}

	// Find source conversation
	sourceConv, err := s.findConversation(ctx, p.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to find source: %w", err)
	}
	if sourceConv == nil {
		return nil, fmt.Errorf("source conversation not found: %s", p.Source)
	}

	// Perform merge
	result, err := s.db.MergeConversations(ctx, targetConv.ID, sourceConv.ID)
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	return map[string]interface{}{
		"merged":       true,
		"target":       targetConv.Company,
		"source":       sourceConv.Company,
		"emails_moved": result.EmailsMoved,
		"total_emails": result.TotalEmails,
	}, nil
}

type archiveConversationParams struct {
	Identifier string `json:"identifier"`
	Unarchive  bool   `json:"unarchive"`
}

func (s *Server) handleArchiveConversation(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p archiveConversationParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if p.Identifier == "" {
		return nil, fmt.Errorf("identifier is required")
	}

	// Find conversation
	conv, err := s.findConversation(ctx, p.Identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("conversation not found: %s", p.Identifier)
	}

	var result *database.ArchiveResult
	if p.Unarchive {
		result, err = s.db.UnarchiveConversation(ctx, conv.ID)
	} else {
		result, err = s.db.ArchiveConversation(ctx, conv.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("operation failed: %w", err)
	}

	action := "archived"
	if p.Unarchive {
		action = "unarchived"
	}

	return map[string]interface{}{
		"success":  true,
		"action":   action,
		"company":  result.Company,
		"archived": result.Archived,
	}, nil
}

// findConversation finds a conversation by company name or ID
func (s *Server) findConversation(ctx context.Context, identifier string) (*database.Conversation, error) {
	// Try by company first
	conv, err := s.db.GetConversationByCompany(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	// Try by ID
	conv, err = s.db.GetConversation(ctx, identifier)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

// Resource handlers

func (s *Server) handleReadResource(ctx context.Context, uri string) (string, error) {
	switch uri {
	case "jobsearch://summary":
		return s.getResourceSummary(ctx)
	case "jobsearch://pending":
		return s.getResourcePending(ctx)
	case "jobsearch://recent":
		return s.getResourceRecent(ctx)
	case "jobsearch://companies":
		return s.getResourceCompanies(ctx)
	default:
		return "", fmt.Errorf("unknown resource: %s", uri)
	}
}

func (s *Server) getResourceSummary(ctx context.Context) (string, error) {
	stats, err := s.db.GetStats(ctx, nil)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf(`Job Search Summary
==================
Total Conversations: %d
  - Waiting on me:   %d
  - Waiting on them: %d
  - Stale (>7 days): %d
  - Closed:          %d

Response Rate: %.1f%%
`, stats.TotalConversations, stats.WaitingOnMe, stats.WaitingOnThem, stats.Stale, stats.Closed,
		stats.ResponseRate*100)

	return summary, nil
}

func (s *Server) getResourcePending(ctx context.Context) (string, error) {
	// Get waiting_on_me
	statusWaiting := database.StatusWaitingOnMe
	waitingOnMe, err := s.db.ListConversations(ctx, database.ListOptions{
		Status: &statusWaiting,
		Limit:  50,
	})
	if err != nil {
		return "", err
	}

	// Get stale
	statusStale := database.StatusStale
	stale, err := s.db.ListConversations(ctx, database.ListOptions{
		Status: &statusStale,
		Limit:  50,
	})
	if err != nil {
		return "", err
	}

	var result string
	result = "Pending Actions\n===============\n\n"

	if len(waitingOnMe) == 0 && len(stale) == 0 {
		result += "No pending actions. All caught up!\n"
		return result, nil
	}

	if len(waitingOnMe) > 0 {
		result += fmt.Sprintf("WAITING ON ME (%d):\n", len(waitingOnMe))
		for _, c := range waitingOnMe {
			days := int(time.Since(c.LastActivityAt).Hours() / 24)
			recruiter := ""
			if c.RecruiterName != nil {
				recruiter = fmt.Sprintf(" (%s)", *c.RecruiterName)
			}
			result += fmt.Sprintf("  - %s%s - %d day(s) ago\n", c.Company, recruiter, days)
		}
		result += "\n"
	}

	if len(stale) > 0 {
		result += fmt.Sprintf("STALE - NEEDS FOLLOW-UP (%d):\n", len(stale))
		for _, c := range stale {
			days := int(time.Since(c.LastActivityAt).Hours() / 24)
			result += fmt.Sprintf("  - %s - %d day(s) since last activity\n", c.Company, days)
		}
	}

	return result, nil
}

func (s *Server) getResourceRecent(ctx context.Context) (string, error) {
	convs, err := s.db.ListConversations(ctx, database.ListOptions{
		Limit: 10,
	})
	if err != nil {
		return "", err
	}

	result := "Recent Activity (Last 10 Conversations)\n========================================\n\n"

	if len(convs) == 0 {
		result += "No conversations yet. Run 'jobsearch sync' to fetch emails.\n"
		return result, nil
	}

	for _, c := range convs {
		days := int(time.Since(c.LastActivityAt).Hours() / 24)
		status := string(c.Status)
		recruiter := ""
		if c.RecruiterName != nil {
			recruiter = *c.RecruiterName
		}
		result += fmt.Sprintf("- %s | %s | %s | %d day(s) ago | %d email(s)\n",
			c.Company, recruiter, status, days, c.EmailCount)
	}

	return result, nil
}

func (s *Server) getResourceCompanies(ctx context.Context) (string, error) {
	convs, err := s.db.ListConversations(ctx, database.ListOptions{
		Limit: 100,
	})
	if err != nil {
		return "", err
	}

	result := "Companies List\n==============\n\n"

	if len(convs) == 0 {
		result += "No companies yet.\n"
		return result, nil
	}

	// Group by status
	byStatus := make(map[database.ConversationStatus][]string)
	for _, c := range convs {
		byStatus[c.Status] = append(byStatus[c.Status], c.Company)
	}

	statusOrder := []database.ConversationStatus{
		database.StatusWaitingOnMe,
		database.StatusWaitingOnThem,
		database.StatusActive,
		database.StatusStale,
	}

	for _, status := range statusOrder {
		companies := byStatus[status]
		if len(companies) > 0 {
			result += fmt.Sprintf("%s (%d):\n", status, len(companies))
			for _, company := range companies {
				result += fmt.Sprintf("  - %s\n", company)
			}
			result += "\n"
		}
	}

	return result, nil
}
