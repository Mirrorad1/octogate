-- Migration: Create pending_settlements table
-- Purpose: Track in-flight payments waiting for blockchain confirmation
-- This table is the heart of the settlement state machine

CREATE TABLE IF NOT EXISTS pending_settlements (
    nonce VARCHAR(66) PRIMARY KEY,
    request_id UUID NOT NULL,

    -- Payment details
    agent_address VARCHAR(42) NOT NULL,           -- 0x... (indexed for queries)
    provider_address VARCHAR(42) NOT NULL,        -- 0x... (indexed for queries)
    expected_amount_usdc DECIMAL(20, 6) NOT NULL, -- e.g., 1.000000 for 1 USDC

    -- State machine tracking
    status VARCHAR(20) NOT NULL DEFAULT 'awaiting_proof',
    -- Statuses:
    --   awaiting_proof: Challenge issued, waiting for agent signature
    --   relay_submitted: Payment proof sent to blockchain
    --   confirmed: PaymentSettled event detected on-chain
    --   expired: Deadline passed, no payment received
    --   failed: Error during settlement (invalid sig, insufficient funds, etc.)

    -- Timestamps
    challenge_issued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    proof_received_at TIMESTAMP,
    relay_tx_hash VARCHAR(66),                    -- Transaction hash if Hub relayed
    on_chain_tx_hash VARCHAR(66),                 -- Transaction hash from blockchain event
    confirmed_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,

    -- Error tracking
    error_reason TEXT,

    -- Metadata
    provider_name VARCHAR(255),                   -- Human-readable for logs
    user_agent VARCHAR(512),                      -- HTTP user-agent for debugging

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Indexes for efficient queries
CREATE INDEX idx_pending_agent ON pending_settlements(agent_address);
CREATE INDEX idx_pending_provider ON pending_settlements(provider_address);
CREATE INDEX idx_pending_status ON pending_settlements(status);
CREATE INDEX idx_pending_expires_at ON pending_settlements(expires_at);
CREATE INDEX idx_pending_request_id ON pending_settlements(request_id);
CREATE UNIQUE INDEX idx_pending_nonce ON pending_settlements(nonce);

-- Constraint: nonce is immutable once created
ALTER TABLE pending_settlements ADD CONSTRAINT check_nonce_immutable
  CHECK (LENGTH(nonce) = 66);
