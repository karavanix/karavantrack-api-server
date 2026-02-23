CREATE TABLE IF NOT EXISTS company_carriers (
    company_id uuid NOT NULL,
    carrier_id uuid NOT NULL,
    alias varchar(255) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (company_id, carrier_id),
    CONSTRAINT company_carriers_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id),
    CONSTRAINT company_carriers_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS company_carriers_company_id_created_at_idx
  ON company_carriers (company_id, created_at DESC);