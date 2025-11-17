-- Migration Rollback: Restore privacy and subscription columns to users table
-- This rollback restores the columns and migrates data back from separate tables

-- Add back the columns to the users table
ALTER TABLE users 
ADD COLUMN gdpr_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN ccpa_consent BOOLEAN DEFAULT FALSE,
ADD COLUMN consent_date TIMESTAMP,
ADD COLUMN data_retention_days INTEGER DEFAULT 2555,
ADD COLUMN subscription_tier TEXT DEFAULT 'free',
ADD COLUMN subscription_end TIMESTAMP;

-- Migrate privacy data back to users table
UPDATE users 
SET 
    gdpr_consent = upm.gdpr_consent,
    ccpa_consent = upm.ccpa_consent,
    consent_date = COALESCE(upm.gdpr_consent_date, upm.ccpa_consent_date),
    data_retention_days = upm.data_retention_days
FROM user_privacy_metadata upm
WHERE users.id = upm.user_id;

-- Migrate subscription data back to users table
UPDATE users 
SET 
    subscription_tier = us.tier,
    subscription_end = us.end_date
FROM user_subscriptions us
WHERE users.id = us.user_id;