CREATE TABLE IF NOT EXISTS companies (
  id          uuid,
  owner_id    uuid NOT NULL,
  name        varchar(255) NOT NULL,
  status      varchar(64) NOT NULL,
  created_at  timestamptz DEFAULT now(),
  updated_at  timestamptz DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT companies_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);
