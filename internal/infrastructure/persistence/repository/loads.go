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

// ---------------------------------------------------------------------------
// ORM models
// ---------------------------------------------------------------------------

type LoadStatusHistoryAttachments struct {
	bun.BaseModel `bun:"table:load_status_history_attachments,alias:lsha"`

	ID           int64     `bun:"id,pk,autoincrement"`
	HistoryID    int64     `bun:"history_id"`
	AttachmentID string    `bun:"attachment_id,type:uuid"`
	CreatedAt    time.Time `bun:"created_at"`
}

type LoadStatusHistories struct {
	bun.BaseModel `bun:"table:load_status_histories,alias:lsh"`

	ID          int64                           `bun:"id,pk,autoincrement"`
	LoadID      string                          `bun:"load_id,type:uuid"`
	UserID      *string                         `bun:"user_id,type:uuid,nullzero"`
	FromStatus  string                          `bun:"from_status"`
	ToStatus    string                          `bun:"to_status"`
	Note        string                          `bun:"note"`
	CreatedAt   time.Time                       `bun:"created_at"`
	Attachments []*LoadStatusHistoryAttachments `bun:"rel:has-many,join:id=history_id"`
}

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

	History []*LoadStatusHistories `bun:"rel:has-many,join:id=load_id"`
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

type loadsRepo struct {
	db bun.IDB
}

func NewLoadsRepo(db bun.IDB) domain.LoadRepository {
	return &loadsRepo{db: db}
}

// Save upserts the load and persists any new history entries (ID == 0) along
// with their new attachments.
func (r *loadsRepo) Save(ctx context.Context, load *domain.Load) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(load)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("carrier_id = EXCLUDED.carrier_id").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}

	// Persist new history entries and their attachments.
	// The uniform rule at every level: ID == 0 means new, non-zero means already persisted.
	for _, h := range load.History {
		if h.ID == 0 {
			histModel := r.toHistoryModel(h, load.ID)
			if err := db.NewInsert().Model(histModel).Returning("id").Scan(ctx); err != nil {
				return postgres.Error(err, histModel)
			}
			h.ID = histModel.ID // write back so the aggregate stays in sync
		}

		// Check attachments regardless of whether history is new or old —
		// attachments can be added to an existing history entry too.
		for _, att := range h.Attachments {
			if att.ID != 0 {
				continue // already persisted
			}
			attModel := r.toAttachmentModel(att, h.ID)
			if err := db.NewInsert().Model(attModel).Returning("id").Scan(ctx); err != nil {
				return postgres.Error(err, attModel)
			}
			att.ID = attModel.ID // write back
		}
	}

	return nil
}

func (r *loadsRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Load, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Loads

	err := db.NewSelect().
		Model(&model).
		Where("l.id = ?", id.String()).
		Relation("History", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lsh.id ASC")
		}).
		Relation("History.Attachments").
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}

	return r.toDomain(&model), nil
}

func (r *loadsRepo) FindActiveByCarrierID(ctx context.Context, carrierID uuid.UUID) (*domain.Load, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Loads

	err := db.NewSelect().
		Model(&model).
		Where(
			"l.carrier_id = ? AND l.status IN (?)",
			carrierID.String(),
			bun.In(activeStatuses()),
		).
		OrderExpr("l.created_at DESC").
		Limit(1).
		Relation("History", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lsh.id ASC")
		}).
		Relation("History.Attachments").
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &Loads{})
	}

	return r.toDomain(&model), nil
}

func (r *loadsRepo) FindActiveByCarrierIDs(ctx context.Context, carrierIDs []uuid.UUID) (map[uuid.UUID]*domain.Load, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []Loads

	ids := make([]string, len(carrierIDs))
	for i, id := range carrierIDs {
		ids[i] = id.String()
	}

	err := db.NewSelect().
		Model(&models).
		Where("l.carrier_id IN (?) AND l.status IN (?)", bun.In(ids), bun.In(activeStatuses())).
		OrderExpr("l.created_at DESC").
		Relation("History", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lsh.id ASC")
		}).
		Relation("History.Attachments").
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &Loads{})
	}

	result := make(map[uuid.UUID]*domain.Load)
	for i := range models {
		load := r.toDomain(&models[i])
		if load == nil {
			continue
		}
		if _, exists := result[load.CarrierID]; !exists {
			result[load.CarrierID] = load
		}
	}

	return result, nil
}

func (r *loadsRepo) FindAll(ctx context.Context, filter domain.LoadFilter) ([]*domain.Load, int, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []Loads

	q := db.NewSelect().
		Model(&models).
		Relation("History", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lsh.id ASC")
		}).
		Relation("History.Attachments")

	q = applyLoadFilter(q, filter)
	q = q.OrderExpr("l.created_at DESC")

	if err := q.Scan(ctx); err != nil {
		return nil, 0, postgres.Error(err, &Loads{})
	}

	countQ := db.NewSelect().TableExpr("loads AS l")
	countQ = applyLoadFilterRaw(countQ, filter)
	total, err := countQ.Count(ctx)
	if err != nil {
		return nil, 0, postgres.Error(err, &Loads{})
	}

	result := make([]*domain.Load, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}

	return result, total, nil
}

