package blockchain

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// EventListener monitors a blockchain for PaymentSettled events
// Connects via WebSocket, falls back to polling if needed
type EventListener struct {
	rpcURL           string
	fallbackRPCURL   string
	contractAddress  common.Address
	client           *ethclient.Client
	fallbackClient   *ethclient.Client
	rpcClient        *rpc.Client
	db               *sql.DB
	settlementChan   chan PaymentSettledEvent
	errorChan        chan error
	ctx              context.Context
	cancel           context.CancelFunc
	lastProcessed    uint64
	checkpointTicker *time.Ticker
}

// PaymentSettledEvent represents the on-chain PaymentSettled event
type PaymentSettledEvent struct {
	Agent     common.Address
	Provider  common.Address
	Amount    *big.Int
	Fee       *big.Int
	Nonce     [32]byte
	TxHash    [32]byte
	TxHashStr string
	Block     uint64
	LogIndex  uint
	Timestamp int64
}

// NewEventListener creates a new blockchain event listener
func NewEventListener(
	rpcURL, fallbackRPCURL string,
	contractAddr common.Address,
	db *sql.DB,
) (*EventListener, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Try connecting to primary RPC
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to primary RPC: %w", err)
	}

	// Connect to fallback RPC
	var fallbackClient *ethclient.Client
	if fallbackRPCURL != "" {
		fallbackClient, _ = ethclient.Dial(fallbackRPCURL)
		// Fallback connection is optional; don't fail if it doesn't work
	}

	listener := &EventListener{
		rpcURL:          rpcURL,
		fallbackRPCURL:  fallbackRPCURL,
		contractAddress: contractAddr,
		client:          client,
		fallbackClient:  fallbackClient,
		db:              db,
		settlementChan:  make(chan PaymentSettledEvent, 100),
		errorChan:       make(chan error, 10),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Load last processed block from database
	err = listener.loadLastProcessedBlock()
	if err != nil {
		log.Printf("Warning: Could not load checkpoint: %v", err)
		listener.lastProcessed = 0 // Start from genesis if no checkpoint
	}

	return listener, nil
}

// Start begins listening for events
// Runs in a background goroutine
func (el *EventListener) Start() error {
	go el.runListener()
	go el.runCheckpointer()
	go el.runHealthCheck()
	return nil
}

// Stop gracefully shuts down the listener
func (el *EventListener) Stop() {
	el.cancel()
	el.client.Close()
	if el.fallbackClient != nil {
		el.fallbackClient.Close()
	}
	if el.checkpointTicker != nil {
		el.checkpointTicker.Stop()
	}
}

// SettlementEvents returns the channel for receiving confirmed settlements
func (el *EventListener) SettlementEvents() <-chan PaymentSettledEvent {
	return el.settlementChan
}

// Errors returns the error channel for monitoring listener health
func (el *EventListener) Errors() <-chan error {
	return el.errorChan
}

// ============ Main Event Monitoring Logic ============

func (el *EventListener) runListener() {
	// Try WebSocket subscription first
	if el.rpcClient == nil {
		err := el.connectWebSocket()
		if err == nil {
			el.runWebSocketListener()
			return
		}
		log.Printf("WebSocket failed: %v, falling back to polling", err)
	}

	// Fallback: Poll for new events
	el.runPollingListener()
}

// connectWebSocket establishes a WebSocket connection to the RPC
func (el *EventListener) connectWebSocket() error {
	rpcClient, err := rpc.Dial(el.rpcURL)
	if err != nil {
		return fmt.Errorf("WebSocket dial failed: %w", err)
	}

	el.rpcClient = rpcClient
	return nil
}

