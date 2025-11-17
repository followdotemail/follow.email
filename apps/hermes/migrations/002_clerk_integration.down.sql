-- Rollback migration for Clerk integration

-- Drop trigger and function
DROP TRIGGER IF EXISTS update_external_accounts_updated_at ON external_accounts;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_external_accounts_user_id;
DROP INDEX IF EXISTS idx_external_accounts_provider;
DROP INDEX IF EXISTS idx_external_accounts_clerk_external_id;
DROP INDEX IF EXISTS idx_users_username;

-- Drop external_accounts table
DROP TABLE IF EXISTS external_accounts;

-- Remove new columns from users table
ALTER TABLE users 
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name,
DROP COLUMN IF EXISTS username,
DROP COLUMN IF EXISTS image_url;