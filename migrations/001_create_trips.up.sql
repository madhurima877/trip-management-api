CREATE TABLE IF NOT EXISTS trips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_number VARCHAR(100) NOT NULL UNIQUE,
    source VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    driver_name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT trips_trip_number_not_blank CHECK (length(btrim(trip_number)) > 0),
    CONSTRAINT trips_source_not_blank CHECK (length(btrim(source)) > 0),
    CONSTRAINT trips_destination_not_blank CHECK (length(btrim(destination)) > 0),
    CONSTRAINT trips_driver_name_not_blank CHECK (length(btrim(driver_name)) > 0),
    CONSTRAINT trips_status_valid CHECK (status IN ('CREATED', 'ASSIGNED', 'IN_TRANSIT', 'COMPLETED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS trips_status_created_at_idx ON trips (status, created_at DESC);
CREATE INDEX IF NOT EXISTS trips_created_at_idx ON trips (created_at DESC);