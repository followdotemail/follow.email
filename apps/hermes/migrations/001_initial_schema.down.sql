-- Drop triggers
DROP TRIGGER IF EXISTS update_followup_schedules_updated_at ON followup_schedules;
DROP TRIGGER IF EXISTS update_followup_templates_updated_at ON followup_templates;
DROP TRIGGER IF EXISTS update_emails_updated_at ON emails;
DROP TRIGGER IF EXISTS update_oauth_tokens_updated_at ON oauth_tokens;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_email_analytics_timestamp;
DROP INDEX IF EXISTS idx_email_analytics_event_type;
DROP INDEX IF EXISTS idx_email_analytics_email_id;
DROP INDEX IF EXISTS idx_email_analytics_user_id;

DROP INDEX IF EXISTS idx_followup_schedules_status;
DROP INDEX IF EXISTS idx_followup_schedules_scheduled_at;
DROP INDEX IF EXISTS idx_followup_schedules_email_id;
DROP INDEX IF EXISTS idx_followup_schedules_user_id;

DROP INDEX IF EXISTS idx_emails_sync_version;
DROP INDEX IF EXISTS idx_emails_ai_priority;
DROP INDEX IF EXISTS idx_emails_requires_followup;
DROP INDEX IF EXISTS idx_emails_followup_status;
DROP INDEX IF EXISTS idx_emails_received_at;
DROP INDEX IF EXISTS idx_emails_sent_at;
DROP INDEX IF EXISTS idx_emails_thread_id;
DROP INDEX IF EXISTS idx_emails_message_id;
DROP INDEX IF EXISTS idx_emails_user_id;

DROP INDEX IF EXISTS idx_oauth_tokens_expires_at;
DROP INDEX IF EXISTS idx_oauth_tokens_provider;
DROP INDEX IF EXISTS idx_oauth_tokens_user_id;

DROP INDEX IF EXISTS idx_users_subscription;
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_users_provider;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables in reverse order of creation (respecting foreign key constraints)
DROP TABLE IF EXISTS email_analytics;
DROP TABLE IF EXISTS followup_schedules;
DROP TABLE IF EXISTS followup_templates;
DROP TABLE IF EXISTS emails;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS users;