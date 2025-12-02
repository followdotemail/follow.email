-- Rollback: Remove category/label separation

-- Drop the new columns from emails
ALTER TABLE emails
    DROP COLUMN IF NOT EXISTS category,
    DROP COLUMN IF NOT EXISTS system_labels,
    DROP COLUMN IF NOT EXISTS user_labels;


-- Drop the user_labels table
DROP TABLE IF EXISTS user_labels;