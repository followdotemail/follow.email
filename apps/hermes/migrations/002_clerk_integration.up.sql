-- Migration for Clerk integration
-- Add new fields to users table for Clerk compatibility

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS first_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS last_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS username VARCHAR(255) UNIQUE,
ADD COLUMN IF NOT EXISTS image_url TEXT;

-- Create external_accounts table for social login connections
CREATE TABLE IF NOT EXISTS external_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    clerk_external_id VARCHAR(255) UNIQUE NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255),
    email_address VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    image_url TEXT,
    username VARCHAR(255),
    public_metadata JSONB,
    label VARCHAR(255),
    verification_status VARCHAR(50) DEFAULT 'verified',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_external_accounts_user_id ON external_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_external_accounts_provider ON external_accounts(provider);
CREATE INDEX IF NOT EXISTS idx_external_accounts_clerk_external_id ON external_accounts(clerk_external_id);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username IS NOT NULL;

-- Update trigger for external_accounts updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_external_accounts_updated_at 
    BEFORE UPDATE ON external_accounts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();