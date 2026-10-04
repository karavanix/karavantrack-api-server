CREATE TABLE IF NOT EXISTS users (
  id             uuid,
  first_name     varchar(255),
  last_name      varchar(255),
  email          varchar(255),
  phone          varchar(64),
  password_hash  text,
  role           varchar(64) NOT NULL,
  status         varchar(64) NOT NULL,
  created_at     timestamptz DEFAULT now(),
  updated_at     timestamptz DEFAULT now(),
  PRIMARY KEY (id),
  CONSTRAINT users_role_check CHECK (role IN ('shipper', 'carrier'))
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users (email);
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_key ON users (phone);
