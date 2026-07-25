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

	// TouchLastUsed records that the given device-token id was used to send
	// a notification. Best-effort: implementations may swallow errors to keep
	// the send path fast, but the Lookup port is the natural place for the
	// call site to live since it already owns the token lifecycle from the
	// scheduler worker's perspective.
	TouchLastUsed(ctx context.Context, id string) error
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

// TouchLastUsed delegates to the underlying repository.
func (p *PostgresLookup) TouchLastUsed(ctx context.Context, id string) error {
	return p.repo.TouchLastUsed(ctx, id)
}
