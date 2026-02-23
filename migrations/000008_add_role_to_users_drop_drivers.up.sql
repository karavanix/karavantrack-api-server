-- Add role column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS role varchar(64) NOT NULL DEFAULT 'cargo_owner';

-- Drop drivers table (role is now on users)
DROP TABLE IF EXISTS drivers CASCADE;

-- Update company_drivers FK to reference users instead of drivers
ALTER TABLE company_drivers DROP CONSTRAINT IF EXISTS company_drivers_driver_id_fkey;
ALTER TABLE company_drivers ADD CONSTRAINT company_drivers_driver_id_fkey
    FOREIGN KEY (driver_id) REFERENCES users(id) ON DELETE CASCADE;
