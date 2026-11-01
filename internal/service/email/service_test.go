package email_test

import (
	"context"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/service/email"
)

type lastMail struct {
	subject, body string
}

func (m *lastMail) SendHTML(ctx context.Context, to, subject, body string) error {
	m.subject, m.body = subject, body
	return nil
}

type nopEmails struct{}

func (nopEmails) Save(ctx context.Context, e *domain.Email) error { return nil }

func newService(t *testing.T, mail *lastMail) *email.Service {
	t.Helper()
	// The templates are parsed the way internal/app/server.go parses them.
	tmpl, err := template.ParseFiles(
		"../../../templates/"+email.TemplateVerification,
		"../../../templates/"+email.TemplatePasswordReset,
	)
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	return email.NewService(mail, nopEmails{}, tmpl, "no-reply@example.com", 15*time.Minute)
}

// Both emails come out in the language asked for, with the code, the
// lifetime in bold and the matching html lang.
func TestBothTemplatesRenderInEveryLanguage(t *testing.T) {
	cases := []struct {
		lang                                  shared.Language
		verifySubject, verifyHeading          string
		resetSubject, resetHeading, boldLimit string
	}{
		{shared.LanguageRussian, "Подтверждение почты — Yool", "Подтвердите ваш email", "Сброс пароля — Yool", "Код для сброса пароля", "<strong style=\"color:#0d1117;font-weight:500;\">15 минут</strong>."},
		{shared.LanguageUzbek, "Emailni tasdiqlash — Yool", "Emailingizni tasdiqlang", "Parolni tiklash — Yool", "Parolni tiklash kodi", "<strong style=\"color:#0d1117;font-weight:500;\">15 daqiqa</strong> amal qiladi."},
		{shared.LanguageEnglish, "Confirm your email — Yool", "Confirm your email", "Reset your password — Yool", "Password reset code", "<strong style=\"color:#0d1117;font-weight:500;\">15 minutes</strong>."},
	}
	for _, tc := range cases {
		t.Run(tc.lang.String(), func(t *testing.T) {
			mail := &lastMail{}
			svc := newService(t, mail)

			if err := svc.SendVerificationCode(context.Background(), uuid.New(), "a@example.com", "123456", tc.lang); err != nil {
				t.Fatalf("SendVerificationCode: %v", err)
			}
			if mail.subject != tc.verifySubject {
				t.Errorf("verification subject = %q, want %q", mail.subject, tc.verifySubject)
			}
			for _, want := range []string{"123456", tc.verifyHeading, tc.boldLimit, `<html lang="` + tc.lang.String() + `">`} {
				if !strings.Contains(mail.body, want) {
					t.Errorf("verification email lacks %q", want)
				}
			}

			if err := svc.SendPasswordResetCode(context.Background(), uuid.New(), "a@example.com", "654321", tc.lang); err != nil {
				t.Fatalf("SendPasswordResetCode: %v", err)
			}
			if mail.subject != tc.resetSubject {
				t.Errorf("reset subject = %q, want %q", mail.subject, tc.resetSubject)
			}
			for _, want := range []string{"654321", tc.resetHeading, tc.boldLimit, `<html lang="` + tc.lang.String() + `">`} {
				if !strings.Contains(mail.body, want) {
					t.Errorf("reset email lacks %q", want)
				}
			}
			if strings.Contains(mail.body, "{{") {
				t.Errorf("reset email left template syntax")
			}
		})
	}
}
