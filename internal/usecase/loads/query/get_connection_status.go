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

type GetConnectionStatusUsecase struct {
	contextDuration       time.Duration
	loadsRepo             domain.LoadRepository
	loadLocationPointRepo domain.LoadLocationPointRepository
	rbacService           rbac.Service
	params                domain.ConnectionParams
}

func NewGetConnectionStatusUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	rbacService rbac.Service,
	params domain.ConnectionParams,
) *GetConnectionStatusUsecase {
	return &GetConnectionStatusUsecase{
		contextDuration:       contextDuration,
		loadsRepo:             loadsRepo,
		loadLocationPointRepo: loadLocationPointRepo,
		rbacService:           rbacService,
		params:                params,
	}
}

type ConnectionStatusResponse struct {
	// State: not_started (the load isn't tracked), moving, stopped (since
	// Since; the phone sends nothing while standing), no_data (no fresh
	// point while not standing) or gps_disabled (Android reported location
	// services off or the permission gone; Reason says which).
	State  string `json:"state" enums:"not_started,moving,stopped,no_data,gps_disabled"`
	Reason string `json:"reason,omitempty" enums:"location_off,permission_denied"`
	// Since: when the truck stopped (stopped) or GPS went off (gps_disabled).
	Since       *time.Time `json:"since,omitempty"`
	LastPointAt *time.Time `json:"last_point_at,omitempty"`
	// BatteryLevel of the driver's phone, 0..1, and whether it's charging,
	// as of the latest point that reported them.
	BatteryLevel *float32 `json:"battery_level,omitempty"`
	IsCharging   *bool    `json:"is_charging,omitempty"`
}

// GetConnectionStatus reports what the driver's phone is doing on a load,
// judged by the GPS points it sent (domain.Load.Connection). requesterID is
// skipped ("") only when the caller already authorized access some other way
// (a public tracking-link token), same convention as GetPosition.
func (u *GetConnectionStatusUsecase) GetConnectionStatus(ctx context.Context, loadID string, requesterID string) (_ *ConnectionStatusResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "GetConnectionStatus",
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

	if requesterID != "" {
		allow, err := u.rbacService.CanAccessLoad(ctx, requesterID, load, domain.CompanyPermissionLoadRead)
		if err != nil {
			return nil, err
		}
		if !allow {
			return nil, inerr.ErrorPermissionDenied
		}
	}

	tail, err := u.loadLocationPointRepo.FindRecentByLoadID(ctx, input.loadID, domain.ConnectionTailSize)
	if err != nil {
		return nil, err
	}

	c := load.Connection(tail, time.Now(), u.params)
	return &ConnectionStatusResponse{
		State:        string(c.State),
		Reason:       c.Reason,
		Since:        c.Since,
		LastPointAt:  c.LastPointAt,
		BatteryLevel: c.BatteryLevel,
		IsCharging:   c.IsCharging,
	}, nil
}
