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
	Role         string    `bun:"role"`
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

func (r *usersRepo) Update(ctx context.Context, user *domain.User) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(user)

	_, err := db.NewUpdate().Model(model).
		Set("first_name = ?", model.FirstName).
		Set("last_name = ?", model.LastName).
		Set("role = ?", model.Role).
		Set("status = ?", model.Status).
		Set("updated_at = ?", model.UpdatedAt).
		Where("id = ?", model.ID).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *usersRepo) Save(ctx context.Context, user *domain.User) error {
	db := postgres.FromContext(ctx, r.db)
	var model = r.toModel(user)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("first_name = EXCLUDED.first_name").
		Set("last_name = EXCLUDED.last_name").
		Set("email = EXCLUDED.email").
		Set("phone = EXCLUDED.phone").
		Set("password_hash = EXCLUDED.password_hash").
		Set("role = EXCLUDED.role").
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

func (r *usersRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.User, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*domain.User), nil
	}

	db := postgres.FromContext(ctx, r.db)
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}

	var models []Users
	err := db.NewSelect().Model(&models).
		Where("id IN (?)", bun.In(strIDs)).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &Users{})
	}

	result := make(map[uuid.UUID]*domain.User, len(models))
	for i := range models {
		user := r.toDomain(&models[i])
		result[user.ID] = user
	}

	return result, nil
}

func (r *usersRepo) FindByFilter(ctx context.Context, filter *domain.UserFilter) ([]*domain.User, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []Users

	q := db.NewSelect().Model(&models)

	if filter.Contact != "" {
		q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("email = ?", filter.Contact).WhereOr("phone = ?", filter.Contact)
		})
	}

	if filter.Query != "" {
		q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("email ILIKE ?", "%"+filter.Query+"%").
				WhereOr("phone ILIKE ?", "%"+filter.Query+"%").
				WhereOr("first_name ILIKE ?", "%"+filter.Query+"%").
				WhereOr("last_name ILIKE ?", "%"+filter.Query+"%")
		})
	}

	if filter.Role != "" {
		q.Where("role = ?", filter.Role.String())
	}

	if filter.Limit > 0 {
		q.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		q.Offset(filter.Offset)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, postgres.Error(err, &Users{})
	}

	result := make([]*domain.User, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}

	return result, nil
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
		Role:         e.Role.String(),
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
		Role:         shared.Role(m.Role),
		Status:       domain.UserStatus(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	return e
}
