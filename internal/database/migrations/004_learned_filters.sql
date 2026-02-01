-- Migration 004: Add learned_filters table for auto-learning blacklist/whitelist
-- This table stores user feedback and auto-learned filter entries

CREATE TABLE IF NOT EXISTS learned_filters (
    id TEXT PRIMARY KEY,
    filter_type TEXT NOT NULL,  -- 'domain_blacklist', 'domain_whitelist', 'subject_blacklist'
    value TEXT NOT NULL,        -- The domain or pattern to filter
    source TEXT NOT NULL,       -- 'user' (manual feedback) or 'auto' (learned from patterns)
    false_positive_count INTEGER DEFAULT 0,  -- Number of times marked as false positive
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(filter_type, value)
);

-- Index for fast lookups during filtering
CREATE INDEX IF NOT EXISTS idx_learned_filters_type_value ON learned_filters(filter_type, value);

-- Table to track classification metrics over time
CREATE TABLE IF NOT EXISTS classification_metrics (
    id TEXT PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    emails_processed INTEGER DEFAULT 0,
    auto_included INTEGER DEFAULT 0,       -- High confidence, no validation needed
    validated INTEGER DEFAULT 0,           -- Required validation pass
    excluded INTEGER DEFAULT 0,            -- Filtered out
    false_positives_marked INTEGER DEFAULT 0,  -- User marked as spam
    domains_auto_blacklisted INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_classification_metrics_date ON classification_metrics(date);
