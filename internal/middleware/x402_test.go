package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octogate/octogate/internal/testutil"
)

// TestNonceFormatRegression verifies nonce format matches database/contract requirements
func TestNonceFormatRegression(t *testing.T) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	// Handler that just passes through
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	// Make request without payment to trigger challenge
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify we got a 402
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", w.Code)
	}

	// Parse the challenge
	var challenge map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&challenge); err != nil {
		t.Fatalf("failed to decode challenge: %v", err)
	}

	// Verify nonce format
	nonce, ok := challenge["nonce"].(string)
	if !ok {
		t.Fatalf("nonce not found or not string in challenge")
	}

	// Most critical test: nonce format must match database constraint and smart contract
	testutil.AssertNonceFormat(t, nonce)
}

// TestPaymentChallengeStructure verifies challenge contains all required fields
func TestPaymentChallengeStructure(t *testing.T) {
	tests := []struct {
		name     string
		payTo    string
		network  string
		price    float64
		validate func(t *testing.T, challenge map[string]interface{})
	}{
		{
			name:    "BaseSepoliaChallenge",
			payTo:   "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
			network: "eip155:84532",
			price:   0.001,
			validate: func(t *testing.T, challenge map[string]interface{}) {
				if challenge["x402Version"] != float64(1) {
					t.Error("missing or wrong x402Version")
				}
				if _, ok := challenge["price"]; !ok {
					t.Error("missing price")
				}
				if challenge["currency"] != "USDC" {
					t.Error("wrong currency")
				}
				if _, ok := challenge["networks"]; !ok {
					t.Error("missing networks")
				}
				if _, ok := challenge["payTo"]; !ok {
					t.Error("missing payTo")
				}
				if _, ok := challenge["nonce"]; !ok {
					t.Error("missing nonce")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := X402Middleware(tt.payTo, tt.network, tt.price)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := middleware(next)

			req := httptest.NewRequest("GET", "/v1/search", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var challenge map[string]interface{}
			json.NewDecoder(w.Body).Decode(&challenge)

			tt.validate(t, challenge)
		})
	}
}

// TestPaymentHeaderBypass verifies payment header is processed (even if verification is stubbed)
func TestPaymentHeaderBypass(t *testing.T) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := middleware(next)

	// Request WITH X-PAYMENT header (even if bogus, should bypass 402)
	req := httptest.NewRequest("GET", "/v1/search", nil)
	req.Header.Set("X-PAYMENT", "test-payment-proof")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should call next handler (not return 402)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !handlerCalled {
		t.Error("next handler was not called")
	}

	// TODO: Add real Coinbase facilitator verification once implemented
	// For now, just verify the bypass logic works
}
