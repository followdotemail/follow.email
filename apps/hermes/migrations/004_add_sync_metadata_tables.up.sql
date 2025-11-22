-- Migration: Add sync metadata tables
-- This migration creates dedicated tables for sync metadata and job tracking

-- Create provider_sync_metadata table
CREATE TABLE provider_sync_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    
    -- Consent and permissions
    consent_granted BOOLEAN DEFAULT FALSE,
    consent_date TIMESTAMP WITH TIME ZONE,
    sync_enabled BOOLEAN DEFAULT FALSE,
    
    -- Sync state tracking
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_full_sync_at TIMESTAMP WITH TIME ZONE,
    sync_version INTEGER DEFAULT 1,
    
    -- Provider-specific sync tokens/IDs
    history_id VARCHAR(255), -- Gmail history ID
    delta_token TEXT,        -- Microsoft Graph delta token
    sync_token TEXT,         -- Generic sync token
    
    -- Sync configuration
    sync_frequency_hours INTEGER DEFAULT 24,
    auto_sync_enabled BOOLEAN DEFAULT TRUE,
    
    -- Sync statistics
    total_emails_synced BIGINT DEFAULT 0,
    last_sync_duration_ms INTEGER DEFAULT 0,
    consecutive_errors INTEGER DEFAULT 0,
    last_error_message TEXT,
    last_error_at TIMESTAMP WITH TIME ZONE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(user_id, provider)
);

-- Create indexes for provider_sync_metadata
CREATE INDEX idx_provider_sync_metadata_user_id ON provider_sync_metadata(user_id);
CREATE INDEX idx_provider_sync_metadata_provider ON provider_sync_metadata(provider);
CREATE INDEX idx_provider_sync_metadata_last_sync ON provider_sync_metadata(last_sync_at);
CREATE INDEX idx_provider_sync_metadata_auto_sync ON provider_sync_metadata(auto_sync_enabled, sync_enabled);

-- Create sync_jobs table
CREATE TABLE sync_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    job_id VARCHAR(255) UNIQUE NOT NULL, -- External job ID (e.g., QStash message ID)
    
    -- Job details
    sync_type VARCHAR(20) NOT NULL CHECK (sync_type IN ('full', 'incremental')),
    status VARCHAR(20) NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    
    -- Progress tracking
    emails_processed INTEGER DEFAULT 0,
    new_emails INTEGER DEFAULT 0,
    updated_emails INTEGER DEFAULT 0,
    skipped_emails INTEGER DEFAULT 0,
    deleted_emails INTEGER DEFAULT 0,
    
    -- Timing
    queued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER DEFAULT 0,
    
    -- Error handling
    error_message TEXT,
    error_code VARCHAR(100),
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    
    -- Sync parameters and results (stored as JSON)
    sync_parameters JSONB,
    result_data JSONB,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for sync_jobs
CREATE INDEX idx_sync_jobs_user_id ON sync_jobs(user_id);
CREATE INDEX idx_sync_jobs_provider ON sync_jobs(provider);
CREATE INDEX idx_sync_jobs_job_id ON sync_jobs(job_id);
CREATE INDEX idx_sync_jobs_status ON sync_jobs(status);
CREATE INDEX idx_sync_jobs_queued_at ON sync_jobs(queued_at);
CREATE INDEX idx_sync_jobs_user_provider_status ON sync_jobs(user_id, provider, status);

-- Migrate existing Gmail sync data from users table to provider_sync_metadata
INSERT INTO provider_sync_metadata (
    user_id,
    provider,
    consent_granted,
    consent_date,
    sync_enabled,
    last_sync_at,
    history_id,
    created_at,
    updated_at
)
SELECT 
    id as user_id,
    'gmail' as provider,
    COALESCE(gmail_consent, false) as consent_granted,
    gmail_consent_date as consent_date,
    COALESCE(gmail_sync_enabled, false) as sync_enabled,
    last_gmail_sync_at as last_sync_at,
    NULLIF(gmail_history_id, '') as history_id,
    created_at,
    updated_at
FROM users 
WHERE gmail_consent = true OR gmail_sync_enabled = true OR gmail_consent_date IS NOT NULL OR last_gmail_sync_at IS NOT NULL OR gmail_history_id IS NOT NULL AND gmail_history_id != '';

-- Create trigger to update updated_at timestamp for provider_sync_metadata
CREATE OR REPLACE FUNCTION update_provider_sync_metadata_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_provider_sync_metadata_updated_at
    BEFORE UPDATE ON provider_sync_metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_provider_sync_metadata_updated_at();

-- Create trigger to update updated_at timestamp for sync_jobs
CREATE OR REPLACE FUNCTION update_sync_jobs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_jobs_updated_at
    BEFORE UPDATE ON sync_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_sync_jobs_updated_at();

-- Add comments for documentation
COMMENT ON TABLE provider_sync_metadata IS 'Stores sync metadata and configuration for different email providers per user';
COMMENT ON TABLE sync_jobs IS 'Tracks individual sync operations for monitoring and debugging';

COMMENT ON COLUMN provider_sync_metadata.history_id IS 'Gmail-specific history ID for incremental sync';
COMMENT ON COLUMN provider_sync_metadata.delta_token IS 'Microsoft Graph delta token for incremental sync';
COMMENT ON COLUMN provider_sync_metadata.sync_token IS 'Generic sync token for other providers';
COMMENT ON COLUMN provider_sync_metadata.consecutive_errors IS 'Number of consecutive sync errors, reset on successful sync';

COMMENT ON COLUMN sync_jobs.job_id IS 'External job identifier (e.g., QStash message ID)';
COMMENT ON COLUMN sync_jobs.sync_parameters IS 'JSON object containing sync request parameters';
COMMENT ON COLUMN sync_jobs.result_data IS 'JSON object containing sync operation results';