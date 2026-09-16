DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM subscribers
        WHERE CHAR_LENGTH(msisdn) > 15
    ) THEN
        RAISE EXCEPTION 'cannot rollback migration 000004: subscribers.msisdn contains values longer than 15 characters';
    END IF;

    ALTER TABLE subscribers
    ALTER COLUMN msisdn TYPE VARCHAR(15);
END
$$;
