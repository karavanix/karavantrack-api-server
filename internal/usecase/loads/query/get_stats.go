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

type GetStatsUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
}

func NewGetStatsUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository) *GetStatsUsecase {
	return &GetStatsUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo}
}

type GetStatsResponse struct {
	Active    int `json:"active"`
	Pending   int `json:"pending"`
	Completed int `json:"completed"`
	Canceled  int `json:"canceled"`
	Total     int `json:"total"`
}

func (u *GetStatsUsecase) GetStats(ctx context.Context, requesterID string, companyID string) (_ *GetStatsResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "GetTrack",
		attribute.String("requester_id", requesterID),
		attribute.String("company_id", companyID),
	)
	defer func() { end(err) }()

	var input struct {
		userID    uuid.UUID
		companyID uuid.UUID
	}
	{
		input.userID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("load_id", "invalid load ID")
		}

		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
	}

	filter := domain.LoadFilter{}
	if input.companyID != uuid.Nil {
		filter.CompanyID = &input.companyID
	}

	stats, err := u.loadsRepo.FindStats(ctx, filter)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find stats", err)
		return nil, err
	}

	result := &GetStatsResponse{
		Active:    stats.Accepted + stats.InTransit + stats.Completed,
		Pending:   stats.Created + stats.Assigned,
		Completed: stats.Confirmed,
		Canceled:  stats.Canceled,
		Total:     stats.Total,
	}

	return result, nil
}
