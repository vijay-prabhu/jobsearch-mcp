package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateConversation inserts a new conversation
func (db *DB) CreateConversation(ctx context.Context, c *Conversation) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO conversations (
			id, company, position, recruiter_name, recruiter_email,
			direction, status, last_activity_at, email_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		c.ID, c.Company, NullString(c.Position), NullString(c.RecruiterName),
		NullString(c.RecruiterEmail), c.Direction, c.Status,
		c.LastActivityAt, c.EmailCount, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetConversation retrieves a conversation by ID
func (db *DB) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	c := &Conversation{}
	var position, recruiterName, recruiterEmail sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, company, position, recruiter_name, recruiter_email,
		       direction, status, last_activity_at, email_count, created_at, updated_at
		FROM conversations WHERE id = ?
	`, id).Scan(
		&c.ID, &c.Company, &position, &recruiterName, &recruiterEmail,
		&c.Direction, &c.Status, &c.LastActivityAt, &c.EmailCount, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Position = StringPtr(position)
	c.RecruiterName = StringPtr(recruiterName)
	c.RecruiterEmail = StringPtr(recruiterEmail)
	return c, nil
}

// GetConversationByCompany retrieves a conversation by company name (case-insensitive)
func (db *DB) GetConversationByCompany(ctx context.Context, company string) (*Conversation, error) {
	c := &Conversation{}
	var position, recruiterName, recruiterEmail sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, company, position, recruiter_name, recruiter_email,
		       direction, status, last_activity_at, email_count, created_at, updated_at
		FROM conversations WHERE LOWER(company) = LOWER(?)
		ORDER BY last_activity_at DESC LIMIT 1
	`, company).Scan(
		&c.ID, &c.Company, &position, &recruiterName, &recruiterEmail,
		&c.Direction, &c.Status, &c.LastActivityAt, &c.EmailCount, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Position = StringPtr(position)
	c.RecruiterName = StringPtr(recruiterName)
	c.RecruiterEmail = StringPtr(recruiterEmail)
	return c, nil
}

// UpdateConversation updates an existing conversation
func (db *DB) UpdateConversation(ctx context.Context, c *Conversation) error {
	c.UpdatedAt = time.Now()

	result, err := db.ExecContext(ctx, `
		UPDATE conversations SET
			company = ?, position = ?, recruiter_name = ?, recruiter_email = ?,
			direction = ?, status = ?, last_activity_at = ?, email_count = ?, updated_at = ?
		WHERE id = ?
	`,
		c.Company, NullString(c.Position), NullString(c.RecruiterName),
		NullString(c.RecruiterEmail), c.Direction, c.Status,
		c.LastActivityAt, c.EmailCount, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found: %s", c.ID)
	}
	return nil
}

// ListConversations retrieves conversations with optional filters
func (db *DB) ListConversations(ctx context.Context, opts ListOptions) ([]Conversation, error) {
	query := `
		SELECT id, company, position, recruiter_name, recruiter_email,
		       direction, status, last_activity_at, email_count, created_at, updated_at
		FROM conversations WHERE 1=1
	`
	args := []interface{}{}

	if opts.Status != nil {
		query += " AND status = ?"
		args = append(args, *opts.Status)
	}
	if opts.Direction != nil {
		query += " AND direction = ?"
		args = append(args, *opts.Direction)
	}
	if opts.Since != nil {
		query += " AND last_activity_at >= ?"
		args = append(args, *opts.Since)
	}
	if opts.Company != nil {
		query += " AND LOWER(company) LIKE LOWER(?)"
		args = append(args, "%"+*opts.Company+"%")
	}

	query += " ORDER BY last_activity_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		c := Conversation{}
		var position, recruiterName, recruiterEmail sql.NullString

		if err := rows.Scan(
			&c.ID, &c.Company, &position, &recruiterName, &recruiterEmail,
			&c.Direction, &c.Status, &c.LastActivityAt, &c.EmailCount, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}

		c.Position = StringPtr(position)
		c.RecruiterName = StringPtr(recruiterName)
		c.RecruiterEmail = StringPtr(recruiterEmail)
		conversations = append(conversations, c)
	}

	return conversations, rows.Err()
}

