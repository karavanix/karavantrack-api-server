package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RegisterDeviceUsecase struct {
	contextDuration time.Duration
	fcmDevicesRepo  domain.FCMDeviceRepository
}

func NewRegisterDeviceUsecase(contextDuration time.Duration, fcmDevicesRepo domain.FCMDeviceRepository) *RegisterDeviceUsecase {
	return &RegisterDeviceUsecase{
		contextDuration: contextDuration,
		fcmDevicesRepo:  fcmDevicesRepo,
	}
}

type RegisterDeviceRequest struct {
	DeviceID    string `json:"device_id" validate:"required"`
	DeviceToken string `json:"device_token" validate:"required"`
	DeviceName  string `json:"device_name"`
	DeviceType  string `json:"device_type"`
}

func (u *RegisterDeviceUsecase) RegisterDevice(ctx context.Context, userID string, req *RegisterDeviceRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "RegisterDevice",
		attribute.String("user_id", userID),
		attribute.String("device_id", req.DeviceID),
	)
	defer func() { end(err) }()

	uid, err := uuid.Parse(userID)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	device, err := domain.NewFCMDevice(uid, req.DeviceID, req.DeviceToken, req.DeviceName, req.DeviceType)
	if err != nil {
		return inerr.NewErrValidation("device", err.Error())
	}

	if err := u.fcmDevicesRepo.Save(ctx, device); err != nil {
		logger.ErrorContext(ctx, "failed to save FCM device", err)
		return err
	}

	return nil
}
