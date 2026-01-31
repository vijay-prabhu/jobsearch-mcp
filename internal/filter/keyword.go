package filter

import (
	"fmt"
	"strings"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// checkSubjectBlacklist checks if the subject matches any blacklisted patterns
func (f *Filter) checkSubjectBlacklist(e *email.Email) *Result {
	subjectLower := strings.ToLower(e.Subject)

	// Check config + learned blacklist
	for _, pattern := range f.GetAllSubjectBlacklist() {
		pattern = strings.ToLower(pattern)

		if strings.Contains(subjectLower, pattern) {
			return &Result{
				Include:    false,
				Layer:      LayerBlacklist,
				Confidence: 0.9,
				Reason:     fmt.Sprintf("Subject matches blacklist pattern: %q", pattern),
			}
		}
	}

	return nil
}

// scoreKeywords calculates a relevance score based on keyword matches
func (f *Filter) scoreKeywords(e *email.Email) Result {
	subjectLower := strings.ToLower(e.Subject)
	bodyLower := strings.ToLower(e.Body)
	snippetLower := strings.ToLower(e.Snippet)

	// Use snippet if body is empty
	if bodyLower == "" {
		bodyLower = snippetLower
	}

	// Get combined keyword lists
	subjectKeywords := f.GetAllSubjectKeywords()
	bodyKeywords := f.GetAllBodyKeywords()

	// Count subject keyword matches
	subjectMatches := 0
	for _, kw := range subjectKeywords {
		if containsWord(subjectLower, strings.ToLower(kw)) {
			subjectMatches++
		}
	}

	// Count body keyword matches
	bodyMatches := 0
	for _, kw := range bodyKeywords {
		if containsWord(bodyLower, strings.ToLower(kw)) {
			bodyMatches++
		}
	}

	// Calculate score
	score := f.scorer.Calculate(
		subjectMatches, len(subjectKeywords),
		bodyMatches, len(bodyKeywords),
	)

	// Determine result based on score
	if score >= f.scorer.config.IncludeThreshold {
		return Result{
			Include:    true,
			Layer:      LayerKeyword,
			Confidence: score,
			Reason:     fmt.Sprintf("Keyword score: %.0f%% (subject: %d, body: %d)", score*100, subjectMatches, bodyMatches),
		}
	}

	if score >= f.scorer.config.UncertainMin {
		return Result{
			Include:    false,
			Layer:      LayerUncertain,
			Confidence: score,
			Reason:     fmt.Sprintf("Uncertain - keyword score: %.0f%% (needs LLM)", score*100),
		}
	}

	return Result{
		Include:    false,
		Layer:      LayerRejected,
		Confidence: 1.0 - score,
		Reason:     fmt.Sprintf("Low keyword score: %.0f%%", score*100),
	}
}

// containsWord checks if text contains the word (with word boundary awareness)
func containsWord(text, word string) bool {
	// Simple contains for multi-word phrases
	if strings.Contains(word, " ") {
		return strings.Contains(text, word)
	}

	// For single words, check for word boundaries
	// This prevents "position" from matching "preposition"
	idx := strings.Index(text, word)
	if idx == -1 {
		return false
	}

	// Check character before (if exists)
	if idx > 0 {
		before := text[idx-1]
		if isWordChar(before) {
			// Try to find another occurrence
			return containsWord(text[idx+len(word):], word)
		}
	}

	// Check character after (if exists)
	endIdx := idx + len(word)
	if endIdx < len(text) {
		after := text[endIdx]
		if isWordChar(after) {
			return containsWord(text[idx+len(word):], word)
		}
	}

	return true
}

// isWordChar returns true for alphanumeric characters
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// ExtractKeywordContext extracts context around matched keywords
func ExtractKeywordContext(text string, keywords []string, contextSize int) []string {
	textLower := strings.ToLower(text)
	var contexts []string

	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		idx := strings.Index(textLower, kwLower)
		if idx == -1 {
			continue
		}

		// Extract context around the keyword
		start := maxInt(0, idx-contextSize)
		end := minInt(len(text), idx+len(kw)+contextSize)

		context := text[start:end]
		if start > 0 {
			context = "..." + context
		}
		if end < len(text) {
			context = context + "..."
		}

		contexts = append(contexts, context)
	}

	return contexts
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
