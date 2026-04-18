CREATE TABLE IF NOT EXISTS attachments (
    id uuid,
    user_id uuid,
    visibility varchar(64) NOT NULL,
    file_name varchar(255) NOT NULL,
    file_ext varchar(4) NOT NULL,
    file_size bigint NOT NULL,
    file_mime varchar(255) NOT NULL,
    status varchar(64) NOT NULL,
    object_bucket varchar(64) NOT NULL,
    object_key varchar(512) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    PRIMARY KEY(id),
    CONSTRAINT attachments_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS attachments_user_id_idx ON attachments(user_id);