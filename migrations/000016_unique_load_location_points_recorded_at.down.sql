DROP INDEX IF EXISTS load_location_points_load_id_recorded_at_key;

CREATE INDEX IF NOT EXISTS load_location_points_load_id_recorded_at_idx
  ON load_location_points (load_id, recorded_at DESC) WHERE recorded_at IS NOT NULL;
