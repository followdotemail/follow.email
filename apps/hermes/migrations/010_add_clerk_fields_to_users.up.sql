-- Migration: Add Clerk authentication fields to users table
-- This migration adds fields needed for Clerk-based authentication and social login

-- Add Clerk-related columns to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS clerk_id VARCHAR(255) UNIQUE,
ADD COLUMN IF NOT EXISTS first_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS last_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS username VARCHAR(255) UNIQUE,
ADD COLUMN IF NOT EXISTS image_url TEXT;

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_users_clerk_id ON users(clerk_id) WHERE clerk_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN users.clerk_id IS 'Clerk authentication service user ID';
COMMENT ON COLUMN users.first_name IS 'User first name from OAuth provider or Clerk profile';
COMMENT ON COLUMN users.last_name IS 'User last name from OAuth provider or Clerk profile';
COMMENT ON COLUMN users.username IS 'Username from OAuth provider (GitHub, etc.)';
COMMENT ON COLUMN users.image_url IS 'Profile picture URL from OAuth provider or Clerk';

