package queries

import (
	"context"
	"fmt"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
)

// ListDeviceTokensQuery returns the caller's device tokens.
// CallerID is the LOCAL user UUID (FK column value), resolved upstream by
// AuthMiddleware. The Firebase UID is not needed here — every lookup after
// the middleware runs against UUID-bound columns.
type ListDeviceTokensQuery struct {
	CallerID string
}

// DeviceTokenQueryHandler serves read operations on device tokens.
type DeviceTokenQueryHandler struct {
	dtRepo devicetoken.Repository
}

// NewDeviceTokenQueryHandler builds the handler.
func NewDeviceTokenQueryHandler(
	dtRepo devicetoken.Repository,
) *DeviceTokenQueryHandler {
	return &DeviceTokenQueryHandler{dtRepo: dtRepo}
}

// ListDeviceTokens returns the caller's device tokens.
func (h *DeviceTokenQueryHandler) ListDeviceTokens(
	ctx context.Context,
	q ListDeviceTokensQuery,
) ([]*devicetoken.DeviceToken, error) {
	if q.CallerID == "" {
		return nil, application.ErrInvalidInput
	}

	out, err := h.dtRepo.FindByUserID(ctx, q.CallerID)
	if err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}
	return out, nil
}