package testutil

import (
	"context"
	"sync"
)

// MockStore implements a simple in-memory Store interface for testing
type MockStore struct {
	mu                sync.RWMutex
	pendingSettlements map[string]map[string]interface{}
	transactions      map[string]map[string]interface{}
	events            []map[string]interface{}
	health            map[string]interface{}
}

// NewMockStore creates a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		pendingSettlements: make(map[string]map[string]interface{}),
		transactions:      make(map[string]map[string]interface{}),
		events:            make([]map[string]interface{}, 0),
		health: map[string]interface{}{
			"database": "ok",
			"status":   "healthy",
		},
	}
}

// RecordPendingSettlement stores a payment challenge
func (m *MockStore) RecordPendingSettlement(ctx context.Context, nonce, requestID, agentAddr, providerAddr string, amountUsdc float64, expiresAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pendingSettlements[nonce] = map[string]interface{}{
		"nonce":                 nonce,
		"request_id":            requestID,
		"agent_address":         agentAddr,
		"provider_address":      providerAddr,
		"expected_amount_usdc":  amountUsdc,
		"expires_at":            expiresAt,
		"status":                "awaiting_proof",
	}
	return nil
}

// UpdateSettlementConfirmed marks a settlement as confirmed
func (m *MockStore) UpdateSettlementConfirmed(ctx context.Context, nonce, txHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if settlement, ok := m.pendingSettlements[nonce]; ok {
		settlement["status"] = "confirmed"
		settlement["tx_hash"] = txHash
		return nil
	}
	return nil
}

// GetPendingSettlement retrieves a pending settlement
func (m *MockStore) GetPendingSettlement(ctx context.Context, nonce string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if settlement, ok := m.pendingSettlements[nonce]; ok {
		return settlement, nil
	}
	return nil, nil
}

// RecordTransaction records a completed settlement
func (m *MockStore) RecordTransaction(ctx context.Context, nonce, agentAddr, providerAddr, txHash string, amountUsdc, feeUsdc, providerAmountUsdc float64, blockNumber int64, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transactions[nonce] = map[string]interface{}{
		"nonce":                  nonce,
		"agent_address":          agentAddr,
		"provider_address":       providerAddr,
		"tx_hash":                txHash,
		"amount_usdc":            amountUsdc,
		"fee_usdc":               feeUsdc,
		"provider_amount_usdc":   providerAmountUsdc,
		"block_number":           blockNumber,
		"status":                 status,
	}
	return nil
}

// RecordEvent adds an event to the log
func (m *MockStore) RecordEvent(ctx context.Context, transactionID, eventType, eventData string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, map[string]interface{}{
		"transaction_id": transactionID,
		"event_type":     eventType,
		"event_data":     eventData,
	})
	return nil
}

// CheckHealth returns health status
func (m *MockStore) CheckHealth(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.health, nil
}

// GetEvents returns all recorded events (for testing)
func (m *MockStore) GetEvents() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.events
}

// GetTransactions returns all recorded transactions (for testing)
func (m *MockStore) GetTransactions() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.transactions
}

// ============ Mock HTTP Client ============

// MockHTTPRequest captures an HTTP request for testing
type MockHTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// MockHTTPClient captures requests for inspection
type MockHTTPClient struct {
	mu       sync.Mutex
	requests []MockHTTPRequest
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		requests: make([]MockHTTPRequest, 0),
	}
}

// RecordRequest records an HTTP request
func (m *MockHTTPClient) RecordRequest(method, url string, headers map[string]string, body []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, MockHTTPRequest{
		Method:  method,
		URL:     url,
		Headers: headers,
		Body:    body,
	})
}

// GetRequests returns all recorded requests
func (m *MockHTTPClient) GetRequests() []MockHTTPRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.requests
}
