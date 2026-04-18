CREATE TABLE IF NOT EXISTS load_status_histories (
    id          bigserial,
    load_id     uuid NOT NULL,
    user_id     uuid,
    from_status varchar(64) NOT NULL,
    to_status   varchar(64) NOT NULL,
    note        text NOT NULL DEFAULT '',
    created_at  timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT load_status_histories_load_id_fkey  FOREIGN KEY (load_id)  REFERENCES loads(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT load_status_histories_user_id_fkey  FOREIGN KEY (user_id)  REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS load_status_histories_load_id_idx ON load_status_histories(load_id);

CREATE TABLE IF NOT EXISTS load_status_history_attachments (
    id            bigserial,
    history_id    bigint NOT NULL,
    attachment_id uuid NOT NULL,
    created_at    timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT load_status_history_attachments_history_id_fkey    FOREIGN KEY (history_id)    REFERENCES load_status_histories(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT load_status_history_attachments_attachment_id_fkey FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS load_status_history_attachments_history_id_idx ON load_status_history_attachments(history_id);
