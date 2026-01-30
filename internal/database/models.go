package database

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ConversationStatus represents the state of a conversation
type ConversationStatus string

const (
	StatusActive        ConversationStatus = "active"
	StatusWaitingOnMe   ConversationStatus = "waiting_on_me"
	StatusWaitingOnThem ConversationStatus = "waiting_on_them"
	StatusStale         ConversationStatus = "stale"
	StatusClosed        ConversationStatus = "closed"
)

// Direction represents email direction
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// Conversation represents a job search conversation thread
type Conversation struct {
	ID             string             `json:"id"`
	Company        string             `json:"company"`
	Position       *string            `json:"position,omitempty"`
	RecruiterName  *string            `json:"recruiter_name,omitempty"`
	RecruiterEmail *string            `json:"recruiter_email,omitempty"`
	Direction      Direction          `json:"direction"`
	Status         ConversationStatus `json:"status"`
	LastActivityAt time.Time          `json:"last_activity_at"`
	EmailCount     int                `json:"email_count"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// DaysSinceActivity returns the number of days since last activity
func (c *Conversation) DaysSinceActivity() int {
	return int(time.Since(c.LastActivityAt).Hours() / 24)
}

// IsStale returns true if the conversation is older than the given days
func (c *Conversation) IsStale(days int) bool {
	return c.DaysSinceActivity() > days
}

// Email represents a single email message
type Email struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	GmailID        string    `json:"gmail_id"`
	ThreadID       string    `json:"thread_id"`
	Subject        *string   `json:"subject,omitempty"`
	FromAddress    string    `json:"from_address"`
	FromName       *string   `json:"from_name,omitempty"`
	ToAddress      *string   `json:"to_address,omitempty"`
	Date           time.Time `json:"date"`
	Direction      Direction `json:"direction"`
	Snippet        *string   `json:"snippet,omitempty"`
	BodyStored     bool      `json:"body_stored"`
	BodyEncrypted  *string   `json:"-"` // Never expose in JSON
	Classification *string   `json:"classification,omitempty"`
	Confidence     *float64  `json:"confidence,omitempty"`
	ExtractedData  *string   `json:"extracted_data,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// GetExtractedData parses the extracted data JSON
func (e *Email) GetExtractedData() (map[string]interface{}, error) {
	if e.ExtractedData == nil {
		return nil, nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(*e.ExtractedData), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// SyncState tracks the sync progress
type SyncState struct {
	ID              int        `json:"id"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	LastHistoryID   *string    `json:"last_history_id,omitempty"`
	EmailsProcessed int        `json:"emails_processed"`
}

// LearnedFilter represents a user or AI-learned filter
type LearnedFilter struct {
	ID         string    `json:"id"`
	FilterType string    `json:"filter_type"`
	Value      string    `json:"value"`
	Source     string    `json:"source"`
	Confidence *float64  `json:"confidence,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Stats represents aggregate statistics
type Stats struct {
	TotalConversations int     `json:"total_conversations"`
	WaitingOnMe        int     `json:"waiting_on_me"`
	WaitingOnThem      int     `json:"waiting_on_them"`
	Stale              int     `json:"stale"`
	Closed             int     `json:"closed"`
	TotalEmails        int     `json:"total_emails"`
	ResponseRate       float64 `json:"response_rate"`
	AvgResponseTime    float64 `json:"avg_response_time_days"`
}

// RecruiterGroup represents conversations grouped by recruiter
type RecruiterGroup struct {
	RecruiterEmail string         `json:"recruiter_email"`
	RecruiterName  *string        `json:"recruiter_name,omitempty"`
	Companies      []string       `json:"companies"`
	Conversations  []Conversation `json:"conversations"`
	TotalEmails    int            `json:"total_emails"`
}

// CompanyGroup represents conversations grouped by company
type CompanyGroup struct {
	Company       string         `json:"company"`
	Conversations []Conversation `json:"conversations"`
	TotalEmails   int            `json:"total_emails"`
}

// ListOptions contains options for listing conversations
type ListOptions struct {
	Status    *ConversationStatus
	Direction *Direction
	Since     *time.Time
	Company   *string
	Limit     int
	Offset    int
}

// NullString is a helper to convert *string to sql.NullString
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// NullFloat64 is a helper to convert *float64 to sql.NullFloat64
func NullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

// StringPtr converts sql.NullString to *string
func StringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// Float64Ptr converts sql.NullFloat64 to *float64
func Float64Ptr(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	return &nf.Float64
}
