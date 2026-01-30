-- JobSearch MCP Database Schema
-- PostgreSQL-compatible schema using SQLite

-- Conversations (threads grouped by context)
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    company TEXT NOT NULL,
    position TEXT,
    recruiter_name TEXT,
    recruiter_email TEXT,
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN (
        'active',
        'waiting_on_me',
        'waiting_on_them',
        'stale',
        'closed'
    )),
    last_activity_at TIMESTAMP NOT NULL,
    email_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Individual emails within conversations
CREATE TABLE IF NOT EXISTS emails (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    gmail_id TEXT UNIQUE NOT NULL,
    thread_id TEXT NOT NULL,
    subject TEXT,
    from_address TEXT NOT NULL,
    from_name TEXT,
    to_address TEXT,
    date TIMESTAMP NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    snippet TEXT,
    body_stored BOOLEAN NOT NULL DEFAULT FALSE,
    body_encrypted TEXT,
    classification TEXT,
    confidence REAL,
    extracted_data TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Sync state tracking
CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_sync_at TIMESTAMP,
    last_history_id TEXT,
    emails_processed INTEGER NOT NULL DEFAULT 0
);

-- Filter learning (future: AI-suggested keywords)
CREATE TABLE IF NOT EXISTS learned_filters (
    id TEXT PRIMARY KEY,
    filter_type TEXT NOT NULL CHECK (filter_type IN (
        'domain_whitelist',
        'domain_blacklist',
        'subject_keyword',
        'body_keyword'
    )),
    value TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('user', 'ai_suggested', 'ai_confirmed')),
    confidence REAL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(filter_type, value)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(status);
CREATE INDEX IF NOT EXISTS idx_conversations_company ON conversations(company);
CREATE INDEX IF NOT EXISTS idx_conversations_last_activity ON conversations(last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_emails_conversation ON emails(conversation_id);
CREATE INDEX IF NOT EXISTS idx_emails_thread ON emails(thread_id);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_gmail_id ON emails(gmail_id);

-- Initialize sync state with singleton row
INSERT OR IGNORE INTO sync_state (id, emails_processed) VALUES (1, 0);
