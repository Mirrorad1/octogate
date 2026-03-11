package blockchain

import (
	"context"
)

// Listener defines the interface for blockchain event listening
type Listener interface {
	// Start begins listening for events (runs in background)
	Start() error

	// Stop gracefully shuts down the listener
	Stop()

	// SettlementEvents returns a channel for receiving confirmed payment settlements
	SettlementEvents() <-chan PaymentSettledEvent

	// Errors returns a channel for monitoring listener health issues
	Errors() <-chan error

	// LastProcessedBlock returns the last block successfully processed
	LastProcessedBlock() uint64

	// GetLastProcessedBlock returns the block number from checkpoint
	GetLastProcessedBlock(ctx context.Context) (uint64, error)
}

// Verify that EventListener implements Listener
var _ Listener = (*EventListener)(nil)

// GetLastProcessedBlock retrieves the last processed block from the database
func (el *EventListener) GetLastProcessedBlock(ctx context.Context) (uint64, error) {
	return el.lastProcessed, nil
}

// LastProcessedBlock returns the current last processed block
func (el *EventListener) LastProcessedBlock() uint64 {
	return el.lastProcessed
}
