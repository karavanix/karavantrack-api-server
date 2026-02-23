package query

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type SearchCarriersUsecase struct {
	contextDuration time.Duration
	usersRepo       domain.UserRepository
}

func NewSearchCarriersUsecase(contextDuration time.Duration, usersRepo domain.UserRepository) *SearchCarriersUsecase {
	return &SearchCarriersUsecase{
		contextDuration: contextDuration,
		usersRepo:       usersRepo,
	}
}

type SearchCarriersRequest struct {
	Query string `json:"query" validate:"required"`
}

type UsersResponse struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *SearchCarriersUsecase) SearchCarriers(ctx context.Context, req *SearchCarriersRequest) (_ []*UsersResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "SearchCarriers")
	defer func() {
		end(err)
	}()

	if req.Query == "" {
		return make([]*UsersResponse, 0), nil
	}

	users, err := u.usersRepo.FindCarriersByQuery(ctx, req.Query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to search users", err)
		return nil, err
	}

	result := make([]*UsersResponse, 0, len(users))
	for _, user := range users {
		result = append(result, &UsersResponse{
			ID:        user.ID.String(),
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     string(user.Email),
			Phone:     string(user.Phone),
			Role:      user.Role.String(),
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	return result, nil
}
