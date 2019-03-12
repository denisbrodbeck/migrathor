/**
* Name: create_users
* Date: 2019-03-05T17:36:12Z
*/

CREATE TABLE users (
    id serial PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE FUNCTION timestamp_created()
    RETURNS trigger AS $body$
    BEGIN
        NEW.created_at := current_timestamp;
        NEW.updated_at := current_timestamp;
        RETURN NEW;
    END;
$body$ LANGUAGE plpgsql;

CREATE TRIGGER timestamp_created BEFORE INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION timestamp_created();
COMMENT ON TRIGGER timestamp_created ON users IS 'Update fields created_at and updated_at';

CREATE FUNCTION timestamp_updated()
    RETURNS trigger AS $body$
    BEGIN
        NEW.updated_at := current_timestamp;
        RETURN NEW;
    END;
$body$ LANGUAGE plpgsql;

CREATE TRIGGER timestamp_updated BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION timestamp_updated();
COMMENT ON TRIGGER timestamp_updated ON users IS 'Update field updated_at';
