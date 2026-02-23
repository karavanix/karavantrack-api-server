CREATE TABLE IF NOT EXISTS company_members (
    company_id uuid NOT NULL,
    member_id uuid NOT NULL,
    alias varchar(255) NOT NULL,
    role varchar(64) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW(),
    PRIMARY KEY (company_id, member_id),
    CONSTRAINT company_members_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT company_members_member_id_fkey FOREIGN KEY (member_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);