package email

import (
	"bytes"
	"context"
	"html/template"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

const emailTypeVerification = "verification"

type Mailer interface {
	SendHTML(ctx context.Context, to, subject, body string) error
}

type Service struct {
	mailer    Mailer
	emailRepo domain.EmailRepository
	tmpl      *template.Template
	from      string
	otpTTL    time.Duration
}

func NewService(mailer Mailer, emailRepo domain.EmailRepository, tmpl *template.Template, from string, otpTTL time.Duration) *Service {
	return &Service{
		mailer:    mailer,
		emailRepo: emailRepo,
		tmpl:      tmpl,
		from:      from,
		otpTTL:    otpTTL,
	}
}

// SendVerificationCode renders the OTP email template, logs it to the DB, and delivers it via SMTP.
func (s *Service) SendVerificationCode(ctx context.Context, userID uuid.UUID, to, code string) error {
	body, err := s.renderVerificationEmail(code)
	if err != nil {
		return err
	}

	record, err := domain.NewEmail(userID, emailTypeVerification, s.from, to, "Verify your email", body)
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

func (s *Service) renderVerificationEmail(code string) (string, error) {
	data := struct {
		Code             string
		ExpiresInMinutes int
	}{
		Code:             code,
		ExpiresInMinutes: int(s.otpTTL.Minutes()),
	}
	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
