DROP TABLE IF EXISTS inputs;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'input_state') THEN
        DROP TYPE input_state;
    END IF;
END$$;