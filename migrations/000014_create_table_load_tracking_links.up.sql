CREATE TABLE IF NOT EXISTS load_tracking_links (
    id uuid,
    load_id uuid NOT NULL,
    token varchar(64) NOT NULL UNIQUE,
    status varchar(16) NOT NULL DEFAULT 'active',
    created_by uuid,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT load_tracking_links_load_id_fkey FOREIGN KEY (load_id) REFERENCES loads(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT load_tracking_links_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS load_tracking_links_load_id_idx ON load_tracking_links(load_id);
