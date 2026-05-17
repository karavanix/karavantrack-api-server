CREATE TABLE IF NOT EXISTS oauth_accounts (
    id                  uuid         NOT NULL,
    user_id             uuid         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider            varchar(64)  NOT NULL,
    provider_account_id text         NOT NULL,
    created_at          timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    UNIQUE (provider, provider_account_id)
);

CREATE INDEX IF NOT EXISTS oauth_accounts_user_id_idx ON oauth_accounts(user_id);
