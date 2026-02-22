package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Presence struct {
	UserID     uuid.UUID
	LastSeenAt time.Time
}

func NewPresence(userID uuid.UUID) (*Presence, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}

	return &Presence{
		UserID:     userID,
		LastSeenAt: time.Now(),
	}, nil
}

type PresenceRepository interface {
	Save(ctx context.Context, e *Presence) error
	Delete(ctx context.Context, userID uuid.UUID) error
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
}
