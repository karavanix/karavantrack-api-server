CREATE TABLE IF NOT EXISTS companies (
    id uuid,
    name varchar(255) NOT NULL,
    status varchar(64) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id)
);