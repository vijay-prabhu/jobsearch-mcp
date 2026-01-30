package filter

// ScorerConfig configures the keyword scoring algorithm
type ScorerConfig struct {
	SubjectWeight    float64 // Weight for subject keyword matches
	BodyWeight       float64 // Weight for body keyword matches
	IncludeThreshold float64 // Score threshold to auto-include
	UncertainMin     float64 // Minimum score to be considered uncertain (vs rejected)
}

// Scorer calculates relevance scores based on keyword matches
type Scorer struct {
	config ScorerConfig
}

// NewScorer creates a new Scorer with the given configuration
func NewScorer(config ScorerConfig) *Scorer {
	return &Scorer{config: config}
}

// Calculate computes a normalized score based on keyword matches
func (s *Scorer) Calculate(subjectMatches, totalSubjectKeywords, bodyMatches, totalBodyKeywords int) float64 {
	if totalSubjectKeywords == 0 && totalBodyKeywords == 0 {
		return 0
	}

	// Calculate weighted score
	subjectScore := 0.0
	if totalSubjectKeywords > 0 {
		subjectScore = float64(subjectMatches) / float64(totalSubjectKeywords)
	}

	bodyScore := 0.0
	if totalBodyKeywords > 0 {
		bodyScore = float64(bodyMatches) / float64(totalBodyKeywords)
	}

	// Weighted average
	totalWeight := s.config.SubjectWeight + s.config.BodyWeight
	weightedScore := (subjectScore*s.config.SubjectWeight + bodyScore*s.config.BodyWeight) / totalWeight

	// Apply bonus for having matches in both subject and body
	if subjectMatches > 0 && bodyMatches > 0 {
		weightedScore = min(1.0, weightedScore*1.2)
	}

	return weightedScore
}

// Explain returns a human-readable explanation of the score
func (s *Scorer) Explain(subjectMatches, totalSubjectKeywords, bodyMatches, totalBodyKeywords int) string {
	score := s.Calculate(subjectMatches, totalSubjectKeywords, bodyMatches, totalBodyKeywords)

	switch {
	case score >= s.config.IncludeThreshold:
		return "high relevance - likely job-related"
	case score >= s.config.UncertainMin:
		return "moderate relevance - needs review"
	default:
		return "low relevance - likely not job-related"
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
