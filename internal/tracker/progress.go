package tracker

// ProgressPhase represents the current sync phase
type ProgressPhase string

const (
	PhaseListingEmails   ProgressPhase = "listing"
	PhaseFetchingEmails  ProgressPhase = "fetching"
	PhaseFiltering       ProgressPhase = "filtering"
	PhaseClassifying     ProgressPhase = "classifying"
	PhaseProcessing      ProgressPhase = "processing"
	PhaseUpdatingStatus  ProgressPhase = "updating_status"
)

// Progress represents the current sync progress
type Progress struct {
	Phase       ProgressPhase
	Current     int    // Current item being processed
	Total       int    // Total items in this phase
	Description string // Human-readable description
}

// ProgressCallback is called with progress updates during sync
type ProgressCallback func(Progress)
