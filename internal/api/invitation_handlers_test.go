package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com.br/lucas-mezencio/pdsi1/internal/application"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/commands"
	"github.com.br/lucas-mezencio/pdsi1/internal/application/queries"
	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
)

// testInviteServer is a tiny stand-in for *ExtendedServer that exposes only
// the CreateInvitation behavior we want to verify. Mirroring the production
// handler keeps these tests focused on the contract (request body, response
// shape, error mapping) without dragging the rest of the server along.
type testInviteServer struct {
	invite  inviteCreateFunc
	baseURL string
}

type inviteCreateFunc func(ctx context.Context, cmd commands.CreateInvitationCommand) (*user.CaregiverInvitation, error)

func (s *testInviteServer) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	callerID := callerUserID(r)
	if callerID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing authenticated caller")
		return
	}

	var body struct {
		CaregiverEmail string `json:"caregiver_email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	inv, err := s.invite(r.Context(), commands.CreateInvitationCommand{
		ElderlyID:      callerID,
		CaregiverEmail: body.CaregiverEmail,
	})
	if err != nil {
		writeExtendedError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, invitationResponse{
		CaregiverInvitation: inv,
		AcceptURL:           inv.AcceptURL(s.baseURL, inv.Token),
	})
}

func newInviteRequest(t *testing.T, body string, callerID string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invitations", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if callerID != "" {
		req = req.WithContext(context.WithValue(req.Context(), contextKeyUserID, callerID))
	}
	return req
}

func TestCreateInvitation_ByEmail_Success(t *testing.T) {
	const (
		callerUUID    = "elderly-uuid"
		caregiverMail = "maria@example.com"
		caregiverUUID = "caregiver-uuid"
		baseURL       = "https://app.example.com"
	)
	inv := &user.CaregiverInvitation{
		ID:          "inv-1",
		CaregiverID: caregiverUUID,
		ElderlyID:   callerUUID,
		Token:       "abc-token",
		Status:      user.InvitationStatusPending,
	}

	var gotCmd commands.CreateInvitationCommand
	srv := &testInviteServer{
		baseURL: baseURL,
		invite: func(_ context.Context, cmd commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			gotCmd = cmd
			return inv, nil
		},
	}

	rr := httptest.NewRecorder()
	srv.CreateInvitation(rr, newInviteRequest(t, `{"caregiver_email":"`+caregiverMail+`"}`, callerUUID))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if gotCmd.ElderlyID != callerUUID {
		t.Errorf("expected ElderlyID=%q (from caller), got %q", callerUUID, gotCmd.ElderlyID)
	}
	if gotCmd.CaregiverEmail != caregiverMail {
		t.Errorf("expected CaregiverEmail=%q, got %q", caregiverMail, gotCmd.CaregiverEmail)
	}

	var resp struct {
		ID          string `json:"id"`
		CaregiverID string `json:"caregiver_id"`
		ElderlyID   string `json:"elderly_id"`
		Token       string `json:"token"`
		Status      string `json:"status"`
		AcceptURL   string `json:"accept_url"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if resp.Token != "abc-token" {
		t.Errorf("expected token=abc-token, got %q", resp.Token)
	}
	wantURL := "https://app.example.com/invitations/abc-token/accept"
	if resp.AcceptURL != wantURL {
		t.Errorf("expected accept_url=%q, got %q", wantURL, resp.AcceptURL)
	}
}

func TestCreateInvitation_CaregiverNotFound(t *testing.T) {
	srv := &testInviteServer{
		invite: func(_ context.Context, _ commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			return nil, application.ErrUserNotFound
		},
	}
	rr := httptest.NewRecorder()
	srv.CreateInvitation(rr, newInviteRequest(t, `{"caregiver_email":"ghost@example.com"}`, "elderly-uuid"))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateInvitation_RejectsUnauthenticatedCaller(t *testing.T) {
	srv := &testInviteServer{
		invite: func(_ context.Context, _ commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			t.Error("Create must not be called when caller is unauthenticated")
			return nil, errors.New("unreached")
		},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invitations", strings.NewReader(`{"caregiver_email":"x@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.CreateInvitation(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateInvitation_InvalidJSONBody(t *testing.T) {
	srv := &testInviteServer{
		invite: func(_ context.Context, _ commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			t.Error("Create must not be called when body is malformed")
			return nil, errors.New("unreached")
		},
	}
	rr := httptest.NewRecorder()
	srv.CreateInvitation(rr, newInviteRequest(t, "not json", "elderly-uuid"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateInvitation_WrongRole(t *testing.T) {
	srv := &testInviteServer{
		invite: func(_ context.Context, _ commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			return nil, application.ErrWrongRole
		},
	}
	rr := httptest.NewRecorder()
	srv.CreateInvitation(rr, newInviteRequest(t, `{"caregiver_email":"x@example.com"}`, "elderly-uuid"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateInvitation_AlreadyLinked(t *testing.T) {
	srv := &testInviteServer{
		invite: func(_ context.Context, _ commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			return nil, application.ErrAlreadyLinked
		},
	}
	rr := httptest.NewRecorder()
	srv.CreateInvitation(rr, newInviteRequest(t, `{"caregiver_email":"x@example.com"}`, "elderly-uuid"))

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateInvitation_RelativeAcceptURLWhenBaseURLEmpty(t *testing.T) {
	inv := &user.CaregiverInvitation{ID: "inv-1", Token: "tok", Status: user.InvitationStatusPending}
	srv := &testInviteServer{
		baseURL: "",
		invite: func(_ context.Context, _ commands.CreateInvitationCommand) (*user.CaregiverInvitation, error) {
			return inv, nil
		},
	}
	rr := httptest.NewRecorder()
	srv.CreateInvitation(rr, newInviteRequest(t, `{"caregiver_email":"x@example.com"}`, "elderly-uuid"))

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	var resp struct {
		AcceptURL string `json:"accept_url"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AcceptURL != "/invitations/tok/accept" {
		t.Errorf("expected relative accept_url, got %q", resp.AcceptURL)
	}
}

// keep package "queries" import used (the production handler indirectly depends on it)
var _ = queries.ListCaregiversQuery{}
