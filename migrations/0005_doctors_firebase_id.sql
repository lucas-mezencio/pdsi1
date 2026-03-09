ALTER TABLE doctors
    ADD COLUMN IF NOT EXISTS firebase_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_doctors_firebase_id_unique
    ON doctors(firebase_id)
    WHERE firebase_id IS NOT NULL;
