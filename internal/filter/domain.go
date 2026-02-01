package filter

import (
	"strings"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// getRelevantAddress returns the address to check for filtering.
// For inbound emails, we check the From address.
// For outbound emails (sent by the user), we check the first To address.
func (f *Filter) getRelevantAddress(e *email.Email) (domain, fullEmail string) {
	// Check if this is an outbound email (sent by the user)
	if f.userEmail != "" && strings.EqualFold(e.From.Email, f.userEmail) {
		// Outbound: check the recipient (who we're emailing)
		if len(e.To) > 0 {
			return e.To[0].Domain(), strings.ToLower(e.To[0].Email)
		}
		// No recipients - can't determine domain
		return "", ""
	}
	// Inbound: check the sender
	return e.Domain(), strings.ToLower(e.From.Email)
}

// checkDomainWhitelist checks if the email is from/to a whitelisted domain
func (f *Filter) checkDomainWhitelist(e *email.Email) *Result {
	domain, relevantEmail := f.getRelevantAddress(e)
	if domain == "" {
		return nil
	}

	// Check config + learned whitelist
	for _, pattern := range f.GetAllDomainWhitelist() {
		pattern = strings.ToLower(pattern)

		// Check if pattern matches domain or is contained in email
		if matchesDomainPattern(domain, relevantEmail, pattern) {
			return &Result{
				Include:    true,
				Layer:      LayerWhitelist,
				Confidence: 1.0,
				Reason:     "Whitelisted domain: " + pattern,
			}
		}
	}

	return nil
}

// checkDomainBlacklist checks if the email is from/to a blacklisted domain/sender
func (f *Filter) checkDomainBlacklist(e *email.Email) *Result {
	domain, relevantEmail := f.getRelevantAddress(e)
	if domain == "" {
		return nil
	}

	// Check config + learned blacklist
	for _, pattern := range f.GetAllDomainBlacklist() {
		pattern = strings.ToLower(pattern)

		// Check if pattern matches domain or is contained in email
		if matchesDomainPattern(domain, relevantEmail, pattern) {
			return &Result{
				Include:    false,
				Layer:      LayerBlacklist,
				Confidence: 1.0,
				Reason:     "Blacklisted sender: " + pattern,
			}
		}
	}

	return nil
}

// matchesDomainPattern checks if a pattern matches the domain or email
func matchesDomainPattern(domain, fullEmail, pattern string) bool {
	// Exact domain match
	if domain == pattern {
		return true
	}

	// Pattern is a subdomain (e.g., "mail.example.com" matches "example.com")
	if strings.HasSuffix(domain, "."+pattern) {
		return true
	}

	// Pattern contains @ - it's a specific email pattern
	if strings.Contains(pattern, "@") {
		// Exact email match
		if fullEmail == pattern {
			return true
		}
		// Prefix match (e.g., "noreply@" matches any "noreply@*")
		if strings.HasSuffix(pattern, "@") && strings.HasPrefix(fullEmail, pattern) {
			return true
		}
	}

	// Pattern is contained in domain (e.g., "greenhouse" matches "greenhouse.io")
	if strings.Contains(domain, strings.TrimSuffix(pattern, ".")) {
		return true
	}

	return false
}

// ExtractCompanyFromDomain attempts to extract company name from domain
func ExtractCompanyFromDomain(domain string) string {
	// Remove common suffixes
	suffixes := []string{
		".com", ".io", ".co", ".net", ".org", ".ai", ".app",
		".jobs", ".careers", ".work", ".hire",
	}

	name := domain
	for _, suffix := range suffixes {
		name = strings.TrimSuffix(name, suffix)
	}

	// Remove common prefixes
	prefixes := []string{
		"mail.", "email.", "jobs.", "careers.", "recruiting.", "talent.",
		"hr.", "hire.", "apply.", "www.",
	}

	for _, prefix := range prefixes {
		name = strings.TrimPrefix(name, prefix)
	}

	// Handle known ATS domains
	atsDomains := map[string]bool{
		"greenhouse":      true,
		"lever":           true,
		"ashbyhq":         true,
		"smartrecruiters": true,
		"workday":         true,
		"myworkdayjobs":   true,
		"icims":           true,
		"taleo":           true,
		"jobvite":         true,
		"breezy":          true,
	}

	if atsDomains[name] {
		return "" // ATS domain, company name should come from email content
	}

	// Capitalize first letter
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}