func (r *loadsRepo) FindStats(ctx context.Context, filter domain.LoadFilter) (*domain.LoadStats, error) {
	db := postgres.FromContext(ctx, r.db)

	var stats domain.LoadStats
	q := db.NewSelect().
		TableExpr("loads").
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS created", domain.LoadStatusCreated).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS assigned", domain.LoadStatusAssigned).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS accepted", domain.LoadStatusAccepted).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS picking_up", domain.LoadStatusPickingUp).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS picked_up", domain.LoadStatusPickedUp).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS in_transit", domain.LoadStatusInTransit).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS dropping_off", domain.LoadStatusDroppingOff).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS dropped_off", domain.LoadStatusDroppedOff).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS confirmed", domain.LoadStatusConfirmed).
		ColumnExpr("COUNT(*) FILTER (WHERE status = ?) AS canceled", domain.LoadStatusCancelled).
		ColumnExpr("COUNT(*) AS total")

	if filter.CompanyID != nil {
		q = q.Where("company_id = ?", filter.CompanyID.String())
	}
	if filter.CarrierID != nil {
		q = q.Where("carrier_id = ?", filter.CarrierID.String())
	}

	if err := q.Scan(ctx, &stats); err != nil {
		return nil, postgres.Error(err, &Loads{})
	}

	return &stats, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func activeStatuses() []string {
	return []string{
		domain.LoadStatusAccepted.String(),
		domain.LoadStatusPickingUp.String(),
		domain.LoadStatusPickedUp.String(),
		domain.LoadStatusInTransit.String(),
		domain.LoadStatusDroppingOff.String(),
		domain.LoadStatusDroppedOff.String(),
	}
}

func applyLoadFilter(q *bun.SelectQuery, filter domain.LoadFilter) *bun.SelectQuery {
	if filter.CompanyID != nil {
		q = q.Where("l.company_id = ?", filter.CompanyID.String())
	}
	if filter.CarrierID != nil {
		q = q.Where("l.carrier_id = ?", filter.CarrierID.String())
	}
	if len(filter.Status) > 0 {
		q = q.Where("l.status IN (?)", bun.In(filter.Status))
	}
	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(50)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}
	return q
}

// applyLoadFilterRaw applies WHERE conditions to a raw table-expr query (used
// for COUNT, which must not carry Limit/Offset).
func applyLoadFilterRaw(q *bun.SelectQuery, filter domain.LoadFilter) *bun.SelectQuery {
	if filter.CompanyID != nil {
		q = q.Where("company_id = ?", filter.CompanyID.String())
	}
	if filter.CarrierID != nil {
		q = q.Where("carrier_id = ?", filter.CarrierID.String())
	}
	if len(filter.Status) > 0 {
		q = q.Where("status IN (?)", bun.In(filter.Status))
	}
	return q
}

// ---------------------------------------------------------------------------
// Mapping: domain ↔ ORM
// ---------------------------------------------------------------------------

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

func (r *loadsRepo) toHistoryModel(h *domain.LoadStatusHistory, loadID uuid.UUID) *LoadStatusHistories {
	m := &LoadStatusHistories{
		ID:         h.ID,
		LoadID:     loadID.String(),
		FromStatus: h.FromStatus.String(),
		ToStatus:   h.ToStatus.String(),
		Note:       h.Note,
		CreatedAt:  h.CreatedAt,
	}
	if h.UserID != uuid.Nil {
		s := h.UserID.String()
		m.UserID = &s
	}
	return m
}

func (r *loadsRepo) toAttachmentModel(att *domain.LoadStatusHistoryAttachment, historyID int64) *LoadStatusHistoryAttachments {
	return &LoadStatusHistoryAttachments{
		ID:           att.ID,
		HistoryID:    historyID,
		AttachmentID: att.AttachmentID.String(),
		CreatedAt:    att.CreatedAt,
	}
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

	e.History = make([]*domain.LoadStatusHistory, len(m.History))
	for i, h := range m.History {
		dh := &domain.LoadStatusHistory{
			ID:         h.ID,
			LoadID:     id,
			FromStatus: domain.LoadStatus(h.FromStatus),
			ToStatus:   domain.LoadStatus(h.ToStatus),
			Note:       h.Note,
			CreatedAt:  h.CreatedAt,
		}
		if h.UserID != nil {
			dh.UserID, _ = uuid.Parse(*h.UserID)
		}
		dh.Attachments = make([]*domain.LoadStatusHistoryAttachment, len(h.Attachments))
		for j, att := range h.Attachments {
			attID, _ := uuid.Parse(att.AttachmentID)
			dh.Attachments[j] = &domain.LoadStatusHistoryAttachment{
				ID:           att.ID,
				HistoryID:    att.HistoryID,
				AttachmentID: attID,
				CreatedAt:    att.CreatedAt,
			}
		}
		e.History[i] = dh
	}

	return e
}
