package db

import (
	"context"
)

// Store defines the interface for database operations
// Both *Client and MockStore implement this interface
type Store interface {
	// Settlement operations
	RecordPendingSettlement(ctx context.Context, nonce, requestID, agentAddr, providerAddr string, amountUsdc float64, expiresAt int64) error
	UpdateSettlementConfirmed(ctx context.Context, nonce, txHash string) error
	GetPendingSettlement(ctx context.Context, nonce string) (map[string]interface{}, error)

	// Transaction recording (immutable audit log)
	RecordTransaction(ctx context.Context, nonce, agentAddr, providerAddr, txHash string, amountUsdc, feeUsdc, providerAmountUsdc float64, blockNumber int64, status string) error

	// Event logging (append-only)
	RecordEvent(ctx context.Context, transactionID, eventType, eventData string) error

	// Health & diagnostics
	CheckHealth(ctx context.Context) (map[string]interface{}, error)

	// Reputation (derived, informational)
	// GetReputation(ctx context.Context, address, entityType string) (map[string]interface{}, error)
	// UpdateReputation(ctx context.Context, address, entityType string) error

	// Lifecycle
	Close() error
}

// Verify that Client implements Store
var _ Store = (*Client)(nil)
