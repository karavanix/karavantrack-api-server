CREATE TABLE IF NOT EXISTS company_members (
    id uuid,
    company_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role varchar(64) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT company_members_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id),
    CONSTRAINT company_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS company_members_company_id_user_id_idx ON company_members(company_id, user_id);
