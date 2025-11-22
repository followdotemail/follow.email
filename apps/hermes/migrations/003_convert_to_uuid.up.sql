-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create a function to generate UUIDs for existing records
CREATE OR REPLACE FUNCTION generate_uuid_for_existing_records() RETURNS void AS $$
BEGIN
    -- We'll handle this table by table to maintain referential integrity
    
    -- First, add UUID columns to all tables
    ALTER TABLE users ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE external_accounts ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE external_accounts ADD COLUMN new_user_id UUID;
    ALTER TABLE oauth_tokens ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE oauth_tokens ADD COLUMN new_user_id UUID;
    ALTER TABLE user_preferences ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE user_preferences ADD COLUMN new_user_id UUID;
    ALTER TABLE emails ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE emails ADD COLUMN new_user_id UUID;
    ALTER TABLE followup_templates ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE followup_templates ADD COLUMN new_user_id UUID;
    ALTER TABLE followup_schedules ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE followup_schedules ADD COLUMN new_user_id UUID;
    ALTER TABLE followup_schedules ADD COLUMN new_email_id UUID;
    ALTER TABLE followup_schedules ADD COLUMN new_template_id UUID;
    ALTER TABLE email_analytics ADD COLUMN new_id UUID DEFAULT gen_random_uuid();
    ALTER TABLE email_analytics ADD COLUMN new_user_id UUID;
    ALTER TABLE email_analytics ADD COLUMN new_email_id UUID;
    
    -- Update foreign key references
    UPDATE external_accounts SET new_user_id = (SELECT new_id FROM users WHERE users.id = external_accounts.user_id);
    UPDATE oauth_tokens SET new_user_id = (SELECT new_id FROM users WHERE users.id = oauth_tokens.user_id);
    UPDATE user_preferences SET new_user_id = (SELECT new_id FROM users WHERE users.id = user_preferences.user_id);
    UPDATE emails SET new_user_id = (SELECT new_id FROM users WHERE users.id = emails.user_id);
    UPDATE followup_templates SET new_user_id = (SELECT new_id FROM users WHERE users.id = followup_templates.user_id);
    UPDATE followup_schedules SET new_user_id = (SELECT new_id FROM users WHERE users.id = followup_schedules.user_id);
    UPDATE followup_schedules SET new_email_id = (SELECT new_id FROM emails WHERE emails.id = followup_schedules.email_id);
    UPDATE followup_schedules SET new_template_id = (SELECT new_id FROM followup_templates WHERE followup_templates.id = followup_schedules.template_id) WHERE template_id IS NOT NULL;
    UPDATE email_analytics SET new_user_id = (SELECT new_id FROM users WHERE users.id = email_analytics.user_id);
    UPDATE email_analytics SET new_email_id = (SELECT new_id FROM emails WHERE emails.id = email_analytics.email_id);
    
END;
$$ LANGUAGE plpgsql;

-- Execute the function
SELECT generate_uuid_for_existing_records();

-- Drop old constraints and indexes
ALTER TABLE external_accounts DROP CONSTRAINT IF EXISTS external_accounts_user_id_fkey;
ALTER TABLE oauth_tokens DROP CONSTRAINT IF EXISTS oauth_tokens_user_id_fkey;
ALTER TABLE user_preferences DROP CONSTRAINT IF EXISTS user_preferences_user_id_fkey;
ALTER TABLE emails DROP CONSTRAINT IF EXISTS emails_user_id_fkey;
ALTER TABLE followup_templates DROP CONSTRAINT IF EXISTS followup_templates_user_id_fkey;
ALTER TABLE followup_schedules DROP CONSTRAINT IF EXISTS followup_schedules_user_id_fkey;
ALTER TABLE followup_schedules DROP CONSTRAINT IF EXISTS followup_schedules_email_id_fkey;
ALTER TABLE followup_schedules DROP CONSTRAINT IF EXISTS followup_schedules_template_id_fkey;
ALTER TABLE email_analytics DROP CONSTRAINT IF EXISTS email_analytics_user_id_fkey;
ALTER TABLE email_analytics DROP CONSTRAINT IF EXISTS email_analytics_email_id_fkey;

-- Drop old primary key constraints
ALTER TABLE users DROP CONSTRAINT users_pkey;
ALTER TABLE external_accounts DROP CONSTRAINT external_accounts_pkey;
ALTER TABLE oauth_tokens DROP CONSTRAINT oauth_tokens_pkey;
ALTER TABLE user_preferences DROP CONSTRAINT user_preferences_pkey;
ALTER TABLE emails DROP CONSTRAINT emails_pkey;
ALTER TABLE followup_templates DROP CONSTRAINT followup_templates_pkey;
ALTER TABLE followup_schedules DROP CONSTRAINT followup_schedules_pkey;
ALTER TABLE email_analytics DROP CONSTRAINT email_analytics_pkey;

