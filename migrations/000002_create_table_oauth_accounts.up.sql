CREATE TABLE IF NOT EXISTS oauth_accounts (
  id                   uuid,
  user_id              uuid NOT NULL,
  provider             varchar(64) NOT NULL,
  provider_account_id  text NOT NULL,
  created_at           timestamptz DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT oauth_accounts_provider_provider_account_id_key UNIQUE (provider, provider_account_id),
  CONSTRAINT oauth_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS oauth_accounts_user_id_idx ON oauth_accounts (user_id);
