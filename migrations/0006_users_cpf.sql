ALTER TABLE users
    ADD COLUMN IF NOT EXISTS cpf TEXT;

-- CPF is unique per person in Brazil. We use a partial unique index so that
-- legacy rows without a CPF (NULL) are not blocked.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_cpf_unique
    ON users(cpf)
    WHERE cpf IS NOT NULL;