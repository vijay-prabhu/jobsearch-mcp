package tracker

import (
	"testing"
	"time"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
)

func TestComputeStatus(t *testing.T) {
	myEmail := "me@example.com"
	staleAfterDays := 7

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-8 * 24 * time.Hour)

	tests := []struct {
		name     string
		emails   []database.Email
		expected database.ConversationStatus
	}{
		{
			name:     "empty emails",
			emails:   []database.Email{},
			expected: database.StatusActive,
		},
		{
			name: "last email from them - waiting on me",
			emails: []database.Email{
				{FromAddress: "recruiter@company.com", Date: yesterday},
			},
			expected: database.StatusWaitingOnMe,
		},
		{
			name: "last email from me - waiting on them",
			emails: []database.Email{
				{FromAddress: myEmail, Date: yesterday},
			},
			expected: database.StatusWaitingOnThem,
		},
		{
			name: "stale conversation",
			emails: []database.Email{
				{FromAddress: "recruiter@company.com", Date: lastWeek},
			},
			expected: database.StatusStale,
		},
		{
			name: "multiple emails - check last one",
			emails: []database.Email{
				{FromAddress: "recruiter@company.com", Date: now.Add(-48 * time.Hour)},
				{FromAddress: myEmail, Date: now.Add(-24 * time.Hour)},
				{FromAddress: "recruiter@company.com", Date: now.Add(-1 * time.Hour)}, // most recent
			},
			expected: database.StatusWaitingOnMe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeStatus(tt.emails, myEmail, staleAfterDays)
			if result != tt.expected {
				t.Errorf("ComputeStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComputeResponseTime(t *testing.T) {
	myEmail := "me@example.com"

	now := time.Now()

	tests := []struct {
		name     string
		emails   []database.Email
		expected float64
	}{
		{
			name:     "single email - no response time",
			emails:   []database.Email{{FromAddress: "recruiter@company.com", Date: now}},
			expected: 0,
		},
		{
			name: "two emails - 1 day response",
			emails: []database.Email{
				{FromAddress: "recruiter@company.com", Date: now.Add(-48 * time.Hour)},
				{FromAddress: myEmail, Date: now.Add(-24 * time.Hour)},
			},
			expected: 1.0,
		},
		{
			name: "same sender - no direction change",
			emails: []database.Email{
				{FromAddress: "recruiter@company.com", Date: now.Add(-48 * time.Hour)},
				{FromAddress: "recruiter@company.com", Date: now.Add(-24 * time.Hour)},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeResponseTime(tt.emails, myEmail)
			if result != tt.expected {
				t.Errorf("ComputeResponseTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsFromMe(t *testing.T) {
	myEmail := "me@example.com"

	tests := []struct {
		email    database.Email
		expected bool
	}{
		{database.Email{FromAddress: "me@example.com"}, true},
		{database.Email{FromAddress: "ME@EXAMPLE.COM"}, true},
		{database.Email{FromAddress: "other@example.com"}, false},
	}

	for _, tt := range tests {
		result := isFromMe(tt.email, myEmail)
		if result != tt.expected {
			t.Errorf("isFromMe(%q) = %v, want %v", tt.email.FromAddress, result, tt.expected)
		}
	}
}
