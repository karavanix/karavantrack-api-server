package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/geo"
	"github.com/uptrace/bun"
)

type LoadTracks struct {
	bun.BaseModel `bun:"table:load_tracks,alias:lt"`

	LoadID            string     `bun:"load_id,pk,type:uuid"`
	DistanceM         float64    `bun:"distance_m"`
	LastPointID       int64      `bun:"last_point_id"`
	MatchedUntil      *time.Time `bun:"matched_until"`
	MatcherVersion    string     `bun:"matcher_version"`
	PointCount        int        `bun:"point_count"`
	MatchedPointCount int        `bun:"matched_point_count"`
	CreatedAt         time.Time  `bun:"created_at"`
	UpdatedAt         time.Time  `bun:"updated_at"`
}

type LoadTrackSegments struct {
	bun.BaseModel `bun:"table:load_track_segments,alias:lts"`

	ID          int64     `bun:"id,pk,autoincrement"`
	LoadID      string    `bun:"load_id,type:uuid"`
	Seq         int       `bun:"seq"`
	Kind        string    `bun:"kind"`
	StartedAt   time.Time `bun:"started_at"`
	EndedAt     time.Time `bun:"ended_at"`
	Geometry    string    `bun:"geometry"`
	DistanceM   float64   `bun:"distance_m"`
	FromPointID int64     `bun:"from_point_id"`
	ToPointID   int64     `bun:"to_point_id"`
}

type loadTracksRepo struct {
	db bun.IDB
}

func NewLoadTracksRepo(db bun.IDB) domain.LoadTrackRepository {
	return &loadTracksRepo{db: db}
}

// Save upserts the track row, then replaces its segments. The upsert only
// applies when the stored track was built from the same or older points
// (last_point_id), so a slow match can't overwrite the result of a newer
// one that finished first; <= rather than < lets a forced rematch with new
// matcher settings replace a track built from the same points.
func (r *loadTracksRepo) Save(ctx context.Context, track *domain.LoadTrack) error {
	db := postgres.FromContext(ctx, r.db)
	model := r.toModel(track)

	res, err := db.NewInsert().Model(model).
		On("CONFLICT (load_id) DO UPDATE").
		Set("distance_m = EXCLUDED.distance_m").
		Set("last_point_id = EXCLUDED.last_point_id").
		Set("matched_until = EXCLUDED.matched_until").
		Set("matcher_version = EXCLUDED.matcher_version").
		Set("point_count = EXCLUDED.point_count").
		Set("matched_point_count = EXCLUDED.matched_point_count").
		Set("updated_at = EXCLUDED.updated_at").
		Where("lt.last_point_id <= EXCLUDED.last_point_id").
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, model)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return inerr.NewErrNoChanges("load track")
	}

	_, err = db.NewDelete().Model((*LoadTrackSegments)(nil)).
		Where("load_id = ?", model.LoadID).
		Exec(ctx)
	if err != nil {
		return postgres.Error(err, &LoadTrackSegments{})
	}

	if len(track.Segments) == 0 {
		return nil
	}
	segments := make([]*LoadTrackSegments, len(track.Segments))
	for i, s := range track.Segments {
		segments[i] = r.segmentToModel(model.LoadID, s)
	}
	_, err = db.NewInsert().Model(&segments).Returning("NULL").Exec(ctx)
	if err != nil {
		return postgres.Error(err, &LoadTrackSegments{})
	}
	return nil
}

func (r *loadTracksRepo) FindByLoadID(ctx context.Context, loadID uuid.UUID) (*domain.LoadTrack, error) {
	db := postgres.FromContext(ctx, r.db)

	var model LoadTracks
	err := db.NewSelect().Model(&model).Where("load_id = ?", loadID.String()).Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &model)
	}

	var segments []LoadTrackSegments
	err = db.NewSelect().Model(&segments).
		Where("load_id = ?", loadID.String()).
		Order("seq ASC").
		Scan(ctx)
	if err != nil {
		return nil, postgres.Error(err, &LoadTrackSegments{})
	}

	track := r.toDomain(&model)
	track.Segments = make([]*domain.LoadTrackSegment, len(segments))
	for i := range segments {
		seg, err := r.segmentToDomain(&segments[i])
		if err != nil {
			return nil, err
		}
		track.Segments[i] = seg
	}
	return track, nil
}

func (r *loadTracksRepo) toModel(e *domain.LoadTrack) *LoadTracks {
	return &LoadTracks{
		LoadID:            e.LoadID.String(),
		DistanceM:         e.DistanceM,
		LastPointID:       e.LastPointID,
		MatchedUntil:      e.MatchedUntil,
		MatcherVersion:    e.MatcherVersion,
		PointCount:        e.PointCount,
		MatchedPointCount: e.MatchedPointCount,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

func (r *loadTracksRepo) toDomain(m *LoadTracks) *domain.LoadTrack {
	loadID, _ := uuid.Parse(m.LoadID)
	return &domain.LoadTrack{
		LoadID:            loadID,
		DistanceM:         m.DistanceM,
		LastPointID:       m.LastPointID,
		MatchedUntil:      m.MatchedUntil,
		MatcherVersion:    m.MatcherVersion,
		PointCount:        m.PointCount,
		MatchedPointCount: m.MatchedPointCount,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func (r *loadTracksRepo) segmentToModel(loadID string, s *domain.LoadTrackSegment) *LoadTrackSegments {
	return &LoadTrackSegments{
		LoadID:      loadID,
		Seq:         s.Seq,
		Kind:        string(s.Kind),
		StartedAt:   s.StartedAt,
		EndedAt:     s.EndedAt,
		Geometry:    geo.EncodePolyline6(s.Geometry),
		DistanceM:   s.DistanceM,
		FromPointID: s.FromPointID,
		ToPointID:   s.ToPointID,
	}
}

func (r *loadTracksRepo) segmentToDomain(m *LoadTrackSegments) (*domain.LoadTrackSegment, error) {
	geometry, err := geo.DecodePolyline6(m.Geometry)
	if err != nil {
		return nil, fmt.Errorf("load track segment %d: %w", m.ID, err)
	}
	return &domain.LoadTrackSegment{
		Seq:         m.Seq,
		Kind:        domain.LoadTrackSegmentKind(m.Kind),
		StartedAt:   m.StartedAt,
		EndedAt:     m.EndedAt,
		Geometry:    geometry,
		DistanceM:   m.DistanceM,
		FromPointID: m.FromPointID,
		ToPointID:   m.ToPointID,
	}, nil
}