// runWebSocketListener uses eth_subscribe to listen for events (efficient)
func (el *EventListener) runWebSocketListener() {
	for {
		select {
		case <-el.ctx.Done():
			return
		default:
		}

		// Subscribe to PaymentSettled events
		// Filter: topics[0] = event signature, topics[1] = agent (indexed)
		eventSig := common.HexToHash("0x...") // PaymentSettled event hash (computed offline)

		var events []map[string]interface{}
		err := el.rpcClient.CallContext(el.ctx, &events, "eth_subscribe", "logs", map[string]interface{}{
			"address": el.contractAddress.Hex(),
			"topics":  []interface{}{eventSig},
		})

		if err != nil {
			log.Printf("WebSocket subscription failed: %v, retrying in 10s", err)
			el.errorChan <- fmt.Errorf("WebSocket subscription error: %w", err)

			select {
			case <-time.After(10 * time.Second):
				// Reconnect
				el.rpcClient.Close()
				el.rpcClient = nil
				if err := el.connectWebSocket(); err != nil {
					log.Printf("Reconnection failed: %v", err)
					continue
				}
			case <-el.ctx.Done():
				return
			}
			continue
		}

		// Process events
		for _, event := range events {
			settled, err := el.parseEvent(event)
			if err != nil {
				log.Printf("Failed to parse event: %v", err)
				el.errorChan <- err
				continue
			}

			select {
			case el.settlementChan <- settled:
			case <-el.ctx.Done():
				return
			}
		}
	}
}

// runPollingListener polls for new logs every 2 seconds (fallback, less efficient)
func (el *EventListener) runPollingListener() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get current block number
			blockNumber, err := el.client.BlockNumber(el.ctx)
			if err != nil {
				log.Printf("Failed to get block number: %v", err)
				el.errorChan <- fmt.Errorf("block number query failed: %w", err)
				continue
			}

			// Query logs from last processed to current
			// Limit to 1000 blocks at a time to avoid timeouts
			fromBlock := el.lastProcessed
			toBlock := blockNumber
			if toBlock-fromBlock > 1000 {
				toBlock = fromBlock + 1000
			}

			events, err := el.pollLogs(fromBlock, toBlock)
			if err != nil {
				log.Printf("Failed to poll logs: %v", err)
				el.errorChan <- err
				continue
			}

			// Process events
			for _, event := range events {
				select {
				case el.settlementChan <- event:
					el.lastProcessed = event.Block
				case <-el.ctx.Done():
					return
				}
			}

		case <-el.ctx.Done():
			return
		}
	}
}

