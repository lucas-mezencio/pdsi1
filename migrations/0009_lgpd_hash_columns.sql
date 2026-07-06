-- LGPD hash sidecar columns.
--
-- Add deterministic SHA-256 hash columns alongside the BYTEA-encrypted
-- columns from 0008 so the application can perform exact-match lookups
-- (and maintain UNIQUE constraints) without decrypting every row.
--
-- The application computes digest(lower(value), 'sha256') at write time
-- and stores the 32-byte result here. The same digest is computed at
-- query time and used in WHERE clauses for FindByEmail /
-- FindByLicenseNumber.

-- ----------------------------- users -----------------------------
ALTER TABLE users ADD COLUMN email_hash BYTEA;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_hash_unique
    ON users(email_hash)
    WHERE email_hash IS NOT NULL;

-- ----------------------------- doctors -----------------------------
ALTER TABLE doctors ADD COLUMN email_hash BYTEA;
ALTER TABLE doctors ADD COLUMN license_number_hash BYTEA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_doctors_email_hash_unique
    ON doctors(email_hash)
    WHERE email_hash IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_doctors_license_number_hash_unique
    ON doctors(license_number_hash)
    WHERE license_number_hash IS NOT NULL;
