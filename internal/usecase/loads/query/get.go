package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	rbacService     rbac.Service
}

func NewGetUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, rbacService rbac.Service) *GetUsecase {
	return &GetUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, rbacService: rbacService}
}

func (u *GetUsecase) Get(ctx context.Context, loadID string, requesterID string) (_ *LoadDetailResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "Get",
		attribute.String("load_id", loadID),
		attribute.String("requester_id", requesterID),
	)
	defer func() { end(err) }()

	var input struct {
		loadID uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return nil, inerr.NewErrValidation("load_id", "invalid load ID")
		}
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return nil, err
	}

	allow, err := u.rbacService.CanAccessLoad(ctx, requesterID, load, domain.CompanyPermissionLoadRead)
	if err != nil {
		return nil, err
	}
	if !allow {
		return nil, inerr.ErrorPermissionDenied
	}

	return loadToDetailResponse(load), nil
}