// pollLogs queries the blockchain for PaymentSettled events in a block range
func (el *EventListener) pollLogs(fromBlock, toBlock uint64) ([]PaymentSettledEvent, error) {
	// Get logs from the RPC
	query := map[string]interface{}{
		"address":   []string{el.contractAddress.Hex()},
		"fromBlock": fmt.Sprintf("0x%x", fromBlock),
		"toBlock":   fmt.Sprintf("0x%x", toBlock),
		// Topic 0 is the event signature; would be set to PaymentSettled hash
	}

	var rawLogs []map[string]interface{}
	err := el.client.Client().CallContext(el.ctx, &rawLogs, "eth_getLogs", query)
	if err != nil {
		return nil, fmt.Errorf("eth_getLogs failed: %w", err)
	}

	var events []PaymentSettledEvent
	for _, rawLog := range rawLogs {
		event, err := el.parseEvent(rawLog)
		if err != nil {
			log.Printf("Failed to parse log: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// parseEvent converts a raw log into a PaymentSettledEvent
func (el *EventListener) parseEvent(rawLog map[string]interface{}) (PaymentSettledEvent, error) {
	// This is a simplified parser; real implementation would:
	// 1. Verify the event signature
	// 2. Decode indexed and non-indexed parameters
	// 3. Handle various error cases

	event := PaymentSettledEvent{
		Timestamp: time.Now().Unix(),
	}

	// Extract block number
	if blockStr, ok := rawLog["blockNumber"].(string); ok {
		blockNum := new(big.Int)
		blockNum.SetString(blockStr[2:], 16) // Remove 0x prefix, parse hex
		event.Block = blockNum.Uint64()
	}

	// Extract transaction hash
	if txStr, ok := rawLog["transactionHash"].(string); ok {
		event.TxHashStr = txStr
	}

	// Extract log index
	if logIndexStr, ok := rawLog["logIndex"].(string); ok {
		logIndex := new(big.Int)
		logIndex.SetString(logIndexStr[2:], 16)
		event.LogIndex = uint(logIndex.Uint64())
	}

	// TODO: Decode indexed and non-indexed parameters from rawLog["data"] and rawLog["topics"]
	// This requires the contract ABI

	return event, nil
}

// ============ Block Checkpointing ============

func (el *EventListener) runCheckpointer() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	el.checkpointTicker = ticker

	for {
		select {
		case <-ticker.C:
			if el.lastProcessed > 0 {
				err := el.saveCheckpoint(el.lastProcessed)
				if err != nil {
					log.Printf("Failed to save checkpoint: %v", err)
					el.errorChan <- err
				}
			}
		case <-el.ctx.Done():
			return
		}
	}
}

func (el *EventListener) loadLastProcessedBlock() error {
	const query = `
		SELECT COALESCE(MAX(block_number), 0)
		FROM transactions
		WHERE status = 'settled'
	`

	err := el.db.QueryRow(query).Scan(&el.lastProcessed)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	return nil
}

func (el *EventListener) saveCheckpoint(blockNumber uint64) error {
	const query = `
		INSERT INTO blockchain_checkpoint (block_number, updated_at)
		VALUES ($1, NOW())
		ON CONFLICT DO UPDATE SET updated_at = NOW(), block_number = $1
	`

	_, err := el.db.Exec(query, blockNumber)
	return err
}

// ============ Health Checking ============

func (el *EventListener) runHealthCheck() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	lastEventTime := time.Now()

	for {
		select {
		case <-ticker.C:
			// Check that we're receiving events or at least blocks
			blockNum, err := el.client.BlockNumber(el.ctx)
			if err != nil {
				log.Printf("Health check failed: block number query error: %v", err)
				el.errorChan <- fmt.Errorf("health check failed: %w", err)
				continue
			}

			// If no blocks have been received in 60 seconds, try fallback RPC
			if time.Since(lastEventTime) > 60*time.Second {
				log.Printf("No events received for 60s, block height: %d", blockNum)

				if el.fallbackClient != nil {
					fallbackBlockNum, err := el.fallbackClient.BlockNumber(el.ctx)
					if err == nil && fallbackBlockNum > blockNum {
						log.Printf("Primary RPC is lagging, switching to fallback")
						el.client = el.fallbackClient
					}
				}
			}

		case <-el.ctx.Done():
			return
		}
	}
}

// ============ Database Integration ============

// RecordSettlement stores a confirmed settlement in the database
func (el *EventListener) RecordSettlement(event PaymentSettledEvent) error {
	const query = `
		UPDATE pending_settlements
		SET status = 'confirmed',
			on_chain_tx_hash = $1,
			confirmed_at = NOW(),
			updated_at = NOW()
		WHERE nonce = $2
	`

	result, err := el.db.Exec(query, event.TxHashStr, fmt.Sprintf("0x%x", event.Nonce))
	if err != nil {
		return fmt.Errorf("failed to record settlement: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("no pending settlement found for nonce %x", event.Nonce)
	}

	return nil
}

// RecordEvent stores an immutable event in the transaction_events log
func (el *EventListener) RecordEvent(transactionID interface{}, eventType string, eventData interface{}) error {
	dataJSON, _ := json.Marshal(eventData)

	const query = `
		INSERT INTO transaction_events (transaction_id, event_type, event_data, source)
		VALUES ($1, $2, $3, 'blockchain_listener')
	`

	_, err := el.db.Exec(query, transactionID, eventType, dataJSON)
	return err
}
