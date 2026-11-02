package email

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

const (
	emailTypeVerification  = "verification"
	emailTypePasswordReset = "password_reset"
)

type Mailer interface {
	SendHTML(ctx context.Context, to, subject, body string) error
}

type Service struct {
	mailer    Mailer
	emailRepo domain.EmailRepository
	tmpl      *Templates
	from      string
	otpTTL    time.Duration
}

// NewService takes the templates LoadTemplates parsed.
func NewService(mailer Mailer, emailRepo domain.EmailRepository, tmpl *Templates, from string, otpTTL time.Duration) *Service {
	return &Service{
		mailer:    mailer,
		emailRepo: emailRepo,
		tmpl:      tmpl,
		from:      from,
		otpTTL:    otpTTL,
	}
}

// SendVerificationCode renders the OTP email in the user's language, logs it to the DB, and delivers it via SMTP.
func (s *Service) SendVerificationCode(ctx context.Context, userID uuid.UUID, to, code string, lang shared.Language) error {
	subject, body, err := s.renderCode(TemplateVerification, lang, code)
	if err != nil {
		return err
	}
	return s.send(ctx, userID, emailTypeVerification, to, subject, body)
}

// SendPasswordResetCode mails the code that lets the user set a new password.
func (s *Service) SendPasswordResetCode(ctx context.Context, userID uuid.UUID, to, code string, lang shared.Language) error {
	subject, body, err := s.renderCode(TemplatePasswordReset, lang, code)
	if err != nil {
		return err
	}
	return s.send(ctx, userID, emailTypePasswordReset, to, subject, body)
}

// send logs the email to the DB, delivers it via SMTP and records the outcome.
func (s *Service) send(ctx context.Context, userID uuid.UUID, emailType, to, subject, body string) error {
	record, err := domain.NewEmail(
		userID,
		emailType,
		s.from,
		to,
		subject,
		body,
	)
	if err != nil {
		return err
	}

	if saveErr := s.emailRepo.Save(ctx, record); saveErr != nil {
		logger.ErrorContext(ctx, "failed to log pending email", saveErr)
	}

	if sendErr := s.mailer.SendHTML(ctx, to, record.Subject, body); sendErr != nil {
		record.Failed()
		if saveErr := s.emailRepo.Save(ctx, record); saveErr != nil {
			logger.ErrorContext(ctx, "failed to update email status to failed", saveErr)
		}
		return sendErr
	}

	record.Sent()
	if saveErr := s.emailRepo.Save(ctx, record); saveErr != nil {
		logger.ErrorContext(ctx, "failed to update email status to sent", saveErr)
	}

	return nil
}

// renderCode fills the template name in lang with the code and its
// lifetime; it returns the subject and the HTML body.
func (s *Service) renderCode(name string, lang shared.Language, code string) (string, string, error) {
	data := struct {
		Code             string
		ExpiresInMinutes int
	}{
		Code:             code,
		ExpiresInMinutes: int(s.otpTTL.Minutes()),
	}
	return s.tmpl.render(name, lang, data)
}
