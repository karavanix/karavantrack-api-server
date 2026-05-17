package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type Emails struct {
	bun.BaseModel `bun:"table:emails,alias:e"`

	ID        int64     `bun:"id,pk,autoincrement"`
	UserID    string    `bun:"user_id,type:uuid"`
	Type      string    `bun:"type"`
	Status    string    `bun:"status"`
	From      string    `bun:"from"`
	To        string    `bun:"to"`
	Bcc       []string  `bun:"bcc,array"`
	Subject   string    `bun:"subject"`
	Content   string    `bun:"content"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type emailsRepo struct {
	db bun.IDB
}

func NewEmailsRepo(db bun.IDB) domain.EmailRepository {
	return &emailsRepo{db: db}
}

func (r *emailsRepo) Save(ctx context.Context, e *domain.Email) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(e)

	if model.ID == 0 {
		_, err := db.NewInsert().Model(model).Returning("id").Exec(ctx)
		if err != nil {
			return postgres.Error(err, model)
		}
		e.ID = model.ID
		return nil
	}

	_, err := db.NewUpdate().Model(model).
		Set("status = ?", model.Status).
		Set("updated_at = ?", model.UpdatedAt).
		Where("id = ?", model.ID).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *emailsRepo) toModel(e *domain.Email) *Emails {
	bcc := e.Bcc
	if bcc == nil {
		bcc = []string{}
	}
	return &Emails{
		ID:        e.ID,
		UserID:    e.UserID.String(),
		Type:      e.Type,
		Status:    e.Status.String(),
		From:      e.From,
		To:        e.To,
		Bcc:       bcc,
		Subject:   e.Subject,
		Content:   e.Content,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func (r *emailsRepo) toDomain(m *Emails) *domain.Email {
	userID, _ := uuid.Parse(m.UserID)
	return &domain.Email{
		ID:        m.ID,
		UserID:    userID,
		Type:      m.Type,
		Status:    domain.EmailStatus(m.Status),
		From:      m.From,
		To:        m.To,
		Bcc:       m.Bcc,
		Subject:   m.Subject,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
