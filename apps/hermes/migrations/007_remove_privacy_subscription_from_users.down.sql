-- Migration Rollback: Restore privacy columns to users table
-- This rollback restores the columns and migrates data back from separate table

-- Add back the columns to the users table
ALTER TABLE users 
ADD COLUMN gdpr_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN ccpa_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN consent_date TIMESTAMP,
ADD COLUMN data_retention_days INTEGER DEFAULT 2555;

-- Migrate privacy data back to users table
UPDATE users 
SET 
    gdpr_consent = upm.gdpr_consent,
    ccpa_consent = upm.ccpa_consent,
    consent_date = COALESCE(upm.gdpr_consent_date, upm.ccpa_consent_date),
    data_retention_days = upm.data_retention_days
FROM user_privacy_metadata upm
WHERE users.id = upm.user_id;