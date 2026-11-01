package notification_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/i18n"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/notification"
)

// The fakes embed the interface so that any method a test does not expect
// panics on the nil interface instead of silently passing.
type fakeUsers struct {
	domain.UserRepository
	users map[uuid.UUID]*domain.User
}

func (f *fakeUsers) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, inerr.NewErrNotFound("user")
}

type fakeDevices struct {
	domain.FCMDeviceRepository
	lookedUp []uuid.UUID
}

func (f *fakeDevices) FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.FCMDevice, error) {
	f.lookedUp = append(f.lookedUp, userID)
	// No devices: delivery stops before FCM, which this test has no client for.
	return nil, nil
}

func TestNotify_LooksUpTheRecipientAndWordsThePush(t *testing.T) {
	id := uuid.New()
	users := &fakeUsers{users: map[uuid.UUID]*domain.User{id: {ID: id, Language: shared.LanguageUzbek}}}
	devices := &fakeDevices{}
	svc := notification.NewService(nil, devices, users)

	err := svc.Notify(context.Background(), id.String(), i18n.PushLoadAssigned,
		map[string]string{i18n.ParamTitle: "Paxta"}, map[string]string{"load_id": "l1"})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if len(devices.lookedUp) != 1 || devices.lookedUp[0] != id {
		t.Fatalf("devices looked up for %v, want exactly %v", devices.lookedUp, id)
	}
}

func TestNotify_SkipsARecipientWhoIsGone(t *testing.T) {
	users := &fakeUsers{users: map[uuid.UUID]*domain.User{}}
	devices := &fakeDevices{}
	svc := notification.NewService(nil, devices, users)

	err := svc.Notify(context.Background(), uuid.NewString(), i18n.PushLoadAssigned,
		map[string]string{i18n.ParamTitle: "Paxta"}, nil)
	if err != nil {
		t.Fatalf("Notify for a deleted user = %v, want nil so the task is not retried", err)
	}
	if len(devices.lookedUp) != 0 {
		t.Fatalf("devices were looked up for a deleted user")
	}
}

func TestNotify_FailsOnAnUnknownKey(t *testing.T) {
	id := uuid.New()
	users := &fakeUsers{users: map[uuid.UUID]*domain.User{id: {ID: id, Language: shared.LanguageRussian}}}
	svc := notification.NewService(nil, &fakeDevices{}, users)

	if err := svc.Notify(context.Background(), id.String(), "push.nope", nil, nil); err == nil {
		t.Fatal("Notify with an unknown key returned nil")
	}
}
