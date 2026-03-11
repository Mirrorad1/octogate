-- Migration: Create reputation and providers tables
-- Purpose: Track scores derived from transaction history (informational only)

CREATE TABLE IF NOT EXISTS reputation (
    entity_address VARCHAR(42) PRIMARY KEY,
    entity_type VARCHAR(20) NOT NULL,            -- "agent" or "provider"

    -- Transaction history
    transaction_count INT DEFAULT 0,
    successful_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,

    -- Performance metrics
    avg_settlement_time_ms INT,                   -- Average time from challenge to settlement
    uptime_percent DECIMAL(5, 2),                 -- For providers: % of time API is reachable
    success_rate DECIMAL(5, 2),                   -- % of successful transactions

    -- Scoring (derived, non-monetary)
    reputation_score INT,
    -- Score calculation:
    -- base = 100
    -- + (successful_count * 0.5) up to +40
    -- - (failed_count * 2) up to -30
    -- + (uptime_percent - 90) up to +20 for providers
    -- = 50 to 150 scale

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_reputation_entity_type ON reputation(entity_type);
CREATE INDEX idx_reputation_score ON reputation(reputation_score);

-- ============================================================

CREATE TABLE IF NOT EXISTS providers (
    provider_address VARCHAR(42) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    endpoint_url VARCHAR(2048),
    description TEXT,

    -- Payment configuration (set by provider)
    price_usdc DECIMAL(20, 6) NOT NULL DEFAULT 0,
    currency VARCHAR(10) DEFAULT 'USDC',
    network VARCHAR(50) NOT NULL DEFAULT 'eip155:8453',  -- Base mainnet default

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    -- Statuses:
    --   active: Accepting payments
    --   paused: Temporarily disabled
    --   deprecated: No longer in use
    --   blacklisted: Blocked due to policy violation

    -- Compliance
    registered_entity VARCHAR(255),                -- Company/individual operating this provider
    contact_email VARCHAR(255),
    payment_wallet VARCHAR(42),                    -- Where payments are sent

    -- Metadata
    reputation_score INT,
    uptime_percent DECIMAL(5, 2),
    transaction_count INT DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_health_check TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_provider_status ON providers(status);
CREATE INDEX idx_provider_reputation ON providers(reputation_score);
CREATE INDEX idx_provider_created ON providers(created_at);

-- ============================================================

-- Constraint: reputation is non-monetary and non-transferable
-- This is documented in the contract terms but enforced at the application layer
-- (no endpoints to trade, sell, or pledge reputation)

-- Derived view: Agent reputation summary
CREATE OR REPLACE VIEW agent_reputation_summary AS
SELECT
    entity_address,
    transaction_count,
    successful_count,
    failed_count,
    CASE
        WHEN transaction_count = 0 THEN NULL
        ELSE ROUND((successful_count::NUMERIC / transaction_count) * 100, 2)
    END as success_rate,
    reputation_score,
    last_updated
FROM reputation
WHERE entity_type = 'agent';

-- Derived view: Provider reputation summary
CREATE OR REPLACE VIEW provider_reputation_summary AS
SELECT
    entity_address,
    transaction_count,
    successful_count,
    failed_count,
    ROUND((successful_count::NUMERIC / transaction_count) * 100, 2) as success_rate,
    uptime_percent,
    reputation_score,
    last_updated
FROM reputation
WHERE entity_type = 'provider';
