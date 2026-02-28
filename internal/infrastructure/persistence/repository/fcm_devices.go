package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/uptrace/bun"
)

type UserFCMDevices struct {
	bun.BaseModel `bun:"table:user_fcm_devices,alias:ufd"`

	ID          int64     `bun:"id,pk,autoincrement"`
	UserID      string    `bun:"user_id,type:uuid,notnull"`
	DeviceID    string    `bun:"device_id,notnull"`
	DeviceName  *string   `bun:"device_name,nullzero"`
	DeviceType  *string   `bun:"device_type,nullzero"`
	DeviceToken string    `bun:"device_token,notnull"`
	ExpiresAt   time.Time `bun:"expires_at"`
	CreatedAt   time.Time `bun:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at"`
}

type fcmDevicesRepo struct {
	db bun.IDB
}

func NewFCMDevicesRepo(db bun.IDB) domain.FCMDeviceRepository {
	return &fcmDevicesRepo{db: db}
}

func (r *fcmDevicesRepo) Save(ctx context.Context, e *domain.FCMDevice) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(e)

	_, err := db.NewInsert().Model(model).
		On("CONFLICT (user_id, device_id) DO UPDATE").
		Set("device_token = EXCLUDED.device_token").
		Set("device_name = EXCLUDED.device_name").
		Set("device_type = EXCLUDED.device_type").
		Set("expires_at = EXCLUDED.expires_at").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	return nil
}

func (r *fcmDevicesRepo) FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.FCMDevice, error) {
	db := postgres.FromContext(ctx, r.db)
	var models []UserFCMDevices
	err := db.NewSelect().Model(&models).
		Where("user_id = ?", userID.String()).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &UserFCMDevices{})
	}

	result := make([]*domain.FCMDevice, len(models))
	for i := range models {
		result[i] = r.toDomain(&models[i])
	}
	return result, nil
}

func (r *fcmDevicesRepo) FindOneByUserIDAndDeviceID(ctx context.Context, userID uuid.UUID, deviceID string) (*domain.FCMDevice, error) {
	db := postgres.FromContext(ctx, r.db)
	var model UserFCMDevices
	err := db.NewSelect().Model(&model).
		Where("user_id = ? AND device_id = ?", userID.String(), deviceID).
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &UserFCMDevices{})
	}
	return r.toDomain(&model), nil
}

func (r *fcmDevicesRepo) RemoveByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	db := postgres.FromContext(ctx, r.db)
	_, err := db.NewDelete().Model((*UserFCMDevices)(nil)).
		Where("id IN (?)", bun.In(ids)).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &UserFCMDevices{})
	}
	return nil
}

func (r *fcmDevicesRepo) toModel(e *domain.FCMDevice) *UserFCMDevices {
	if e == nil {
		return nil
	}
	m := &UserFCMDevices{
		ID:          e.ID,
		UserID:      e.UserID.String(),
		DeviceToken: e.DeviceToken,
		DeviceID:    e.DeviceID,
		ExpiresAt:   e.ExpiresAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	if e.DeviceName != "" {
		m.DeviceName = &e.DeviceName
	}
	if e.DeviceType != "" {
		m.DeviceType = &e.DeviceType
	}
	return m
}

func (r *fcmDevicesRepo) toDomain(m *UserFCMDevices) *domain.FCMDevice {
	if m == nil {
		return nil
	}
	userID, _ := uuid.Parse(m.UserID)
	e := &domain.FCMDevice{
		ID:          m.ID,
		UserID:      userID,
		DeviceID:    m.DeviceID,
		DeviceToken: m.DeviceToken,
		ExpiresAt:   m.ExpiresAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.DeviceName != nil {
		e.DeviceName = *m.DeviceName
	}
	if m.DeviceType != nil {
		e.DeviceType = *m.DeviceType
	}
	return e
}
