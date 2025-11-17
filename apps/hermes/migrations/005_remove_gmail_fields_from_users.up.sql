-- Migration: Remove Gmail sync fields from users table
-- These fields have been moved to provider_sync_metadata table

-- Remove Gmail sync columns from users table
ALTER TABLE users 
DROP COLUMN IF EXISTS gmail_consent,
DROP COLUMN IF EXISTS gmail_consent_date,
DROP COLUMN IF EXISTS gmail_sync_enabled,
DROP COLUMN IF EXISTS last_gmail_sync_at,
DROP COLUMN IF EXISTS gmail_history_id;