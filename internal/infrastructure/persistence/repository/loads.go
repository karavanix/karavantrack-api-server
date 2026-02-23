package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/shogo82148/pointer"
	"github.com/uptrace/bun"
)

type Loads struct {
	bun.BaseModel `bun:"table:loads,alias:l"`

	ID               string     `bun:"id,type:uuid,pk"`
	CompanyID        *string    `bun:"company_id,nullzero"`
	MemberID         *string    `bun:"member_id,nullzero"`
	CarrierID        *string    `bun:"carrier_id,nullzero"`
	ReferenceID      *string    `bun:"reference_id,nullzero"`
	Title            *string    `bun:"title,nullzero"`
	Description      *string    `bun:"description,nullzero"`
	Status           string     `bun:"status"`
	PickupAddress    *string    `bun:"pickup_address,nullzero"`
	PickupLat        float64    `bun:"pickup_lat"`
	PickupLng        float64    `bun:"pickup_lng"`
	PickupAddressID  *string    `bun:"pickup_address_id,nullzero"`
	PickupAt         *time.Time `bun:"pickup_at"`
	DropoffAddress   *string    `bun:"dropoff_address,nullzero"`
	DropoffLat       float64    `bun:"dropoff_lat"`
	DropoffLng       float64    `bun:"dropoff_lng"`
	DropoffAddressID *string    `bun:"dropoff_address_id,nullzero"`
	DropoffAt        *time.Time `bun:"dropoff_at"`
	CreatedAt        time.Time  `bun:"created_at"`
	UpdatedAt        time.Time  `bun:"updated_at"`
}

type loadsRepo struct {
	db bun.IDB
}

func NewLoadsRepo(db bun.IDB) domain.LoadRepository {
	return &loadsRepo{db: db}
}

func (r *loadsRepo) Save(ctx context.Context, load *domain.Load) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(load)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE SET").
		Set("carrier_id = EXCLUDED.carrier_id").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *loadsRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Loads
	err := db.NewSelect().Model(&model).Where("id = ?", id.String()).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}
	return r.toDomain(&model), nil
}

func (r *loadsRepo) FindAll(ctx context.Context, filter domain.LoadFilter) ([]*domain.Load, int, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []Loads
	q := db.NewSelect().Model(&models)

	if filter.CompanyID != nil {
		q = q.Where("company_id = ?", filter.CompanyID.String())
	}
	if filter.CarrierID != nil {
		q = q.Where("carrier_id = ?", filter.CarrierID.String())
	}
	if filter.Status != nil {
		q = q.Where("status = ?", filter.Status.String())
	}

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(50)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}

	q = q.Order("created_at DESC")

	err := q.Scan(ctx)
	if err != nil {
		return nil, 0, postgres.Error(err, &Loads{})
	}

	result := make([]*domain.Load, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, postgres.Error(err, &Loads{})
	}

	return result, total, nil
}

func (r *loadsRepo) toModel(e *domain.Load) *Loads {
	if e == nil {
		return nil
	}

	m := &Loads{
		ID:               e.ID.String(),
		Status:           e.Status.String(),
		PickupAddress:    pointer.StringOrNil(e.PickupAddress),
		PickupLat:        e.PickupLat,
		PickupLng:        e.PickupLng,
		DropoffAddress:   pointer.StringOrNil(e.DropoffAddress),
		DropoffLat:       e.DropoffLat,
		DropoffLng:       e.DropoffLng,
		Title:            pointer.StringOrNil(e.Title),
		Description:      pointer.StringOrNil(e.Description),
		ReferenceID:      pointer.StringOrNil(e.ReferenceID),
		PickupAddressID:  pointer.StringOrNil(e.PickupAddressID),
		DropoffAddressID: pointer.StringOrNil(e.DropoffAddressID),
		PickupAt:         e.PickupAt,
		DropoffAt:        e.DropoffAt,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}

	if e.CompanyID != uuid.Nil {
		s := e.CompanyID.String()
		m.CompanyID = &s
	}
	if e.MemberID != uuid.Nil {
		s := e.MemberID.String()
		m.MemberID = &s
	}
	if e.CarrierID != uuid.Nil {
		s := e.CarrierID.String()
		m.CarrierID = &s
	}

	return m
}

func (r *loadsRepo) toDomain(m *Loads) *domain.Load {
	if m == nil {
		return nil
	}

	id, _ := uuid.Parse(m.ID)

	e := &domain.Load{
		ID:               id,
		ReferenceID:      pointer.StringValue(m.ReferenceID),
		Title:            pointer.StringValue(m.Title),
		Description:      pointer.StringValue(m.Description),
		Status:           domain.LoadStatus(m.Status),
		PickupAddress:    pointer.StringValue(m.PickupAddress),
		PickupLat:        m.PickupLat,
		PickupLng:        m.PickupLng,
		PickupAddressID:  pointer.StringValue(m.PickupAddressID),
		PickupAt:         m.PickupAt,
		DropoffAddress:   pointer.StringValue(m.DropoffAddress),
		DropoffLat:       m.DropoffLat,
		DropoffLng:       m.DropoffLng,
		DropoffAddressID: pointer.StringValue(m.DropoffAddressID),
		DropoffAt:        m.DropoffAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}

	if m.CompanyID != nil {
		e.CompanyID, _ = uuid.Parse(*m.CompanyID)
	}
	if m.MemberID != nil {
		e.MemberID, _ = uuid.Parse(*m.MemberID)
	}
	if m.CarrierID != nil {
		e.CarrierID, _ = uuid.Parse(*m.CarrierID)
	}

	return e
}
