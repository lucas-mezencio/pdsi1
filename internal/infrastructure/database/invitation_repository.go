package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// InvitationRepository implements user.InvitationRepository using PostgreSQL.
type InvitationRepository struct {
	db *sql.DB
}

// NewInvitationRepository creates a new InvitationRepository.
func NewInvitationRepository(db *sql.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Save creates or updates a caregiver invitation.
func (r *InvitationRepository) Save(ctx context.Context, inv *user.CaregiverInvitation) error {
	query := `
		INSERT INTO caregiver_invitations (id, caregiver_id, elderly_id, token, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			status     = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		inv.ID,
		inv.CaregiverID,
		inv.ElderlyID,
		inv.Token,
		string(inv.Status),
		inv.CreatedAt,
		inv.UpdatedAt,
	)
	return err
}

// FindByToken retrieves an invitation by its token.
func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*user.CaregiverInvitation, error) {
	query := `
		SELECT id, caregiver_id, elderly_id, token, status, created_at, updated_at
		FROM caregiver_invitations
		WHERE LOWER(token) = LOWER($1)
	`

	var inv user.CaregiverInvitation
	var status string
	if err := r.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID,
		&inv.CaregiverID,
		&inv.ElderlyID,
		&inv.Token,
		&status,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrInvitationNotFound
		}
		return nil, err
	}

	inv.Status = user.InvitationStatus(status)
	return &inv, nil
}

// FindByElderlyID retrieves all invitations for an elderly user.
func (r *InvitationRepository) FindByElderlyID(ctx context.Context, elderlyID string) ([]*user.CaregiverInvitation, error) {
	query := `
		SELECT id, caregiver_id, elderly_id, token, status, created_at, updated_at
		FROM caregiver_invitations
		WHERE elderly_id = $1
		ORDER BY created_at DESC
	`
	return r.queryInvitations(ctx, query, elderlyID)
}

// FindByCaregiverID retrieves all invitations for a caregiver.
func (r *InvitationRepository) FindByCaregiverID(ctx context.Context, caregiverID string) ([]*user.CaregiverInvitation, error) {
	query := `
		SELECT id, caregiver_id, elderly_id, token, status, created_at, updated_at
		FROM caregiver_invitations
		WHERE caregiver_id = $1
		ORDER BY created_at DESC
	`
	return r.queryInvitations(ctx, query, caregiverID)
}

func (r *InvitationRepository) queryInvitations(ctx context.Context, query string, arg string) ([]*user.CaregiverInvitation, error) {
	rows, err := r.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*user.CaregiverInvitation
	for rows.Next() {
		var inv user.CaregiverInvitation
		var status string
		if err := rows.Scan(
			&inv.ID,
			&inv.CaregiverID,
			&inv.ElderlyID,
			&inv.Token,
			&status,
			&inv.CreatedAt,
			&inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		inv.Status = user.InvitationStatus(status)
		result = append(result, &inv)
	}

	return result, rows.Err()
}
