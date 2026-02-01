-- Additional performance indexes

-- Compound index for filtered list queries (status + archived)
CREATE INDEX IF NOT EXISTS idx_conversations_status_archived
ON conversations(status, archived);

-- Index for recruiter email grouping
CREATE INDEX IF NOT EXISTS idx_conversations_recruiter_email
ON conversations(recruiter_email);

-- Index for email from address (used in filtering)
CREATE INDEX IF NOT EXISTS idx_emails_from_address
ON emails(from_address);

-- Index for learned filters lookup
CREATE INDEX IF NOT EXISTS idx_learned_filters_type_source
ON learned_filters(filter_type, source);
