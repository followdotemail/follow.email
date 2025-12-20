-- Remove email security/authentication metadata fields
ALTER TABLE emails DROP COLUMN IF EXISTS mailed_by;
ALTER TABLE emails DROP COLUMN IF EXISTS signed_by;
ALTER TABLE emails DROP COLUMN IF EXISTS security_info;
