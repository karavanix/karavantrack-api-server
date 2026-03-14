package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type CompanyCarrier struct {
	CompanyID uuid.UUID
	CarrierID uuid.UUID
	Alias     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCompanyCarrier(companyID, carrierID uuid.UUID, alias string) (*CompanyCarrier, error) {
	if companyID == uuid.Nil {
		return nil, errors.New("company ID cannot be nil")
	}
	if carrierID == uuid.Nil {
		return nil, errors.New("carrier ID cannot be nil")
	}
	if alias == "" {
		return nil, errors.New("alias is required")
	}

	return &CompanyCarrier{
		CompanyID: companyID,
		CarrierID: carrierID,
		Alias:     alias,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

type CompanyCarrierFilter struct {
	Query  string
	Limit  int
	Offset int
}

type CompanyCarrierRepository interface {
	Save(ctx context.Context, cs *CompanyCarrier) error
	FindByCompanyID(ctx context.Context, companyID uuid.UUID) ([]*CompanyCarrier, error)
	FindByCompanyIDWithFilter(ctx context.Context, companyID uuid.UUID, filter *CompanyCarrierFilter) ([]*CompanyCarrier, error)
	FindByCarrierID(ctx context.Context, carrierID uuid.UUID) ([]*CompanyCarrier, error)
	FindByCompanyIDAndCarrierID(ctx context.Context, companyID, carrierID uuid.UUID) (*CompanyCarrier, error)
	DeleteByCompanyIDAndCarrierID(ctx context.Context, companyID, carrierID uuid.UUID) error
}