-- Drop old columns
ALTER TABLE users DROP COLUMN id;
ALTER TABLE external_accounts DROP COLUMN id, DROP COLUMN user_id;
ALTER TABLE oauth_tokens DROP COLUMN id, DROP COLUMN user_id;
ALTER TABLE user_preferences DROP COLUMN id, DROP COLUMN user_id;
ALTER TABLE emails DROP COLUMN id, DROP COLUMN user_id;
ALTER TABLE followup_templates DROP COLUMN id, DROP COLUMN user_id;
ALTER TABLE followup_schedules DROP COLUMN id, DROP COLUMN user_id, DROP COLUMN email_id, DROP COLUMN template_id;
ALTER TABLE email_analytics DROP COLUMN id, DROP COLUMN user_id, DROP COLUMN email_id;

-- Rename new columns to original names
ALTER TABLE users RENAME COLUMN new_id TO id;
ALTER TABLE external_accounts RENAME COLUMN new_id TO id;
ALTER TABLE external_accounts RENAME COLUMN new_user_id TO user_id;
ALTER TABLE oauth_tokens RENAME COLUMN new_id TO id;
ALTER TABLE oauth_tokens RENAME COLUMN new_user_id TO user_id;
ALTER TABLE user_preferences RENAME COLUMN new_id TO id;
ALTER TABLE user_preferences RENAME COLUMN new_user_id TO user_id;
ALTER TABLE emails RENAME COLUMN new_id TO id;
ALTER TABLE emails RENAME COLUMN new_user_id TO user_id;
ALTER TABLE followup_templates RENAME COLUMN new_id TO id;
ALTER TABLE followup_templates RENAME COLUMN new_user_id TO user_id;
ALTER TABLE followup_schedules RENAME COLUMN new_id TO id;
ALTER TABLE followup_schedules RENAME COLUMN new_user_id TO user_id;
ALTER TABLE followup_schedules RENAME COLUMN new_email_id TO email_id;
ALTER TABLE followup_schedules RENAME COLUMN new_template_id TO template_id;
ALTER TABLE email_analytics RENAME COLUMN new_id TO id;
ALTER TABLE email_analytics RENAME COLUMN new_user_id TO user_id;
ALTER TABLE email_analytics RENAME COLUMN new_email_id TO email_id;

-- Add new primary key constraints
ALTER TABLE users ADD PRIMARY KEY (id);
ALTER TABLE external_accounts ADD PRIMARY KEY (id);
ALTER TABLE oauth_tokens ADD PRIMARY KEY (id);
ALTER TABLE user_preferences ADD PRIMARY KEY (id);
ALTER TABLE emails ADD PRIMARY KEY (id);
ALTER TABLE followup_templates ADD PRIMARY KEY (id);
ALTER TABLE followup_schedules ADD PRIMARY KEY (id);
ALTER TABLE email_analytics ADD PRIMARY KEY (id);

-- Add foreign key constraints
ALTER TABLE external_accounts ADD CONSTRAINT external_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE oauth_tokens ADD CONSTRAINT oauth_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE user_preferences ADD CONSTRAINT user_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE emails ADD CONSTRAINT emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE followup_templates ADD CONSTRAINT followup_templates_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE followup_schedules ADD CONSTRAINT followup_schedules_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE followup_schedules ADD CONSTRAINT followup_schedules_email_id_fkey FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE;
ALTER TABLE followup_schedules ADD CONSTRAINT followup_schedules_template_id_fkey FOREIGN KEY (template_id) REFERENCES followup_templates(id) ON DELETE SET NULL;
ALTER TABLE email_analytics ADD CONSTRAINT email_analytics_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE email_analytics ADD CONSTRAINT email_analytics_email_id_fkey FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE;

-- Recreate indexes
CREATE INDEX idx_external_accounts_user_id ON external_accounts(user_id);
CREATE INDEX idx_oauth_tokens_user_id ON oauth_tokens(user_id);
CREATE UNIQUE INDEX idx_user_preferences_user_id ON user_preferences(user_id);
CREATE INDEX idx_emails_user_id ON emails(user_id);
CREATE INDEX idx_emails_from_email ON emails(from_email);
CREATE INDEX idx_emails_received_at ON emails(received_at);
CREATE INDEX idx_emails_sent_at ON emails(sent_at);
CREATE INDEX idx_emails_thread_id ON emails(thread_id);
CREATE INDEX idx_followup_templates_user_id ON followup_templates(user_id);
CREATE INDEX idx_followup_schedules_user_id ON followup_schedules(user_id);
CREATE INDEX idx_followup_schedules_email_id ON followup_schedules(email_id);
CREATE INDEX idx_followup_schedules_scheduled_at ON followup_schedules(scheduled_at);
CREATE INDEX idx_email_analytics_user_id ON email_analytics(user_id);
CREATE INDEX idx_email_analytics_email_id ON email_analytics(email_id);
CREATE INDEX idx_email_analytics_event_type ON email_analytics(event_type);
CREATE INDEX idx_email_analytics_event_timestamp ON email_analytics(event_timestamp);

-- Drop the helper function
DROP FUNCTION generate_uuid_for_existing_records();