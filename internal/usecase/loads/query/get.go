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

type GetUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
}

func NewGetUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository) *GetUsecase {
	return &GetUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo}
}

func (u *GetUsecase) Get(ctx context.Context, loadIDStr string) (_ *LoadResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Get",
		attribute.String("load_id", loadIDStr),
	)
	defer func() { end(err) }()

	loadID, err := uuid.Parse(loadIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("load_id", "invalid load ID")
	}

	load, err := u.loadsRepo.FindByID(ctx, loadID)
	if err != nil {
		return nil, err
	}

	return loadToResponse(load), nil
}
