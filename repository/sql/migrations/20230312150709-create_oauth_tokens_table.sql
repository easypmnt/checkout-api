
-- +migrate Up
-- +migrate StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE
OR REPLACE FUNCTION tokens_update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN NEW .updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';
CREATE TABLE IF NOT EXISTS tokens (
	token_type VARCHAR NOT NULL,
	credential VARCHAR NOT NULL,
	access_token_id uuid NOT NULL,
	refresh_token_id uuid NOT NULL,
	access_expires_at TIMESTAMP NOT NULL,
	refresh_expires_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
	PRIMARY KEY (token_type, credential)
);
CREATE UNIQUE INDEX tokens_access_token_id ON tokens USING BTREE (access_token_id);
CREATE UNIQUE INDEX tokens_refresh_token_id ON tokens USING BTREE (refresh_token_id);
CREATE TRIGGER update_tokens_modtime BEFORE
UPDATE ON tokens FOR EACH ROW EXECUTE PROCEDURE tokens_update_updated_at_column();
-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
DROP TRIGGER IF EXISTS update_tokens_modtime ON tokens;
DROP TABLE IF EXISTS tokens;
DROP FUNCTION IF EXISTS tokens_update_updated_at_column();
-- +migrate StatementEnd
