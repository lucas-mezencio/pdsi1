package commands

import (
    "context"
    "fmt"

    "github.com.br/lucas-mezencio/pdsi1/internal/application"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
    "github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// RegisterDeviceTokenCommand registers a new device token for the caller.
// CallerID is the LOCAL user UUID resolved by AuthMiddleware (NOT the
// Firebase UID). Pass it to FK-bound columns like user_device_tokens.user_id.
type RegisterDeviceTokenCommand struct {
    CallerID string
    Token    string
}

// DeleteDeviceTokenCommand removes a device token owned by the caller.
type DeleteDeviceTokenCommand struct {
    CallerID string
    TokenID  string
}

// SetDeviceTokenEnabledCommand toggles the enabled flag of a token.
type SetDeviceTokenEnabledCommand struct {
    CallerID string
    TokenID  string
    Enabled  bool
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

// RegisterDeviceToken upserts (or conflicts) a token for the caller.
func (h *DeviceTokenCommandHandler) RegisterDeviceToken(
    ctx context.Context,
    cmd RegisterDeviceTokenCommand,
) (*devicetoken.DeviceToken, error) {
    if cmd.CallerID == "" {
        return nil, application.ErrInvalidInput
    }

    dt, err := devicetoken.New(cmd.CallerID, cmd.Token)
    if err != nil {
        return nil, application.ErrInvalidInput
    }

    saved, err := h.dtRepo.Save(ctx, dt)
    if err != nil {
        if err == devicetoken.ErrConflict {
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
    if cmd.CallerID == "" {
        return application.ErrInvalidInput
    }

    existing, err := h.dtRepo.FindByID(ctx, cmd.TokenID)
    if err != nil {
        if err == devicetoken.ErrNotFound {
            return application.ErrNotFound
        }
        return fmt.Errorf("find device token: %w", err)
    }
    if existing.UserID != cmd.CallerID {
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
    if cmd.CallerID == "" {
        return nil, application.ErrInvalidInput
    }

    existing, err := h.dtRepo.FindByID(ctx, cmd.TokenID)
    if err != nil {
        if err == devicetoken.ErrNotFound {
            return nil, application.ErrNotFound
        }
        return nil, fmt.Errorf("find device token: %w", err)
    }
    if existing.UserID != cmd.CallerID {
        return nil, application.ErrForbidden
    }

    updated, err := h.dtRepo.SetEnabled(ctx, cmd.TokenID, cmd.Enabled)
    if err != nil {
        if err == devicetoken.ErrNotFound {
            return nil, application.ErrNotFound
        }
        return nil, fmt.Errorf("set device token enabled: %w", err)
    }
    return updated, nil
}