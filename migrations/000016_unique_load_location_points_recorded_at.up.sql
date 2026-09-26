-- A phone retries a whole batch when the response is lost, so the same fix
-- can arrive twice. (load_id, recorded_at) identifies a fix: the phone sets
-- recorded_at from the GPS timestamp, and one load has one phone.
--
-- Keep the first copy of any duplicate already stored, otherwise the unique
-- index can't be built.
DELETE FROM load_location_points p
USING load_location_points dup
WHERE p.load_id = dup.load_id
  AND p.recorded_at = dup.recorded_at
  AND p.id > dup.id;

-- Same columns and order as the index it replaces, so the "latest points of
-- a load" queries keep using it.
DROP INDEX IF EXISTS load_location_points_load_id_recorded_at_idx;

CREATE UNIQUE INDEX IF NOT EXISTS load_location_points_load_id_recorded_at_key
  ON load_location_points (load_id, recorded_at DESC);
