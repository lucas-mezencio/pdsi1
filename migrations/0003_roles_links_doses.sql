ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'ELDERLY';

CREATE TABLE IF NOT EXISTS user_links (
    caregiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    elderly_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (caregiver_id, elderly_id)
);

CREATE TABLE IF NOT EXISTS caregiver_invitations (
    id           UUID PRIMARY KEY,
    caregiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    elderly_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token        TEXT NOT NULL UNIQUE,
    status       TEXT NOT NULL DEFAULT 'PENDING',
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS dose_records (
    id              UUID PRIMARY KEY,
    prescription_id UUID NOT NULL REFERENCES prescriptions(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    medicament_name TEXT NOT NULL,
    dosage          TEXT NOT NULL,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    confirmed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_user_links_caregiver_id            ON user_links(caregiver_id);
CREATE INDEX IF NOT EXISTS idx_user_links_elderly_id              ON user_links(elderly_id);
CREATE INDEX IF NOT EXISTS idx_caregiver_invitations_token        ON caregiver_invitations(token);
CREATE INDEX IF NOT EXISTS idx_caregiver_invitations_elderly_id   ON caregiver_invitations(elderly_id);
CREATE INDEX IF NOT EXISTS idx_caregiver_invitations_caregiver_id ON caregiver_invitations(caregiver_id);
CREATE INDEX IF NOT EXISTS idx_dose_records_prescription_id       ON dose_records(prescription_id);
CREATE INDEX IF NOT EXISTS idx_dose_records_user_id               ON dose_records(user_id);
CREATE INDEX IF NOT EXISTS idx_dose_records_scheduled_at          ON dose_records(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_dose_records_status                ON dose_records(status);
