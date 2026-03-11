package payment

import (
	"strings"
	"testing"
)

// TestPaymentChallengeHashFormat verifies challenge hash format
func TestPaymentChallengeHashFormat(t *testing.T) {
	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + strings.Repeat("a", 64),
	}

	hash := challenge.Hash()

	// Should be in format: "x402-payment:{payTo}:{nonce}:{price}:{currency}"
	if !strings.HasPrefix(hash, "x402-payment:") {
		t.Error("hash should start with x402-payment:")
	}

	parts := strings.Split(hash, ":")
	if len(parts) != 5 {
		t.Errorf("hash should have 5 parts separated by :, got %d", len(parts))
	}

	t.Logf("Challenge Hash: %s", hash)
}

// TestPaymentSignatureStructure verifies payment signature structure
func TestPaymentSignatureStructure(t *testing.T) {
	sig := &PaymentSignature{
		Agent:     "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Signature: "0x" + strings.Repeat("b", 130),
		Timestamp: 1234567890,
	}

	// Verify structure is valid
	if sig.Agent == "" {
		t.Error("agent should not be empty")
	}

	if sig.Signature == "" {
		t.Error("signature should not be empty")
	}

	// Test header format
	header := sig.ToX402Header("eip155:84532")
	if !strings.HasPrefix(header, "0x") {
		t.Error("header should contain agent address starting with 0x")
	}

	parts := strings.Split(header, ":")
	if len(parts) < 3 {
		t.Errorf("header should have at least 3 parts, got %d", len(parts))
	}
}

// TestPaymentChallengeCreation verifies challenge can be created
func TestPaymentChallengeCreation(t *testing.T) {
	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + strings.Repeat("c", 64),
	}

	if challenge.X402Version != 1 {
		t.Error("x402Version should be 1")
	}

	if challenge.Price != "0.001" {
		t.Error("price should be 0.001")
	}

	if challenge.Currency != "USDC" {
		t.Error("currency should be USDC")
	}

	if len(challenge.Networks) != 1 {
		t.Error("networks should have 1 entry")
	}

	if !strings.HasPrefix(challenge.Nonce, "0x") {
		t.Error("nonce should start with 0x")
	}
}
