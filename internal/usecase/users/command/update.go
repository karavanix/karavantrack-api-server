package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/domain/shared"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type UpdateUsecase struct {
	contextDuration time.Duration
	usersRepo       domain.UserRepository
}

func NewUpdateUsecase(contextDuration time.Duration, usersRepo domain.UserRepository) *UpdateUsecase {
	return &UpdateUsecase{
		contextDuration: contextDuration,
		usersRepo:       usersRepo,
	}
}

type UpdateRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// Language (ru, uz or en) of the user's emails and pushes; the clients
	// send the one they are switched to. Empty keeps the current one.
	Language string `json:"language"`
}

func (u *UpdateUsecase) Update(ctx context.Context, userIDStr string, req *UpdateRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "Update",
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	var input struct {
		userID   uuid.UUID
		language shared.Language
	}
	{
		input.userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return inerr.NewErrValidation("user_id", "invalid user ID")
		}

		if req.Language != "" {
			input.language, err = shared.NewLanguage(req.Language)
			if err != nil {
				return inerr.NewErrValidation("language", err.Error())
			}
		}
	}

	user, err := u.usersRepo.FindByID(ctx, input.userID)
	if err != nil {
		return err
	}

	user.Update(req.FirstName, req.LastName)
	if input.language != "" {
		user.ChangeLanguage(input.language)
	}

	if err := u.usersRepo.Update(ctx, user); err != nil {
		logger.ErrorContext(ctx, "failed to update user", err)
		return err
	}

	return nil
}
