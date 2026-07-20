package notification

import (
	"context"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/devicetoken"
)

// Token is the minimum data the scheduler worker needs to push a notification.
type Token struct {
	DeviceTokenID string
	FCMToken      string
}

// Lookup returns active device tokens for a user.
type Lookup interface {
	ActiveTokens(ctx context.Context, userID string) ([]Token, error)
}

// PostgresLookup implements Lookup against the DeviceTokenRepository.
type PostgresLookup struct {
	repo devicetoken.Repository
}

// NewPostgresLookup builds a Lookup backed by the given repository.
func NewPostgresLookup(repo devicetoken.Repository) *PostgresLookup {
	return &PostgresLookup{repo: repo}
}

// ActiveTokens returns only the enabled tokens for the user.
func (p *PostgresLookup) ActiveTokens(ctx context.Context, userID string) ([]Token, error) {
	rows, err := p.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Token, 0, len(rows))
	for _, r := range rows {
		if !r.Enabled {
			continue
		}
		out = append(out, Token{
			DeviceTokenID: r.ID,
			FCMToken:      r.Token,
		})
	}
	return out, nil
}
