
-- +migrate Up
-- +migrate StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE
OR REPLACE FUNCTION payments_update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN NEW .updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TYPE payment_status AS ENUM ('new', 'pending', 'completed', 'failed', 'canceled');

CREATE TABLE IF NOT EXISTS payments (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR DEFAULT NULL,
    currency VARCHAR NOT NULL,
    total_amount BIGINT NOT NULL,
    status payment_status NOT NULL DEFAULT 'new'::payment_status,
    message VARCHAR DEFAULT NULL,
    memo VARCHAR DEFAULT NULL,
    expires_at TIMESTAMP DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP DEFAULT NULL
);
CREATE UNIQUE INDEX payments_external_id ON payments USING BTREE (external_id) WHERE external_id IS NOT NULL;
CREATE TRIGGER update_payments_modtime BEFORE
UPDATE ON payments FOR EACH ROW EXECUTE PROCEDURE payments_update_updated_at_column();

CREATE TABLE IF NOT EXISTS payment_destinations (
    payment_id uuid NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    destination VARCHAR NOT NULL,
    amount BIGINT DEFAULT NULL,
    percentage SMALLINT DEFAULT NULL,
    total_amount BIGINT NOT NULL DEFAULT 0,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    apply_bonus BOOLEAN NOT NULL DEFAULT FALSE,
    max_bonus_amount BIGINT NOT NULL DEFAULT 0,
    max_bonus_percentage SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (payment_id, destination)
);
-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
DROP TRIGGER IF EXISTS update_payments_modtime ON payments;
DROP TABLE IF EXISTS payment_destinations;
DROP TABLE IF EXISTS payments;
DROP FUNCTION IF EXISTS payments_update_updated_at_column();
DROP TYPE IF EXISTS payment_status;
-- +migrate StatementEnd
