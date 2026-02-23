package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type CompanyDriver struct {
	CompanyID uuid.UUID
	DriverID  uuid.UUID
	Alias     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCompanyDriver(companyID, driverID uuid.UUID, alias string) (*CompanyDriver, error) {
	if companyID == uuid.Nil {
		return nil, errors.New("company ID cannot be nil")
	}
	if driverID == uuid.Nil {
		return nil, errors.New("driver ID cannot be nil")
	}
	if alias == "" {
		return nil, errors.New("alias is required")
	}

	return &CompanyDriver{
		CompanyID: companyID,
		DriverID:  driverID,
		Alias:     alias,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

type CompanyDriverRepository interface {
	Save(ctx context.Context, cd *CompanyDriver) error
	FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*CompanyDriver, error)
	FindByDriverID(ctx context.Context, driverID uuid.UUID) ([]*CompanyDriver, error)
	Delete(ctx context.Context, companyID, driverID uuid.UUID) error
}
