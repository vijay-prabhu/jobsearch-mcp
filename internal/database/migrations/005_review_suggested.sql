-- Migration 005: Add review_suggested flag for confidence tiering
-- Conversations with medium confidence (0.6-0.8) are flagged for review

ALTER TABLE conversations ADD COLUMN review_suggested BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_conversations_review_suggested ON conversations(review_suggested);
