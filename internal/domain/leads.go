package domain

import (
	"context"
	"errors"
	"time"
)

// Lead is a sales contact request submitted from the public marketing
// landing page (karavantrack-landing), not tied to any authenticated user.
type Lead struct {
	ID        int64
	Name      string
	Company   string
	Phone     string
	Fleet     string
	UserAgent string
	CreatedAt time.Time
}

func NewLead(name, company, phone, fleet, userAgent string) (*Lead, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if company == "" {
		return nil, errors.New("company cannot be empty")
	}
	if phone == "" {
		return nil, errors.New("phone cannot be empty")
	}
	if fleet == "" {
		return nil, errors.New("fleet cannot be empty")
	}

	return &Lead{
		Name:      name,
		Company:   company,
		Phone:     phone,
		Fleet:     fleet,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}, nil
}

type LeadRepository interface {
	Save(ctx context.Context, l *Lead) error
}
