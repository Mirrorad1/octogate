package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"time"
)

// PaymentChallenge represents a generated x402 payment challenge
type PaymentChallenge struct {
	X402Version int
	Price       string
	Currency    string
	Networks    []string
	PayTo       string
	Nonce       string
}

// NewPaymentChallengeFactory creates valid payment challenges for testing
type PaymentChallengeFactory struct {
	PayTo    string
	Network  string
	Price    string
	Currency string
}

// NewPaymentChallengeFactory returns a factory with defaults
func NewPaymentChallengeFactory() *PaymentChallengeFactory {
	return &PaymentChallengeFactory{
		PayTo:    "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Network:  "eip155:84532",
		Price:    "0.001",
		Currency: "USDC",
	}
}

// Build creates a valid PaymentChallenge with cryptographic nonce
func (f *PaymentChallengeFactory) Build() (*PaymentChallenge, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, err
	}
	nonce := "0x" + hex.EncodeToString(nonceBytes)

	return &PaymentChallenge{
		X402Version: 1,
		Price:       f.Price,
		Currency:    f.Currency,
		Networks:    []string{f.Network},
		PayTo:       f.PayTo,
		Nonce:       nonce,
	}, nil
}

// WithPayTo sets the PayTo address
func (f *PaymentChallengeFactory) WithPayTo(addr string) *PaymentChallengeFactory {
	f.PayTo = addr
	return f
}

// WithPrice sets the price
func (f *PaymentChallengeFactory) WithPrice(price string) *PaymentChallengeFactory {
	f.Price = price
	return f
}

// ============ Agent Wallet Factory ============

// AgentWallet represents a test agent wallet
type AgentWallet struct {
	Address    string
	PrivateKey string
	Network    string
	CreatedAt  time.Time
}

// NewAgentWalletFactory creates test wallets
type AgentWalletFactory struct {
	Network string
}

// NewAgentWalletFactory returns a factory with defaults
func NewAgentWalletFactory() *AgentWalletFactory {
	return &AgentWalletFactory{
		Network: "eip155:84532",
	}
}

// Build creates a test wallet (mock, not real)
func (f *AgentWalletFactory) Build() *AgentWallet {
	return &AgentWallet{
		Address:    "0x" + hex.EncodeToString(make([]byte, 20)),
		PrivateKey: "0x" + hex.EncodeToString(make([]byte, 32)),
		Network:    f.Network,
		CreatedAt:  time.Now(),
	}
}

// ============ Settlement Event Factory ============

// SettlementEvent represents an on-chain PaymentSettled event
type SettlementEvent struct {
	Agent    string
	Provider string
	Amount   *big.Int
	Fee      *big.Int
	Nonce    string
	TxHash   string
	Block    uint64
}

// NewSettlementEventFactory creates test settlement events
type SettlementEventFactory struct {
	TxHashCounter int
}

// NewSettlementEventFactory returns a factory
func NewSettlementEventFactory() *SettlementEventFactory {
	return &SettlementEventFactory{}
}

// Build creates a test settlement event
func (f *SettlementEventFactory) Build(nonce string) *SettlementEvent {
	txHash := "0x" + hex.EncodeToString([]byte{byte(f.TxHashCounter)})
	f.TxHashCounter++

	return &SettlementEvent{
		Agent:    "0x" + hex.EncodeToString(make([]byte, 20)),
		Provider: "0x" + hex.EncodeToString(make([]byte, 20)),
		Amount:   big.NewInt(1000000), // 1 USDC (6 decimals)
		Fee:      big.NewInt(100000),  // 0.1 USDC
		Nonce:    nonce,
		TxHash:   txHash,
		Block:    1,
	}
}

// ============ Context Factory ============

// NewTestContext returns a context with a 5-second timeout for tests
func NewTestContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
