-- Revert: remove role column from users
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- Recreate drivers table
CREATE TABLE IF NOT EXISTS drivers (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS drivers_user_id_idx ON drivers(user_id);

-- Revert company_drivers FK to reference drivers
ALTER TABLE company_drivers DROP CONSTRAINT IF EXISTS company_drivers_driver_id_fkey;
ALTER TABLE company_drivers ADD CONSTRAINT company_drivers_driver_id_fkey
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE;
