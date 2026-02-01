# ADR 007: Classification Validation Pipeline

## Status
Accepted

## Date
2026-01-31

## Context

The email classification system was experiencing a significant false positive rate (~21%, or 27 out of 126 conversations). Non-job-related emails such as retail promotions (Walmart), travel notifications (Qatar Airways), entertainment (Cineplex), and marketing newsletters were being incorrectly classified as job-related.

### Root Causes Identified

1. **Keyword overlap**: Marketing emails contain job-related keywords like "opportunity", "career", "exclusive offer"
2. **Single-pass classification**: One LLM call with no validation
3. **No learning mechanism**: Same domains repeatedly produce false positives
4. **No user feedback loop**: No way to correct mistakes and learn from them

### Approaches Considered

| Approach | Problem Solved | Pros | Cons |
|----------|---------------|------|------|
| LLM Validation Pass | Wrong classifications | Catches errors | 2x API cost |
| Multi-prompt Ensemble | Prompt bias | More robust | 3x API cost |
| Domain Learning | Repeat offenders | Zero cost after learning | Needs initial data |
| Confidence Tiering | Uncertainty handling | Low cost | Needs calibration |
| User Feedback Loop | No ground truth | Learns from corrections | Requires user effort |
| Embedding Clustering | Unknown patterns | Finds anomalies | Complex to implement |

### Key Insight

These approaches are **not mutually exclusive**—they solve different problems at different stages of the classification pipeline. A hybrid approach uses each technique where it's most effective.

### Structured Single Call vs Multi-prompt Ensemble

Instead of 3 separate LLM calls (ensemble), we can achieve similar accuracy with a single structured call that asks multiple classification questions:

```json
{
  "is_direct_opportunity": true/false,
  "is_recruiter_outreach": true/false,
  "is_interview_related": true/false,
  "is_job_alert_newsletter": true/false,
  "is_marketing_promo": true/false,
  "reasoning": "...",
  "confidence": 0.0-1.0
}
```

This provides ensemble-like signal at single-call cost.

## Decision

Implement a **5-stage classification validation pipeline**:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        EMAIL CLASSIFICATION PIPELINE                         │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────────────────┤
│  PRE-FILTER │   CLASSIFY  │  VALIDATE   │   DECIDE    │       LEARN         │
│  (Stage 1)  │  (Stage 2)  │  (Stage 3)  │  (Stage 4)  │     (Stage 5)       │
├─────────────┼─────────────┼─────────────┼─────────────┼─────────────────────┤
│   Domain    │    LLM      │  Structured │ Confidence  │  User Feedback      │
│  Blacklist  │ Classifier  │  Validation │  Tiering    │  + Domain Learning  │
│             │             │             │             │  + Embedding Audit  │
├─────────────┼─────────────┼─────────────┼─────────────┼─────────────────────┤
│   Cost: 0   │  Cost: 1    │ Cost: 0-1   │   Cost: 0   │   Cost: Periodic    │
│             │  LLM call   │  LLM call   │             │                     │
└─────────────┴─────────────┴─────────────┴─────────────┴─────────────────────┘
```

### Stage 1: Pre-filter (Zero LLM cost)
- Check domain against whitelist → auto-include
- Check domain against blacklist → auto-exclude
- Check domain against learned_blacklist → auto-exclude

### Stage 2: Initial Classification (1 LLM call)
- Existing LLM classifier
- Returns: is_job_related, confidence, extracted metadata

### Stage 3: Conditional Validation (0-1 LLM calls)
- Skip if confidence >= 0.85 AND known recruiter domain
- Run structured validation if confidence < 0.7 OR unfamiliar domain
- Structured validation returns multi-signal classification

### Stage 4: Confidence Tiering (Zero cost)
- High confidence (>0.8): Auto-include
- Medium confidence (0.6-0.8): Include, flag for review
- Low confidence (<0.6): Exclude, log for audit

### Stage 5: Learning Loop (Async/Periodic)
- User feedback: `mark-spam` and `mark-missed` commands
- Domain learning: Auto-blacklist after 3+ false positives from same domain
- Embedding audit: Periodic clustering to find anomalies (future)

## Implementation Plan

### Phase 1: Foundation (Priority: High)
1. Expand domain blacklist with known false positive domains
2. Add `learned_filters` table for persistent learned blacklist
3. Implement `mark-spam` CLI command
4. Implement domain learning (auto-blacklist threshold)

### Phase 2: Validation (Priority: High)
5. Implement structured validation prompt
6. Add conditional validation logic (based on confidence)
7. Update classifier client to support validation pass

### Phase 3: Tiering & Metrics (Priority: Medium)
8. Add confidence tiering logic
9. Add `review_suggested` flag to conversations
10. Track classification metrics (precision, recall estimates)

### Phase 4: Advanced Learning (Priority: Low)
11. Implement `mark-missed` for false negatives
12. Add embedding-based audit (requires embedding model)
13. Dashboard for classification quality metrics

## Consequences

### Positive
- Reduced false positive rate (target: <5%)
- Self-improving system through user feedback
- Minimal additional LLM cost (average ~1.3 calls vs 1.0)
- No manual blacklist maintenance required

### Negative
- Increased code complexity
- Requires user participation for feedback loop
- Validation adds latency for uncertain emails

### Risks
- Cold start: Learned blacklist empty initially
- Over-learning: Aggressive blacklisting might cause false negatives
- Mitigation: Require 3+ false positives before auto-blacklist

## Metrics

Track these metrics to measure success:
- False positive rate (target: <5%)
- False negative rate (target: <2%)
- Average LLM calls per email
- User feedback volume (mark-spam/mark-missed usage)
- Auto-blacklisted domains count

## References
- [ADR 003: Multi-layer Email Filtering](003-multi-layer-email-filtering.md)
- [ADR 004: Local-first LLM](004-local-first-llm.md)
