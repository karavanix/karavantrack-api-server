package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// ORM model
// ---------------------------------------------------------------------------

type Attachments struct {
	bun.BaseModel `bun:"table:attachments,alias:a"`

	ID           string     `bun:"id,type:uuid,pk"`
	UserID       *string    `bun:"user_id,type:uuid,nullzero"`
	Visibility   string     `bun:"visibility"`
	FileName     string     `bun:"file_name"`
	FileExt      string     `bun:"file_ext"`
	FileSize     int64      `bun:"file_size"`
	FileMime     string     `bun:"file_mime"`
	Status       string     `bun:"status"`
	ObjectBucket string     `bun:"object_bucket"`
	ObjectKey    string     `bun:"object_key"`
	CreatedAt    time.Time  `bun:"created_at"`
	UpdatedAt    time.Time  `bun:"updated_at"`
	DeletedAt    *time.Time `bun:"deleted_at"`
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

type attachmentsRepo struct {
	db bun.IDB
}

func NewAttachmentsRepo(db bun.IDB) domain.AttachmentRepository {
	return &attachmentsRepo{db: db}
}

func (r *attachmentsRepo) Save(ctx context.Context, attachment *domain.Attachment) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(attachment)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("visibility = EXCLUDED.visibility").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Set("deleted_at = EXCLUDED.deleted_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}

	return nil
}

func (r *attachmentsRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Attachments

	err := db.NewSelect().Model(&model).
		Where("a.id = ? AND a.deleted_at IS NULL", id.String()).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}

	return r.toDomain(&model), nil
}

func (r *attachmentsRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Attachment, error) {
	result := make(map[uuid.UUID]*domain.Attachment, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}

	db := postgres.FromContext(ctx, r.db)
	var models []Attachments

	err := db.NewSelect().Model(&models).
		Where("a.id IN (?) AND a.deleted_at IS NULL", bun.List(strIDs)).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &Attachments{})
	}

	for i := range models {
		attachment := r.toDomain(&models[i])
		result[attachment.ID] = attachment
	}

	return result, nil
}

func (r *attachmentsRepo) FindAllByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Attachment, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []Attachments

	err := db.NewSelect().Model(&models).
		Where("a.user_id = ? AND a.deleted_at IS NULL", userID.String()).
		Order("a.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &Attachments{})
	}

	result := make([]*domain.Attachment, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}

	return result, nil
}

func (r *attachmentsRepo) Delete(ctx context.Context, id uuid.UUID) error {
	db := postgres.FromContext(ctx, r.db)

	_, err := db.NewUpdate().Model((*Attachments)(nil)).
		Set("deleted_at = ?", time.Now()).
		Where("id = ? AND deleted_at IS NULL", id.String()).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &Attachments{})
	}

	return nil
}

// ---------------------------------------------------------------------------
// Mapping: domain <-> ORM
// ---------------------------------------------------------------------------

func (r *attachmentsRepo) toModel(e *domain.Attachment) *Attachments {
	if e == nil {
		return nil
	}

	m := &Attachments{
		ID:           e.ID.String(),
		Visibility:   e.Visibility.String(),
		FileName:     e.FileName,
		FileExt:      e.FileExt,
		FileSize:     e.FileSize,
		FileMime:     e.FileMime,
		Status:       e.Status.String(),
		ObjectBucket: e.ObjectBucket,
		ObjectKey:    e.ObjectKey,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		DeletedAt:    e.DeletedAt,
	}

	if e.UserID != uuid.Nil {
		s := e.UserID.String()
		m.UserID = &s
	}

	return m
}

func (r *attachmentsRepo) toDomain(m *Attachments) *domain.Attachment {
	if m == nil {
		return nil
	}

	id, _ := uuid.Parse(m.ID)

	e := &domain.Attachment{
		ID:           id,
		Visibility:   domain.Visibility(m.Visibility),
		FileName:     m.FileName,
		FileExt:      m.FileExt,
		FileSize:     m.FileSize,
		FileMime:     m.FileMime,
		Status:       domain.Status(m.Status),
		ObjectBucket: m.ObjectBucket,
		ObjectKey:    m.ObjectKey,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}

	if m.UserID != nil {
		e.UserID, _ = uuid.Parse(*m.UserID)
	}

	return e
}
