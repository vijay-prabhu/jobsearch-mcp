package tracker

import "time"

// ProgressPhase represents the current sync phase
type ProgressPhase string

const (
	PhaseListingEmails  ProgressPhase = "listing"
	PhaseFetchingEmails ProgressPhase = "fetching"
	PhaseFiltering      ProgressPhase = "filtering"
	PhaseClassifying    ProgressPhase = "classifying"
	PhaseValidating     ProgressPhase = "validating"
	PhaseProcessing     ProgressPhase = "processing"
	PhaseUpdatingStatus ProgressPhase = "updating_status"
)

// Progress represents the current sync progress
type Progress struct {
	Phase       ProgressPhase
	Current     int       // Current item being processed
	Total       int       // Total items in this phase
	Description string    // Human-readable description
	StartedAt   time.Time // When this phase started (for ETA calculation)
}

// ProgressCallback is called with progress updates during sync
type ProgressCallback func(Progress)

// ETA returns the estimated time remaining based on current progress
func (p Progress) ETA() time.Duration {
	if p.Current == 0 || p.Total == 0 || p.StartedAt.IsZero() {
		return 0
	}
	elapsed := time.Since(p.StartedAt)
	rate := float64(p.Current) / elapsed.Seconds()
	if rate <= 0 {
		return 0
	}
	remaining := p.Total - p.Current
	return time.Duration(float64(remaining)/rate) * time.Second
}

// Percentage returns the completion percentage (0-100)
func (p Progress) Percentage() int {
	if p.Total == 0 {
		return 0
	}
	return (p.Current * 100) / p.Total
}
