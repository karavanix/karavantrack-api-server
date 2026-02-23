package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Driver struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewDriver(userID uuid.UUID) (*Driver, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}

	return &Driver{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

type DriverRepository interface {
	Save(ctx context.Context, driver *Driver) error
	FindByID(ctx context.Context, id uuid.UUID) (*Driver, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Driver, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
