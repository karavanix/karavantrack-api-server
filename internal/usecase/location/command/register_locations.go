package command

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/events"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/broker"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RegisterLocationsUsecase struct {
	contextTimeout        time.Duration
	windowParams          domain.TrackingWindowParams
	bkr                   broker.Broker
	eventsFactory         *events.Factory
	loadsRepo             domain.LoadRepository
	loadLocationPointRepo domain.LoadLocationPointRepository
	matchScheduler        *tasks.MatchLoadTrackScheduler
}

func NewRegisterLocationsUsecase(
	contextTimeout time.Duration,
	windowParams domain.TrackingWindowParams,
	bkr broker.Broker,
	eventsFactory *events.Factory,
	loadsRepo domain.LoadRepository,
	loadLocationPointRepo domain.LoadLocationPointRepository,
	matchScheduler *tasks.MatchLoadTrackScheduler,
) *RegisterLocationsUsecase {
	return &RegisterLocationsUsecase{
		contextTimeout:        contextTimeout,
		windowParams:          windowParams,
		bkr:                   bkr,
		eventsFactory:         eventsFactory,
		loadsRepo:             loadsRepo,
		loadLocationPointRepo: loadLocationPointRepo,
		matchScheduler:        matchScheduler,
	}
}

// RegisterLocationsPoint is one record of the tracking library, rendered by
// its locationTemplate. load_id is stamped into the record when it's taken,
// so a batch may hold points of several loads.
type RegisterLocationsPoint struct {
	UUID       string    `json:"uuid"`
	LoadID     string    `json:"load_id"`
	RecordedAt time.Time `json:"recorded_at"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  *float32  `json:"accuracy_m"`
	AltitudeM  *float32  `json:"altitude_m"`
	// SpeedMps and HeadingDeg are -1 when unknown.
	SpeedMps   *float32 `json:"speed_mps"`
	HeadingDeg *float32 `json:"heading_deg"`
	// Event is "" for a regular location.
	Event              string   `json:"event"`
	IsMoving           *bool    `json:"is_moving"`
	ActivityType       string   `json:"activity_type"`
	ActivityConfidence *int16   `json:"activity_confidence"`
	OdometerM          *float64 `json:"odometer_m"`
	// BatteryLevel is a fraction, 0..1.
	BatteryLevel *float32 `json:"battery_level"`
	IsCharging   *bool    `json:"is_charging"`
	IsMock       *bool    `json:"is_mock"`
	// Provider is what Android attaches to a providerchange record.
	Provider json.RawMessage `json:"provider,omitempty" swaggertype:"object"`

	decodeErr error
}

// UnmarshalJSON never fails on a point that is valid JSON but doesn't fit
// the fields (a string where a number belongs): the point is dropped alone
// instead of failing the batch. The library deletes records only on a 2xx,
// so a batch that keeps failing would block the phone's queue for good.
func (p *RegisterLocationsPoint) UnmarshalJSON(data []byte) error {
	type plain RegisterLocationsPoint
	if err := json.Unmarshal(data, (*plain)(p)); err != nil {
		*p = RegisterLocationsPoint{decodeErr: err}
	}
	return nil
}

type RegisterLocationsRequest struct {
	CarrierID string                   `json:"-"`
	Points    []RegisterLocationsPoint `json:"points"`
}

type RegisterLocationsResponse struct {
	// Accepted points were stored (or had already been stored).
	Accepted int `json:"accepted"`
	// Dropped points are malformed or fall outside their load's tracking
	// window; they're not coming back, so the phone deletes them all the same.
	Dropped int `json:"dropped"`
	// LoadStatus is the status of the load of the batch's latest point; null
	// when that load doesn't exist or isn't this driver's.
	LoadStatus *string `json:"load_status"`
	// StopTracking tells the phone to stop: that load is confirmed,
	// cancelled, waiting for a confirmation too long, or not this driver's.
	StopTracking bool `json:"stop_tracking"`
}

// RegisterLocations stores the driver's GPS points that fall within their
// load's tracking window and drops the rest. Only a broken request is an
// error: anything the phone can't fix by sending again is dropped and
// counted, so the phone can clear it from its queue.
func (u *RegisterLocationsUsecase) RegisterLocations(ctx context.Context, req *RegisterLocationsRequest) (_ *RegisterLocationsResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("location"), "RegisterLocations",
		attribute.String("carrier_id", req.CarrierID),
		attribute.Int("point_count", len(req.Points)),
	)
	defer func() { end(err) }()

	var input struct {
		carrierID uuid.UUID
	}
	{
		input.carrierID, err = uuid.Parse(req.CarrierID)
		if err != nil {
			return nil, inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}
	}

	now := time.Now()
	resp := &RegisterLocationsResponse{}
	drops := map[dropKey]int{}
	drop := func(reason string, loadID uuid.UUID) {
		resp.Dropped++
		drops[dropKey{reason: reason, loadID: loadID}]++
	}

	// loads holds the driver's loads the batch refers to; nil for a load
	// that doesn't exist or belongs to someone else.
	loads := map[uuid.UUID]*domain.Load{}
	var latest struct {
		loadID     uuid.UUID
		recordedAt time.Time
	}
	var points []*domain.LoadLocationPoint
	for _, p := range req.Points {
		point, reason := u.parsePoint(p, input.carrierID)
		if point == nil {
			drop(reason, uuid.Nil)
			continue
		}

		load, seen := loads[point.LoadID]
		if !seen {
			load, err = u.findCarrierLoad(ctx, point.LoadID, input.carrierID)
			if err != nil {
				return nil, err
			}
			loads[point.LoadID] = load
		}

		if point.RecordedAt.After(latest.recordedAt) {
			latest.loadID, latest.recordedAt = point.LoadID, point.RecordedAt
		}

		switch {
		case load == nil:
			drop("load not found or not the driver's", point.LoadID)
		case !load.AcceptsTrackingPointAt(point.RecordedAt, u.windowParams):
			drop("outside the tracking window", point.LoadID)
		default:
			points = append(points, point)
		}
	}

	for key, count := range drops {
		logger.InfoContext(ctx, "dropping location points", "reason", key.reason, "load_id", key.loadID.String(), "count", count)
	}

	if latest.loadID != uuid.Nil {
		if load := loads[latest.loadID]; load != nil {
			status := load.Status.String()
			resp.LoadStatus = &status
			resp.StopTracking = load.ShouldStopTracking(now, u.windowParams)
		} else {
			resp.StopTracking = true
		}
	}

	if len(points) == 0 {
		return resp, nil
	}

	if err := u.loadLocationPointRepo.BatchSave(ctx, points); err != nil {
		logger.ErrorContext(ctx, "failed to save location points", err)
		return nil, err
	}
	resp.Accepted = len(points)

	scheduled := map[uuid.UUID]bool{}
	for _, point := range points {
		if scheduled[point.LoadID] {
			continue
		}
		scheduled[point.LoadID] = true
		if err := u.matchScheduler.Schedule(ctx, point.LoadID.String()); err != nil {
			logger.ErrorContext(ctx, "failed to schedule load track matching", err)
		}
	}

	for _, point := range points {
		u.publish(ctx, point)
	}

	return resp, nil
}

type dropKey struct {
	reason string
	loadID uuid.UUID
}

// parsePoint turns a record into a point of the given carrier, or says why it
// can't be one.
func (u *RegisterLocationsUsecase) parsePoint(p RegisterLocationsPoint, carrierID uuid.UUID) (*domain.LoadLocationPoint, string) {
	if p.decodeErr != nil {
		return nil, "undecodable point"
	}
	pointUUID, err := uuid.Parse(p.UUID)
	if err != nil {
		return nil, "invalid uuid"
	}
	loadID, err := uuid.Parse(p.LoadID)
	if err != nil {
		return nil, "invalid load_id"
	}
	if p.RecordedAt.IsZero() {
		return nil, "missing recorded_at"
	}

	point, err := domain.NewLoadLocationPoint(loadID, carrierID, p.Lat, p.Lng, p.AccuracyM, unknownIfNegative(p.SpeedMps), unknownIfNegative(p.HeadingDeg), p.RecordedAt)
	if err != nil {
		return nil, "invalid coordinates"
	}
	point.UUID = pointUUID
	point.AltitudeM = p.AltitudeM
	point.Event = p.Event
	point.IsMoving = p.IsMoving
	point.ActivityType = p.ActivityType
	point.ActivityConfidence = p.ActivityConfidence
	point.OdometerM = p.OdometerM
	point.BatteryLevel = p.BatteryLevel
	point.IsCharging = p.IsCharging
	point.IsMock = p.IsMock
	if len(p.Provider) > 0 && string(p.Provider) != "null" {
		point.Provider = p.Provider
	}
	return point, ""
}

// findCarrierLoad returns the load with its history, or nil when it doesn't
// exist or isn't the carrier's.
func (u *RegisterLocationsUsecase) findCarrierLoad(ctx context.Context, loadID, carrierID uuid.UUID) (*domain.Load, error) {
	load, err := u.loadsRepo.FindByID(ctx, loadID)
	if errors.Is(err, inerr.ErrNotFound{}) {
		return nil, nil
	}
	if err != nil {
		logger.ErrorContext(ctx, "failed to find load", err)
		return nil, err
	}
	if load.CarrierID != carrierID {
		return nil, nil
	}
	return load, nil
}

func (u *RegisterLocationsUsecase) publish(ctx context.Context, point *domain.LoadLocationPoint) {
	event, err := u.eventsFactory.LoadLocationPointCreatedEvent(&events.LoadLocationPointCreatedEvent{
		LoadID:     point.LoadID.String(),
		CarrierID:  point.CarrierID.String(),
		Lat:        point.Lat,
		Lng:        point.Lng,
		AccuracyM:  point.AccuracyM,
		SpeedMps:   point.SpeedMps,
		HeadingDeg: point.HeadingDeg,
		RecordedAt: point.RecordedAt,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to create load location point event", err)
		return
	}
	if err := u.bkr.Publish(ctx, event); err != nil {
		logger.ErrorContext(ctx, "failed to publish load location point event", err)
	}
}

// unknownIfNegative maps the library's -1 ("unknown") to nil.
func unknownIfNegative(v *float32) *float32 {
	if v == nil || *v < 0 {
		return nil
	}
	return v
}
