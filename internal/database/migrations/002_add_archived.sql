-- Add archived column to conversations
ALTER TABLE conversations ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;

-- Index for filtering archived conversations
CREATE INDEX IF NOT EXISTS idx_conversations_archived ON conversations(archived);
