-- LGPD column-level encryption.
--
-- Convert PII columns from TEXT to BYTEA. Existing rows lose their plaintext
-- values; the application must re-create or backfill any pre-existing data
-- after this migration runs. All affected UNIQUE constraints and partial
-- indexes are dropped here and re-created in 0009 on SHA-256 hash sidecar
-- columns (email_hash, license_number_hash). The users.cpf partial unique
-- index is dropped without replacement (no cpf_hash sidecar per spec).
--
-- Tables/columns affected:
--   users                  email, phone, cpf
--   doctors                email, phone, license_number
--   medicaments            name, dosage
--   dose_records           medicament_name, dosage
--   notification_events    medicament_name, dosage

-- ----------------------------- users -----------------------------
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS idx_users_cpf_unique;

ALTER TABLE users DROP COLUMN email;
ALTER TABLE users DROP COLUMN phone;
ALTER TABLE users DROP COLUMN cpf;

ALTER TABLE users ADD COLUMN email  BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE users ADD COLUMN phone  BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE users ADD COLUMN cpf    BYTEA;

-- ----------------------------- doctors -----------------------------
ALTER TABLE doctors DROP CONSTRAINT IF EXISTS doctors_email_key;
ALTER TABLE doctors DROP CONSTRAINT IF EXISTS doctors_license_number_key;

ALTER TABLE doctors DROP COLUMN email;
ALTER TABLE doctors DROP COLUMN phone;
ALTER TABLE doctors DROP COLUMN license_number;

ALTER TABLE doctors ADD COLUMN email          BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE doctors ADD COLUMN phone          BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE doctors ADD COLUMN license_number BYTEA NOT NULL DEFAULT '\x'::bytea;

-- ----------------------------- medicaments -----------------------------
ALTER TABLE medicaments DROP COLUMN name;
ALTER TABLE medicaments DROP COLUMN dosage;

ALTER TABLE medicaments ADD COLUMN name   BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE medicaments ADD COLUMN dosage BYTEA NOT NULL DEFAULT '\x'::bytea;

-- ----------------------------- dose_records -----------------------------
ALTER TABLE dose_records DROP COLUMN medicament_name;
ALTER TABLE dose_records DROP COLUMN dosage;

ALTER TABLE dose_records ADD COLUMN medicament_name BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE dose_records ADD COLUMN dosage          BYTEA NOT NULL DEFAULT '\x'::bytea;

-- ----------------------------- notification_events -----------------------------
ALTER TABLE notification_events DROP COLUMN medicament_name;
ALTER TABLE notification_events DROP COLUMN dosage;

ALTER TABLE notification_events ADD COLUMN medicament_name BYTEA NOT NULL DEFAULT '\x'::bytea;
ALTER TABLE notification_events ADD COLUMN dosage          BYTEA NOT NULL DEFAULT '\x'::bytea;
