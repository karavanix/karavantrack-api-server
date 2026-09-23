package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type GetStatsUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
	rbacService     rbac.Service
}

func NewGetStatsUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository, rbacService rbac.Service) *GetStatsUsecase {
	return &GetStatsUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo, rbacService: rbacService}
}

type GetStatsResponse struct {
	Created     int `json:"created"`
	Assigned    int `json:"assigned"`
	Accepted    int `json:"accepted"`
	PickingUp   int `json:"picking_up"`
	PickedUp    int `json:"picked_up"`
	InTransit   int `json:"in_transit"`
	DroppingOff int `json:"dropping_off"`
	DroppedOff  int `json:"dropped_off"`
	Confirmed   int `json:"confirmed"`
	Cancelled   int `json:"canceled"`
	Total       int `json:"total"`
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

	allow, err := u.rbacService.HasPermission(ctx, companyID, requesterID, domain.CompanyPermissionLoadRead)
	if err != nil {
		return nil, err
	}
	if !allow {
		return nil, inerr.ErrorPermissionDenied
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
		Created:     stats.Created,
		Assigned:    stats.Assigned,
		Accepted:    stats.Accepted,
		PickingUp:   stats.PickingUp,
		PickedUp:    stats.PickedUp,
		InTransit:   stats.InTransit,
		DroppingOff: stats.DroppingOff,
		DroppedOff:  stats.DroppedOff,
		Confirmed:   stats.Confirmed,
		Cancelled:   stats.Canceled,
		Total:       stats.Total,
	}

	return result, nil
}
