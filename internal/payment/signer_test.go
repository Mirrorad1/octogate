package payment

import (
	"strings"
	"testing"
)

// TestEIP191SignerCreation verifies signer can be created from private key
func TestEIP191SignerCreation(t *testing.T) {
	// Test private key (dummy, for testing only)
	privKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a"

	signer, err := NewEIP191Signer(privKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	if signer.AgentAddress() == "" {
		t.Error("agent address should not be empty")
	}

	if !strings.HasPrefix(signer.AgentAddress(), "0x") {
		t.Error("agent address should start with 0x")
	}

	t.Logf("Agent Address: %s", signer.AgentAddress())
}

// TestChallengeSigningFlow verifies complete signing flow
func TestChallengeSigningFlow(t *testing.T) {
	privKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a"
	signer, err := NewEIP191Signer(privKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	// Create a payment challenge
	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + "a"*64,
	}

	// Sign the challenge
	signature, err := signer.SignChallenge(challenge)
	if err != nil {
		t.Fatalf("failed to sign challenge: %v", err)
	}

	// Verify signature structure
	if signature.Agent == "" {
		t.Error("signature agent should not be empty")
	}
	if signature.Signature == "" {
		t.Error("signature should not be empty")
	}
	if !strings.HasPrefix(signature.Signature, "0x") {
		t.Error("signature should start with 0x")
	}
	if len(signature.Signature) != 132 { // 0x + 130 hex chars (65 bytes)
		t.Errorf("signature length should be 132, got %d", len(signature.Signature))
	}

	t.Logf("Signature created: %s...", signature.Signature[:20])
}

// TestX402HeaderFormat verifies header formatting
func TestX402HeaderFormat(t *testing.T) {
	privKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a"
	signer, _ := NewEIP191Signer(privKey)

	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + "b"*64,
	}

	signature, _ := signer.SignChallenge(challenge)
	header := signature.ToX402Header("eip155:84532")

	// Format: "{agent}:{network}:{signature}"
	parts := strings.Split(header, ":")
	if len(parts) < 3 {
		t.Errorf("header should have at least 3 parts separated by :, got %d", len(parts))
	}

	// First part should be agent address
	if !strings.HasPrefix(parts[0], "0x") {
		t.Error("agent in header should start with 0x")
	}

	t.Logf("X402 Header: %s...", header[:30])
}

// TestSignatureVerification verifies signature can be verified
func TestSignatureVerification(t *testing.T) {
	privKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a"
	signer, _ := NewEIP191Signer(privKey)

	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + "c"*64,
	}

	signature, _ := signer.SignChallenge(challenge)

	// Verify signature
	message := challenge.Hash()
	verified, err := VerifySignature(message, signature.Signature, signature.Agent)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if !verified {
		t.Error("signature should verify")
	}

	t.Log("Signature verification passed")
}

// TestChallengeHash verifies hash format
func TestChallengeHash(t *testing.T) {
	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + "d"*64,
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
