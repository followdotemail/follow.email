-- Migration rollback: Remove gmail_consent table
-- This rollback restores Gmail sync data to the users table

-- Add Gmail sync columns back to users table
ALTER TABLE users 
ADD COLUMN gmail_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN gmail_consent_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN gmail_sync_enabled BOOLEAN DEFAULT FALSE,
ADD COLUMN last_gmail_sync_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN gmail_history_id VARCHAR(255) DEFAULT '';

-- Migrate data back from gmail_consent to users table
UPDATE users 
SET 
    gmail_consent = gc.gmail_consent,
    gmail_consent_date = gc.gmail_consent_date,
    gmail_sync_enabled = gc.gmail_sync_enabled,
    last_gmail_sync_at = gc.last_gmail_sync_at,
    gmail_history_id = gc.gmail_history_id,
    updated_at = NOW()
FROM gmail_consent gc 
WHERE users.id = gc.user_id;

-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_gmail_consent_updated_at ON gmail_consent;
DROP FUNCTION IF EXISTS update_gmail_consent_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_gmail_consent_user_id;
DROP INDEX IF EXISTS idx_gmail_consent_enabled;

-- Drop the gmail_consent table
DROP TABLE IF EXISTS gmail_consent;

