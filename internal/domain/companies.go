package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type CompanyStatus string

const (
	CompanyStatusActive   CompanyStatus = "active"
	CompanyStatusInactive CompanyStatus = "inactive"
)

func (s CompanyStatus) String() string {
	return string(s)
}

type Company struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Name      string
	Status    CompanyStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCompany(ownerID uuid.UUID, name string) (*Company, error) {
	if ownerID == uuid.Nil {
		return nil, errors.New("owner ID cannot be nil")
	}
	if name == "" {
		return nil, errors.New("company name is required")
	}

	return &Company{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      name,
		Status:    CompanyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (c *Company) Update(name string) {
	if name != "" {
		c.Name = name
	}
	c.UpdatedAt = time.Now()
}

func (c *Company) Deactivate() {
	c.Status = CompanyStatusInactive
	c.UpdatedAt = time.Now()
}

type CompanyRepository interface {
	Save(ctx context.Context, company *Company) error
	FindByID(ctx context.Context, id uuid.UUID) (*Company, error)
	FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*Company, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
}