// CreateEmail inserts a new email
func (db *DB) CreateEmail(ctx context.Context, e *Email) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO emails (
			id, conversation_id, gmail_id, thread_id, subject, from_address, from_name,
			to_address, date, direction, snippet, body_stored, body_encrypted,
			classification, confidence, extracted_data, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		e.ID, e.ConversationID, e.GmailID, e.ThreadID, NullString(e.Subject),
		e.FromAddress, NullString(e.FromName), NullString(e.ToAddress),
		e.Date, e.Direction, NullString(e.Snippet), e.BodyStored, NullString(e.BodyEncrypted),
		NullString(e.Classification), NullFloat64(e.Confidence), NullString(e.ExtractedData), e.CreatedAt,
	)
	return err
}

// GetEmailByGmailID retrieves an email by Gmail ID
func (db *DB) GetEmailByGmailID(ctx context.Context, gmailID string) (*Email, error) {
	e := &Email{}
	var subject, fromName, toAddress, snippet, bodyEncrypted, classification, extractedData sql.NullString
	var confidence sql.NullFloat64

	err := db.QueryRowContext(ctx, `
		SELECT id, conversation_id, gmail_id, thread_id, subject, from_address, from_name,
		       to_address, date, direction, snippet, body_stored, body_encrypted,
		       classification, confidence, extracted_data, created_at
		FROM emails WHERE gmail_id = ?
	`, gmailID).Scan(
		&e.ID, &e.ConversationID, &e.GmailID, &e.ThreadID, &subject, &e.FromAddress, &fromName,
		&toAddress, &e.Date, &e.Direction, &snippet, &e.BodyStored, &bodyEncrypted,
		&classification, &confidence, &extractedData, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	e.Subject = StringPtr(subject)
	e.FromName = StringPtr(fromName)
	e.ToAddress = StringPtr(toAddress)
	e.Snippet = StringPtr(snippet)
	e.BodyEncrypted = StringPtr(bodyEncrypted)
	e.Classification = StringPtr(classification)
	e.Confidence = Float64Ptr(confidence)
	e.ExtractedData = StringPtr(extractedData)
	return e, nil
}

// ListEmailsForConversation retrieves all emails for a conversation
func (db *DB) ListEmailsForConversation(ctx context.Context, convID string) ([]Email, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, conversation_id, gmail_id, thread_id, subject, from_address, from_name,
		       to_address, date, direction, snippet, body_stored,
		       classification, confidence, extracted_data, created_at
		FROM emails WHERE conversation_id = ?
		ORDER BY date ASC
	`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []Email
	for rows.Next() {
		e := Email{}
		var subject, fromName, toAddress, snippet, classification, extractedData sql.NullString
		var confidence sql.NullFloat64

		if err := rows.Scan(
			&e.ID, &e.ConversationID, &e.GmailID, &e.ThreadID, &subject, &e.FromAddress, &fromName,
			&toAddress, &e.Date, &e.Direction, &snippet, &e.BodyStored,
			&classification, &confidence, &extractedData, &e.CreatedAt,
		); err != nil {
			return nil, err
		}

		e.Subject = StringPtr(subject)
		e.FromName = StringPtr(fromName)
		e.ToAddress = StringPtr(toAddress)
		e.Snippet = StringPtr(snippet)
		e.Classification = StringPtr(classification)
		e.Confidence = Float64Ptr(confidence)
		e.ExtractedData = StringPtr(extractedData)
		emails = append(emails, e)
	}

	return emails, rows.Err()
}

// GetStats retrieves aggregate statistics
func (db *DB) GetStats(ctx context.Context, since *time.Time) (*Stats, error) {
	stats := &Stats{}

	whereClause := ""
	args := []interface{}{}
	if since != nil {
		whereClause = "WHERE last_activity_at >= ?"
		args = append(args, *since)
	}

	// Get conversation counts by status
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'waiting_on_me' THEN 1 ELSE 0 END) as waiting_on_me,
			SUM(CASE WHEN status = 'waiting_on_them' THEN 1 ELSE 0 END) as waiting_on_them,
			SUM(CASE WHEN status = 'stale' THEN 1 ELSE 0 END) as stale,
			SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END) as closed
		FROM conversations %s
	`, whereClause)

	if err := db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalConversations, &stats.WaitingOnMe, &stats.WaitingOnThem,
		&stats.Stale, &stats.Closed,
	); err != nil {
		return nil, err
	}

	// Get total email count
	emailQuery := "SELECT COUNT(*) FROM emails"
	if since != nil {
		emailQuery += " WHERE date >= ?"
	}
	if err := db.QueryRowContext(ctx, emailQuery, args...).Scan(&stats.TotalEmails); err != nil {
		return nil, err
	}

	return stats, nil
}

