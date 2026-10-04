CREATE TABLE IF NOT EXISTS emails (
  id          bigserial,
  user_id     uuid NOT NULL,
  type        varchar(64) NOT NULL,
  status      varchar(32) NOT NULL DEFAULT 'pending',
  "from"      varchar(255) NOT NULL,
  "to"        varchar(255) NOT NULL,
  bcc         text[] NOT NULL DEFAULT '{}',
  subject     varchar(255) NOT NULL,
  content     text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS emails_user_id_idx ON emails (user_id);
