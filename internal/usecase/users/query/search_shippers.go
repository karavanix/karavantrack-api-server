package query

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type SearchShippersUsecase struct {
	contextDuration time.Duration
	usersRepo       domain.UserRepository
}

func NewSearchShippersUsecase(contextDuration time.Duration, usersRepo domain.UserRepository) *SearchShippersUsecase {
	return &SearchShippersUsecase{
		contextDuration: contextDuration,
		usersRepo:       usersRepo,
	}
}

type SearchShippersRequest struct {
	Query string `json:"query" validate:"required"`
}

func (u *SearchShippersUsecase) SearchShippers(ctx context.Context, req *SearchShippersRequest) (_ []*UsersResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("users"), "SearchShippers")
	defer func() {
		end(err)
	}()

	if req.Query == "" {
		return make([]*UsersResponse, 0), nil
	}

	users, err := u.usersRepo.FindShippersByQuery(ctx, req.Query)
	if err != nil {
		logger.ErrorContext(ctx, "failed to search shippers", err)
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
