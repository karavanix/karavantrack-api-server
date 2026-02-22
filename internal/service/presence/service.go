package presence

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

type service struct {
	contextDuration time.Duration
	presenceRepo    domain.PresenceRepository
}

func NewService(contextDuration time.Duration, presenceRepo domain.PresenceRepository) Service {
	return &service{
		contextDuration: contextDuration,
		presenceRepo:    presenceRepo,
	}
}

func (s *service) Online(ctx context.Context, userID string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("presence"), "Online",
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		userID uuid.UUID
	}
	{
		input.userID, err = uuid.Parse(userID)
		if err != nil {
			return inerr.NewErrValidation("user_id", err.Error())
		}
	}

	presence, err := domain.NewPresence(input.userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create presence", err)
		return err
	}

	err = s.presenceRepo.Save(ctx, presence)
	if err != nil {
		logger.ErrorContext(ctx, "failed to save presence", err)
		return err
	}

	return nil
}

func (s *service) Offline(ctx context.Context, userID string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("presence"), "Offline",
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		userID uuid.UUID
	}
	{
		input.userID, err = uuid.Parse(userID)
		if err != nil {
			return inerr.NewErrValidation("user_id", err.Error())
		}
	}

	err = s.presenceRepo.Delete(ctx, input.userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to delete presence", err)
		return err
	}

	return nil
}

func (s *service) IsOnline(ctx context.Context, userID string) (_ bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("presence"), "IsOnline",
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		userID uuid.UUID
	}
	{
		input.userID, err = uuid.Parse(userID)
		if err != nil {
			return false, inerr.NewErrValidation("user_id", err.Error())
		}
	}

	isOnline, err := s.presenceRepo.IsOnline(ctx, input.userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check presence", err)
		return false, err
	}

	return isOnline, nil
}
