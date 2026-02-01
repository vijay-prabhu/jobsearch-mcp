package filter

import (
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// Layer identifies which filtering layer made the decision
type Layer string

const (
	LayerWhitelist Layer = "whitelist"
	LayerBlacklist Layer = "blacklist"
	LayerKeyword   Layer = "keyword"
	LayerUncertain Layer = "uncertain"
	LayerRejected  Layer = "rejected"
	LayerLLM       Layer = "llm"
)

// Result represents the outcome of filtering an email
type Result struct {
	Include    bool    // Whether to include this email
	Layer      Layer   // Which layer made the decision
	Confidence float64 // Confidence in the decision (0.0-1.0)
	Reason     string  // Human-readable reason
}

// FilteredEmail combines an email with its filter result
type FilteredEmail struct {
	Email  email.Email
	Result Result
}

// Filter applies multi-layer filtering to emails
type Filter struct {
	config    config.FilterConfig
	scorer    *Scorer
	userEmail string // User's email for detecting outbound emails

	// Learned filters (added at runtime)
	learnedDomainWhitelist  []string
	learnedDomainBlacklist  []string
	learnedSubjectBlacklist []string
	learnedSubjectKeywords  []string
	learnedBodyKeywords     []string
}

// New creates a new Filter with the given configuration
func New(cfg config.FilterConfig) *Filter {
	return &Filter{
		config: cfg,
		scorer: NewScorer(ScorerConfig{
			SubjectWeight:    2.0, // Subject keywords weighted higher
			BodyWeight:       1.0,
			IncludeThreshold: 0.3,  // Include if score >= 30%
			UncertainMin:     0.02, // Uncertain if score >= 2% (let LLM decide)
		}),
	}
}

// SetUserEmail sets the user's email for detecting outbound emails
func (f *Filter) SetUserEmail(email string) {
	f.userEmail = email
}

// AddLearnedFilters adds learned filters to the filter configuration
func (f *Filter) AddLearnedFilters(filterType string, values []string) {
	switch filterType {
	case "domain_whitelist":
		f.learnedDomainWhitelist = append(f.learnedDomainWhitelist, values...)
	case "domain_blacklist":
		f.learnedDomainBlacklist = append(f.learnedDomainBlacklist, values...)
	case "subject_blacklist":
		f.learnedSubjectBlacklist = append(f.learnedSubjectBlacklist, values...)
	case "subject_keyword":
		f.learnedSubjectKeywords = append(f.learnedSubjectKeywords, values...)
	case "body_keyword":
		f.learnedBodyKeywords = append(f.learnedBodyKeywords, values...)
	}
}

// GetAllDomainWhitelist returns config + learned domain whitelist
func (f *Filter) GetAllDomainWhitelist() []string {
	return append(f.config.DomainWhitelist, f.learnedDomainWhitelist...)
}

// GetAllDomainBlacklist returns config + learned domain blacklist
func (f *Filter) GetAllDomainBlacklist() []string {
	return append(f.config.DomainBlacklist, f.learnedDomainBlacklist...)
}

// GetAllSubjectBlacklist returns config + learned subject blacklist
func (f *Filter) GetAllSubjectBlacklist() []string {
	return append(f.config.SubjectBlacklist, f.learnedSubjectBlacklist...)
}

// GetAllSubjectKeywords returns config + learned subject keywords
func (f *Filter) GetAllSubjectKeywords() []string {
	return append(f.config.SubjectKeywords, f.learnedSubjectKeywords...)
}

// GetAllBodyKeywords returns config + learned body keywords
func (f *Filter) GetAllBodyKeywords() []string {
	return append(f.config.BodyKeywords, f.learnedBodyKeywords...)
}

// Apply runs the email through the filtering pipeline
func (f *Filter) Apply(e *email.Email) Result {
	// Layer 1: Domain whitelist (auto-include)
	if result := f.checkDomainWhitelist(e); result != nil {
		return *result
	}

	// Layer 2a: Domain blacklist (auto-exclude)
	if result := f.checkDomainBlacklist(e); result != nil {
		return *result
	}

	// Layer 2b: Subject blacklist (auto-exclude)
	if result := f.checkSubjectBlacklist(e); result != nil {
		return *result
	}

	// Layer 3: Keyword scoring
	return f.scoreKeywords(e)
}

// ApplyBatch applies filtering to multiple emails
func (f *Filter) ApplyBatch(emails []email.Email) []FilteredEmail {
	results := make([]FilteredEmail, 0, len(emails))

	for _, e := range emails {
		result := f.Apply(&e)
		results = append(results, FilteredEmail{
			Email:  e,
			Result: result,
		})
	}

	return results
}

// FilterIncluded returns only emails that should be included
func FilterIncluded(filtered []FilteredEmail) []FilteredEmail {
	var included []FilteredEmail
	for _, f := range filtered {
		if f.Result.Include {
			included = append(included, f)
		}
	}
	return included
}

// FilterUncertain returns emails that need LLM classification
func FilterUncertain(filtered []FilteredEmail) []FilteredEmail {
	var uncertain []FilteredEmail
	for _, f := range filtered {
		if f.Result.Layer == LayerUncertain {
			uncertain = append(uncertain, f)
		}
	}
	return uncertain
}

// Stats returns filtering statistics
type Stats struct {
	Total       int
	Whitelisted int
	Blacklisted int
	ByKeyword   int
	Uncertain   int
	Rejected    int
}

// GetStats returns statistics about filtered emails
func GetStats(filtered []FilteredEmail) Stats {
	stats := Stats{Total: len(filtered)}

	for _, f := range filtered {
		switch f.Result.Layer {
		case LayerWhitelist:
			stats.Whitelisted++
		case LayerBlacklist:
			stats.Blacklisted++
		case LayerKeyword:
			stats.ByKeyword++
		case LayerUncertain:
			stats.Uncertain++
		case LayerRejected:
			stats.Rejected++
		}
	}

	return stats
}
