package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com.br/lucas-mezencio/pdsi1/internal/domain/user"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/notification"
)

// stubSender records each Send call and reports a configurable error.
type stubSender struct {
	mu        sync.Mutex
	calls     []notification.Notification
	returnErr error
}

func (s *stubSender) Send(_ context.Context, n notification.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, n)
	return s.returnErr
}

func (s *stubSender) callsSnapshot() []notification.Notification {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]notification.Notification, len(s.calls))
	copy(out, s.calls)
	return out
}

// stubUserRepo returns a single user for the given ID.
type stubUserRepo struct {
	users       map[string]*user.User
	caregivers  map[string][]*user.User
	findByIDErr error
}

func (s *stubUserRepo) Save(context.Context, *user.User) error { return nil }
func (s *stubUserRepo) FindByID(_ context.Context, id string) (*user.User, error) {
	if s.findByIDErr != nil {
		return nil, s.findByIDErr
	}
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepo) FindByEmail(context.Context, string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepo) FindByFirebaseID(context.Context, string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepo) FindAll(context.Context) ([]*user.User, error) { return nil, nil }
func (s *stubUserRepo) Delete(context.Context, string) error          { return nil }
func (s *stubUserRepo) Exists(_ context.Context, id string) (bool, error) {
	_, ok := s.users[id]
	return ok, nil
}
func (s *stubUserRepo) FindCaregivers(_ context.Context, elderlyID string) ([]*user.User, error) {
	return s.caregivers[elderlyID], nil
}
func (s *stubUserRepo) FindCharges(context.Context, string) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepo) IsLinked(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *stubUserRepo) LinkUsers(context.Context, string, string) error   { return nil }
func (s *stubUserRepo) UnlinkUsers(context.Context, string, string) error { return nil }

// stubEventStore records NotificationEvent saves so tests can assert what
// the consumer persisted for skipped/failed deliveries.
type stubEventStore struct {
	mu      sync.Mutex
	saved   []NotificationEvent
	saveErr error
}

func (s *stubEventStore) Save(_ context.Context, event NotificationEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = append(s.saved, event)
	return nil
}

func (s *stubEventStore) savedSnapshot() []NotificationEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]NotificationEvent, len(s.saved))
	copy(out, s.saved)
	return out
}

// stubLookup returns active tokens and records TouchLastUsed calls.
type stubLookup struct {
	mu           sync.Mutex
	tokensByUser map[string][]notification.Token
	touched      []string
	returnErr    error
}

func (s *stubLookup) ActiveTokens(_ context.Context, userID string) ([]notification.Token, error) {
	return s.tokensByUser[userID], nil
}

func (s *stubLookup) TouchLastUsed(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touched = append(s.touched, id)
	return s.returnErr
}

func (s *stubLookup) touchedSnapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.touched))
	copy(out, s.touched)
	return out
}

func TestStartNotificationConsumer_TouchLastUsedOnSuccessfulSend(t *testing.T) {
	pubSub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, watermill.NopLogger{})

	sender := &stubSender{}
	lookup := &stubLookup{
		tokensByUser: map[string][]notification.Token{
			"user-1": {{DeviceTokenID: "dt-1", FCMToken: "fcm-aaa"}, {DeviceTokenID: "dt-2", FCMToken: "fcm-bbb"}},
		},
	}
	users := &stubUserRepo{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Role: user.RoleElderly},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- StartNotificationConsumer(ctx, pubSub, sender, users, lookup, nil, nil)
	}()

	job := NotificationJob{
		ID:             "job-1",
		PrescriptionID: "rx-1",
		UserID:         "user-1",
		MedicamentName: "Aspirin",
		Dosage:         "100mg",
		ScheduledAt:    time.Now(),
	}
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	if err := pubSub.Publish(NotificationTopic, message.NewMessage("msg-1", payload)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait until sender has recorded 2 sends (one per token).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sender.callsSnapshot()) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	gotSends := sender.callsSnapshot()
	if len(gotSends) != 2 {
		t.Fatalf("expected 2 Send calls, got %d", len(gotSends))
	}
	touched := lookup.touchedSnapshot()
	if len(touched) != 2 {
		t.Fatalf("expected 2 TouchLastUsed calls (one per token), got %d: %v", len(touched), touched)
	}
	want := map[string]bool{"dt-1": true, "dt-2": true}
	for _, id := range touched {
		if !want[id] {
			t.Fatalf("unexpected TouchLastUsed id %q (want dt-1 or dt-2)", id)
		}
	}

	cancel()
	pubSub.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("consumer returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not return after cancel")
	}
}

