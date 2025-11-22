-- This migration reverts UUID primary keys back to serial/int primary keys
-- WARNING: This will lose data if UUIDs cannot be converted back to integers

-- Create a function to handle the conversion back to integers
CREATE OR REPLACE FUNCTION convert_uuid_to_serial() RETURNS void AS $$
BEGIN
    -- Add new serial columns to all tables
    ALTER TABLE users ADD COLUMN new_id SERIAL;
    ALTER TABLE external_accounts ADD COLUMN new_id SERIAL;
    ALTER TABLE external_accounts ADD COLUMN new_user_id INTEGER;
    ALTER TABLE oauth_tokens ADD COLUMN new_id SERIAL;
    ALTER TABLE oauth_tokens ADD COLUMN new_user_id INTEGER;
    ALTER TABLE user_preferences ADD COLUMN new_id SERIAL;
    ALTER TABLE user_preferences ADD COLUMN new_user_id INTEGER;
    ALTER TABLE emails ADD COLUMN new_id SERIAL;
    ALTER TABLE emails ADD COLUMN new_user_id INTEGER;
    ALTER TABLE followup_templates ADD COLUMN new_id SERIAL;
    ALTER TABLE followup_templates ADD COLUMN new_user_id INTEGER;
    ALTER TABLE followup_schedules ADD COLUMN new_id SERIAL;
    ALTER TABLE followup_schedules ADD COLUMN new_user_id INTEGER;
    ALTER TABLE followup_schedules ADD COLUMN new_email_id INTEGER;
    ALTER TABLE followup_schedules ADD COLUMN new_template_id INTEGER;
    ALTER TABLE email_analytics ADD COLUMN new_id SERIAL;
    ALTER TABLE email_analytics ADD COLUMN new_user_id INTEGER;
    ALTER TABLE email_analytics ADD COLUMN new_email_id INTEGER;
    
    -- Update foreign key references using row_number to create sequential IDs
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
SELECT convert_uuid_to_serial();

-- Drop foreign key constraints
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

-- Drop primary key constraints
ALTER TABLE users DROP CONSTRAINT users_pkey;
ALTER TABLE external_accounts DROP CONSTRAINT external_accounts_pkey;
ALTER TABLE oauth_tokens DROP CONSTRAINT oauth_tokens_pkey;
ALTER TABLE user_preferences DROP CONSTRAINT user_preferences_pkey;
ALTER TABLE emails DROP CONSTRAINT emails_pkey;
ALTER TABLE followup_templates DROP CONSTRAINT followup_templates_pkey;
ALTER TABLE followup_schedules DROP CONSTRAINT followup_schedules_pkey;
ALTER TABLE email_analytics DROP CONSTRAINT email_analytics_pkey;

-- Drop UUID columns
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

-- Add primary key constraints
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

-- Drop the helper function
DROP FUNCTION convert_uuid_to_serial();