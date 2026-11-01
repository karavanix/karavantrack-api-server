package email_test

import (
	"context"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
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

// The templates are parsed the way internal/app/server.go parses them.
func TestBothTemplatesRenderTheCode(t *testing.T) {
	tmpl, err := template.ParseFiles(
		"../../../templates/"+email.TemplateVerification,
		"../../../templates/"+email.TemplatePasswordReset,
	)
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	mail := &lastMail{}
	svc := email.NewService(mail, nopEmails{}, tmpl, "no-reply@example.com", 15*time.Minute)

	if err := svc.SendVerificationCode(context.Background(), uuid.New(), "a@example.com", "123456"); err != nil {
		t.Fatalf("SendVerificationCode: %v", err)
	}
	if !strings.Contains(mail.body, "123456") || !strings.Contains(mail.body, "Подтвердите") {
		t.Errorf("verification email lacks its code or heading")
	}

	if err := svc.SendPasswordResetCode(context.Background(), uuid.New(), "a@example.com", "654321"); err != nil {
		t.Fatalf("SendPasswordResetCode: %v", err)
	}
	if !strings.Contains(mail.body, "654321") || !strings.Contains(mail.body, "сброса пароля") {
		t.Errorf("reset email lacks its code or heading")
	}
	if !strings.Contains(mail.subject, "Сброс пароля") {
		t.Errorf("reset subject = %q", mail.subject)
	}
}
