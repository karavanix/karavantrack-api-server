package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetActiveUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	urlResolver     *attachmentURLResolver
}

func NewGetActiveUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, attachmentsRepo domain.AttachmentRepository, s3Client *s3.S3Client) *GetActiveUsecase {
	return &GetActiveUsecase{
		contextDuration: contextDuration,
		loadsRepo:       loadsRepo,
		urlResolver:     newAttachmentURLResolver(attachmentsRepo, s3Client),
	}
}

func (u *GetActiveUsecase) GetActive(ctx context.Context, carrierID string) (_ *LoadDetailResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "GetActive",
		attribute.String("carrier_id", carrierID),
	)
	defer func() { end(err) }()

	var input struct {
		carrierID uuid.UUID
	}
	{
		input.carrierID, err = uuid.Parse(carrierID)
		if err != nil {
			return nil, inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}
	}

	load, err := u.loadsRepo.FindActiveByCarrierID(ctx, input.carrierID)
	if err != nil {
		return nil, err
	}

	return loadToDetailResponse(ctx, load, u.urlResolver), nil
}
