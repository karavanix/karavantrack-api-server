package notification

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/i18n"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/firebase"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

type Service interface {
	// SendToUser delivers an already-worded notification to every device of
	// the user.
	SendToUser(ctx context.Context, userID string, notification *firebase.Notification) error
	// Notify words the push text key in the user's language and delivers it.
	// A user who no longer exists gets nothing and is not an error.
	Notify(ctx context.Context, userID string, key i18n.Key, params map[string]string, metadata map[string]string) error
}

type service struct {
	fcm            *firebase.FCM
	fcmDevicesRepo domain.FCMDeviceRepository
	usersRepo      domain.UserRepository
}

func NewService(fcm *firebase.FCM, fcmDevicesRepo domain.FCMDeviceRepository, usersRepo domain.UserRepository) Service {
	return &service{
		fcm:            fcm,
		fcmDevicesRepo: fcmDevicesRepo,
		usersRepo:      usersRepo,
	}
}

func (s *service) Notify(ctx context.Context, userID string, key i18n.Key, params map[string]string, metadata map[string]string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	user, err := s.usersRepo.FindByID(ctx, uid)
	if err != nil {
		if errors.Is(err, inerr.ErrNotFound{}) {
			logger.InfoContext(ctx, "push recipient no longer exists, skipping push", "user_id", userID, "key", string(key))
			return nil
		}
		logger.WarnContext(ctx, "failed to find push recipient", err, "user_id", userID)
		return err
	}

	push, err := i18n.RenderPush(key, user.Language, params)
	if err != nil {
		logger.ErrorContext(ctx, "failed to render push notification", err, "user_id", userID, "key", string(key))
		return err
	}

	return s.SendToUser(ctx, userID, firebase.CreateNotificationWithMetadata(push.Title, push.Body, metadata))
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
