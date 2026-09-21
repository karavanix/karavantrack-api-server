package query

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/liveack"
	"github.com/karavanix/karavantrack-api-server/internal/service/presence"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/internal/service/watcher"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	ConnectionStateNotStarted   = "not_started"
	ConnectionStateLive         = "live"
	ConnectionStateEconomy      = "economy"
	ConnectionStateDisconnected = "disconnected"
	ConnectionStateGpsDisabled  = "gps_disabled"
)

// activeTrackingStatuses mirrors the set of load statuses gps_notify already
// treats as "should have a moving driver" (see FindWithStaleGps) — outside
// this set there's nothing to report a connection status about.
var activeTrackingStatuses = map[domain.LoadStatus]struct{}{
	domain.LoadStatusPickingUp:   {},
	domain.LoadStatusPickedUp:    {},
	domain.LoadStatusInTransit:   {},
	domain.LoadStatusDroppingOff: {},
}

type GetConnectionStatusUsecase struct {
	contextDuration       time.Duration
	loadsRepo             domain.LoadRepository
	loadLocationPointRepo domain.LoadLocationPointRepository
	rbacService           rbac.Service
	presenceService       presence.Service
	watcherService        watcher.Service
	liveAckService        liveack.Service
}

func NewGetConnectionStatusUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	rbacService rbac.Service,
	presenceService presence.Service,
	watcherService watcher.Service,
	liveAckService liveack.Service,
) *GetConnectionStatusUsecase {
	return &GetConnectionStatusUsecase{
		contextDuration:       contextDuration,
		loadsRepo:             loadsRepo,
		loadLocationPointRepo: loadLocationPointRepo,
		rbacService:           rbacService,
		presenceService:       presenceService,
		watcherService:        watcherService,
		liveAckService:        liveAckService,
	}
}

type ConnectionStatusResponse struct {
	State       string     `json:"state"`
	Reason      string     `json:"reason,omitempty"`
	LastPointAt *time.Time `json:"last_point_at,omitempty"`
}

// GetConnectionStatus reports whether the driver's phone is actually
// reachable and streaming GPS for a load, as opposed to whether the
// requester's own browser has a WebSocket open. requesterID is skipped
// ("") only when the caller already authorized access some other way (a
// public tracking-link token), same convention as GetPosition.
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

	resp := &ConnectionStatusResponse{}

	if point, err := u.loadLocationPointRepo.FindLatestByLoadID(ctx, input.loadID); err == nil {
		resp.LastPointAt = &point.RecordedAt
	} else if !errors.Is(err, inerr.ErrNotFound{}) {
		return nil, err
	}

	if _, active := activeTrackingStatuses[load.Status]; !active || load.CarrierID == uuid.Nil {
		resp.State = ConnectionStateNotStarted
		return resp, nil
	}

	online, err := u.presenceService.IsOnline(ctx, load.CarrierID.String())
	if err != nil {
		return nil, err
	}
	if !online {
		resp.State = ConnectionStateDisconnected
		return resp, nil
	}

	ack, err := u.liveAckService.Get(ctx, loadID)
	if err != nil {
		return nil, err
	}
	if ack != nil && ack.Status == liveack.StatusFailed {
		resp.State = ConnectionStateGpsDisabled
		resp.Reason = ack.Reason
		return resp, nil
	}

	watcherCount, err := u.watcherService.Count(ctx, loadID)
	if err != nil {
		return nil, err
	}
	if watcherCount > 0 && ack != nil && ack.Status == liveack.StatusStarted {
		resp.State = ConnectionStateLive
		return resp, nil
	}

	resp.State = ConnectionStateEconomy
	return resp, nil
}