// TestStartNotificationConsumer_NoTokens_AcksAndSkips verifies that when the
// elderly user has zero active device tokens, the consumer Acks the message
// (does NOT Nack, which would trigger the watermill redis-stream ResendLoop
// infinite loop seen in prod), records a SKIPPED_NO_TOKENS event, and exits
// cleanly on cancel.
func TestStartNotificationConsumer_NoTokens_AcksAndSkips(t *testing.T) {
	pubSub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, watermill.NopLogger{})

	sender := &stubSender{}
	lookup := &stubLookup{
		tokensByUser: map[string][]notification.Token{
			"user-1": nil,
		},
	}
	users := &stubUserRepo{
		users: map[string]*user.User{
			"user-1": {ID: "user-1", Role: user.RoleElderly},
		},
	}
	store := &stubEventStore{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- StartNotificationConsumer(ctx, pubSub, sender, users, lookup, nil, store)
	}()

	job := NotificationJob{
		ID:             "job-skip",
		PrescriptionID: "rx-skip",
		UserID:         "user-1",
		MedicamentName: "Aspirin",
		Dosage:         "100mg",
		ScheduledAt:    time.Now(),
	}
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	if err := pubSub.Publish(NotificationTopic, message.NewMessage("msg-skip", payload)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait for the skip event to be recorded (consumer must Ack the message
	// and not loop on it).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.savedSnapshot()) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if calls := sender.callsSnapshot(); len(calls) != 0 {
		t.Fatalf("expected 0 Send calls for tokens-less user, got %d", len(calls))
	}

	saved := store.savedSnapshot()
	if len(saved) != 1 {
		t.Fatalf("expected 1 skip event recorded, got %d", len(saved))
	}
	if saved[0].UserID != "user-1" {
		t.Errorf("skip event user_id = %q, want %q", saved[0].UserID, "user-1")
	}
	if saved[0].Status != StatusSkippedNoTokens {
		t.Errorf("skip event status = %q, want %q", saved[0].Status, StatusSkippedNoTokens)
	}

	// Critical regression guard: the consumer must not be stuck in a Nack
	// ResendLoop. It must exit promptly on cancel.
	cancel()
	pubSub.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("consumer returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consumer hung after cancel (possible infinite Nack loop)")
	}

	// No additional Send calls should have happened after cancel.
	if calls := sender.callsSnapshot(); len(calls) != 0 {
		t.Fatalf("sender.Send called %d times after no-tokens skip", len(calls))
	}
}

// TestStartNotificationConsumer_NoTokens_StillFansOutToCaregivers verifies
// that when the elderly user has zero active device tokens, the caregiver
// fan-out still runs. Caregivers are the safety net when the elderly person
// has not installed the mobile app, so skipping them because the elderly
// user has no tokens defeats the purpose of the caregiver relationship.
func TestStartNotificationConsumer_NoTokens_StillFansOutToCaregivers(t *testing.T) {
	pubSub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, watermill.NopLogger{})

	sender := &stubSender{}
	lookup := &stubLookup{
		tokensByUser: map[string][]notification.Token{
			"elderly-1":  nil,
			"caregiver-1": {{DeviceTokenID: "cg-dt-1", FCMToken: "cg-fcm-1"}},
			"caregiver-2": {{DeviceTokenID: "cg-dt-2", FCMToken: "cg-fcm-2"}},
		},
	}
	users := &stubUserRepo{
		users: map[string]*user.User{
			"elderly-1":   {ID: "elderly-1", Role: user.RoleElderly},
			"caregiver-1": {ID: "caregiver-1", Role: user.RoleCaregiver},
			"caregiver-2": {ID: "caregiver-2", Role: user.RoleCaregiver},
		},
		caregivers: map[string][]*user.User{
			"elderly-1": {
				{ID: "caregiver-1", Role: user.RoleCaregiver},
				{ID: "caregiver-2", Role: user.RoleCaregiver},
			},
		},
	}
	store := &stubEventStore{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- StartNotificationConsumer(ctx, pubSub, sender, users, lookup, nil, store)
	}()

	job := NotificationJob{
		ID:             "job-cg-fanout",
		PrescriptionID: "rx-cg",
		UserID:         "elderly-1",
		MedicamentName: "Aspirin",
		Dosage:         "100mg",
		ScheduledAt:    time.Now(),
	}
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	if err := pubSub.Publish(NotificationTopic, message.NewMessage("msg-cg", payload)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait until sender has recorded the 2 caregiver sends (one per caregiver).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sender.callsSnapshot()) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	calls := sender.callsSnapshot()
	if len(calls) != 2 {
		t.Fatalf("expected 2 caregiver Send calls (elderly has no tokens), got %d", len(calls))
	}

	gotTokens := map[string]bool{}
	for _, c := range calls {
		gotTokens[c.FirebaseToken] = true
	}
	if !gotTokens["cg-fcm-1"] || !gotTokens["cg-fcm-2"] {
		t.Fatalf("expected caregiver FCM tokens cg-fcm-1 and cg-fcm-2, got %v", gotTokens)
	}

	// Skip event for the elderly user must still be recorded.
	saved := store.savedSnapshot()
	if len(saved) != 1 {
		t.Fatalf("expected 1 skip event for elderly user, got %d", len(saved))
	}
	if saved[0].Status != StatusSkippedNoTokens {
		t.Errorf("skip event status = %q, want %q", saved[0].Status, StatusSkippedNoTokens)
	}
	if saved[0].UserID != "elderly-1" {
		t.Errorf("skip event user_id = %q, want %q", saved[0].UserID, "elderly-1")
	}

	cancel()
	pubSub.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("consumer returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consumer hung after cancel")
	}
}

