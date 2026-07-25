package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// ListDeviceTokensQuery returns the caller's device tokens.
type ListDeviceTokensQuery struct {
	CallerFirebaseID string
}

// DeviceTokenQueryHandler serves read operations on device tokens.
type DeviceTokenQueryHandler struct {
	dtRepo devicetoken.Repository
	uRepo  user.Repository
}

// NewDeviceTokenQueryHandler builds the handler.
func NewDeviceTokenQueryHandler(
	dtRepo devicetoken.Repository,
	uRepo user.Repository,
) *DeviceTokenQueryHandler {
	return &DeviceTokenQueryHandler{dtRepo: dtRepo, uRepo: uRepo}
}

// ListDeviceTokens resolves the caller and returns their tokens.
func (h *DeviceTokenQueryHandler) ListDeviceTokens(
	ctx context.Context,
	q ListDeviceTokensQuery,
) ([]*devicetoken.DeviceToken, error) {
	if q.CallerFirebaseID == "" {
		return nil, application.ErrInvalidInput
	}

	u, err := h.uRepo.FindByFirebaseID(ctx, q.CallerFirebaseID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, application.ErrUserNotFound
		}
		return nil, fmt.Errorf("resolve caller: %w", err)
	}

	out, err := h.dtRepo.FindByUserID(ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}
	return out, nil
}