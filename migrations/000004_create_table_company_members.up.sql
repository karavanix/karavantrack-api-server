CREATE TABLE IF NOT EXISTS company_members (
    company_id uuid NOT NULL,
    user_id uuid NOT NULL,
    alias varchar(255) NOT NULL,
    role varchar(64) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (company_id, user_id),
    CONSTRAINT company_members_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT company_members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);