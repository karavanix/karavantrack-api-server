package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type FCMDevice struct {
	ID          int64
	UserID      uuid.UUID
	DeviceID    string
	DeviceName  string
	DeviceType  string
	DeviceToken string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewFCMDevice(userID uuid.UUID, deviceID, deviceToken, deviceName, deviceType string) (*FCMDevice, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if deviceID == "" {
		return nil, errors.New("device ID cannot be empty")
	}
	if deviceToken == "" {
		return nil, errors.New("device token cannot be empty")
	}

	return &FCMDevice{
		UserID:      userID,
		DeviceID:    deviceID,
		DeviceToken: deviceToken,
		DeviceName:  deviceName,
		DeviceType:  deviceType,
		ExpiresAt:   time.Now().Add(time.Hour * 24 * 30),
		CreatedAt:   time.Now(),
	}, nil
}

func (f *FCMDevice) IsExpired() bool {
	return time.Now().After(f.ExpiresAt)
}

type FCMDeviceRepository interface {
	Save(ctx context.Context, e *FCMDevice) error
	FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]*FCMDevice, error)
	FindOneByUserIDAndDeviceID(ctx context.Context, userID uuid.UUID, deviceID string) (*FCMDevice, error)
	RemoveByIDs(ctx context.Context, ids []int64) error
}
