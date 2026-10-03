-- A load's route as driven: raw GPS points matched to roads, rebuilt from
-- load_location_points on every match. The raw points are never modified.
CREATE TABLE IF NOT EXISTS load_tracks (
  load_id              uuid NOT NULL,
  distance_m           real NOT NULL DEFAULT 0,
  last_point_id        bigint NOT NULL,
  matched_until        timestamptz,
  matcher_version      varchar(64) NOT NULL,
  point_count          integer NOT NULL DEFAULT 0,
  matched_point_count  integer NOT NULL DEFAULT 0,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (load_id),
  CONSTRAINT load_tracks_load_id_fkey FOREIGN KEY (load_id) REFERENCES loads(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS load_track_segments (
  id             bigserial,
  load_id        uuid NOT NULL,
  seq            integer NOT NULL,
  kind           varchar(16) NOT NULL,
  started_at     timestamptz NOT NULL,
  ended_at       timestamptz NOT NULL,
  geometry       text NOT NULL,
  distance_m     real NOT NULL DEFAULT 0,
  from_point_id  bigint NOT NULL,
  to_point_id    bigint NOT NULL,
  PRIMARY KEY (id),
  CONSTRAINT load_track_segments_load_id_seq_key UNIQUE (load_id, seq),
  CONSTRAINT load_track_segments_load_id_fkey FOREIGN KEY (load_id) REFERENCES load_tracks(load_id) ON DELETE CASCADE ON UPDATE CASCADE
);
