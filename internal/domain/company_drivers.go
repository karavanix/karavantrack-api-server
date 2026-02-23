package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// CompanyDriver links a driver-role user to a company.
// DriverID refers to users.id where the user has role = "driver".
type CompanyDriver struct {
	CompanyID uuid.UUID
	DriverID  uuid.UUID // references users(id) where role='driver'
	Alias     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCompanyDriver(companyID, driverUserID uuid.UUID, alias string) (*CompanyDriver, error) {
	if companyID == uuid.Nil {
		return nil, errors.New("company ID cannot be nil")
	}
	if driverUserID == uuid.Nil {
		return nil, errors.New("driver user ID cannot be nil")
	}
	if alias == "" {
		return nil, errors.New("alias is required")
	}

	return &CompanyDriver{
		CompanyID: companyID,
		DriverID:  driverUserID,
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
