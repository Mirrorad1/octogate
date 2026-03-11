package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Client wraps database operations for the Hub
type Client struct {
	db *sql.DB
}

// New creates a new database client from environment variables
func New() (*Client, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &Client{db: db}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// DB returns the underlying sql.DB for direct queries
func (c *Client) DB() *sql.DB {
	return c.db
}

// ============ Settlement Queries ============

// RecordPendingSettlement stores a payment challenge in the database
func (c *Client) RecordPendingSettlement(
	ctx context.Context,
	nonce, requestID, agentAddr, providerAddr string,
	amountUsdc float64,
	expiresAt int64,
) error {
	const query = `
		INSERT INTO pending_settlements
		(nonce, request_id, agent_address, provider_address, expected_amount_usdc, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, to_timestamp($6), 'awaiting_proof')
	`

	_, err := c.db.ExecContext(ctx, query, nonce, requestID, agentAddr, providerAddr, amountUsdc, expiresAt)
	return err
}

// UpdateSettlementConfirmed marks a settlement as confirmed on-chain
func (c *Client) UpdateSettlementConfirmed(ctx context.Context, nonce, txHash string) error {
	const query = `
		UPDATE pending_settlements
		SET status = 'confirmed',
			on_chain_tx_hash = $1,
			confirmed_at = NOW(),
			updated_at = NOW()
		WHERE nonce = $2
	`

	result, err := c.db.ExecContext(ctx, query, txHash, nonce)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("settlement not found: %s", nonce)
	}

	return nil
}

// GetPendingSettlement retrieves a pending settlement by nonce
func (c *Client) GetPendingSettlement(ctx context.Context, nonce string) (map[string]interface{}, error) {
	const query = `
		SELECT nonce, request_id, agent_address, provider_address, expected_amount_usdc,
		       status, challenge_issued_at, expires_at
		FROM pending_settlements
		WHERE nonce = $1
	`

	var result map[string]interface{}
	row := c.db.QueryRowContext(ctx, query, nonce)

	var (
		n, rid, agent, provider string
		amount                  float64
		status                  string
		issuedAt, expiresAt     sql.NullTime
	)

	err := row.Scan(&n, &rid, &agent, &provider, &amount, &status, &issuedAt, &expiresAt)
	if err != nil {
		return nil, err
	}

	result = map[string]interface{}{
		"nonce":                 n,
		"request_id":            rid,
		"agent_address":         agent,
		"provider_address":      provider,
		"expected_amount_usdc":  amount,
		"status":                status,
		"challenge_issued_at":   issuedAt.Time,
		"expires_at":            expiresAt.Time,
	}

	return result, nil
}

// ============ Transaction Recording ============

// RecordTransaction records a completed settlement as an immutable transaction
func (c *Client) RecordTransaction(
	ctx context.Context,
	nonce, agentAddr, providerAddr, txHash string,
	amountUsdc, feeUsdc, providerAmountUsdc float64,
	blockNumber int64,
	status string,
) error {
	const query = `
		INSERT INTO transactions
		(nonce, agent_address, provider_address, amount_usdc, fee_usdc,
		 provider_amount_usdc, tx_hash, block_number, status, settled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`

	_, err := c.db.ExecContext(
		ctx,
		query,
		nonce, agentAddr, providerAddr, amountUsdc, feeUsdc, providerAmountUsdc,
		txHash, blockNumber, status,
	)

	return err
}

// ============ Event Recording (Append-Only) ============

// RecordEvent adds an immutable event to the transaction log
func (c *Client) RecordEvent(ctx context.Context, transactionID, eventType, eventData string) error {
	const query = `
		INSERT INTO transaction_events
		(transaction_id, event_type, event_data, source)
		VALUES ($1, $2, $3::jsonb, 'hub')
	`

	_, err := c.db.ExecContext(ctx, query, transactionID, eventType, eventData)
	return err
}

// ============ Reputation Queries ============

// GetReputation retrieves reputation score for an entity
func (c *Client) GetReputation(ctx context.Context, address, entityType string) (map[string]interface{}, error) {
	const query = `
		SELECT entity_address, entity_type, transaction_count, successful_count,
		       failed_count, reputation_score, last_updated
		FROM reputation
		WHERE entity_address = $1 AND entity_type = $2
	`

	var (
		addr, etype        string
		txCount, succCount  int
		failCount, score    int
		lastUpdated         sql.NullTime
	)

	row := c.db.QueryRowContext(ctx, query, address, entityType)
	err := row.Scan(&addr, &etype, &txCount, &succCount, &failCount, &score, &lastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No reputation yet
		}
		return nil, err
	}

	return map[string]interface{}{
		"address":               addr,
		"type":                  etype,
		"transaction_count":     txCount,
		"successful_count":      succCount,
		"failed_count":          failCount,
		"reputation_score":      score,
		"last_updated":          lastUpdated.Time,
	}, nil
}

// UpdateReputation updates an entity's reputation score (derived, informational only)
func (c *Client) UpdateReputation(ctx context.Context, address, entityType string) error {
	const query = `
		UPDATE reputation
		SET transaction_count = (
				SELECT COUNT(*) FROM transactions WHERE agent_address = $1 OR provider_address = $1
			),
			successful_count = (
				SELECT COUNT(*) FROM transactions WHERE (agent_address = $1 OR provider_address = $1) AND status = 'settled'
			),
			failed_count = (
				SELECT COUNT(*) FROM transactions WHERE (agent_address = $1 OR provider_address = $1) AND status = 'failed'
			),
			last_updated = NOW()
		WHERE entity_address = $1 AND entity_type = $2
	`

	_, err := c.db.ExecContext(ctx, query, address, entityType)
	if err != nil {
		// If update fails, try insert (new entity)
		return c.ensureReputation(ctx, address, entityType)
	}

	return nil
}

func (c *Client) ensureReputation(ctx context.Context, address, entityType string) error {
	const query = `
		INSERT INTO reputation (entity_address, entity_type, transaction_count, successful_count, failed_count, reputation_score)
		VALUES ($1, $2, 0, 0, 0, 100)
		ON CONFLICT (entity_address) DO NOTHING
	`

	_, err := c.db.ExecContext(ctx, query, address, entityType)
	return err
}

// ============ Health & Diagnostics ============

// Stats returns database pool statistics
func (c *Client) Stats() sql.DBStats {
	return c.db.Stats()
}

// CheckHealth performs a comprehensive health check
func (c *Client) CheckHealth(ctx context.Context) (map[string]interface{}, error) {
	stats := c.db.Stats()

	// Count pending settlements
	var pendingCount int
	err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pending_settlements WHERE status = 'awaiting_proof'").Scan(&pendingCount)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"database":              "ok",
		"open_connections":      stats.OpenConnections,
		"in_use":                stats.InUse,
		"idle":                  stats.Idle,
		"pending_settlements":   pendingCount,
		"connection_wait_count": stats.WaitCount,
		"connection_wait_dur":   stats.WaitDuration.String(),
	}, nil
}
