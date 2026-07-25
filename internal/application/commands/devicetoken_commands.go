package commands

import (
    "context"
    "errors"
    "fmt"

    "github.com.br/lucas-mezencio/pdsi1/internal/application"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// RegisterDeviceTokenCommand registers a new device token for the caller.
type RegisterDeviceTokenCommand struct {
    CallerFirebaseID string
    Token            string
}

// DeleteDeviceTokenCommand removes a device token owned by the caller.
type DeleteDeviceTokenCommand struct {
    CallerFirebaseID string
    TokenID          string
}

// SetDeviceTokenEnabledCommand toggles the enabled flag of a token.
type SetDeviceTokenEnabledCommand struct {
    CallerFirebaseID string
    TokenID          string
    Enabled          bool
}

// DeviceTokenCommandHandler routes write operations on device tokens.
type DeviceTokenCommandHandler struct {
    dtRepo devicetoken.Repository
    uRepo  user.Repository
}

// NewDeviceTokenCommandHandler builds the handler with its dependencies.
func NewDeviceTokenCommandHandler(
    dtRepo devicetoken.Repository,
    uRepo user.Repository,
) *DeviceTokenCommandHandler {
    return &DeviceTokenCommandHandler{dtRepo: dtRepo, uRepo: uRepo}
}

func (h *DeviceTokenCommandHandler) resolveCaller(ctx context.Context, fid string) (string, error) {
    if fid == "" {
        return "", application.ErrInvalidInput
    }
    u, err := h.uRepo.FindByFirebaseID(ctx, fid)
    if err != nil {
        if errors.Is(err, user.ErrUserNotFound) {
            return "", application.ErrUserNotFound
        }
        return "", err
    }
    return u.ID, nil
}

// RegisterDeviceToken upserts (or conflicts) a token for the caller.
func (h *DeviceTokenCommandHandler) RegisterDeviceToken(
    ctx context.Context,
    cmd RegisterDeviceTokenCommand,
) (*devicetoken.DeviceToken, error) {
    localID, err := h.resolveCaller(ctx, cmd.CallerFirebaseID)
    if err != nil {
        return nil, err
    }

    dt, err := devicetoken.New(localID, cmd.Token)
    if err != nil {
        return nil, application.ErrInvalidInput
    }

    saved, err := h.dtRepo.Save(ctx, dt)
    if err != nil {
        if errors.Is(err, devicetoken.ErrConflict) {
            return nil, application.ErrConflict
        }
        return nil, fmt.Errorf("save device token: %w", err)
    }
    return saved, nil
}

// DeleteDeviceToken removes the caller's token. 403 if it belongs to another user.
func (h *DeviceTokenCommandHandler) DeleteDeviceToken(
    ctx context.Context,
    cmd DeleteDeviceTokenCommand,
) error {
    localID, err := h.resolveCaller(ctx, cmd.CallerFirebaseID)
    if err != nil {
        return err
    }

    existing, err := h.dtRepo.FindByID(ctx, cmd.TokenID)
    if err != nil {
        if errors.Is(err, devicetoken.ErrNotFound) {
            return application.ErrNotFound
        }
        return fmt.Errorf("find device token: %w", err)
    }
    if existing.UserID != localID {
        return application.ErrForbidden
    }

    if err := h.dtRepo.Delete(ctx, cmd.TokenID); err != nil {
        return fmt.Errorf("delete device token: %w", err)
    }
    return nil
}

// SetDeviceTokenEnabled toggles the enabled flag.
func (h *DeviceTokenCommandHandler) SetDeviceTokenEnabled(
    ctx context.Context,
    cmd SetDeviceTokenEnabledCommand,
) (*devicetoken.DeviceToken, error) {
    localID, err := h.resolveCaller(ctx, cmd.CallerFirebaseID)
    if err != nil {
        return nil, err
    }

    existing, err := h.dtRepo.FindByID(ctx, cmd.TokenID)
    if err != nil {
        if errors.Is(err, devicetoken.ErrNotFound) {
            return nil, application.ErrNotFound
        }
        return nil, fmt.Errorf("find device token: %w", err)
    }
    if existing.UserID != localID {
        return nil, application.ErrForbidden
    }

    updated, err := h.dtRepo.SetEnabled(ctx, cmd.TokenID, cmd.Enabled)
    if err != nil {
        if errors.Is(err, devicetoken.ErrNotFound) {
            return nil, application.ErrNotFound
        }
        return nil, fmt.Errorf("set device token enabled: %w", err)
    }
    return updated, nil
}