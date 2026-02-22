CREATE TABLE IF NOT EXISTS drivers (
    id uuid,
    user_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT drivers_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS drivers_user_id_idx ON drivers(user_id);
