package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type LoadTrackingLinkStatus string

const (
	LoadTrackingLinkStatusActive  LoadTrackingLinkStatus = "active"
	LoadTrackingLinkStatusRevoked LoadTrackingLinkStatus = "revoked"
)

func (s LoadTrackingLinkStatus) String() string {
	return string(s)
}

// LoadTrackingLink is a public, no-login, read-only link a broker can
// generate and hand to their client to follow a single load's progress. It
// has no expiry — it remains valid for the life of the load unless revoked.
type LoadTrackingLink struct {
	ID        uuid.UUID
	LoadID    uuid.UUID
	Token     string
	Status    LoadTrackingLinkStatus
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewLoadTrackingLink(loadID uuid.UUID, createdBy uuid.UUID, token string) (*LoadTrackingLink, error) {
	if loadID == uuid.Nil {
		return nil, errors.New("load ID is required")
	}
	if token == "" {
		return nil, errors.New("token is required")
	}

	now := time.Now()
	return &LoadTrackingLink{
		ID:        uuid.New(),
		LoadID:    loadID,
		Token:     token,
		Status:    LoadTrackingLinkStatusActive,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (l *LoadTrackingLink) IsActive() bool {
	return l.Status == LoadTrackingLinkStatusActive
}

type LoadTrackingLinkRepository interface {
	Save(ctx context.Context, link *LoadTrackingLink) error
	FindByToken(ctx context.Context, token string) (*LoadTrackingLink, error)
	// FindActiveByLoadID returns the current active tracking link for a
	// load, if one exists (used to keep link creation idempotent).
	FindActiveByLoadID(ctx context.Context, loadID uuid.UUID) (*LoadTrackingLink, error)
}
