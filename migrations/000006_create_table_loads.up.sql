CREATE TABLE IF NOT EXISTS loads (
    id uuid,
    company_id uuid NOT NULL,
    driver_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT loads_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id),
    CONSTRAINT loads_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES drivers(id)
);   