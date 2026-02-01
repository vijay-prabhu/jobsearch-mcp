# ADR-004: Local-First LLM with Cloud Fallback

## Status
Accepted

## Context
Email classification requires an LLM for accurate results. Options:
1. Cloud-only (OpenAI, Anthropic)
2. Local-only (Ollama)
3. Hybrid with fallback

Considerations:
- Privacy: job search emails contain sensitive information
- Cost: processing hundreds of emails can be expensive
- Latency: local inference avoids network round-trips
- Accuracy: larger cloud models may be more accurate

## Decision
Default to local LLM (Ollama) with optional cloud fallback (OpenAI).

### Configuration
```toml
[llm]
primary = "ollama"    # or "openai"
fallback = "openai"   # optional

[llm.ollama]
model = "llama3.2:1b"
host = "http://localhost:11434"

[llm.openai]
model = "gpt-4o-mini"
# API key from OPENAI_API_KEY environment variable
```

## Consequences

### Positive
- Privacy by default - emails never leave local machine
- Zero ongoing cost with Ollama
- Works offline
- Fast classification (~1s per email)
- User can opt into cloud for higher accuracy
- Classification caching reduces redundant LLM calls

### Negative
- Requires Ollama installation
- Local models less accurate than GPT-4
- Hardware requirements for local inference

### Model Selection
Default model is `llama3.2:1b` chosen for:
- Small enough to run on any modern laptop
- Fast inference (~1s per email)
- Sufficient accuracy for classification task
- Open source and free

Users with capable hardware can use larger models:
- `llama3.2:3b` - better accuracy
- `llama3.1:8b` - near cloud quality

## Alternatives Considered

### Cloud-Only
- Highest accuracy
- Privacy concerns with job search data
- Ongoing API costs
- Rejected as default, kept as fallback

### Local-Only
- Maximum privacy
- No fallback if local model fails
- Rejected as too restrictive

### Embedding-Based Classification
- Could use local embeddings + classifier
- Less flexible than prompt-based
- Rejected for flexibility
