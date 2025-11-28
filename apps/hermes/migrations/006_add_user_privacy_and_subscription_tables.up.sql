-- Migration: Add user privacy metadata table
-- This migration normalizes the users table by moving privacy data to a separate table

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

-- Create indexes for better performance
CREATE INDEX idx_user_privacy_metadata_user_id ON user_privacy_metadata(user_id);

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

-- Create trigger to automatically update updated_at timestamp
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