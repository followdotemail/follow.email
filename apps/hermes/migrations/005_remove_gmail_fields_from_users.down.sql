-- Migration rollback: Restore Gmail sync fields to users table
-- This restores the Gmail sync columns that were removed

-- Add Gmail sync columns back to users table
ALTER TABLE users 
ADD COLUMN gmail_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN gmail_consent_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN gmail_sync_enabled BOOLEAN DEFAULT FALSE,
ADD COLUMN last_gmail_sync_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN gmail_history_id VARCHAR(255) DEFAULT '';

-- Migrate data back from provider_sync_metadata to users table
UPDATE users 
SET 
    gmail_consent = psm.consent_granted,
    gmail_consent_date = psm.consent_date,
    gmail_sync_enabled = psm.sync_enabled,
    last_gmail_sync_at = psm.last_sync_at,
    gmail_history_id = COALESCE(psm.history_id, ''),
    updated_at = NOW()
FROM provider_sync_metadata psm 
WHERE users.id = psm.user_id 
  AND psm.provider = 'gmail';