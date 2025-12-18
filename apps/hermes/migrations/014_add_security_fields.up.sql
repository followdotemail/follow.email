-- Add email security/authentication metadata fields
ALTER TABLE emails ADD COLUMN IF NOT EXISTS mailed_by VARCHAR(255);
ALTER TABLE emails ADD COLUMN IF NOT EXISTS signed_by VARCHAR(255);
ALTER TABLE emails ADD COLUMN IF NOT EXISTS security_info VARCHAR(100);

-- Add comment for documentation
COMMENT ON COLUMN emails.mailed_by IS 'Email sending service domain (e.g., amazonses.com from SPF/DKIM headers)';
COMMENT ON COLUMN emails.signed_by IS 'DKIM signature domain (e.g., swiggy.in from DKIM-Signature header)';
COMMENT ON COLUMN emails.security_info IS 'Encryption/security info (e.g., Standard encryption (TLS))';
