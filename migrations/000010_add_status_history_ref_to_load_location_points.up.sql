ALTER TABLE IF EXISTS load_location_points
    ADD COLUMN IF NOT EXISTS load_status_history_id bigint REFERENCES load_status_histories(id);
