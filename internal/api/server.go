package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	gen "github.com.br/lucas-mezencio/pdsi1/internal/api/gen"
	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
)

// Server implements the generated API interface.
type Server struct {
	userCommands         *commands.UserCommandHandler
	userQueries          *queries.UserQueryHandler
	doctorCommands       *commands.DoctorCommandHandler
	doctorQueries        *queries.DoctorQueryHandler
	prescriptionCommands *commands.PrescriptionCommandHandler
	prescriptionQueries  *queries.PrescriptionQueryHandler
}

// NewServer constructs a Server with handlers.
func NewServer(
	userCommands *commands.UserCommandHandler,
	userQueries *queries.UserQueryHandler,
	doctorCommands *commands.DoctorCommandHandler,
	doctorQueries *queries.DoctorQueryHandler,
	prescriptionCommands *commands.PrescriptionCommandHandler,
	prescriptionQueries *queries.PrescriptionQueryHandler,
) *Server {
	return &Server{
		userCommands:         userCommands,
		userQueries:          userQueries,
		doctorCommands:       doctorCommands,
		doctorQueries:        doctorQueries,
		prescriptionCommands: prescriptionCommands,
		prescriptionQueries:  prescriptionQueries,
	}
}

type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {
	items, err := s.userQueries.List(r.Context(), queries.ListUsersQuery{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Custom struct to accept the optional `role` field alongside generated schema fields.
	var body struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		FirebaseToken string `json:"firebase_token"`
		Role          string `json:"role"` // "ELDERLY" | "CAREGIVER" (optional, defaults to "ELDERLY")
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	created, err := s.userCommands.Create(r.Context(), commands.CreateUserCommand{
		Name:          body.Name,
		Email:         body.Email,
		Phone:         body.Phone,
		FirebaseToken: body.FirebaseToken,
		Role:          body.Role,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) GetUserById(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	entity, err := s.userQueries.GetByID(r.Context(), queries.GetUserByIDQuery{ID: userId.String()})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	var body gen.UpdateUserRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updated, err := s.userCommands.Update(r.Context(), commands.UpdateUserCommand{
		ID:    userId.String(),
		Name:  body.Name,
		Email: string(body.Email),
		Phone: body.Phone,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	if err := s.userCommands.Delete(r.Context(), commands.DeleteUserCommand{ID: userId.String()}); err != nil {
		writeCommandError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) UpdateFirebaseToken(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	var body gen.UpdateFirebaseTokenJSONBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updated, err := s.userCommands.UpdateFirebaseToken(r.Context(), commands.UpdateUserFirebaseTokenCommand{
		ID:            userId.String(),
		FirebaseToken: body.FirebaseToken,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) ToggleNotifications(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	var body gen.ToggleNotificationsJSONBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updated, err := s.userCommands.ToggleNotifications(r.Context(), commands.ToggleUserNotificationsCommand{
		ID:      userId.String(),
		Enabled: body.Enabled,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) ListDoctors(w http.ResponseWriter, r *http.Request) {
	items, err := s.doctorQueries.List(r.Context(), queries.ListDoctorsQuery{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list doctors", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) CreateDoctor(w http.ResponseWriter, r *http.Request) {
	var body gen.CreateDoctorRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	created, err := s.doctorCommands.Create(r.Context(), commands.CreateDoctorCommand{
		Name:          body.Name,
		Email:         string(body.Email),
		Phone:         body.Phone,
		Specialty:     derefString(body.Specialty),
		LicenseNumber: body.LicenseNumber,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) GetDoctorById(w http.ResponseWriter, r *http.Request, doctorId gen.DoctorId) {
	entity, err := s.doctorQueries.GetByID(r.Context(), queries.GetDoctorByIDQuery{ID: doctorId.String()})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (s *Server) UpdateDoctor(w http.ResponseWriter, r *http.Request, doctorId gen.DoctorId) {
	var body gen.UpdateDoctorRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updated, err := s.doctorCommands.Update(r.Context(), commands.UpdateDoctorCommand{
		ID:        doctorId.String(),
		Name:      body.Name,
		Email:     string(body.Email),
		Phone:     body.Phone,
		Specialty: derefString(body.Specialty),
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) DeleteDoctor(w http.ResponseWriter, r *http.Request, doctorId gen.DoctorId) {
	if err := s.doctorCommands.Delete(r.Context(), commands.DeleteDoctorCommand{ID: doctorId.String()}); err != nil {
		writeCommandError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) CreatePrescription(w http.ResponseWriter, r *http.Request) {
	var body gen.CreatePrescriptionRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	created, err := s.prescriptionCommands.Create(r.Context(), commands.CreatePrescriptionCommand{
		UserID:      body.UserId.String(),
		MedicID:     body.MedicId.String(),
		Medicaments: toDomainMedicaments(body.Medicaments),
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) ListPrescriptions(w http.ResponseWriter, r *http.Request, params gen.ListPrescriptionsParams) {
	items, err := s.prescriptionQueries.List(r.Context(), queries.ListPrescriptionsQuery{
		UserID:  uuidPtrToString(params.UserId),
		MedicID: uuidPtrToStringDoctor(params.MedicId),
		Active:  params.Active,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list prescriptions", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) GetPrescriptionById(w http.ResponseWriter, r *http.Request, prescriptionId gen.PrescriptionId) {
	entity, err := s.prescriptionQueries.GetByID(r.Context(), queries.GetPrescriptionByIDQuery{ID: prescriptionId.String()})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (s *Server) UpdatePrescription(w http.ResponseWriter, r *http.Request, prescriptionId gen.PrescriptionId) {
	var body gen.UpdatePrescriptionRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updated, err := s.prescriptionCommands.UpdateMedicaments(r.Context(), commands.UpdatePrescriptionCommand{
		ID:          prescriptionId.String(),
		Medicaments: toDomainMedicaments(body.Medicaments),
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) DeletePrescription(w http.ResponseWriter, r *http.Request, prescriptionId gen.PrescriptionId) {
	if err := s.prescriptionCommands.Delete(r.Context(), commands.DeletePrescriptionCommand{ID: prescriptionId.String()}); err != nil {
		writeCommandError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ActivatePrescription(w http.ResponseWriter, r *http.Request, prescriptionId gen.PrescriptionId) {
	updated, err := s.prescriptionCommands.Activate(r.Context(), commands.ActivatePrescriptionCommand{ID: prescriptionId.String()})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) DeactivatePrescription(w http.ResponseWriter, r *http.Request, prescriptionId gen.PrescriptionId) {
	updated, err := s.prescriptionCommands.Deactivate(r.Context(), commands.DeactivatePrescriptionCommand{ID: prescriptionId.String()})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string, details string) {
	writeJSON(w, status, errorResponse{
		Error:   message,
		Details: details,
	})
}

func writeCommandError(w http.ResponseWriter, err error) {
	if errors.Is(err, application.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if errors.Is(err, application.ErrUserNotFound) || errors.Is(err, application.ErrDoctorNotFound) || errors.Is(err, application.ErrPrescriptionNotFound) {
		writeError(w, http.StatusNotFound, "not found", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error", err.Error())
}

func writeQueryError(w http.ResponseWriter, err error) {
	if errors.Is(err, application.ErrInvalidInput) {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if errors.Is(err, application.ErrUserNotFound) || errors.Is(err, application.ErrDoctorNotFound) || errors.Is(err, application.ErrPrescriptionNotFound) {
		writeError(w, http.StatusNotFound, "not found", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error", err.Error())
}

func uuidPtrToString(value *gen.UserId) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func uuidPtrToStringDoctor(value *gen.DoctorId) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func toDomainMedicaments(items []gen.Medicament) []prescription.Medicament {
	result := make([]prescription.Medicament, 0, len(items))
	for _, item := range items {
		result = append(result, prescription.Medicament{
			Name:      item.Name,
			Dosage:    item.Dosage,
			Frequency: item.Frequency,
			Times:     item.Time,
			Doses:     item.Doses,
		})
	}
	return result
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
