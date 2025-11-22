-- Revert labels column from JSONB back to TEXT
ALTER TABLE emails ALTER COLUMN labels TYPE TEXT USING labels::TEXT;