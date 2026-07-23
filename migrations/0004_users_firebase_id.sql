ALTER TABLE users
    ADD COLUMN IF NOT EXISTS firebase_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_firebase_id_unique
    ON users(firebase_id)
    WHERE firebase_id IS NOT NULL;
