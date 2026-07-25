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
		done <- StartNotificationConsumer(ctx, pubSub, sender, users, lookup, nil)
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
