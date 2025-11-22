-- Migration: Add user privacy metadata and subscription tables
-- This migration normalizes the users table by moving privacy and subscription data to separate tables

-- Create user_privacy_metadata table
CREATE TABLE user_privacy_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    
    -- GDPR Compliance
    gdpr_consent BOOLEAN DEFAULT FALSE,
    gdpr_consent_date TIMESTAMP,
    gdpr_withdrawn BOOLEAN DEFAULT FALSE,
    gdpr_withdrawn_date TIMESTAMP,
    
    -- CCPA Compliance
    ccpa_consent BOOLEAN DEFAULT FALSE,
    ccpa_consent_date TIMESTAMP,
    ccpa_opt_out BOOLEAN DEFAULT FALSE,
    ccpa_opt_out_date TIMESTAMP,
    
    -- Data Retention
    data_retention_days INTEGER DEFAULT 2555, -- 7 years default
    data_retention_set BOOLEAN DEFAULT FALSE,
    
    -- Audit Trail
    consent_ip_address TEXT,
    consent_user_agent TEXT,
    last_updated_by TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create user_subscriptions table
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    
    -- Subscription Details
    tier TEXT DEFAULT 'free' CHECK (tier IN ('free', 'pro', 'enterprise')),
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'canceled', 'expired', 'pending', 'suspended')),
    
    -- Billing Cycle
    start_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_date TIMESTAMP,
    next_billing_date TIMESTAMP,
    billing_cycle TEXT, -- 'monthly', 'yearly', 'lifetime'
    
    -- Pricing
    amount DECIMAL(10,2) DEFAULT 0.00,
    currency TEXT DEFAULT 'USD',
    
    -- External Integration
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    stripe_price_id TEXT,
    
    -- Usage Tracking
    emails_processed INTEGER DEFAULT 0,
    emails_limit INTEGER DEFAULT 100, -- Monthly limit
    api_calls_used INTEGER DEFAULT 0,
    api_calls_limit INTEGER DEFAULT 1000, -- Monthly limit
    
    -- Trial Information
    is_trial_active BOOLEAN DEFAULT FALSE,
    trial_start_date TIMESTAMP,
    trial_end_date TIMESTAMP,
    
    -- Cancellation
    canceled_at TIMESTAMP,
    cancellation_reason TEXT,
    canceled_by TEXT,
    
    -- Auto-renewal
    auto_renew BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX idx_user_privacy_metadata_user_id ON user_privacy_metadata(user_id);
CREATE INDEX idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX idx_user_subscriptions_tier ON user_subscriptions(tier);
CREATE INDEX idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX idx_user_subscriptions_stripe_customer_id ON user_subscriptions(stripe_customer_id);

-- Migrate existing privacy data from users table to user_privacy_metadata
INSERT INTO user_privacy_metadata (
    user_id,
    gdpr_consent,
    gdpr_consent_date,
    ccpa_consent,
    ccpa_consent_date,
    data_retention_days,
    data_retention_set,
    created_at,
    updated_at
)
SELECT 
    id,
    gdpr_consent,
    CASE 
        WHEN consent_date IS NOT NULL AND gdpr_consent = TRUE THEN consent_date
        ELSE NULL
    END,
    ccpa_consent,
    CASE 
        WHEN consent_date IS NOT NULL AND ccpa_consent = TRUE THEN consent_date
        ELSE NULL
    END,
    data_retention_days,
    CASE 
        WHEN data_retention_days != 2555 THEN TRUE
        ELSE FALSE
    END,
    created_at,
    updated_at
FROM users
WHERE id IS NOT NULL;

-- Migrate existing subscription data from users table to user_subscriptions
INSERT INTO user_subscriptions (
    user_id,
    tier,
    status,
    start_date,
    end_date,
    created_at,
    updated_at
)
SELECT 
    id,
    COALESCE(subscription_tier, 'free'),
    CASE 
        WHEN subscription_end IS NULL OR subscription_end > CURRENT_TIMESTAMP THEN 'active'
        ELSE 'expired'
    END,
    created_at, -- Use user creation date as subscription start
    subscription_end,
    created_at,
    updated_at
FROM users
WHERE id IS NOT NULL;

-- Create triggers to automatically update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_privacy_metadata_updated_at 
    BEFORE UPDATE ON user_privacy_metadata 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_subscriptions_updated_at 
    BEFORE UPDATE ON user_subscriptions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();