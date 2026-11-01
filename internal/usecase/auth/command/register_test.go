package command_test

import (
	"context"
	"errors"
	"html/template"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/email"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth/command"
)

// fakeUsers implements domain.UserRepository; only FindByEmail and Save are used.
type fakeUsers struct {
	domain.UserRepository
	byEmail map[shared.Email]*domain.User
}

func (r *fakeUsers) FindByEmail(ctx context.Context, e shared.Email) (*domain.User, error) {
	if u, ok := r.byEmail[e]; ok {
		return u, nil
	}
	return nil, inerr.NewErrNotFound("user")
}

func (r *fakeUsers) Save(ctx context.Context, u *domain.User) error {
	r.byEmail[u.Email] = u
	return nil
}

type fakeAttributions struct {
	saved []*domain.UserAttribution
}

func (r *fakeAttributions) Save(ctx context.Context, a *domain.UserAttribution) error {
	r.saved = append(r.saved, a)
	return nil
}

// noTx runs fn in place.
type noTx struct{}

func (noTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type memOTPStore map[string]otp.OTPRecord

func (s memOTPStore) Save(ctx context.Context, key string, r *otp.OTPRecord, ttl time.Duration) error {
	s[key] = *r
	return nil
}

func (s memOTPStore) Get(ctx context.Context, key string) (*otp.OTPRecord, error) {
	r, ok := s[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &r, nil
}

func (s memOTPStore) Delete(ctx context.Context, key string) error {
	delete(s, key)
	return nil
}

type nopMailer struct{}

func (nopMailer) SendHTML(ctx context.Context, to, subject, body string) error { return nil }

type nopEmails struct{}

func (nopEmails) Save(ctx context.Context, e *domain.Email) error { return nil }

func newRegister(users *fakeUsers, attributions *fakeAttributions) *command.RegisterUsecase {
	otpSvc := otp.NewService(memOTPStore{}, otp.Config{
		Secret: []byte("test-secret"), TTL: 15 * time.Minute, MaxAttempts: 3, Length: 6,
	})
	// Named like the file internal/app/server.go parses it from.
	tmpl := template.Must(template.New("email_verification.html").Parse("{{.Code}}"))
	emailSvc := email.NewService(nopMailer{}, nopEmails{}, tmpl, "no-reply@example.com", 15*time.Minute)
	return command.NewRegisterUsecase(time.Second, noTx{}, users, attributions, otpSvc, emailSvc)
}

func registerRequest(attribution *command.AttributionRequest) *command.RegisterRequest {
	return &command.RegisterRequest{
		FirstName:   "Ali",
		Email:       "ali@example.com",
		Password:    "Secret-pass-123",
		Role:        "shipper",
		Attribution: attribution,
	}
}

func TestRegisterRecordsAttributionOfNewUser(t *testing.T) {
	users := &fakeUsers{byEmail: map[shared.Email]*domain.User{}}
	attributions := &fakeAttributions{}

	err := newRegister(users, attributions).Register(context.Background(), registerRequest(&command.AttributionRequest{
		Source: "Telegram", Medium: "cpc", Campaign: "launch_ru_2026_11", Referrer: "t.me", LandingLang: "ru",
	}))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if len(attributions.saved) != 1 {
		t.Fatalf("saved %d attributions, want 1", len(attributions.saved))
	}
	a := attributions.saved[0]
	user := users.byEmail["ali@example.com"]
	if a.UserID != user.ID {
		t.Errorf("attribution UserID = %v, want the new user %v", a.UserID, user.ID)
	}
	if a.Source != "telegram" || a.Medium != "cpc" || a.Campaign != "launch_ru_2026_11" || a.Referrer != "t.me" || a.LandingLang != "ru" {
		t.Errorf("attribution = %+v, want the request's tags", a.AttributionTags)
	}
}

func TestRegisterWithoutAttributionRecordsDirect(t *testing.T) {
	users := &fakeUsers{byEmail: map[shared.Email]*domain.User{}}
	attributions := &fakeAttributions{}

	if err := newRegister(users, attributions).Register(context.Background(), registerRequest(nil)); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if len(attributions.saved) != 1 || attributions.saved[0].Source != domain.AttributionSourceDirect {
		t.Fatalf("saved %+v, want one attribution with source direct", attributions.saved)
	}
}

func TestRegisterAgainOnPendingEmailKeepsFirstAttribution(t *testing.T) {
	users := &fakeUsers{byEmail: map[shared.Email]*domain.User{}}
	attributions := &fakeAttributions{}
	register := newRegister(users, attributions)

	if err := register.Register(context.Background(), registerRequest(&command.AttributionRequest{Source: "telegram"})); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := register.Register(context.Background(), registerRequest(&command.AttributionRequest{Source: "google"})); err != nil {
		t.Fatalf("second Register: %v", err)
	}

	if len(attributions.saved) != 1 || attributions.saved[0].Source != "telegram" {
		t.Fatalf("saved %+v, want only the first registration's telegram", attributions.saved)
	}
}
