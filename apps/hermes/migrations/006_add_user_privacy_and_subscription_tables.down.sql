-- Migration Rollback: Remove user privacy metadata and subscription tables
-- This rollback restores privacy and subscription data to the users table

-- First, add back the columns to the users table
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

-- Drop triggers first
DROP TRIGGER IF EXISTS update_user_privacy_metadata_updated_at ON user_privacy_metadata;
DROP TRIGGER IF EXISTS update_user_subscriptions_updated_at ON user_subscriptions;

-- Drop the trigger function if no other tables are using it
-- Note: We'll keep the function as it might be used by other tables

-- Drop indexes
DROP INDEX IF EXISTS idx_user_privacy_metadata_user_id;
DROP INDEX IF EXISTS idx_user_subscriptions_user_id;
DROP INDEX IF EXISTS idx_user_subscriptions_tier;
DROP INDEX IF EXISTS idx_user_subscriptions_status;
DROP INDEX IF EXISTS idx_user_subscriptions_stripe_customer_id;

-- Drop the new tables
DROP TABLE IF EXISTS user_subscriptions;
DROP TABLE IF EXISTS user_privacy_metadata;