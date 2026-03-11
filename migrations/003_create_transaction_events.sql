-- Migration: Create transaction_events table
-- Purpose: Append-only event log for full audit trail
-- Each state transition is immutably recorded

CREATE TABLE IF NOT EXISTS transaction_events (
    id BIGSERIAL PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    nonce VARCHAR(66) NOT NULL,

    -- Event type indicates the state transition
    event_type VARCHAR(50) NOT NULL,
    -- Event types:
    --   challenge_issued: Hub issued payment challenge
    --   proof_received: Hub received signed payment from agent
    --   relay_submitted: Hub submitted to blockchain
    --   on_chain_confirmed: PaymentSettled event detected
    --   settlement_delivered: Results delivered to agent
    --   payment_expired: Deadline passed
    --   payment_failed: Settlement failed (sig invalid, balance, etc.)

    event_data JSONB,                              -- Flexible data per event type
    occurred_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Metadata
    source VARCHAR(50),                            -- "hub", "blockchain_listener", "admin"
    error_message TEXT
);

-- Append-only constraint: no updates, only inserts
-- Enforced at application layer (no UPDATE or DELETE permissions)

-- Index for efficient queries
CREATE INDEX idx_event_transaction ON transaction_events(transaction_id);
CREATE INDEX idx_event_nonce ON transaction_events(nonce);
CREATE INDEX idx_event_type ON transaction_events(event_type);
CREATE INDEX idx_event_occurred ON transaction_events(occurred_at);
CREATE INDEX idx_event_source ON transaction_events(source);

-- Example event_data payloads:
--
-- challenge_issued:
--   { "nonce": "0x...", "price": "0.001", "deadline": 1234567890 }
--
-- proof_received:
--   { "signature_valid": true, "signature_length": 65 }
--
-- relay_submitted:
--   { "relay_tx_hash": "0x...", "gas_limit": 80000 }
--
-- on_chain_confirmed:
--   { "tx_hash": "0x...", "block_number": 12345678, "log_index": 0 }
--
-- settlement_failed:
--   { "reason": "insufficient_balance", "agent_balance": "0.5", "required": "1.0" }
