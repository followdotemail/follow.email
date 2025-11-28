-- Migration Rollback: Remove user privacy metadata table
-- This rollback restores privacy data to the users table

-- First, add back the columns to the users table
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

-- Drop trigger
DROP TRIGGER IF EXISTS update_user_privacy_metadata_updated_at ON user_privacy_metadata;

-- Drop the trigger function if no other tables are using it
-- Note: We'll keep the function as it might be used by other tables

-- Drop index
DROP INDEX IF EXISTS idx_user_privacy_metadata_user_id;

-- Drop the table
DROP TABLE IF EXISTS user_privacy_metadata;