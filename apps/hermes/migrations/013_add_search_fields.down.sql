-- Remove search fields from emails table
DROP TRIGGER IF EXISTS emails_search_vector_trigger ON emails;
DROP FUNCTION IF EXISTS emails_search_vector_update();
DROP INDEX IF EXISTS idx_emails_search_vector;
ALTER TABLE emails DROP COLUMN IF EXISTS search_vector;
ALTER TABLE emails DROP COLUMN IF EXISTS body_snippet;
