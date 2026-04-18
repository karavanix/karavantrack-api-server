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
	LoadStatusCreated     LoadStatus = "created"
	LoadStatusAssigned    LoadStatus = "assigned"
	LoadStatusAccepted    LoadStatus = "accepted"
	LoadStatusPickingUp   LoadStatus = "picking_up"
	LoadStatusPickedUp    LoadStatus = "picked_up"
	LoadStatusInTransit   LoadStatus = "in_transit"
	LoadStatusDroppingOff LoadStatus = "dropping_off"
	LoadStatusDroppedOff  LoadStatus = "dropped_off"
	LoadStatusConfirmed   LoadStatus = "confirmed"
	LoadStatusCancelled   LoadStatus = "cancelled"
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

	History []*LoadStatusHistory
}

type LoadStatusHistory struct {
	ID         int64
	LoadID     uuid.UUID
	UserID     uuid.UUID
	FromStatus LoadStatus
	ToStatus   LoadStatus
	Note       string
	CreatedAt  time.Time

	Attachments []*LoadStatusHistoryAttachment
}

type LoadStatusHistoryAttachment struct {
	ID           int64
	HistoryID    int64
	AttachmentID uuid.UUID
	CreatedAt    time.Time
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
		History: []*LoadStatusHistory{
			{
				FromStatus: LoadStatusCreated,
				ToStatus:   LoadStatusCreated,
				CreatedAt:  time.Now(),
			},
		},
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
func (l *Load) Assign(note string, carrierID uuid.UUID, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusCreated {
		return errors.New("can only assign carrier to a created load")
	}
	if carrierID == uuid.Nil {
		return errors.New("carrier ID is required")
	}
	l.CarrierID = carrierID
	l.Status = LoadStatusAssigned
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusCreated, LoadStatusAssigned, note, attachmentIDs...))
	return nil
}

// Accept marks the load as accepted by the carrier.
func (l *Load) Accept(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusAssigned {
		return errors.New("can only accept an assigned load")
	}
	l.Status = LoadStatusAccepted
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusAssigned, LoadStatusAccepted, note, attachmentIDs...))
	return nil
}

// BeginPickup transitions accepted → picking_up (carrier is driving to pickup location).
func (l *Load) BeginPickup(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusAccepted {
		return errors.New("can only begin pickup on an accepted load")
	}
	l.Status = LoadStatusPickingUp
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusAccepted, LoadStatusPickingUp, note, attachmentIDs...))
	return nil
}

// ConfirmPickup transitions picking_up → picked_up (cargo loaded onto truck).
func (l *Load) ConfirmPickup(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusPickingUp {
		return errors.New("can only confirm pickup on a picking_up load")
	}
	l.Status = LoadStatusPickedUp
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusPickingUp, LoadStatusPickedUp, note, attachmentIDs...))
	return nil
}

// StartTrip transitions picked_up → in_transit (truck en route to destination).
// Also accepts legacy accepted status during client transition window.
func (l *Load) StartTrip(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusPickedUp && l.Status != LoadStatusAccepted {
		return errors.New("can only start trip on a picked_up load")
	}
	l.Status = LoadStatusInTransit
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusPickedUp, LoadStatusInTransit, note, attachmentIDs...))
	return nil
}

// BeginDropoff transitions in_transit → dropping_off (carrier arrived at destination).
func (l *Load) BeginDropoff(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusInTransit {
		return errors.New("can only begin dropoff on an in_transit load")
	}
	l.Status = LoadStatusDroppingOff
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusInTransit, LoadStatusDroppingOff, note, attachmentIDs...))
	return nil
}

// ConfirmDropoff transitions dropping_off → dropped_off (cargo unloaded).
func (l *Load) ConfirmDropoff(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusDroppingOff {
		return errors.New("can only confirm dropoff on a dropping_off load")
	}
	l.Status = LoadStatusDroppedOff
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusDroppingOff, LoadStatusDroppedOff, note, attachmentIDs...))
	return nil
}

// ConfirmByOwner confirms the load completion by the cargo owner.
// Accepts dropped_off (new) or completed (legacy) status.
func (l *Load) ConfirmByOwner(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status != LoadStatusDroppedOff {
		return errors.New("can only confirm a dropped_off load")
	}
	l.Status = LoadStatusConfirmed
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusDroppedOff, LoadStatusConfirmed, note, attachmentIDs...))
	return nil
}

// Cancel cancels the load.
func (l *Load) Cancel(note string, attachmentIDs ...uuid.UUID) error {
	if l.Status == LoadStatusConfirmed || l.Status == LoadStatusCancelled {
		return errors.New("cannot cancel a confirmed or already cancelled load")
	}
	l.Status = LoadStatusCancelled
	l.UpdatedAt = time.Now()
	l.History = append(l.History, l.newHistory(LoadStatusDroppedOff, LoadStatusCancelled, note, attachmentIDs...))
	return nil
}

func (l *Load) newHistory(from, to LoadStatus, note string, attachmentIDs ...uuid.UUID) *LoadStatusHistory {
	now := time.Now()

	h := &LoadStatusHistory{
		LoadID:     l.ID,
		FromStatus: from,
		ToStatus:   to,
		Note:       note,
		CreatedAt:  now,
	}

	if len(attachmentIDs) > 0 {
		for _, attachmentID := range attachmentIDs {
			h.Attachments = append(h.Attachments, &LoadStatusHistoryAttachment{
				HistoryID:    h.ID,
				AttachmentID: attachmentID,
				CreatedAt:    now,
			})
		}
	}

	return h
}

type LoadFilter struct {
	CompanyID *uuid.UUID
	CarrierID *uuid.UUID
	Status    []LoadStatus
	Limit     int
	Offset    int
}

type LoadStats struct {
	Created     int
	Assigned    int
	Accepted    int
	PickingUp   int
	PickedUp    int
	InTransit   int
	DroppingOff int
	DroppedOff  int
	Confirmed   int
	Canceled    int
	Total       int
}

type LoadRepository interface {
	Save(ctx context.Context, load *Load) error
	FindByID(ctx context.Context, id uuid.UUID) (*Load, error)
	FindActiveByCarrierID(ctx context.Context, carrierID uuid.UUID) (*Load, error)
	FindActiveByCarrierIDs(ctx context.Context, carrierIDs []uuid.UUID) (map[uuid.UUID]*Load, error)
	FindAll(ctx context.Context, filter LoadFilter) ([]*Load, int, error)
	FindStats(ctx context.Context, filter LoadFilter) (*LoadStats, error)
}
