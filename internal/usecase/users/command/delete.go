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

type DeleteUsecase struct {
	contextDuration time.Duration
	usersRepo       domain.UserRepository
}

func NewDeleteUsecase(contextDuration time.Duration, usersRepo domain.UserRepository) *DeleteUsecase {
	return &DeleteUsecase{
		contextDuration: contextDuration,
		usersRepo:       usersRepo,
	}
}

func (u *DeleteUsecase) Delete(ctx context.Context, userIDStr string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "Delete",
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	if err := u.usersRepo.Delete(ctx, userID); err != nil {
		logger.ErrorContext(ctx, "failed to delete user", err)
		return err
	}

	return nil
}
