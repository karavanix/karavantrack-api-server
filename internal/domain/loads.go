package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shogo82148/pointer"
)

type LoadStatus string

const (
	LoadStatusCreated   LoadStatus = "created"
	LoadStatusAssigned  LoadStatus = "assigned"
	LoadStatusAccepted  LoadStatus = "accepted"
	LoadStatusInTransit LoadStatus = "in_transit"
	LoadStatusCompleted LoadStatus = "completed"
	LoadStatusConfirmed LoadStatus = "confirmed"
	LoadStatusCancelled LoadStatus = "cancelled"
)

func (s LoadStatus) String() string {
	return string(s)
}

type Load struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	MemberID         uuid.UUID
	CarrierID        uuid.UUID
	ReferenceID      string
	Title            string
	Description      string
	Status           LoadStatus
	PickupAddress    string
	PickupLat        float64
	PickupLng        float64
	PickupAddressID  string
	PickupAt         *time.Time
	DropoffAddress   string
	DropoffLat       float64
	DropoffLng       float64
	DropoffAddressID string
	DropoffAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewLoad(
	companyID uuid.UUID,
	memberID uuid.UUID,
	title string,
	description string,
	pickupAddress string,
	pickupLat, pickupLng float64,
	dropoffAddress string,
	dropoffLat, dropoffLng float64,
) (*Load, error) {
	if companyID == uuid.Nil {
		return nil, errors.New("company ID is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}

	return &Load{
		ID:             uuid.New(),
		CompanyID:      companyID,
		MemberID:       memberID,
		Status:         LoadStatusCreated,
		Title:          title,
		Description:    description,
		PickupAddress:  pickupAddress,
		PickupLat:      pickupLat,
		PickupLng:      pickupLng,
		DropoffAddress: dropoffAddress,
		DropoffLat:     dropoffLat,
		DropoffLng:     dropoffLng,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (l *Load) SetReferenceID(referenceID string) {
	l.ReferenceID = referenceID
}

func (l *Load) SetDeadlines(pickupAt, dropoffAt time.Time) {
	l.PickupAt = pointer.TimeOrNil(pickupAt)
	l.DropoffAt = pointer.TimeOrNil(dropoffAt)
}

// Assign assigns a carrier to the load.
func (l *Load) Assign(carrierID uuid.UUID) error {
	if l.Status != LoadStatusCreated {
		return errors.New("can only assign carrier to a created load")
	}
	if carrierID == uuid.Nil {
		return errors.New("carrier ID is required")
	}
	l.CarrierID = carrierID
	l.Status = LoadStatusAssigned
	l.UpdatedAt = time.Now()
	return nil
}

// Accept marks the load as accepted by the carrier.
func (l *Load) Accept() error {
	if l.Status != LoadStatusAssigned {
		return errors.New("can only accept an assigned load")
	}
	l.Status = LoadStatusAccepted
	l.UpdatedAt = time.Now()
	return nil
}

// StartTrip transitions to in-transit, activating GPS tracking.
func (l *Load) StartTrip() error {
	if l.Status != LoadStatusAccepted {
		return errors.New("can only start trip on an accepted load")
	}
	l.Status = LoadStatusInTransit
	l.UpdatedAt = time.Now()
	return nil
}

// CompleteByCarrier marks the load as completed by the carrier.
func (l *Load) CompleteByCarrier() error {
	if l.Status != LoadStatusInTransit {
		return errors.New("can only complete an in-transit load")
	}
	l.Status = LoadStatusCompleted
	l.UpdatedAt = time.Now()
	return nil
}

// ConfirmByOwner confirms the load completion by the cargo owner.
func (l *Load) ConfirmByOwner() error {
	if l.Status != LoadStatusCompleted {
		return errors.New("can only confirm a carrier completed load")
	}
	l.Status = LoadStatusConfirmed
	l.UpdatedAt = time.Now()
	return nil
}

// Cancel cancels the load.
func (l *Load) Cancel() error {
	if l.Status == LoadStatusConfirmed || l.Status == LoadStatusCancelled {
		return errors.New("cannot cancel a confirmed or already cancelled load")
	}
	l.Status = LoadStatusCancelled
	l.UpdatedAt = time.Now()
	return nil
}

type LoadFilter struct {
	CompanyID *uuid.UUID
	CarrierID *uuid.UUID
	Status    []LoadStatus
	Limit     int
	Offset    int
}

type LoadStats struct {
	Created   int
	Assigned  int
	Accepted  int
	InTransit int
	Completed int
	Confirmed int
	Canceled  int
	Total     int
}

type LoadRepository interface {
	Save(ctx context.Context, load *Load) error
	FindByID(ctx context.Context, id uuid.UUID) (*Load, error)
	FindActiveByCarrierID(ctx context.Context, carrierID uuid.UUID) (*Load, error)
	FindActiveByCarrierIDs(ctx context.Context, carrierIDs []uuid.UUID) (map[uuid.UUID]*Load, error)
	FindAll(ctx context.Context, filter LoadFilter) ([]*Load, int, error)
	FindStats(ctx context.Context, filter LoadFilter) (*LoadStats, error)
}
