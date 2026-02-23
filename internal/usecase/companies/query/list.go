package query

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

type ListByUserUsecase struct {
	contextDuration time.Duration
	companiesRepo   domain.CompanyRepository
	membersRepo     domain.CompanyMemberRepository
}

func NewListByUserUsecase(
	contextDuration time.Duration,
	companiesRepo domain.CompanyRepository,
	membersRepo domain.CompanyMemberRepository,
) *ListByUserUsecase {
	return &ListByUserUsecase{
		contextDuration: contextDuration,
		companiesRepo:   companiesRepo,
		membersRepo:     membersRepo,
	}
}

func (u *ListByUserUsecase) List(ctx context.Context, userIDStr string) (_ []*CompanyResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "ListByUser",
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("user_id", "invalid user ID")
	}

	memberships, err := u.membersRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find memberships", err)
		return nil, err
	}

	result := make([]*CompanyResponse, 0, len(memberships))
	for _, m := range memberships {
		company, err := u.companiesRepo.FindByID(ctx, m.CompanyID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to find company", err)
			continue
		}
		result = append(result, &CompanyResponse{
			ID:        company.ID.String(),
			OwnerID:   company.OwnerID.String(),
			Name:      company.Name,
			Status:    company.Status.String(),
			CreatedAt: company.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}
