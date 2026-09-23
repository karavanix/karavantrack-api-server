CREATE TABLE leads (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    company    VARCHAR(255) NOT NULL,
    phone      VARCHAR(64)  NOT NULL,
    fleet      VARCHAR(64)  NOT NULL,
    user_agent VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
