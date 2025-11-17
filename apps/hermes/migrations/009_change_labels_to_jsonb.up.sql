-- Change labels column from TEXT to JSONB for better JSON handling
ALTER TABLE emails ALTER COLUMN labels TYPE JSONB USING labels::JSONB;