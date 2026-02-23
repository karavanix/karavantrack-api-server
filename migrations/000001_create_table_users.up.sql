CREATE TABLE IF NOT EXISTS users (
    id uuid,
    first_name varchar(255),
    last_name varchar(255),
    email varchar(255),
    phone varchar(64),
    password_hash text NOT NULL,
    status varchar(64) NOT NULL,
    role varchar(64) NOT NULL CHECK (role IN ('shipper', 'carrier')),
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_idx ON users(phone);
CREATE INDEX IF NOT EXISTS users_role_idx ON users(role);