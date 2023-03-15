
-- +migrate Up
-- +migrate StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE
OR REPLACE FUNCTION transactions_update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN NEW .updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TYPE transaction_status AS ENUM ('pending', 'completed', 'failed', 'expired');

CREATE TABLE IF NOT EXISTS transactions ( 
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id uuid NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    reference VARCHAR NOT NULL,
    source_wallet VARCHAR NOT NULL,
    source_mint VARCHAR NOT NULL,
    destination_wallet VARCHAR NOT NULL,
    destination_mint VARCHAR NOT NULL,
    amount BIGINT NOT NULL,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL,
    accrued_bonus_amount BIGINT NOT NULL DEFAULT 0,
    message VARCHAR DEFAULT NULL,
    memo VARCHAR DEFAULT NULL,
    apply_bonus BOOLEAN DEFAULT NULL,
    tx_signature VARCHAR DEFAULT NULL,
    status transaction_status NOT NULL DEFAULT 'pending'::transaction_status,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP DEFAULT NULL
);
CREATE UNIQUE INDEX transactions_reference ON transactions USING BTREE (reference);
CREATE INDEX transactions_payment_source ON transactions USING BTREE (payment_id, source_wallet, source_mint) WHERE status = 'pending'::transaction_status;
CREATE TRIGGER update_transactions_modtime BEFORE
UPDATE ON transactions FOR EACH ROW EXECUTE PROCEDURE transactions_update_updated_at_column();
-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
DROP TRIGGER IF EXISTS update_transactions_modtime ON transactions;
DROP TABLE IF EXISTS transactions;
DROP FUNCTION IF EXISTS transactions_update_updated_at_column();
DROP TYPE IF EXISTS transaction_status;
-- +migrate StatementEnd
