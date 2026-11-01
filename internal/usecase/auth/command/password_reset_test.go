package command_test

import (
	"context"
	"errors"
	"html/template"
	"regexp"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/email"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/auth/command"
)

// resetUsers implements domain.UserRepository; only FindByEmail and Update are used.
type resetUsers struct {
	domain.UserRepository
	user    *domain.User
	updated *domain.User
}

func (r *resetUsers) FindByEmail(ctx context.Context, e shared.Email) (*domain.User, error) {
	if r.user != nil && r.user.Email == e {
		return r.user, nil
	}
	return nil, inerr.NewErrNotFound("user")
}

func (r *resetUsers) Update(ctx context.Context, u *domain.User) error {
	r.updated = u
	return nil
}

type resetOTPStore map[string]otp.OTPRecord

func (s resetOTPStore) Save(ctx context.Context, key string, r *otp.OTPRecord, ttl time.Duration) error {
	s[key] = *r
	return nil
}

func (s resetOTPStore) Get(ctx context.Context, key string) (*otp.OTPRecord, error) {
	r, ok := s[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &r, nil
}

func (s resetOTPStore) Delete(ctx context.Context, key string) error {
	delete(s, key)
	return nil
}

// resetMailer keeps the bodies it was asked to send.
type resetMailer struct {
	bodies []string
}

func (m *resetMailer) SendHTML(ctx context.Context, to, subject, body string) error {
	m.bodies = append(m.bodies, body)
	return nil
}

type resetEmails struct{}

func (resetEmails) Save(ctx context.Context, e *domain.Email) error { return nil }

type resetRevocation struct {
	revoked map[string]time.Time
}

func (r *resetRevocation) RevokeAllBefore(ctx context.Context, userID string, before time.Time) error {
	r.revoked[userID] = before
	return nil
}

func (r *resetRevocation) IsRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error) {
	return false, nil
}

const resetEmail = "ali@example.com"

var sixDigits = regexp.MustCompile(`\d{6}`)

type resetFixture struct {
	users      *resetUsers
	store      resetOTPStore
	mailer     *resetMailer
	revocation *resetRevocation
	forgot     *command.ForgotPasswordUsecase
	reset      *command.ResetPasswordUsecase
}

func newResetFixture(t *testing.T, user *domain.User) *resetFixture {
	t.Helper()
	f := &resetFixture{
		users:      &resetUsers{user: user},
		store:      resetOTPStore{},
		mailer:     &resetMailer{},
		revocation: &resetRevocation{revoked: map[string]time.Time{}},
	}
	otpSvc := otp.NewService(f.store, otp.Config{
		Secret: []byte("test-secret"), TTL: 15 * time.Minute, MaxAttempts: 3, Length: 6, Cooldown: time.Minute,
	})
	tmpl := template.Must(template.New(email.TemplatePasswordReset).Parse("{{.Code}}"))
	emailSvc := email.NewService(f.mailer, resetEmails{}, tmpl, "no-reply@example.com", 15*time.Minute)
	f.forgot = command.NewForgotPasswordUsecase(time.Second, f.users, otpSvc, emailSvc)
	f.reset = command.NewResetPasswordUsecase(time.Second, f.users, otpSvc, f.revocation)
	return f
}

func activeUser(t *testing.T) *domain.User {
	t.Helper()
	password, _ := shared.NewPassword("old-password")
	user, err := domain.NewUser("Ali", "", shared.Email(resetEmail), "", password, shared.RoleShipper)
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	return user
}

// mailedCode is the code in the last email sent.
func (f *resetFixture) mailedCode(t *testing.T) string {
	t.Helper()
	if len(f.mailer.bodies) == 0 {
		t.Fatal("no email was sent")
	}
	return sixDigits.FindString(f.mailer.bodies[len(f.mailer.bodies)-1])
}

