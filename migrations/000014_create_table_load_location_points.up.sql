-- Raw GPS points as the phone recorded them. uuid is the tracking library's
-- record id and makes a re-sent batch idempotent; it is NULL for points that
-- don't come from the library (a location attached to a status change).
-- provider is the location-services state Android attaches to a
-- providerchange record (NULL on every other point).
CREATE TABLE IF NOT EXISTS load_location_points (
  id                      bigserial,
  uuid                    uuid,
  load_id                 uuid NOT NULL,
  carrier_id              uuid,
  load_status_history_id  bigint,
  recorded_at             timestamptz NOT NULL,
  lat                     numeric(11, 8) NOT NULL,
  lng                     numeric(11, 8) NOT NULL,
  accuracy_m              real,
  altitude_m              real,
  speed_mps               real,
  heading_deg             real,
  event                   varchar(16),
  is_moving               boolean,
  activity_type           varchar(16),
  activity_confidence     smallint,
  odometer_m              double precision,
  battery_level           real,
  is_charging             boolean,
  is_mock                 boolean,
  provider                jsonb,
  created_at              timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT load_location_points_uuid_key UNIQUE (uuid),
  CONSTRAINT load_location_points_load_id_fkey FOREIGN KEY (load_id) REFERENCES loads(id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT load_location_points_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT load_location_points_load_status_history_id_fkey FOREIGN KEY (load_status_history_id) REFERENCES load_status_histories(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS load_location_points_load_id_recorded_at_idx
  ON load_location_points (load_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS load_location_points_carrier_id_recorded_at_idx
  ON load_location_points (carrier_id, recorded_at DESC);
