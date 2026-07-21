-- Backfill existing single tokens into the new device-token table.
INSERT INTO user_device_tokens (id, user_id, token, enabled, created_at, updated_at)
SELECT gen_random_uuid(), id, firebase_token, TRUE, NOW(), NOW()
FROM users
WHERE firebase_token IS NOT NULL
  AND firebase_token <> ''
  AND NOT EXISTS (
      SELECT 1 FROM user_device_tokens WHERE user_device_tokens.token = users.firebase_token
  );

ALTER TABLE users DROP COLUMN IF EXISTS firebase_token;
ALTER TABLE users DROP COLUMN IF EXISTS notifications_enabled;
