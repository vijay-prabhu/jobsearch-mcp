package tracker

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
)

// ComputeStatus determines the conversation status based on email history
func ComputeStatus(emails []database.Email, myEmail string, staleAfterDays int) database.ConversationStatus {
	if len(emails) == 0 {
		return database.StatusActive
	}

	// Sort by date descending
	sorted := make([]database.Email, len(emails))
	copy(sorted, emails)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.After(sorted[j].Date)
	})

	lastEmail := sorted[0]
	daysSince := time.Since(lastEmail.Date).Hours() / 24

	// Check if stale
	if daysSince > float64(staleAfterDays) {
		return database.StatusStale
	}

	// Check who sent the last email
	if isFromMe(lastEmail, myEmail) {
		return database.StatusWaitingOnThem
	}

	return database.StatusWaitingOnMe
}

// isFromMe checks if the email was sent by the user
func isFromMe(e database.Email, myEmail string) bool {
	return strings.EqualFold(e.FromAddress, myEmail)
}

// ComputeResponseTime calculates the average response time in days
func ComputeResponseTime(emails []database.Email, myEmail string) float64 {
	if len(emails) < 2 {
		return 0
	}

	// Sort by date ascending
	sorted := make([]database.Email, len(emails))
	copy(sorted, emails)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	var totalDays float64
	var responseCount int

	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1]
		curr := sorted[i]

		// Only count responses (direction changes)
		prevFromMe := isFromMe(prev, myEmail)
		currFromMe := isFromMe(curr, myEmail)

		if prevFromMe != currFromMe {
			days := curr.Date.Sub(prev.Date).Hours() / 24
			totalDays += days
			responseCount++
		}
	}

	if responseCount == 0 {
		return 0
	}

	return totalDays / float64(responseCount)
}

// GetLastActivitySummary returns a human-readable summary of the last activity
func GetLastActivitySummary(emails []database.Email, myEmail string) string {
	if len(emails) == 0 {
		return "No activity"
	}

	// Sort by date descending
	sorted := make([]database.Email, len(emails))
	copy(sorted, emails)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.After(sorted[j].Date)
	})

	lastEmail := sorted[0]
	days := int(time.Since(lastEmail.Date).Hours() / 24)

	direction := "received"
	if isFromMe(lastEmail, myEmail) {
		direction = "sent"
	}

	switch {
	case days == 0:
		return "Today - " + direction
	case days == 1:
		return "Yesterday - " + direction
	case days < 7:
		return formatDays(days) + " ago - " + direction
	case days < 30:
		weeks := days / 7
		return formatWeeks(weeks) + " ago - " + direction
	default:
		return formatDays(days) + " ago - " + direction
	}
}

func formatDays(days int) string {
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func formatWeeks(weeks int) string {
	if weeks == 1 {
		return "1 week"
	}
	return fmt.Sprintf("%d weeks", weeks)
}
