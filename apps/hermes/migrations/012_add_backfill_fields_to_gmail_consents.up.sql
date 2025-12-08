ALTER TABLE gmail_consent ADD COLUMN next_page_token TEXT DEFAULT '';
ALTER TABLE gmail_consent ADD COLUMN backfill_status VARCHAR(50) DEFAULT 'idle';
