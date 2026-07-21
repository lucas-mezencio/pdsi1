package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	gen "github.com.br/lucas-mezencio/pdsi1/internal/api/gen"
	"github.com.br/lucas-mezencio/pdsi1/internal/api/dto"
	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/prescription"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// Server implements the generated API interface.
type Server struct {
	userCommands         *commands.UserCommandHandler
	userQueries          *queries.UserQueryHandler
	doctorCommands       *commands.DoctorCommandHandler
	doctorQueries        *queries.DoctorQueryHandler
	prescriptionCommands *commands.PrescriptionCommandHandler
	prescriptionQueries  *queries.PrescriptionQueryHandler
	authCommands         *commands.AuthCommandHandler
}

// NewServer constructs a Server with handlers.
func NewServer(
	userCommands *commands.UserCommandHandler,
	userQueries *queries.UserQueryHandler,
	doctorCommands *commands.DoctorCommandHandler,
	doctorQueries *queries.DoctorQueryHandler,
	prescriptionCommands *commands.PrescriptionCommandHandler,
	prescriptionQueries *queries.PrescriptionQueryHandler,
	authCommands *commands.AuthCommandHandler,
) *Server {
	return &Server{
		userCommands:         userCommands,
		userQueries:          userQueries,
		doctorCommands:       doctorCommands,
		doctorQueries:        doctorQueries,
		prescriptionCommands: prescriptionCommands,
		prescriptionQueries:  prescriptionQueries,
		authCommands:         authCommands,
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
	writeJSON(w, http.StatusOK, dto.UserResponsesFromDomain(items))
}

func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	// firebase_id is NEVER accepted as input. The Firebase account is created
	// server-side via AuthenticationProvider. Push delivery tokens are
	// managed separately via the device-token endpoints.
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		CPF      string `json:"cpf"`
		Password string `json:"password"`
		Role     string `json:"role"` // "ELDERLY" | "CAREGIVER" (optional, defaults to "ELDERLY")
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	slog.DebugContext(r.Context(), "creating user",
		"email", body.Email,
		"role", body.Role,
	)

	created, err := s.userCommands.Create(r.Context(), commands.CreateUserCommand{
		Name:     body.Name,
		Email:    body.Email,
		Phone:    body.Phone,
		CPF:      body.CPF,
		Password: body.Password,
		Role:     body.Role,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}

	slog.DebugContext(r.Context(), "user created", "user_id", created.ID)
	writeJSON(w, http.StatusCreated, dto.UserResponseFromDomain(created))
}

func (s *Server) GetUserById(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	entity, err := s.userQueries.GetByID(r.Context(), queries.GetUserByIDQuery{ID: userId.String()})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.UserResponseFromDomain(entity))
}

func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	// Custom struct: gen.UpdateUserRequest does not include CPF, so we decode into
	// an extended body and only update CPF when it is provided.
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
		CPF   string `json:"cpf"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updated, err := s.userCommands.Update(r.Context(), commands.UpdateUserCommand{
		ID:    userId.String(),
		Name:  body.Name,
		Email: body.Email,
		Phone: body.Phone,
		CPF:   body.CPF,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.UserResponseFromDomain(updated))
}

func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request, userId gen.UserId) {
	if err := s.userCommands.Delete(r.Context(), commands.DeleteUserCommand{ID: userId.String()}); err != nil {
		writeCommandError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListDoctors(w http.ResponseWriter, r *http.Request) {
	items, err := s.doctorQueries.List(r.Context(), queries.ListDoctorsQuery{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list doctors", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto.DoctorResponsesFromDomain(items))
}

func (s *Server) CreateDoctor(w http.ResponseWriter, r *http.Request) {
	// firebase_id and password are NEVER accepted as input. Doctor is a
	// semantic entity that identifies who issued a prescription; it is
	// not an authenticatable principal.
	var body struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		Specialty     string `json:"specialty"`
		LicenseNumber string `json:"license_number"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	created, err := s.doctorCommands.Create(r.Context(), commands.CreateDoctorCommand{
		Name:          body.Name,
		Email:         body.Email,
		Phone:         body.Phone,
		Specialty:     body.Specialty,
		LicenseNumber: body.LicenseNumber,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.DoctorResponseFromDomain(created))
}

func (s *Server) GetDoctorById(w http.ResponseWriter, r *http.Request, doctorId gen.DoctorId) {
	entity, err := s.doctorQueries.GetByID(r.Context(), queries.GetDoctorByIDQuery{ID: doctorId.String()})
	if err != nil {
		writeQueryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.DoctorResponseFromDomain(entity))
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

	userID := body.UserId.String()
	medicID := body.MedicId.String()
	slog.DebugContext(r.Context(), "creating prescription",
		"user_id", userID,
		"medic_id", medicID,
		"medicament_count", len(body.Medicaments),
	)

	created, err := s.prescriptionCommands.Create(r.Context(), commands.CreatePrescriptionCommand{
		UserID:      userID,
		MedicID:     medicID,
		Medicaments: toDomainMedicaments(body.Medicaments),
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	slog.DebugContext(r.Context(), "prescription created", "prescription_id", created.ID)
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

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var body gen.LoginRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	slog.DebugContext(r.Context(), "login attempt", "email", string(body.Email))

	user, token, err := s.authCommands.Login(r.Context(), commands.LoginCommand{
		Email:    string(body.Email),
		Password: body.Password,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	slog.DebugContext(r.Context(), "login succeeded", "user_id", user.ID)
	writeJSON(w, http.StatusOK, gen.AuthResponse{
		Token: token,
		User:  userToGen(user),
	})
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	// Custom struct: gen.RegisterRequest does not include CPF, so we decode into
	// an extended body. CPF is optional and validated when present.
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		CPF      string `json:"cpf"`
		Password string `json:"password"`
		Role     string `json:"role"` // optional, defaults to "ELDERLY"
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	user, err := s.authCommands.Register(r.Context(), commands.RegisterCommand{
		Name:     body.Name,
		Email:    body.Email,
		Password: body.Password,
		Phone:    body.Phone,
		CPF:      body.CPF,
		Role:     body.Role,
	})
	if err != nil {
		writeCommandError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.UserResponseFromDomain(user))
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	// Auth token removal is handled client-side; server just acknowledges.
	w.WriteHeader(http.StatusNoContent)
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

// userToGen converts a domain user into the OpenAPI-generated User type so
// that handlers can build responses that match the published contract
// (gen.AuthResponse, etc.).
func userToGen(u *user.User) gen.User {
	if u == nil {
		return gen.User{}
	}
	genUser := gen.User{
		Name:      u.Name,
		Email:     openapi_types.Email(u.Email),
		Phone:     u.Phone,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if id, err := uuid.Parse(u.ID); err == nil {
		genUser.Id = gen.UserId(id)
	}
	return genUser
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
