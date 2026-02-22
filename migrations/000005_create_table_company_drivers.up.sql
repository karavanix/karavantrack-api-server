CREATE TABLE IF NOT EXISTS company_drivers (
    company_id uuid NOT NULL,
    driver_id uuid NOT NULL,
    alias varchar(255) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (company_id, driver_id),
    CONSTRAINT company_drivers_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id),
    CONSTRAINT company_drivers_driver_id_fkey FOREIGN KEY (driver_id) REFERENCES drivers(id)
);