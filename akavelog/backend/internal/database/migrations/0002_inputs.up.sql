-- Enable UUIDs (if not already done in an earlier migration)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Input state enum
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'input_state') THEN
        CREATE TYPE input_state AS ENUM ('RUNNING', 'STOPPED', 'PAUSED');
    END IF;
END$$;

-- Main inputs table
CREATE TABLE IF NOT EXISTS inputs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    configuration JSONB NOT NULL,
    global BOOLEAN NOT NULL DEFAULT FALSE,
    node_id TEXT,
    creator_user_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    desired_state input_state NOT NULL DEFAULT 'RUNNING'
);

CREATE INDEX IF NOT EXISTS idx_inputs_type ON inputs(type);
CREATE INDEX IF NOT EXISTS idx_inputs_node_id ON inputs(node_id);