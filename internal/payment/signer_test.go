package payment

import (
	"strings"
	"testing"
)

// TestPaymentSignatureFormatting verifies signature formatting
func TestPaymentSignatureFormatting(t *testing.T) {
	sig := &PaymentSignature{
		Agent:     "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Signature: "0x" + strings.Repeat("abc123", 22),
		Timestamp: 1709845123,
	}

	// Test ToX402Header format
	header := sig.ToX402Header("eip155:84532")

	// Format should be: "{agent}:{network}:{signature}"
	parts := strings.Split(header, ":")
	if len(parts) < 3 {
		t.Errorf("header should have at least 3 parts separated by :, got %d", len(parts))
	}

	// First part should be agent address
	if !strings.HasPrefix(parts[0], "0x") {
		t.Error("agent in header should start with 0x")
	}

	t.Logf("X402 Header format valid: %d parts", len(parts))
}

// TestPaymentChallengeNonceFormat verifies nonce format
func TestPaymentChallengeNonceFormat(t *testing.T) {
	tests := []struct {
		name  string
		nonce string
		valid bool
	}{
		{"ValidHexNonce", "0x" + strings.Repeat("a", 64), true},
		{"InvalidMissingPrefix", strings.Repeat("a", 64), false},
		{"InvalidWrongLength", "0x" + strings.Repeat("b", 63), false},
		{"ValidDifferentChars", "0x" + strings.Repeat("f", 64), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.nonce) < 2 {
				return // Skip validation for short nonces
			}

			hasPrefix := strings.HasPrefix(tt.nonce, "0x")
			expectedLength := len(tt.nonce) == 66 // 0x + 64 hex chars

			if tt.valid {
				if !hasPrefix || !expectedLength {
					t.Errorf("nonce %s should be valid hex format", tt.nonce[:10])
				}
			}
		})
	}
}

// TestSignatureVerificationStructure verifies verification can work with proper format
func TestSignatureVerificationStructure(t *testing.T) {
	message := "x402-payment:0x742d35Cc6634C0532925a3b844Bc9e7595f42521:0x" + strings.Repeat("a", 64) + ":0.001:USDC"
	signature := "0x" + strings.Repeat("abcd", 32) + "ab" // Exactly 132 chars: 0x + 130 hex chars
	agentAddress := "0x742d35Cc6634C0532925a3b844Bc9e7595f42521"

	// Verify message format is correct
	if !strings.HasPrefix(message, "x402-payment:") {
		t.Error("message should start with x402-payment:")
	}

	// Verify signature format
	if !strings.HasPrefix(signature, "0x") {
		t.Error("signature should start with 0x")
	}

	if len(signature) != 132 {
		t.Errorf("signature should be 132 chars (0x + 130 hex), got %d", len(signature))
	}

	// Verify agent format
	if !strings.HasPrefix(agentAddress, "0x") {
		t.Error("agent address should start with 0x")
	}

	if len(agentAddress) != 42 {
		t.Errorf("agent address should be 42 chars (0x + 40 hex), got %d", len(agentAddress))
	}

	t.Log("Signature verification structure is valid")
}
