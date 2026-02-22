CREATE TABLE IF NOT EXISTS users (
    id uuid,
    first_name varchar(255) NOT NULL,
    last_name varchar(255) NOT NULL,
    email varchar(255) NOT NULL,
    phone varchar(64) NOT NULL,
    password_hash text NOT NULL,
    status varchar(64) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_idx ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_idx ON users(phone);