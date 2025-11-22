-- Migration rollback: Remove sync metadata tables
-- This migration removes the sync metadata tables and restores data to users table if needed

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_provider_sync_metadata_updated_at ON provider_sync_metadata;
DROP TRIGGER IF EXISTS trigger_sync_jobs_updated_at ON sync_jobs;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_provider_sync_metadata_updated_at();
DROP FUNCTION IF EXISTS update_sync_jobs_updated_at();

-- Before dropping tables, migrate critical data back to users table if needed
-- Update users table with the latest sync metadata from provider_sync_metadata
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

-- Drop indexes
DROP INDEX IF EXISTS idx_sync_jobs_user_provider_status;
DROP INDEX IF EXISTS idx_sync_jobs_queued_at;
DROP INDEX IF EXISTS idx_sync_jobs_status;
DROP INDEX IF EXISTS idx_sync_jobs_job_id;
DROP INDEX IF EXISTS idx_sync_jobs_provider;
DROP INDEX IF EXISTS idx_sync_jobs_user_id;

DROP INDEX IF EXISTS idx_provider_sync_metadata_auto_sync;
DROP INDEX IF EXISTS idx_provider_sync_metadata_last_sync;
DROP INDEX IF EXISTS idx_provider_sync_metadata_provider;
DROP INDEX IF EXISTS idx_provider_sync_metadata_user_id;

-- Drop tables
DROP TABLE IF EXISTS sync_jobs;
DROP TABLE IF EXISTS provider_sync_metadata;