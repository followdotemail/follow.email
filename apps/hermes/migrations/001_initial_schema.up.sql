-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('gmail', 'outlook')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    
    -- Privacy and compliance
    gdpr_consent BOOLEAN DEFAULT false,
    ccpa_consent BOOLEAN DEFAULT false,
    consent_date TIMESTAMP WITH TIME ZONE,
    data_retention_days INTEGER DEFAULT 365,
    
    -- Subscription and billing
    subscription_tier VARCHAR(50) DEFAULT 'free' CHECK (subscription_tier IN ('free', 'pro', 'enterprise')),
    subscription_end TIMESTAMP WITH TIME ZONE
);

-- Create oauth_tokens table
CREATE TABLE oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('gmail', 'outlook')),
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50) DEFAULT 'Bearer',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    scope TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, provider)
);

-- Create user_preferences table
CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    auto_followup_enabled BOOLEAN DEFAULT true,
    followup_delay_hours INTEGER DEFAULT 24,
    max_followup_attempts INTEGER DEFAULT 3,
    ai_response_enabled BOOLEAN DEFAULT true,
    email_notifications BOOLEAN DEFAULT true,
    webhook_notifications BOOLEAN DEFAULT false,
    webhook_url TEXT
);

-- Create emails table
CREATE TABLE emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id VARCHAR(255) NOT NULL,
    thread_id VARCHAR(255),
    subject TEXT,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255),
    to_emails JSONB,
    cc_emails JSONB,
    bcc_emails JSONB,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Email content storage references
    s3_body_key VARCHAR(500),
    s3_attachments_key VARCHAR(500),
    
    -- Email metadata
    is_read BOOLEAN DEFAULT false,
    is_important BOOLEAN DEFAULT false,
    has_attachments BOOLEAN DEFAULT false,
    email_size BIGINT DEFAULT 0,
    mime_type VARCHAR(100),
    
    -- Follow-up tracking
    requires_followup BOOLEAN DEFAULT false,
    followup_status VARCHAR(50) DEFAULT 'pending' CHECK (followup_status IN ('pending', 'sent', 'completed', 'cancelled')),
    last_followup_at TIMESTAMP WITH TIME ZONE,
    followup_count INTEGER DEFAULT 0,
    
    -- AI analysis
    ai_summary TEXT,
    ai_sentiment VARCHAR(20) CHECK (ai_sentiment IN ('positive', 'neutral', 'negative')),
    ai_priority VARCHAR(20) CHECK (ai_priority IN ('high', 'medium', 'low')),
    ai_analyzed_at TIMESTAMP WITH TIME ZONE,
    
    -- Sync metadata
    provider_sync_id VARCHAR(255),
    last_sync_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sync_version INTEGER DEFAULT 1,
    
    UNIQUE(user_id, message_id)
);

-- Create followup_templates table
CREATE TABLE followup_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    subject_template TEXT NOT NULL,
    body_template TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, name)
);

-- Create followup_schedules table
CREATE TABLE followup_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    template_id UUID REFERENCES followup_templates(id) ON DELETE SET NULL,
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
    attempt_count INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Generated content
    generated_subject VARCHAR(500),
    generated_body TEXT
);

-- Create email_analytics table
CREATE TABLE email_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('sent', 'delivered', 'opened', 'clicked', 'replied', 'bounced')),
    event_data JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(email_id, event_type, occurred_at)
);

-- Create indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_provider ON users(provider);
CREATE INDEX idx_users_active ON users(is_active);
CREATE INDEX idx_users_subscription ON users(subscription_tier);

CREATE INDEX idx_oauth_tokens_user_id ON oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_provider ON oauth_tokens(provider);
CREATE INDEX idx_oauth_tokens_expires_at ON oauth_tokens(expires_at);

CREATE INDEX idx_emails_user_id ON emails(user_id);
CREATE INDEX idx_emails_message_id ON emails(message_id);
CREATE INDEX idx_emails_thread_id ON emails(thread_id);
CREATE INDEX idx_emails_sent_at ON emails(sent_at);
CREATE INDEX idx_emails_received_at ON emails(received_at);
CREATE INDEX idx_emails_followup_status ON emails(followup_status);
CREATE INDEX idx_emails_requires_followup ON emails(requires_followup);
CREATE INDEX idx_emails_ai_priority ON emails(ai_priority);
CREATE INDEX idx_emails_sync_version ON emails(sync_version);

CREATE INDEX idx_followup_schedules_user_id ON followup_schedules(user_id);
CREATE INDEX idx_followup_schedules_email_id ON followup_schedules(email_id);
CREATE INDEX idx_followup_schedules_scheduled_at ON followup_schedules(scheduled_at);
CREATE INDEX idx_followup_schedules_status ON followup_schedules(status);

CREATE INDEX idx_email_analytics_user_id ON email_analytics(user_id);
CREATE INDEX idx_email_analytics_email_id ON email_analytics(email_id);
CREATE INDEX idx_email_analytics_event_type ON email_analytics(event_type);
CREATE INDEX idx_email_analytics_timestamp ON email_analytics(event_timestamp);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON oauth_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_emails_updated_at BEFORE UPDATE ON emails FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_followup_templates_updated_at BEFORE UPDATE ON followup_templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_followup_schedules_updated_at BEFORE UPDATE ON followup_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();