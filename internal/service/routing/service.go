package routing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/retry"
)

// MatcherVersion is stored with every track. Bump it when the matching
// logic or its settings change, so tracks built the old way can be found
// and rematched.
const MatcherVersion = "valhalla-auto/v2"

// minSplitPoints: a failing piece this small isn't split further; it's kept
// as a raw line instead.
const minSplitPoints = 3

type Config struct {
	GapThreshold        time.Duration
	StopRadiusM         float64
	StopMinDuration     time.Duration
	DepartureRadiusM    float64
	MaxAccuracyM        float64
	RadiusMultiplier    float64
	MinRadiusM          float64
	MaxRadiusM          float64
	BreakageSpeedMps    float64
	BreakageFactor      float64
	MaxRequestDistanceM float64
	// Retry applies to an unreachable engine only; other errors are
	// handled by splitting or falling back to raw points.
	Retry retry.RetryConfig
}

func ConfigFrom(cfg *config.Config) Config {
	m := cfg.Matching
	return Config{
		GapThreshold:        m.GapThreshold,
		StopRadiusM:         m.StopRadiusM,
		StopMinDuration:     m.StopMinDuration,
		DepartureRadiusM:    m.DepartureRadiusM,
		MaxAccuracyM:        m.MaxAccuracyM,
		RadiusMultiplier:    m.RadiusMultiplier,
		MinRadiusM:          m.MinRadiusM,
		MaxRadiusM:          m.MaxRadiusM,
		BreakageSpeedMps:    m.BreakageSpeedMps,
		BreakageFactor:      m.BreakageFactor,
		MaxRequestDistanceM: m.MaxRequestDistanceM,
		Retry: retry.RetryConfig{
			MaxAttempts:   3,
			RetryInterval: time.Second,
			MaxInterval:   5 * time.Second,
			Multiplier:    2,
		},
	}
}

// SplitParams is how tracks are cut into moving, stop and gap pieces.
func (c Config) SplitParams() domain.TrackSplitParams {
	return domain.TrackSplitParams{
		MaxAccuracyM:     c.MaxAccuracyM,
		GapThreshold:     c.GapThreshold,
		StopRadiusM:      c.StopRadiusM,
		StopMinDuration:  c.StopMinDuration,
		DepartureRadiusM: c.DepartureRadiusM,
	}
}

// Route is a driving route between two points.
type Route struct {
	Geometry  []geo.Point
	DistanceM float64
	Duration  time.Duration
}

type Service interface {
	// MatchLoadTrack builds a load's track from all of its raw points. It
	// returns an error only when the routing engine is unreachable; parts
	// the engine can't match become raw segments.
	MatchLoadTrack(ctx context.Context, loadID uuid.UUID, points domain.LoadLocationTrack) (*domain.LoadTrack, error)
	// Route builds a driving route from one point to another.
	Route(ctx context.Context, from, to geo.Point) (*Route, error)
}

type service struct {
	provider ports.ValhallaProvider
	cfg      Config
}

func NewService(provider ports.ValhallaProvider, cfg Config) Service {
	cfg.Retry.ShouldRetry = func(err error) bool { return errors.Is(err, inerr.ErrRoutingUnavailable) }
	return &service{provider: provider, cfg: cfg}
}

func (s *service) Route(ctx context.Context, from, to geo.Point) (*Route, error) {
	res, err := retry.Retry(ctx, s.cfg.Retry, func(ctx context.Context) (*ports.RouteResult, error) {
		return s.provider.Route(ctx, &ports.RouteRequest{From: from, To: to})
	})
	if err != nil {
		return nil, err
	}
	return &Route{Geometry: res.Shape, DistanceM: res.DistanceM, Duration: res.Duration}, nil
}

func (s *service) MatchLoadTrack(ctx context.Context, loadID uuid.UUID, points domain.LoadLocationTrack) (*domain.LoadTrack, error) {
	pieces := points.Split(s.cfg.SplitParams())

	m := &matching{sent: map[int64]bool{}, matched: map[int64]bool{}}
	for _, piece := range pieces {
		first, last := piece.First(), piece.Last()
		switch piece.Kind {
		case domain.TrackPieceGap:
			if err := m.add(domain.LoadTrackSegmentGap, []geo.Point{first.Point(), last.Point()}, first, last); err != nil {
				return nil, err
			}
		case domain.TrackPieceStop:
			if err := m.add(domain.LoadTrackSegmentStop, []geo.Point{geo.Centroid(pointsOf(piece.Points))}, first, last); err != nil {
				return nil, err
			}
		case domain.TrackPieceMoving:
			for _, window := range s.windows(piece.Points) {
				if err := s.matchWindow(ctx, m, window); err != nil {
					return nil, err
				}
			}
		}
	}

	return domain.NewLoadTrack(loadID, MatcherVersion, points.LastID(), len(m.sent), len(m.matched), m.segments)
}

