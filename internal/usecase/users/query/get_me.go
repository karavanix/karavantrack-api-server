package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetMeUsecase struct {
	contextDuration time.Duration
	usersRepo       domain.UserRepository
}

func NewGetMeUsecase(contextDuration time.Duration, usersRepo domain.UserRepository) *GetMeUsecase {
	return &GetMeUsecase{
		contextDuration: contextDuration,
		usersRepo:       usersRepo,
	}
}

type MeResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func (u *GetMeUsecase) GetMe(ctx context.Context, userIDStr string) (_ *MeResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "GetMe",
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("user_id", "invalid user ID")
	}

	user, err := u.usersRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &MeResponse{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email.String(),
		Phone:     user.Phone.String(),
		Status:    user.Status.String(),
		Role:      user.Role.String(),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}
