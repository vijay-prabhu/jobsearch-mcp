package tracker

import (
	"context"
	"strings"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/filter"
)

// Learner extracts patterns from classified emails and suggests filters
type Learner struct {
	db *database.DB
}

// NewLearner creates a new Learner
func NewLearner(db *database.DB) *Learner {
	return &Learner{db: db}
}

// LearnFromEmail extracts patterns from a job-related email and suggests filters
func (l *Learner) LearnFromEmail(ctx context.Context, e *email.Email, confidence float64) error {
	// Only learn from high-confidence classifications
	if confidence < 0.7 {
		return nil
	}

	// Extract and suggest domain
	if err := l.suggestDomain(ctx, e, confidence); err != nil {
		return err
	}

	// Extract and suggest keywords from subject
	if err := l.suggestSubjectKeywords(ctx, e, confidence); err != nil {
		return err
	}

	return nil
}

// suggestDomain suggests adding the email domain to whitelist
func (l *Learner) suggestDomain(ctx context.Context, e *email.Email, confidence float64) error {
	domain := e.Domain()
	if domain == "" {
		return nil
	}

	// Skip if it's a known ATS or common domain
	if isCommonDomain(domain) || isATSDomain(domain) {
		return nil
	}

	// Check if already exists (in any form)
	exists, err := l.db.LearnedFilterExists(ctx, database.FilterTypeDomainWhitelist, domain)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// Create suggestion
	f := &database.LearnedFilter{
		FilterType:         database.FilterTypeDomainWhitelist,
		Value:              domain,
		Source:             database.FilterSourceAISuggested,
		FalsePositiveCount: 0,
	}

	return l.db.CreateLearnedFilter(ctx, f)
}

// suggestSubjectKeywords extracts potential keywords from email subject
func (l *Learner) suggestSubjectKeywords(ctx context.Context, e *email.Email, confidence float64) error {
	subject := strings.ToLower(e.Subject)

	// Look for recruiting-related phrases
	phrases := extractRecruitingPhrases(subject)

	for _, phrase := range phrases {
		// Check if already exists
		exists, err := l.db.LearnedFilterExists(ctx, database.FilterTypeSubjectKeyword, phrase)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		// Create suggestion
		f := &database.LearnedFilter{
			FilterType:         database.FilterTypeSubjectKeyword,
			Value:              phrase,
			Source:             database.FilterSourceAISuggested,
			FalsePositiveCount: 0,
		}

		if err := l.db.CreateLearnedFilter(ctx, f); err != nil {
			return err
		}
	}

	return nil
}

// LearnFromFeedback learns from user feedback on misclassified emails
func (l *Learner) LearnFromFeedback(ctx context.Context, e *email.Email, isFalsePositive bool) error {
	domain := e.Domain()

	if isFalsePositive {
		// Email was wrongly included - add domain to blacklist
		if domain != "" && !isCommonDomain(domain) {
			exists, err := l.db.LearnedFilterExists(ctx, database.FilterTypeDomainBlacklist, domain)
			if err != nil {
				return err
			}
			if !exists {
				f := &database.LearnedFilter{
					FilterType:         database.FilterTypeDomainBlacklist,
					Value:              domain,
					Source:             database.FilterSourceUser,
					FalsePositiveCount: 1,
				}
				if err := l.db.CreateLearnedFilter(ctx, f); err != nil {
					return err
				}
			}
		}

		// Also check for blacklistable subject patterns
		subject := strings.ToLower(e.Subject)
		blacklistPhrases := extractBlacklistPhrases(subject)
		for _, phrase := range blacklistPhrases {
			exists, err := l.db.LearnedFilterExists(ctx, database.FilterTypeSubjectBlacklist, phrase)
			if err != nil {
				return err
			}
			if !exists {
				f := &database.LearnedFilter{
					FilterType:         database.FilterTypeSubjectBlacklist,
					Value:              phrase,
					Source:             database.FilterSourceUser,
					FalsePositiveCount: 1,
				}
				if err := l.db.CreateLearnedFilter(ctx, f); err != nil {
					return err
				}
			}
		}
	} else {
		// Email was wrongly excluded (false negative) - add domain to whitelist
		if domain != "" && !isCommonDomain(domain) && !isATSDomain(domain) {
			exists, err := l.db.LearnedFilterExists(ctx, database.FilterTypeDomainWhitelist, domain)
			if err != nil {
				return err
			}
			if !exists {
				f := &database.LearnedFilter{
					FilterType:         database.FilterTypeDomainWhitelist,
					Value:              domain,
					Source:             database.FilterSourceUser,
					FalsePositiveCount: 0,
				}
				if err := l.db.CreateLearnedFilter(ctx, f); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// extractRecruitingPhrases finds recruiting-related phrases in text
func extractRecruitingPhrases(text string) []string {
	var phrases []string

	// Look for common recruiting patterns
	patterns := []string{
		"exciting opportunity",
		"perfect fit",
		"great fit",
		"your background",
		"your experience",
		"your profile",
		"reaching out",
		"touch base",
		"quick chat",
		"quick call",
		"open role",
		"open position",
		"new role",
		"new position",
		"career opportunity",
		"job opportunity",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			phrases = append(phrases, pattern)
		}
	}

	return phrases
}

// extractBlacklistPhrases finds newsletter/spam patterns in text
func extractBlacklistPhrases(text string) []string {
	var phrases []string

	patterns := []string{
		"job alert",
		"new jobs",
		"jobs for you",
		"weekly digest",
		"daily digest",
		"newsletter",
		"unsubscribe",
		"view in browser",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			phrases = append(phrases, pattern)
		}
	}

	return phrases
}

// isCommonDomain checks if domain is too common to be useful as a filter
func isCommonDomain(domain string) bool {
	common := map[string]bool{
		"gmail.com":      true,
		"yahoo.com":      true,
		"hotmail.com":    true,
		"outlook.com":    true,
		"icloud.com":     true,
		"protonmail.com": true,
		"mail.com":       true,
	}
	return common[strings.ToLower(domain)]
}

// isATSDomain checks if domain is an ATS (handled separately)
func isATSDomain(domain string) bool {
	return filter.ExtractCompanyFromDomain(domain) == ""
}
