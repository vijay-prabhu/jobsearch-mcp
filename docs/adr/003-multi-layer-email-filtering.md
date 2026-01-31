# ADR-003: Multi-Layer Email Filtering

## Status
Accepted

## Context
Identifying job-related emails from a Gmail inbox is challenging:
- High false positive rate from job alerts, newsletters, LinkedIn notifications
- Legitimate recruiter emails vary widely in format
- LLM classification is expensive and slow
- Users have different filtering preferences

## Decision
Implement a multi-layer filtering pipeline that processes emails through successive filters:

```
Email → Whitelist → Blacklist → Keywords → LLM Classification
```

Each layer can include, exclude, or pass to next layer.

### Layer Details

1. **Whitelist (instant include)**
   - Known good domains: greenhouse.io, lever.co, ashbyhq.com
   - Specific recruiter email addresses
   - Bypasses all other filters

2. **Blacklist (instant exclude)**
   - Newsletter domains: substack.com, tldrnewsletter.com
   - Notification addresses: noreply@linkedin.com
   - Subject patterns: "job alert", "weekly digest"

3. **Keyword Scoring**
   - Subject keywords: "opportunity", "role", "position"
   - Body keywords: "your background", "schedule a call"
   - Score > threshold → include

4. **LLM Classification (final arbiter)**
   - Only called if previous layers are inconclusive
   - Extracts company, position, recruiter name
   - Most expensive but most accurate

## Consequences

### Positive
- Fast rejection of obvious non-job emails
- Reduces LLM API calls (cost and latency)
- Users can customize filters without touching LLM
- Graceful degradation if LLM unavailable
- Deterministic behavior for whitelisted/blacklisted

### Negative
- More configuration surface area
- False negatives possible if blacklist too aggressive
- Keyword matching is language-dependent

### Configuration
All layers configurable in `~/.config/jobsearch/config.toml`:

```toml
[filter]
domain_whitelist = ["greenhouse.io", "lever.co"]
domain_blacklist = ["substack.com", "noreply@linkedin.com"]
subject_blacklist = ["job alert", "weekly digest"]
subject_keywords = ["opportunity", "role", "position"]
body_keywords = ["your background", "schedule a call"]
```

## Alternatives Considered

### LLM-Only Filtering
- Simpler implementation
- High latency and cost for every email
- Rejected for performance

### Rule-Based Only
- No LLM needed
- Poor accuracy for edge cases
- Rejected for accuracy