func TestResetPasswordWithMailedCode(t *testing.T) {
	ctx := context.Background()
	user := activeUser(t)
	f := newResetFixture(t, user)

	if err := f.forgot.ForgotPassword(ctx, &command.ForgotPasswordRequest{Email: resetEmail}); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}
	code := f.mailedCode(t)

	err := f.reset.ResetPassword(ctx, &command.ResetPasswordRequest{Email: resetEmail, Code: code, NewPassword: "new-password"})
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	if f.users.updated == nil {
		t.Fatal("user was not saved")
	}
	if shared.Password("new-password").Verify(f.users.updated.PasswordHash) != nil {
		t.Error("saved hash does not match the new password")
	}
	if _, ok := f.revocation.revoked[user.ID.String()]; !ok {
		t.Error("tokens were not revoked")
	}
	// The code is spent.
	err = f.reset.ResetPassword(ctx, &command.ResetPasswordRequest{Email: resetEmail, Code: code, NewPassword: "third-password"})
	if !errors.Is(err, inerr.ErrOTPNotFound) {
		t.Errorf("second ResetPassword with the same code = %v, want ErrOTPNotFound", err)
	}
}

func TestResetPasswordWrongCode(t *testing.T) {
	ctx := context.Background()
	f := newResetFixture(t, activeUser(t))

	if err := f.forgot.ForgotPassword(ctx, &command.ForgotPasswordRequest{Email: resetEmail}); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}
	wrong := "000000"
	if f.mailedCode(t) == wrong {
		wrong = "111111"
	}

	err := f.reset.ResetPassword(ctx, &command.ResetPasswordRequest{Email: resetEmail, Code: wrong, NewPassword: "new-password"})
	if !errors.Is(err, inerr.ErrOTPMismatch) {
		t.Fatalf("ResetPassword = %v, want ErrOTPMismatch", err)
	}
	if f.users.updated != nil || len(f.revocation.revoked) != 0 {
		t.Error("a wrong code changed the password or revoked tokens")
	}
}

func TestForgotPasswordSendsNothingToUnknownOrPendingEmail(t *testing.T) {
	ctx := context.Background()

	unknown := newResetFixture(t, nil)
	if err := unknown.forgot.ForgotPassword(ctx, &command.ForgotPasswordRequest{Email: resetEmail}); err != nil {
		t.Fatalf("ForgotPassword(unknown) = %v, want nil", err)
	}

	pendingUser, err := domain.NewPendingUser("Ali", "", shared.Email(resetEmail), "", "hash", shared.RoleShipper)
	if err != nil {
		t.Fatalf("NewPendingUser: %v", err)
	}
	pending := newResetFixture(t, pendingUser)
	if err := pending.forgot.ForgotPassword(ctx, &command.ForgotPasswordRequest{Email: resetEmail}); err != nil {
		t.Fatalf("ForgotPassword(pending) = %v, want nil", err)
	}

	if len(unknown.mailer.bodies)+len(pending.mailer.bodies) != 0 {
		t.Error("an email went to an unknown or pending address")
	}
}

func TestForgotPasswordLetsAppleUserSetPassword(t *testing.T) {
	ctx := context.Background()
	appleUser, err := domain.NewUserFromApple("Ali", "", shared.Email(resetEmail), shared.RoleShipper)
	if err != nil {
		t.Fatalf("NewUserFromApple: %v", err)
	}
	f := newResetFixture(t, appleUser)

	if err := f.forgot.ForgotPassword(ctx, &command.ForgotPasswordRequest{Email: resetEmail}); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}
	err = f.reset.ResetPassword(ctx, &command.ResetPasswordRequest{Email: resetEmail, Code: f.mailedCode(t), NewPassword: "first-password"})
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if shared.Password("first-password").Verify(f.users.updated.PasswordHash) != nil {
		t.Error("Apple user did not get the password")
	}
}

func TestForgotPasswordTwiceWithinCooldownSendsOneCode(t *testing.T) {
	ctx := context.Background()
	f := newResetFixture(t, activeUser(t))

	for i := 0; i < 2; i++ {
		if err := f.forgot.ForgotPassword(ctx, &command.ForgotPasswordRequest{Email: resetEmail}); err != nil {
			t.Fatalf("ForgotPassword #%d: %v", i+1, err)
		}
	}

	if len(f.mailer.bodies) != 1 {
		t.Fatalf("sent %d emails, want 1", len(f.mailer.bodies))
	}
}
