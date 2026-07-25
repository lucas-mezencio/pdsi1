-- Backfill existing single tokens into the new device-token table.
-- The INSERT is wrapped in a column-existence check so the migration is
-- idempotent: after the first run drops the column, subsequent runs
-- (e.g. when other binaries like cmd/testnotify re-run Migrate() against
-- an already-migrated DB) skip the backfill cleanly.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'users'
          AND column_name = 'firebase_token'
    ) THEN
        INSERT INTO user_device_tokens (id, user_id, token, enabled, created_at, updated_at)
        SELECT gen_random_uuid(), id, firebase_token, TRUE, NOW(), NOW()
        FROM users
        WHERE firebase_token IS NOT NULL
          AND firebase_token <> ''
          AND NOT EXISTS (
              SELECT 1 FROM user_device_tokens
              WHERE user_device_tokens.token = users.firebase_token
          );
    END IF;
END $$;

ALTER TABLE users DROP COLUMN IF EXISTS firebase_token;
ALTER TABLE users DROP COLUMN IF EXISTS notifications_enabled;