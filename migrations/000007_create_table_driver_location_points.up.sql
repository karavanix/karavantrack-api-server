CREATE TABLE IF NOT EXISTS driver_location_points (
  id           bigserial,
  load_id      uuid NOT NULL,
  driver_id    uuid NOT NULL,
  lat          numeric(11, 8) NOT NULL,
  lng          numeric(11, 8) NOT NULL,
  accuracy_m   real,
  speed_mps    real,
  heading_deg  real,
  recorded_at  timestamptz NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT driver_location_points_load_id_fkey FOREIGN KEY (load_id) REFERENCES loads(id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT driver_location_points_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS driver_location_points_load_id_recorded_at_idx
  ON driver_location_points (load_id, recorded_at DESC) WHERE recorded_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS driver_location_points_driver_id_recorded_at_idx
  ON driver_location_points (driver_id, recorded_at DESC) WHERE recorded_at IS NOT NULL;