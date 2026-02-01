# ADR-006: Parallel Processing and Progress Output

## Status
Accepted

## Context
Syncing emails involves multiple slow operations:
1. Fetching email metadata from Gmail API
2. Fetching full email content (when needed)
3. Classifying emails with LLM
4. Processing and storing results

Sequential processing of hundreds of emails is slow:
- Gmail API latency: ~100-200ms per request
- LLM classification: ~1s per email
- A sync of 500 emails could take 10+ minutes

Users need feedback during long operations to understand:
- What phase the sync is in
- How much progress has been made
- Whether the operation is still running

## Decision

### Parallel Processing
Implement concurrent processing with bounded parallelism:

```go
// Gmail fetching: 10 concurrent connections
const concurrentFetches = 10

// LLM classification: 5 concurrent requests
const concurrentClassifications = 5
```

Use semaphore pattern with buffered channels:
```go
sem := make(chan struct{}, concurrentFetches)
for _, id := range messageIDs {
    sem <- struct{}{}
    go func(id string) {
        defer func() { <-sem }()
        // fetch email
    }(id)
}
```

### Progress Callback Architecture
Define progress phases and callback type:

```go
type ProgressPhase string

const (
    PhaseListingEmails   ProgressPhase = "listing"
    PhaseFetchingEmails  ProgressPhase = "fetching"
    PhaseFiltering       ProgressPhase = "filtering"
    PhaseClassifying     ProgressPhase = "classifying"
    PhaseProcessing      ProgressPhase = "processing"
    PhaseUpdatingStatus  ProgressPhase = "updating_status"
)

type Progress struct {
    Phase       ProgressPhase
    Current     int
    Total       int
    Description string
}

type ProgressCallback func(Progress)
```

Pass callbacks through layers:
- CLI → Tracker → Provider
- CLI → Tracker → Classifier

### Terminal-Aware Output
Detect terminal vs non-terminal output:
```go
isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
if isTerminal {
    // Overwrite line with \r\033[K
} else {
    // Print new line only on phase change
}
```

### ETA Calculation
Track start time per phase and calculate remaining time:
```go
func (p Progress) ETA() time.Duration {
    elapsed := time.Since(p.StartedAt)
    rate := float64(p.Current) / elapsed.Seconds()
    remaining := p.Total - p.Current
    return time.Duration(float64(remaining)/rate) * time.Second
}
```

### Visual Enhancements
- **Spinner animation** for indeterminate phases (listing, filtering, updating)
- **ANSI color coding** per phase for visual distinction
- **ETA display** for phases with known progress (fetching, classifying, processing)

## Consequences

### Positive
- **4-5x faster syncs** - 500 emails in ~2 minutes vs 10+
- **User feedback** - Clear progress indication with emoji phases
- **ETA predictions** - Time remaining shown during long operations
- **Visual clarity** - Color coding and spinner animations in terminals
- **Efficient API usage** - Gmail API handles 10 concurrent well
- **LLM protection** - 5 concurrent prevents Ollama overload
- **Works everywhere** - Terminal and non-terminal output modes

### Negative
- **Memory usage** - Concurrent goroutines hold more emails in memory
- **Error handling** - Need to aggregate errors from parallel operations
- **Rate limiting** - Gmail API has quotas (but 10 concurrent is safe)

### Concurrency Limits Rationale

**Gmail: 10 concurrent**
- Gmail API quota: 250 quota units/user/second
- MessageGet uses 5 units → 50 messages/second possible
- 10 concurrent is conservative, leaves room for bursts
- Higher values risk rate limiting

**LLM: 5 concurrent**
- Ollama runs on local hardware with limited resources
- 5 concurrent balances throughput vs resource usage
- Prevents memory pressure on 8GB machines
- OpenAI fallback could handle more, but consistent limits are simpler

## Alternatives Considered

### Unbounded Parallelism
- Maximum speed
- Risk of rate limiting and memory exhaustion
- Rejected for stability

### Streaming Progress (MCP)
- Send progress updates via MCP protocol
- More complex implementation
- Deferred to future enhancement

### Batch API Calls
- Gmail batch API for multiple requests
- More complex error handling
- Rejected for simplicity, parallel individual calls sufficient
