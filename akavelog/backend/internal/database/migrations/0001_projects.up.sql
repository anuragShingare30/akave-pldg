-- @dev Up migration for projects table
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL UNIQUE,
    owner_email TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);