// TestStartNotificationConsumer_SendError_BoundedRetryThenAck verifies that
// when the sender returns an error and the message metadata shows the
// attempt counter has reached the maximum, the consumer records a
// SKIPPED_RETRIES_EXHAUSTED event, still fans out to caregivers (which
// have their own independent tokens), Acks the message (no infinite loop),
// and exits cleanly on cancel.
func TestStartNotificationConsumer_SendError_BoundedRetryThenAck(t *testing.T) {
	pubSub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, watermill.NopLogger{})

	sendErr := errors.New("fcm 503")
	sender := &stubSender{returnErr: sendErr}
	lookup := &stubLookup{
		tokensByUser: map[string][]notification.Token{
			"elderly-1":    {{DeviceTokenID: "e-dt-1", FCMToken: "e-fcm-1"}},
			"caregiver-1": {{DeviceTokenID: "cg-dt-1", FCMToken: "cg-fcm-1"}},
		},
	}
	users := &stubUserRepo{
		users: map[string]*user.User{
			"elderly-1":   {ID: "elderly-1", Role: user.RoleElderly},
			"caregiver-1": {ID: "caregiver-1", Role: user.RoleCaregiver},
		},
		caregivers: map[string][]*user.User{
			"elderly-1": {{ID: "caregiver-1", Role: user.RoleCaregiver}},
		},
	}
	store := &stubEventStore{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- StartNotificationConsumer(ctx, pubSub, sender, users, lookup, nil, store)
	}()

	job := NotificationJob{
		ID:             "job-retry-exhausted",
		PrescriptionID: "rx-retry",
		UserID:         "elderly-1",
		MedicamentName: "Aspirin",
		Dosage:         "100mg",
		ScheduledAt:    time.Now(),
	}
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}

	// Pre-set attempt counter to maxAttempts-1 so the FIRST send failure
	// immediately exhausts retries (the unit test focuses on the terminal
	// path, not on driving N redeliveries through the test broker).
	msg := message.NewMessage("msg-retry", payload)
	msg.Metadata = map[string]string{
		"x-attempt": "2",
	}
	if err := pubSub.Publish(NotificationTopic, msg); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Wait for at least one caregiver Send call (proves fan-out ran after
	// retries were exhausted) AND the SKIPPED_RETRIES_EXHAUSTED event.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		calls := sender.callsSnapshot()
		saved := store.savedSnapshot()
		if len(calls) >= 2 && len(saved) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	calls := sender.callsSnapshot()
	if len(calls) < 2 {
		t.Fatalf("expected at least 2 Send calls (elderly + caregiver), got %d", len(calls))
	}

	saved := store.savedSnapshot()
	if len(saved) != 1 {
		t.Fatalf("expected 1 skip event recorded, got %d", len(saved))
	}
	if saved[0].Status != StatusSkippedRetriesDone {
		t.Errorf("skip event status = %q, want %q", saved[0].Status, StatusSkippedRetriesDone)
	}
	if saved[0].UserID != "elderly-1" {
		t.Errorf("skip event user_id = %q, want %q", saved[0].UserID, "elderly-1")
	}

	// The caregiver Send should have succeeded independently even though the
	// elderly Send failed.
	gotCG := false
	for _, c := range calls {
		if c.FirebaseToken == "cg-fcm-1" {
			gotCG = true
		}
	}
	if !gotCG {
		t.Errorf("expected caregiver FCM token cg-fcm-1 to be sent, got %v", calls)
	}

	cancel()
	pubSub.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("consumer returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consumer hung after cancel")
	}
}
