-- Migration rollback: Remove Clerk authentication fields from users table
-- This rollback removes the Clerk-related columns added in migration 010

-- Drop indexes first
DROP INDEX IF EXISTS idx_users_clerk_id;
DROP INDEX IF EXISTS idx_users_username;

-- Remove Clerk-related columns from users table
ALTER TABLE users 
DROP COLUMN IF EXISTS clerk_id,
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name,
DROP COLUMN IF EXISTS username,
DROP COLUMN IF EXISTS image_url;