// Search searches conversations by text
func (db *DB) Search(ctx context.Context, query string) ([]Conversation, error) {
	searchPattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT c.id, c.company, c.position, c.recruiter_name, c.recruiter_email,
		       c.direction, c.status, c.last_activity_at, c.email_count, c.created_at, c.updated_at
		FROM conversations c
		LEFT JOIN emails e ON c.id = e.conversation_id
		WHERE LOWER(c.company) LIKE ?
		   OR LOWER(c.position) LIKE ?
		   OR LOWER(c.recruiter_name) LIKE ?
		   OR LOWER(c.recruiter_email) LIKE ?
		   OR LOWER(e.subject) LIKE ?
		ORDER BY c.last_activity_at DESC
	`, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		c := Conversation{}
		var position, recruiterName, recruiterEmail sql.NullString

		if err := rows.Scan(
			&c.ID, &c.Company, &position, &recruiterName, &recruiterEmail,
			&c.Direction, &c.Status, &c.LastActivityAt, &c.EmailCount, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}

		c.Position = StringPtr(position)
		c.RecruiterName = StringPtr(recruiterName)
		c.RecruiterEmail = StringPtr(recruiterEmail)
		conversations = append(conversations, c)
	}

	return conversations, rows.Err()
}

// GetSyncState retrieves the current sync state
func (db *DB) GetSyncState(ctx context.Context) (*SyncState, error) {
	state := &SyncState{}
	var lastSyncAt sql.NullTime
	var lastHistoryID sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, last_sync_at, last_history_id, emails_processed
		FROM sync_state WHERE id = 1
	`).Scan(&state.ID, &lastSyncAt, &lastHistoryID, &state.EmailsProcessed)
	if err != nil {
		return nil, err
	}

	if lastSyncAt.Valid {
		state.LastSyncAt = &lastSyncAt.Time
	}
	state.LastHistoryID = StringPtr(lastHistoryID)
	return state, nil
}

// UpdateSyncState updates the sync state
func (db *DB) UpdateSyncState(ctx context.Context, state *SyncState) error {
	_, err := db.ExecContext(ctx, `
		UPDATE sync_state SET
			last_sync_at = ?, last_history_id = ?, emails_processed = ?
		WHERE id = 1
	`, state.LastSyncAt, NullString(state.LastHistoryID), state.EmailsProcessed)
	return err
}

// GetConversationByThreadID finds a conversation that contains emails with the given thread ID
func (db *DB) GetConversationByThreadID(ctx context.Context, threadID string) (*Conversation, error) {
	c := &Conversation{}
	var position, recruiterName, recruiterEmail sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT c.id, c.company, c.position, c.recruiter_name, c.recruiter_email,
		       c.direction, c.status, c.last_activity_at, c.email_count, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN emails e ON c.id = e.conversation_id
		WHERE e.thread_id = ?
		LIMIT 1
	`, threadID).Scan(
		&c.ID, &c.Company, &position, &recruiterName, &recruiterEmail,
		&c.Direction, &c.Status, &c.LastActivityAt, &c.EmailCount, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.Position = StringPtr(position)
	c.RecruiterName = StringPtr(recruiterName)
	c.RecruiterEmail = StringPtr(recruiterEmail)
	return c, nil
}

// IncrementEmailCount increments the email count for a conversation
func (db *DB) IncrementEmailCount(ctx context.Context, convID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE conversations SET email_count = email_count + 1, updated_at = ?
		WHERE id = ?
	`, time.Now(), convID)
	return err
}

// Learned Filters

