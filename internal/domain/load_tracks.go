package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
)

type LoadTrackSegmentKind string

const (
	// LoadTrackSegmentMatched follows the roads the map matcher placed the
	// points on. Drawn as a solid line.
	LoadTrackSegmentMatched LoadTrackSegmentKind = "matched"
	// LoadTrackSegmentRaw is where the matcher couldn't find a road path:
	// the raw GPS polyline, drawn as a thin line.
	LoadTrackSegmentRaw LoadTrackSegmentKind = "raw"
	// LoadTrackSegmentGap is silence longer than the gap threshold: a
	// straight dashed line between the last point before and the first
	// after, labelled "no data for N min".
	LoadTrackSegmentGap LoadTrackSegmentKind = "gap"
	// LoadTrackSegmentStop is a stop: a single point (the stop's center),
	// drawn as a marker labelled "stopped for N min".
	LoadTrackSegmentStop LoadTrackSegmentKind = "stop"
)

// LoadTrackSegment is one chronological part of a load's track, built from
// the raw points FromPointID..ToPointID.
type LoadTrackSegment struct {
	Seq         int
	Kind        LoadTrackSegmentKind
	StartedAt   time.Time
	EndedAt     time.Time
	Geometry    []geo.Point
	DistanceM   float64
	FromPointID int64
	ToPointID   int64
}

func NewLoadTrackSegment(kind LoadTrackSegmentKind, geometry []geo.Point, from, to *LoadLocationPoint) (*LoadTrackSegment, error) {
	if from == nil || to == nil {
		return nil, errors.New("segment boundary points are required")
	}
	if to.RecordedAt.Before(from.RecordedAt) {
		return nil, errors.New("segment ends before it starts")
	}

	var distance float64
	switch kind {
	case LoadTrackSegmentMatched, LoadTrackSegmentRaw:
		if len(geometry) < 2 {
			return nil, fmt.Errorf("%s segment needs at least 2 points, got %d", kind, len(geometry))
		}
		distance = geo.LengthM(geometry)
	case LoadTrackSegmentGap:
		if len(geometry) != 2 {
			return nil, fmt.Errorf("gap segment needs exactly 2 points, got %d", len(geometry))
		}
		distance = geo.DistanceM(geometry[0], geometry[1])
	case LoadTrackSegmentStop:
		if len(geometry) != 1 {
			return nil, fmt.Errorf("stop segment needs exactly 1 point, got %d", len(geometry))
		}
	default:
		return nil, fmt.Errorf("unknown segment kind %q", kind)
	}

	return &LoadTrackSegment{
		Kind:        kind,
		StartedAt:   from.RecordedAt,
		EndedAt:     to.RecordedAt,
		Geometry:    geometry,
		DistanceM:   distance,
		FromPointID: from.ID,
		ToPointID:   to.ID,
	}, nil
}

// LoadTrack is a load's route as driven: its raw GPS points matched to roads
// and cut into segments. It's derived data — rebuilt from the raw points on
// every match, never edited — so the raw points stay the source of truth.
type LoadTrack struct {
	LoadID uuid.UUID
	// DistanceM is the distance driven: matched and raw segments. Gaps are
	// left out, since we don't know how the truck got across them.
	DistanceM float64
	// LastPointID is the highest raw point ID the track was built from; a
	// stored point with a higher ID means the track is out of date.
	LastPointID int64
	// MatchedUntil is when the last point covered by the track was
	// recorded. Anything newer is drawn from raw points up to the live
	// marker. Nil for a track with no segments.
	MatchedUntil *time.Time
	// MatcherVersion names the matcher and its settings, so tracks built by
	// an older version can be found and rematched.
	MatcherVersion string
	// PointCount is how many points went into the matcher (after cleaning);
	// MatchedPointCount is how many of them it placed on a road.
	PointCount        int
	MatchedPointCount int
	CreatedAt         time.Time
	UpdatedAt         time.Time

	Segments []*LoadTrackSegment
}

func NewLoadTrack(
	loadID uuid.UUID,
	matcherVersion string,
	lastPointID int64,
	pointCount int,
	matchedPointCount int,
	segments []*LoadTrackSegment,
) (*LoadTrack, error) {
	if loadID == uuid.Nil {
		return nil, errors.New("loadID is required")
	}
	if matcherVersion == "" {
		return nil, errors.New("matcherVersion is required")
	}
	if matchedPointCount < 0 || matchedPointCount > pointCount {
		return nil, fmt.Errorf("matched point count %d out of range 0..%d", matchedPointCount, pointCount)
	}

	track := &LoadTrack{
		LoadID:            loadID,
		LastPointID:       lastPointID,
		MatcherVersion:    matcherVersion,
		PointCount:        pointCount,
		MatchedPointCount: matchedPointCount,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Segments:          segments,
	}

	for i, seg := range segments {
		if seg == nil {
			return nil, fmt.Errorf("segment %d is nil", i)
		}
		if i > 0 && seg.StartedAt.Before(segments[i-1].EndedAt) {
			return nil, fmt.Errorf("segment %d starts before segment %d ends", i, i-1)
		}
		if seg.ToPointID > lastPointID {
			return nil, fmt.Errorf("segment %d uses point %d, newer than the track's last point %d", i, seg.ToPointID, lastPointID)
		}
		seg.Seq = i
		if seg.Kind == LoadTrackSegmentMatched || seg.Kind == LoadTrackSegmentRaw {
			track.DistanceM += seg.DistanceM
		}
	}
	if n := len(segments); n > 0 {
		until := segments[n-1].EndedAt
		track.MatchedUntil = &until
	}

	return track, nil
}

type LoadTrackRepository interface {
	// Save replaces the load's stored track with this one. It must run in a
	// transaction. It returns inerr.ErrNoChanges, leaving the stored track
	// as is, when that track was built from newer points than this one.
	Save(ctx context.Context, track *LoadTrack) error
	FindByLoadID(ctx context.Context, loadID uuid.UUID) (*LoadTrack, error)
}
