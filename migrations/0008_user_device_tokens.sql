CREATE TABLE IF NOT EXISTS user_device_tokens (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token         TEXT NOT NULL UNIQUE,
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    last_used_at  TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_user_device_tokens_user_id
    ON user_device_tokens(user_id);