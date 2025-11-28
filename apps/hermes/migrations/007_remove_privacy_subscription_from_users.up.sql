-- Migration: Remove privacy columns from users table
-- This migration removes the columns that have been moved to separate table

-- Remove privacy columns from users table
ALTER TABLE users 
DROP COLUMN IF EXISTS gdpr_consent,
DROP COLUMN IF EXISTS ccpa_consent,
DROP COLUMN IF EXISTS consent_date,
DROP COLUMN IF EXISTS data_retention_days;