CREATE TABLE IF NOT EXISTS loads (
    id uuid,
    company_id uuid,
    member_id uuid,
    driver_id uuid,
    reference_id varchar(255),
    title varchar(255),
    description text,
    status varchar(255) NOT NULL,
    pickup_address text,
    pickup_lat numeric(11, 8) NOT NULL,
    pickup_lng numeric(11, 8) NOT NULL,
    pickup_address_id text,
    pickup_at timestamp with time zone,
    dropoff_address text,
    dropoff_lat numeric(11, 8) NOT NULL,
    dropoff_lng numeric(11, 8) NOT NULL,
    dropoff_address_id text,
    dropoff_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT loads_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT loads_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT loads_member_id_fkey FOREIGN KEY (member_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS loads_company_id_status_created_at_idx ON loads(company_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS loads_driver_id_status_created_at_idx ON loads(driver_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS loads_company_id_reference_id_idx ON loads(company_id, reference_id);
