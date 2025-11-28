-- Migration: Add gmail_consent table
-- This migration creates a dedicated table for Gmail consent and sync metadata

-- Create gmail_consent table
CREATE TABLE gmail_consent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    
    -- Consent and sync status
    gmail_consent BOOLEAN DEFAULT FALSE,
    gmail_consent_date TIMESTAMP WITH TIME ZONE,
    gmail_sync_enabled BOOLEAN DEFAULT FALSE,
    
    -- Sync tracking
    last_gmail_sync_at TIMESTAMP WITH TIME ZONE,
    gmail_history_id VARCHAR(255) DEFAULT '',
    
    -- Audit trail
    consent_ip_address VARCHAR(255),
    consent_user_agent TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX idx_gmail_consent_user_id ON gmail_consent(user_id);
CREATE INDEX idx_gmail_consent_enabled ON gmail_consent(gmail_consent, gmail_sync_enabled);

-- Migrate existing Gmail sync data from users table to gmail_consent
INSERT INTO gmail_consent (
    user_id,
    gmail_consent,
    gmail_consent_date,
    gmail_sync_enabled,
    last_gmail_sync_at,
    gmail_history_id,
    created_at,
    updated_at
)
SELECT 
    id as user_id,
    COALESCE(gmail_consent, false) as gmail_consent,
    gmail_consent_date,
    COALESCE(gmail_sync_enabled, false) as gmail_sync_enabled,
    last_gmail_sync_at,
    COALESCE(gmail_history_id, '') as gmail_history_id,
    created_at,
    updated_at
FROM users 
WHERE gmail_consent = true 
   OR gmail_sync_enabled = true 
   OR gmail_consent_date IS NOT NULL 
   OR last_gmail_sync_at IS NOT NULL 
   OR (gmail_history_id IS NOT NULL AND gmail_history_id != '');

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_gmail_consent_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_gmail_consent_updated_at
    BEFORE UPDATE ON gmail_consent
    FOR EACH ROW
    EXECUTE FUNCTION update_gmail_consent_updated_at();

-- Add comments for documentation
COMMENT ON TABLE gmail_consent IS 'Stores Gmail OAuth consent and sync metadata per user';
COMMENT ON COLUMN gmail_consent.gmail_history_id IS 'Gmail history ID for incremental sync using history.list API';
COMMENT ON COLUMN gmail_consent.consent_ip_address IS 'IP address from which consent was granted (for audit trail)';
COMMENT ON COLUMN gmail_consent.consent_user_agent IS 'User agent string from consent request (for audit trail)';

