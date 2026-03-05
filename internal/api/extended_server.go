package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// ExtendedServer handles API endpoints that are not part of the generated spec.
type ExtendedServer struct {
	userRepo         user.Repository
	inviteCommands   *commands.InvitationCommandHandler
	doseCommands     *commands.DoseRecordCommandHandler
	doseQueries      *queries.DoseRecordQueryHandler
	linkedUserQueries *queries.LinkedUserQueryHandler
}

// NewExtendedServer creates an ExtendedServer.
func NewExtendedServer(
	userRepo user.Repository,
	inviteCommands *commands.InvitationCommandHandler,
	doseCommands *commands.DoseRecordCommandHandler,
	doseQueries *queries.DoseRecordQueryHandler,
	linkedUserQueries *queries.LinkedUserQueryHandler,
) *ExtendedServer {
	return &ExtendedServer{
		userRepo:          userRepo,
		inviteCommands:    inviteCommands,
		doseCommands:      doseCommands,
		doseQueries:       doseQueries,
		linkedUserQueries: linkedUserQueries,
	}
}

// --- Invitation endpoints ---

// CreateInvitation handles POST /invitations
func (s *ExtendedServer) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ElderlyID   string `json:"elderly_id"`
		CaregiverID string `json:"caregiver_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	inv, err := s.inviteCommands.Create(r.Context(), commands.CreateInvitationCommand{
		ElderlyID:   body.ElderlyID,
		CaregiverID: body.CaregiverID,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

// GetInvitationByToken handles GET /invitations/{token}
func (s *ExtendedServer) GetInvitationByToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	inv, err := s.linkedUserQueries.GetInvitationByToken(r.Context(), queries.GetInvitationByTokenQuery{Token: token})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// AcceptInvitation handles POST /invitations/{token}/accept
func (s *ExtendedServer) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	inv, err := s.inviteCommands.Accept(r.Context(), commands.AcceptInvitationCommand{Token: token})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// RejectInvitation handles POST /invitations/{token}/reject
func (s *ExtendedServer) RejectInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	inv, err := s.inviteCommands.Reject(r.Context(), commands.RejectInvitationCommand{Token: token})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// --- Linked user endpoints ---

// ListCaregivers handles GET /users/{userId}/caregivers
func (s *ExtendedServer) ListCaregivers(w http.ResponseWriter, r *http.Request) {
	elderlyID := chi.URLParam(r, "userId")
	callerID := callerUserID(r)

	items, err := s.linkedUserQueries.ListCaregivers(r.Context(), queries.ListCaregiversQuery{
		ElderlyID: elderlyID,
		CallerID:  callerID,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ListCharges handles GET /users/{userId}/charges
func (s *ExtendedServer) ListCharges(w http.ResponseWriter, r *http.Request) {
	caregiverID := chi.URLParam(r, "userId")
	callerID := callerUserID(r)

	items, err := s.linkedUserQueries.ListCharges(r.Context(), queries.ListChargesQuery{
		CaregiverID: caregiverID,
		CallerID:    callerID,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// UnlinkUsers handles DELETE /users/{userId}/caregivers/{caregiverId}
func (s *ExtendedServer) UnlinkUsers(w http.ResponseWriter, r *http.Request) {
	elderlyID := chi.URLParam(r, "userId")
	caregiverID := chi.URLParam(r, "caregiverId")
	callerID := callerUserID(r)
	callerRoleVal := callerRole(r)

	// Only the elderly user themselves or a caregiver in the link can unlink.
	if callerID != "" && callerID != elderlyID && callerID != caregiverID {
		writeError(w, http.StatusForbidden, "access denied", "only the linked parties can remove the link")
		return
	}
	_ = callerRoleVal

	if err := s.inviteCommands.Unlink(r.Context(), commands.UnlinkUsersCommand{
		CaregiverID: caregiverID,
		ElderlyID:   elderlyID,
	}); err != nil {
		writeExtendedError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Dose record endpoints ---

// ListDoseRecords handles GET /users/{userId}/dose-records
func (s *ExtendedServer) ListDoseRecords(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	callerID := callerUserID(r)

	records, err := s.doseQueries.ListByUser(r.Context(), queries.ListDoseRecordsQuery{
		UserID:   userID,
		CallerID: callerID,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

// ConfirmDose handles POST /dose-records/{doseRecordId}/confirm
func (s *ExtendedServer) ConfirmDose(w http.ResponseWriter, r *http.Request) {
	doseRecordID := chi.URLParam(r, "doseRecordId")
	callerID := callerUserID(r)

	record, err := s.doseCommands.Confirm(r.Context(), commands.ConfirmDoseCommand{
		DoseRecordID: doseRecordID,
		CallerID:     callerID,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// MarkDoseMissed handles POST /dose-records/{doseRecordId}/miss
func (s *ExtendedServer) MarkDoseMissed(w http.ResponseWriter, r *http.Request) {
	doseRecordID := chi.URLParam(r, "doseRecordId")
	callerID := callerUserID(r)

	record, err := s.doseCommands.Miss(r.Context(), commands.MissDoseCommand{
		DoseRecordID: doseRecordID,
		CallerID:     callerID,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func writeExtendedError(w http.ResponseWriter, err error) {
	if errors.Is(err, application.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if errors.Is(err, application.ErrForbidden) {
		writeError(w, http.StatusForbidden, "access denied", err.Error())
		return
	}
	if errors.Is(err, application.ErrWrongRole) {
		writeError(w, http.StatusUnprocessableEntity, "wrong role", err.Error())
		return
	}
	if errors.Is(err, application.ErrAlreadyLinked) {
		writeError(w, http.StatusConflict, "already linked", err.Error())
		return
	}
	if errors.Is(err, application.ErrInvitationNotPending) {
		writeError(w, http.StatusConflict, "invitation not pending", err.Error())
		return
	}
	if errors.Is(err, application.ErrUserNotFound) ||
		errors.Is(err, application.ErrInvitationNotFound) ||
		errors.Is(err, application.ErrDoseRecordNotFound) {
		writeError(w, http.StatusNotFound, "not found", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error", err.Error())
}
