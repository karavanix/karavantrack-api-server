package notification

import (
	"context"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

type Service interface {
	SendToUser(ctx context.Context, userID string, notification *firebase.Notification) error
}

type service struct {
	fcm            *firebase.FCM
	fcmDevicesRepo domain.FCMDeviceRepository
}

func NewService(fcm *firebase.FCM, fcmDevicesRepo domain.FCMDeviceRepository) Service {
	return &service{
		fcm:            fcm,
		fcmDevicesRepo: fcmDevicesRepo,
	}
}

func (s *service) SendToUser(ctx context.Context, userID string, notification *firebase.Notification) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	devices, err := s.fcmDevicesRepo.FindAllByUserID(ctx, uid)
	if err != nil {
		logger.WarnContext(ctx, "failed to find FCM devices for user", err, "user_id", userID)
		return err
	}

	if len(devices) == 0 {
		logger.InfoContext(ctx, "no FCM devices found for user, skipping push", "user_id", userID)
		return nil
	}

	// Collect non-expired tokens
	tokens := make([]string, 0, len(devices))
	expiredIDs := make([]int64, 0)
	for _, device := range devices {
		if device.IsExpired() {
			expiredIDs = append(expiredIDs, device.ID)
			continue
		}
		tokens = append(tokens, device.DeviceToken)
	}

	// Clean up expired devices
	if len(expiredIDs) > 0 {
		if err := s.fcmDevicesRepo.RemoveByIDs(ctx, expiredIDs); err != nil {
			logger.WarnContext(ctx, "failed to remove expired FCM devices", err, "user_id", userID)
		}
	}

	if len(tokens) == 0 {
		logger.InfoContext(ctx, "all FCM devices expired for user, skipping push", "user_id", userID)
		return nil
	}

	// Send via FCM
	result, err := s.fcm.SendBatch(ctx, tokens, notification)
	if err != nil {
		logger.ErrorContext(ctx, "failed to send push notification", err, "user_id", userID)
		return err
	}

	logger.InfoContext(ctx, "push notification sent",
		"user_id", userID,
		"success_count", result.SuccessCount,
		"failure_count", result.FailureCount,
		"invalid_tokens", len(result.InvalidTokens),
	)

	// Remove invalid tokens
	if len(result.InvalidTokens) > 0 {
		invalidIDs := make([]int64, 0)
		for _, device := range devices {
			for _, invalidToken := range result.InvalidTokens {
				if device.DeviceToken == invalidToken {
					invalidIDs = append(invalidIDs, device.ID)
				}
			}
		}
		if len(invalidIDs) > 0 {
			if err := s.fcmDevicesRepo.RemoveByIDs(ctx, invalidIDs); err != nil {
				logger.WarnContext(ctx, "failed to remove invalid FCM devices", err, "user_id", userID)
			}
		}
	}

	return nil
}
