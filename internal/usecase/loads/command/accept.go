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

type AcceptUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
}

func NewAcceptUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository) *AcceptUsecase {
	return &AcceptUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo}
}

func (u *AcceptUsecase) Accept(ctx context.Context, loadIDStr string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Accept",
		attribute.String("load_id", loadIDStr),
	)
	defer func() { end(err) }()

	loadID, err := uuid.Parse(loadIDStr)
	if err != nil {
		return inerr.NewErrValidation("load_id", "invalid load ID")
	}

	load, err := u.loadsRepo.FindByID(ctx, loadID)
	if err != nil {
		return err
	}

	if err := load.Accept(); err != nil {
		return inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Update(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return err
	}

	// TODO: enqueue push notification to cargo owner

	return nil
}