// matching accumulates the result of one MatchLoadTrack call.
type matching struct {
	segments []*domain.LoadTrackSegment
	// sent and matched are point IDs sent to the engine and placed on a
	// road by it; sets, because neighboring windows share a point.
	sent    map[int64]bool
	matched map[int64]bool
}

func (m *matching) add(kind domain.LoadTrackSegmentKind, geometry []geo.Point, from, to *domain.LoadLocationPoint) error {
	seg, err := domain.NewLoadTrackSegment(kind, geometry, from, to)
	if err != nil {
		return fmt.Errorf("building %s segment: %w", kind, err)
	}
	m.segments = append(m.segments, seg)
	return nil
}

// windows splits a moving stretch into requests under Valhalla's path length
// limit. Neighboring windows share their boundary point, so the line stays
// connected without having to stitch overlapping results.
func (s *service) windows(points domain.LoadLocationTrack) []domain.LoadLocationTrack {
	var out []domain.LoadLocationTrack
	start, distance := 0, 0.0
	for i := 1; i < len(points); i++ {
		distance += geo.DistanceM(points[i-1].Point(), points[i].Point())
		if distance > s.cfg.MaxRequestDistanceM && i-1 > start {
			out = append(out, points[start:i])
			start, distance = i-1, geo.DistanceM(points[i-1].Point(), points[i].Point())
		}
	}
	return append(out, points[start:])
}

// matchWindow matches one request's worth of points. When the engine finds
// no path or the request is over its limits, the window is split in half
// (sharing the middle point) and each half is matched on its own; a piece
// too small to split is kept as a raw line.
func (s *service) matchWindow(ctx context.Context, m *matching, points domain.LoadLocationTrack) error {
	if len(points) < 2 {
		return nil
	}
	for _, p := range points {
		m.sent[p.ID] = true
	}

	res, err := retry.Retry(ctx, s.cfg.Retry, func(ctx context.Context) (*ports.TraceAttributesResult, error) {
		return s.provider.TraceAttributes(ctx, s.traceRequest(points))
	})
	switch {
	case err == nil:
		return s.addTrace(m, points, res)
	case errors.Is(err, inerr.ErrRoutingNoMatch), errors.Is(err, inerr.ErrRoutingLimitExceeded):
		if len(points) <= minSplitPoints {
			return m.add(domain.LoadTrackSegmentRaw, pointsOf(points), points[0], points[len(points)-1])
		}
		mid := len(points) / 2
		if err := s.matchWindow(ctx, m, points[:mid+1]); err != nil {
			return err
		}
		return s.matchWindow(ctx, m, points[mid:])
	case errors.Is(err, inerr.ErrRoutingBadRequest):
		// Our request is malformed: a bug to fix, but the track should
		// still show where the truck went.
		logger.ErrorContext(ctx, "routing engine rejected a matching request", err, "points", len(points))
		return m.add(domain.LoadTrackSegmentRaw, pointsOf(points), points[0], points[len(points)-1])
	default:
		return err
	}
}

func (s *service) traceRequest(points domain.LoadLocationTrack) *ports.TraceAttributesRequest {
	req := &ports.TraceAttributesRequest{
		Points:            make([]ports.TracePoint, len(points)),
		BreakageDistanceM: s.cfg.BreakageSpeedMps * s.cfg.GapThreshold.Seconds() * s.cfg.BreakageFactor,
	}
	for i, p := range points {
		radius := s.cfg.MaxRadiusM
		if p.AccuracyM != nil {
			radius = min(s.cfg.MaxRadiusM, max(s.cfg.MinRadiusM, float64(*p.AccuracyM)*s.cfg.RadiusMultiplier))
		}
		req.Points[i] = ports.TracePoint{Point: p.Point(), Time: p.RecordedAt, RadiusM: radius}
	}
	return req
}

func pointsOf(points domain.LoadLocationTrack) []geo.Point {
	out := make([]geo.Point, len(points))
	for i, p := range points {
		out[i] = p.Point()
	}
	return out
}
