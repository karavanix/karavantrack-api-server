package email_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/service/email"
)

const templatesDir = "../../../templates"

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
	// Loaded the way internal/app/server.go loads them.
	tmpl, err := email.LoadTemplates(templatesDir)
	if err != nil {
		t.Fatalf("LoadTemplates: %v", err)
	}
	return email.NewService(mail, nopEmails{}, tmpl, "no-reply@example.com", 15*time.Minute)
}

// Both emails come out in the language asked for: its own subject, heading,
// the code, the lifetime and the matching html lang.
func TestBothEmailsRenderInEveryLanguage(t *testing.T) {
	cases := []struct {
		lang                         shared.Language
		verifySubject, verifyHeading string
		resetSubject, resetHeading   string
		minutes                      string
	}{
		{shared.LanguageRussian, "Подтверждение почты — Yool", "Подтвердите ваш email", "Сброс пароля — Yool", "Код для сброса пароля", "15 минут"},
		{shared.LanguageUzbek, "Emailni tasdiqlash — Yool", "Emailingizni tasdiqlang", "Parolni tiklash — Yool", "Parolni tiklash kodi", "15 daqiqa"},
		{shared.LanguageEnglish, "Confirm your email — Yool", "Confirm your email", "Reset your password — Yool", "Password reset code", "15 minutes"},
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
			for _, want := range []string{"123456", tc.verifyHeading, tc.minutes, `<html lang="` + tc.lang.String() + `">`, "<title>" + tc.verifySubject + "</title>"} {
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
			for _, want := range []string{"654321", tc.resetHeading, tc.minutes, `<html lang="` + tc.lang.String() + `">`, "<title>" + tc.resetSubject + "</title>"} {
				if !strings.Contains(mail.body, want) {
					t.Errorf("reset email lacks %q", want)
				}
			}
		})
	}
}

// A language the server does not know gets the Russian email, not an error.
func TestUnknownLanguageFallsBackToRussian(t *testing.T) {
	mail := &lastMail{}
	svc := newService(t, mail)
	if err := svc.SendVerificationCode(context.Background(), uuid.New(), "a@example.com", "123456", shared.Language("de")); err != nil {
		t.Fatalf("SendVerificationCode: %v", err)
	}
	if mail.subject != "Подтверждение почты — Yool" {
		t.Errorf("subject = %q, want the Russian one", mail.subject)
	}
}

// A translation is a copy of the Russian file: it must use exactly the same
// placeholders, so a copy that lost {{.Code}} fails here, not in a mailbox.
func TestEveryLanguageUsesTheSamePlaceholders(t *testing.T) {
	action := regexp.MustCompile(`\{\{[^}]*\}\}`)
	placeholders := func(path string) []string {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		found := action.FindAllString(string(raw), -1)
		slices.Sort(found)
		return slices.Compact(found)
	}
	for _, name := range []string{email.TemplateVerification, email.TemplatePasswordReset} {
		want := placeholders(filepath.Join(templatesDir, shared.DefaultLanguage.String(), name))
		for _, lang := range shared.Languages() {
			got := placeholders(filepath.Join(templatesDir, lang.String(), name))
			if !slices.Equal(got, want) {
				t.Errorf("%s/%s placeholders = %v, want the Russian file's %v", lang, name, got, want)
			}
		}
	}
}

// A language without its email files stops the server at startup.
func TestLoadTemplatesFailsOnAMissingLanguage(t *testing.T) {
	dir := t.TempDir()
	for _, lang := range []string{"ru", "en"} { // no uz
		if err := os.MkdirAll(filepath.Join(dir, lang), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{email.TemplateVerification, email.TemplatePasswordReset} {
			src, err := os.ReadFile(filepath.Join(templatesDir, lang, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, lang, name), src, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := email.LoadTemplates(dir); err == nil || !strings.Contains(err.Error(), "uz") {
		t.Fatalf("LoadTemplates without uz = %v, want an error naming uz", err)
	}
}

// A file without a subject block is refused at startup too.
func TestLoadTemplatesFailsWithoutASubject(t *testing.T) {
	dir := t.TempDir()
	for _, lang := range shared.Languages() {
		if err := os.MkdirAll(filepath.Join(dir, lang.String()), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{email.TemplateVerification, email.TemplatePasswordReset} {
			if err := os.WriteFile(filepath.Join(dir, lang.String(), name), []byte("<p>{{.Code}}</p>"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := email.LoadTemplates(dir); err == nil || !strings.Contains(err.Error(), "subject") {
		t.Fatalf("LoadTemplates without a subject = %v, want an error about it", err)
	}
}
