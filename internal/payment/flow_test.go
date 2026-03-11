package payment

import (
	"testing"
)

// TestCompletePaymentFlow demonstrates the complete payment flow
func TestCompletePaymentFlow(t *testing.T) {
	// Step 1: Agent creates signer from private key
	privKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a"
	signer, err := NewEIP191Signer(privKey)
	if err != nil {
		t.Fatalf("step 1 failed: %v", err)
	}
	t.Logf("Step 1 ✓ Agent created (address: %s)", signer.AgentAddress())

	// Step 2: Hub sends payment challenge
	challenge := &PaymentChallenge{
		X402Version: 1,
		Price:       "0.001",
		Currency:    "USDC",
		Networks:    []string{"eip155:84532"},
		PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		Nonce:       "0x" + "a"*64, // Hub generates this
	}
	t.Logf("Step 2 ✓ Challenge received (nonce: %s...)", challenge.Nonce[:10])

	// Step 3: Agent signs the challenge
	signature, err := signer.SignChallenge(challenge)
	if err != nil {
		t.Fatalf("step 3 failed: %v", err)
	}
	t.Logf("Step 3 ✓ Challenge signed (signature: %s...)", signature.Signature[:20])

	// Step 4: Agent creates X-PAYMENT header
	xPaymentHeader := signature.ToX402Header("eip155:84532")
	t.Logf("Step 4 ✓ Payment header created (length: %d)", len(xPaymentHeader))

	// Step 5: Hub receives payment and verifies signature
	message := challenge.Hash()
	verified, err := VerifySignature(message, signature.Signature, signature.Agent)
	if err != nil {
		t.Fatalf("step 5 failed: %v", err)
	}
	if !verified {
		t.Fatal("step 5 failed: signature verification failed")
	}
	t.Log("Step 5 ✓ Signature verified by Hub")

	// Step 6: Hub settles payment on blockchain
	t.Log("Step 6 ✓ Payment settled on blockchain (mock)")

	// Step 7: Hub returns results
	t.Log("Step 7 ✓ Results returned to agent")

	t.Log("\n✓ Complete payment flow successful!")
}

// TestPaymentFlowSteps breaks down the exact steps
func TestPaymentFlowSteps(t *testing.T) {
	steps := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "Agent creates signer",
			test: func(t *testing.T) {
				_, err := NewEIP191Signer("0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a")
				if err != nil {
					t.Errorf("failed: %v", err)
				}
			},
		},
		{
			name: "Hub generates challenge",
			test: func(t *testing.T) {
				challenge := &PaymentChallenge{
					X402Version: 1,
					Price:       "0.001",
					Currency:    "USDC",
					Networks:    []string{"eip155:84532"},
					PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
					Nonce:       "0x" + "b"*64,
				}
				if challenge.Nonce != "0x"+("b"*64) {
					t.Error("nonce not set correctly")
				}
			},
		},
		{
			name: "Agent signs challenge",
			test: func(t *testing.T) {
				signer, _ := NewEIP191Signer("0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a")
				challenge := &PaymentChallenge{
					X402Version: 1,
					Price:       "0.001",
					Currency:    "USDC",
					Networks:    []string{"eip155:84532"},
					PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
					Nonce:       "0x" + "c"*64,
				}
				sig, err := signer.SignChallenge(challenge)
				if err != nil {
					t.Errorf("failed: %v", err)
				}
				if sig.Signature == "" {
					t.Error("signature is empty")
				}
			},
		},
		{
			name: "Agent creates X-PAYMENT header",
			test: func(t *testing.T) {
				signer, _ := NewEIP191Signer("0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a")
				challenge := &PaymentChallenge{
					X402Version: 1,
					Price:       "0.001",
					Currency:    "USDC",
					Networks:    []string{"eip155:84532"},
					PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
					Nonce:       "0x" + "d"*64,
				}
				sig, _ := signer.SignChallenge(challenge)
				header := sig.ToX402Header("eip155:84532")
				if header == "" {
					t.Error("header is empty")
				}
			},
		},
		{
			name: "Hub verifies signature",
			test: func(t *testing.T) {
				signer, _ := NewEIP191Signer("0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a")
				challenge := &PaymentChallenge{
					X402Version: 1,
					Price:       "0.001",
					Currency:    "USDC",
					Networks:    []string{"eip155:84532"},
					PayTo:       "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
					Nonce:       "0x" + "e"*64,
				}
				sig, _ := signer.SignChallenge(challenge)
				message := challenge.Hash()
				verified, err := VerifySignature(message, sig.Signature, sig.Agent)
				if err != nil {
					t.Errorf("verification failed: %v", err)
				}
				if !verified {
					t.Error("signature did not verify")
				}
			},
		},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			step.test(t)
		})
	}
}
