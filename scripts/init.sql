-- Payment table
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    merchant_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    method VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    transaction_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

-- Indexes for query optimization
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_merchant_id ON payments(merchant_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX idx_payments_transaction_id ON payments(transaction_id);

-- Composite index for common queries
CREATE INDEX idx_payments_user_status ON payments(user_id, status);

-- Payment metadata table (for extensibility)
CREATE TABLE IF NOT EXISTS payment_metadata (
    id SERIAL PRIMARY KEY,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_metadata_payment_id ON payment_metadata(payment_id);

-- Audit log table
CREATE TABLE IF NOT EXISTS payment_audit_log (
    id SERIAL PRIMARY KEY,
    payment_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    old_status VARCHAR(50),
    new_status VARCHAR(50),
    changed_by VARCHAR(255),
    changed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB
);

CREATE INDEX idx_audit_log_payment_id ON payment_audit_log(payment_id);
CREATE INDEX idx_audit_log_event_type ON payment_audit_log(event_type);
CREATE INDEX idx_audit_log_changed_at ON payment_audit_log(changed_at DESC);
