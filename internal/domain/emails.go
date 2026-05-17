package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type EmailStatus string

const (
	EmailStatusPending EmailStatus = "pending"
	EmailStatusSent    EmailStatus = "sent"
	EmailStatusFailed  EmailStatus = "failed"
	EmailStatusBounced EmailStatus = "bounced"
)

func (e EmailStatus) String() string {
	return string(e)
}

func (e EmailStatus) IsValid() bool {
	return e == EmailStatusPending || e == EmailStatusSent || e == EmailStatusFailed || e == EmailStatusBounced
}

type Email struct {
	ID        int64
	UserID    uuid.UUID
	Type      string
	Status    EmailStatus
	From      string
	To        string
	Bcc       []string
	Subject   string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewEmail(userID uuid.UUID, emailType string, from string, to string, subject string, content string) (*Email, error) {
	if from == "" {
		return nil, errors.New("from cannot be empty")
	}
	if to == "" {
		return nil, errors.New("to cannot be empty")
	}
	if subject == "" {
		return nil, errors.New("subject cannot be empty")
	}
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}

	return &Email{
		UserID:    userID,
		Type:      emailType,
		Status:    EmailStatusPending,
		From:      from,
		To:        to,
		Subject:   subject,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (e *Email) SetBcc(bcc []string) error {
	if len(bcc) == 0 {
		return errors.New("bcc cannot be empty")
	}
	e.Bcc = bcc
	return nil
}

func (e *Email) Sent() {
	e.Status = EmailStatusSent
	e.UpdatedAt = time.Now()
}

func (e *Email) Failed() {
	e.Status = EmailStatusFailed
	e.UpdatedAt = time.Now()
}

func (e *Email) Bounced() {
	e.Status = EmailStatusBounced
	e.UpdatedAt = time.Now()
}

type EmailRepository interface {
	Save(ctx context.Context, e *Email) error
}
