CREATE TABLE IF NOT EXISTS leads (
  id          bigserial,
  name        varchar(255) NOT NULL,
  company     varchar(255) NOT NULL,
  phone       varchar(64) NOT NULL,
  fleet       varchar(64) NOT NULL,
  user_agent  varchar(255) NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id)
);
