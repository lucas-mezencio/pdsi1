package devicetoken

import "context"

// Repository persists DeviceTokens.
type Repository interface {
    // Save upserts by token. When the token already exists for a different
    // user_id, returns ErrConflict. When it exists for the same user_id,
    // updates enabled=true and updated_at, returns the existing row.
    Save(ctx context.Context, t *DeviceToken) (*DeviceToken, error)

    // FindByID returns the token with the given id, or ErrNotFound.
    FindByID(ctx context.Context, id string) (*DeviceToken, error)

    // FindByUserID returns all tokens for the user (enabled and disabled).
    FindByUserID(ctx context.Context, userID string) ([]*DeviceToken, error)

    // Delete removes the token by id. No-op if missing.
    Delete(ctx context.Context, id string) error

    // SetEnabled toggles the enabled flag and bumps UpdatedAt. Returns the
    // updated row, or ErrNotFound.
    SetEnabled(ctx context.Context, id string, enabled bool) (*DeviceToken, error)

    // TouchLastUsed sets last_used_at to the current time for the given id.
    // Best-effort: callers should not fail the surrounding operation when
    // this returns an error. No-op if the id does not exist.
    TouchLastUsed(ctx context.Context, id string) error
}
