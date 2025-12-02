-- Migration: Separate Gmail categories from labels
-- This migration properly distinguishes between Gmail categories, system labels and user labels.

-- Add Clerk-related columns to users table
ALTER TABLE emails
    ADD COLUMN IF NOT EXISTS category VARCHAR(50), -- Single category: "personal", "social", "promotions", "updates", "forums".
    ADD COLUMN IF NOT EXISTS system_labels JSONB DEFAULT '[]', -- ["INBOX", "SPAM", "TRASH", "DRAFTS", "SENT", "STARRED", "UNREAD", "ARCHIVED", "IMPORTANT", "UNSEEN"]
    ADD COLUMN IF NOT EXISTS user_labels JSONB DEFAULT '[]';  -- ["uuid1", "uuid2"] (references user_labels table)


CREATE TABLE IF NOT EXISTS user_labels (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    gmail_label_id VARCHAR(255) NOT NULL,
    label_name VARCHAR(255) NOT NULL,
    color JSON NULL,
    message_list_visibility VARCHAR(50) NULL,       -- From Gmail API
    label_list_visibility VARCHAR(50) NULL,         -- From Gmail API
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, gmail_label_id)
);

CREATE INDEX IF NOT EXISTS idx_user_labels_user_id ON user_labels(user_id);