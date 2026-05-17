CREATE TABLE emails (
    id         BIGSERIAL PRIMARY KEY,
    user_id    UUID         NOT NULL REFERENCES users(id),
    type       VARCHAR(64)  NOT NULL,
    status     VARCHAR(32)  NOT NULL DEFAULT 'pending',
    "from"     VARCHAR(255) NOT NULL,
    "to"       VARCHAR(255) NOT NULL,
    bcc        TEXT[]       NOT NULL DEFAULT '{}',
    subject    VARCHAR(255) NOT NULL,
    content    TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