// FilterType constants
const (
	FilterTypeDomainWhitelist  = "domain_whitelist"
	FilterTypeDomainBlacklist  = "domain_blacklist"
	FilterTypeSubjectKeyword   = "subject_keyword"
	FilterTypeBodyKeyword      = "body_keyword"
	FilterTypeSubjectBlacklist = "subject_blacklist"
)

// FilterSource constants
const (
	FilterSourceUser        = "user"
	FilterSourceAISuggested = "ai_suggested"
	FilterSourceAIConfirmed = "ai_confirmed"
)

// CreateLearnedFilter inserts a new learned filter
func (db *DB) CreateLearnedFilter(ctx context.Context, f *LearnedFilter) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	f.CreatedAt = time.Now()

	_, err := db.ExecContext(ctx, `
		INSERT INTO learned_filters (id, filter_type, value, source, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(filter_type, value) DO UPDATE SET
			source = excluded.source,
			confidence = excluded.confidence
	`, f.ID, f.FilterType, f.Value, f.Source, NullFloat64(f.Confidence), f.CreatedAt)
	return err
}

// GetLearnedFilter retrieves a learned filter by ID
func (db *DB) GetLearnedFilter(ctx context.Context, id string) (*LearnedFilter, error) {
	f := &LearnedFilter{}
	var confidence sql.NullFloat64

	err := db.QueryRowContext(ctx, `
		SELECT id, filter_type, value, source, confidence, created_at
		FROM learned_filters WHERE id = ?
	`, id).Scan(&f.ID, &f.FilterType, &f.Value, &f.Source, &confidence, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	f.Confidence = Float64Ptr(confidence)
	return f, nil
}

// LearnedFilterListOptions contains options for listing learned filters
type LearnedFilterListOptions struct {
	FilterType *string
	Source     *string
	Limit      int
}

// ListLearnedFilters retrieves learned filters with optional filters
func (db *DB) ListLearnedFilters(ctx context.Context, opts LearnedFilterListOptions) ([]LearnedFilter, error) {
	query := `SELECT id, filter_type, value, source, confidence, created_at FROM learned_filters WHERE 1=1`
	args := []interface{}{}

	if opts.FilterType != nil {
		query += " AND filter_type = ?"
		args = append(args, *opts.FilterType)
	}
	if opts.Source != nil {
		query += " AND source = ?"
		args = append(args, *opts.Source)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []LearnedFilter
	for rows.Next() {
		f := LearnedFilter{}
		var confidence sql.NullFloat64

		if err := rows.Scan(&f.ID, &f.FilterType, &f.Value, &f.Source, &confidence, &f.CreatedAt); err != nil {
			return nil, err
		}

		f.Confidence = Float64Ptr(confidence)
		filters = append(filters, f)
	}

	return filters, rows.Err()
}

// LearnedFilterExists checks if a filter with the given type and value exists
func (db *DB) LearnedFilterExists(ctx context.Context, filterType, value string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM learned_filters WHERE filter_type = ? AND LOWER(value) = LOWER(?)
	`, filterType, value).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteLearnedFilter deletes a learned filter by ID
func (db *DB) DeleteLearnedFilter(ctx context.Context, id string) error {
	result, err := db.ExecContext(ctx, `DELETE FROM learned_filters WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("filter not found: %s", id)
	}
	return nil
}

// ApproveLearnedFilter changes a filter's source from ai_suggested to ai_confirmed
func (db *DB) ApproveLearnedFilter(ctx context.Context, id string) error {
	result, err := db.ExecContext(ctx, `
		UPDATE learned_filters SET source = ? WHERE id = ? AND source = ?
	`, FilterSourceAIConfirmed, id, FilterSourceAISuggested)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("filter not found or not a suggestion: %s", id)
	}
	return nil
}

// GetLearnedFiltersByType returns all confirmed filters of a specific type
func (db *DB) GetLearnedFiltersByType(ctx context.Context, filterType string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT value FROM learned_filters
		WHERE filter_type = ? AND (source = ? OR source = ?)
		ORDER BY value
	`, filterType, FilterSourceUser, FilterSourceAIConfirmed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	return values, rows.Err()
}
