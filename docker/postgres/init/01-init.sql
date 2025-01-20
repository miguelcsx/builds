-- docker/postgres/init/01-init.sql

-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Set up proper permissions
ALTER USER builds WITH SUPERUSER;

-- Create schema
CREATE SCHEMA IF NOT EXISTS builds;

-- Set search path
ALTER DATABASE builds SET search_path TO builds,public;

-- Create necessary indices and functions
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';
