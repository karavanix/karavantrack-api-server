package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/shogo82148/pointer"
	"github.com/uptrace/bun"
)

type Users struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           string    `bun:"id,type:uuid,pk"`
	FirstName    *string   `bun:"first_name,nullzero"`
	LastName     *string   `bun:"last_name,nullzero"`
	Email        *string   `bun:"email,nullzero"`
	Phone        *string   `bun:"phone,nullzero"`
	PasswordHash string    `bun:"password_hash"`
	Status       string    `bun:"status"`
	CreatedAt    time.Time `bun:"created_at"`
	UpdatedAt    time.Time `bun:"updated_at"`
}

type usersRepo struct {
	db bun.IDB
}

func NewUsersRepo(db bun.IDB) domain.UserRepository {
	return &usersRepo{db: db}
}

func (r *usersRepo) Save(ctx context.Context, user *domain.User) error {
	db := postgres.FromContext(ctx, r.db)
	var model = r.toModel(user)

	_, err := db.NewInsert().Model(model).
		On("ON CONFLICT (id) DO UPDATE SET").
		Set("first_name = EXCLUDED.first_name").
		Set("last_name = EXCLUDED.last_name").
		Set("email = EXCLUDED.email").
		Set("phone = EXCLUDED.phone").
		Set("password_hash = EXCLUDED.password_hash").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *usersRepo) FindByEmail(ctx context.Context, email shared.Email) (*domain.User, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Users
	err := db.NewSelect().Model(&model).
		Where("email = ?", email.String()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return r.toDomain(&model), nil
}

func (r *usersRepo) FindByPhone(ctx context.Context, phone shared.Phone) (*domain.User, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Users
	err := db.NewSelect().Model(&model).
		Where("phone = ?", phone.String()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return r.toDomain(&model), nil
}

func (r *usersRepo) FindByEmailOrPhone(ctx context.Context, email shared.Email, phone shared.Phone) (*domain.User, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Users
	err := db.NewSelect().Model(&model).
		Where("email = ? OR phone = ?", email.String(), phone.String()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return r.toDomain(&model), nil
}

func (r *usersRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	db := postgres.FromContext(ctx, r.db)
	var model Users
	err := db.NewSelect().Model(&model).
		Where("id = ?", id.String()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return r.toDomain(&model), nil
}

func (r *usersRepo) toModel(e *domain.User) *Users {
	if e == nil {
		return nil
	}

	users := &Users{
		ID:           e.ID.String(),
		FirstName:    pointer.StringOrNil(e.FirstName),
		LastName:     pointer.StringOrNil(e.LastName),
		Email:        pointer.StringOrNil(e.Email.String()),
		Phone:        pointer.StringOrNil(e.Phone.String()),
		PasswordHash: e.PasswordHash,
		Status:       e.Status.String(),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}

	return users
}

func (r *usersRepo) toDomain(m *Users) *domain.User {
	if m == nil {
		return nil
	}

	id, _ := uuid.Parse(m.ID)

	e := &domain.User{
		ID:           id,
		FirstName:    pointer.StringValue(m.FirstName),
		LastName:     pointer.StringValue(m.LastName),
		Email:        shared.Email(pointer.StringValue(m.Email)),
		Phone:        shared.Phone(pointer.StringValue(m.Phone)),
		PasswordHash: m.PasswordHash,
		Status:       domain.UserStatus(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	return e
}
