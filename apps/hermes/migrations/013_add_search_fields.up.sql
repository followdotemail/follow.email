-- Add search fields to emails table for full-text search
ALTER TABLE emails ADD COLUMN IF NOT EXISTS body_snippet TEXT DEFAULT '';
ALTER TABLE emails ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN index for fast full-text search
CREATE INDEX IF NOT EXISTS idx_emails_search_vector ON emails USING GIN(search_vector);

-- Create a trigger to automatically update search_vector when subject or body_snippet changes
CREATE OR REPLACE FUNCTION emails_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.subject, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.from_email, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.from_name, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.body_snippet, '')), 'C');
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS emails_search_vector_trigger ON emails;
CREATE TRIGGER emails_search_vector_trigger
    BEFORE INSERT OR UPDATE ON emails
    FOR EACH ROW EXECUTE FUNCTION emails_search_vector_update();

-- Update existing rows to populate search_vector
UPDATE emails SET search_vector =
    setweight(to_tsvector('english', COALESCE(subject, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(from_email, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(from_name, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(body_snippet, '')), 'C');
