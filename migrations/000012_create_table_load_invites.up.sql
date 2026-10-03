CREATE TABLE IF NOT EXISTS load_invites (
  id           uuid,
  load_id      uuid NOT NULL,
  token        varchar(64) NOT NULL,
  status       varchar(16) NOT NULL DEFAULT 'pending',
  created_by   uuid,
  accepted_by  uuid,
  accepted_at  timestamptz,
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT load_invites_token_key UNIQUE (token),
  CONSTRAINT load_invites_load_id_fkey FOREIGN KEY (load_id) REFERENCES loads(id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT load_invites_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT load_invites_accepted_by_fkey FOREIGN KEY (accepted_by) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS load_invites_load_id_idx ON load_invites (load_id);
