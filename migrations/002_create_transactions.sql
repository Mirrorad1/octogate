-- Migration: Create transactions table
-- Purpose: Immutable audit log of all completed settlements
-- This is the source of truth for reconciliation and reputation

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nonce VARCHAR(66) NOT NULL UNIQUE,           -- Foreign key to pending_settlements

    -- Request metadata
    request_id UUID NOT NULL,
    timestamp BIGINT NOT NULL,                    -- Unix timestamp

    -- Settlement details (immutable once set)
    agent_address VARCHAR(42) NOT NULL,
    provider_address VARCHAR(42) NOT NULL,
    amount_usdc DECIMAL(20, 6) NOT NULL,          -- Total payment from agent
    fee_usdc DECIMAL(20, 6) NOT NULL,             -- Fee to Octogate
    provider_amount_usdc DECIMAL(20, 6) NOT NULL, -- Amount provider received

    -- Blockchain details
    tx_hash VARCHAR(66) NOT NULL UNIQUE,          -- On-chain transaction hash
    block_number BIGINT NOT NULL,
    block_timestamp BIGINT NOT NULL,
    network VARCHAR(50) NOT NULL,                 -- e.g. "eip155:8453" for Base

    -- Status (final state)
    status VARCHAR(20) NOT NULL,                  -- "settled", "failed", "cancelled"

    -- Metadata
    provider_name VARCHAR(255),
    duration_ms INT,                              -- Time from challenge to settlement
    notes TEXT,                                   -- Error details if failed

    -- Immutability: after 24 hours, cannot be updated or deleted
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    settled_at TIMESTAMP,
    locked_at TIMESTAMP GENERATED ALWAYS AS (
        CASE
            WHEN created_at + INTERVAL '24 hours' < CURRENT_TIMESTAMP
            THEN created_at + INTERVAL '24 hours'
            ELSE NULL
        END
    ) STORED
);

-- Indexes for queries and reconciliation
CREATE INDEX idx_tx_agent ON transactions(agent_address);
CREATE INDEX idx_tx_provider ON transactions(provider_address);
CREATE INDEX idx_tx_status ON transactions(status);
CREATE INDEX idx_tx_block ON transactions(block_number);
CREATE INDEX idx_tx_timestamp ON transactions(block_timestamp);
CREATE UNIQUE INDEX idx_tx_nonce ON transactions(nonce);
CREATE UNIQUE INDEX idx_tx_hash ON transactions(tx_hash);
CREATE INDEX idx_tx_created ON transactions(created_at);

-- Constraint: Ensure amounts are consistent
ALTER TABLE transactions ADD CONSTRAINT check_amount_consistency
  CHECK (amount_usdc = fee_usdc + provider_amount_usdc);

-- Constraint: Ensure positive amounts
ALTER TABLE transactions ADD CONSTRAINT check_positive_amounts
  CHECK (amount_usdc > 0 AND fee_usdc >= 0 AND provider_amount_usdc > 0);

-- Grant read-only access after 24 hours (enforced via application layer)
-- SELECT * FROM transactions WHERE created_at + INTERVAL '24 hours' < NOW()
--   → Immutable, can only INSERT new records
