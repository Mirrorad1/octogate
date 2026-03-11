package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octogate/octogate/internal/testutil"
)

// TestPaymentVerificationFlow tests the complete payment challenge -> verify -> pass flow
func TestPaymentVerificationFlow(t *testing.T) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := middleware(next)

	t.Run("InvalidPaymentReturns402", func(t *testing.T) {
		// Request with invalid payment header
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-PAYMENT", "")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402, got %d", w.Code)
		}
		if handlerCalled {
			t.Error("next handler should not be called on invalid payment")
		}
	})

	t.Run("ValidPaymentCallsNext", func(t *testing.T) {
		handlerCalled = false
		// Request with valid payment header
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-PAYMENT", "valid-payment-proof-0x123abc")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if !handlerCalled {
			t.Error("next handler should be called on valid payment")
		}
		if w.Header().Get("X-PAYMENT-RESPONSE") != "verified" {
			t.Error("X-PAYMENT-RESPONSE header not set")
		}
	})
}

// TestChallengeNonceFormat verifies nonce in payment challenge
func TestChallengeNonceFormat(t *testing.T) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	// Request without payment → should return 402 with challenge
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", w.Code)
	}

	var challenge map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&challenge); err != nil {
		t.Fatalf("failed to decode challenge: %v", err)
	}

	// Verify nonce format (this is CRITICAL)
	nonce, ok := challenge["nonce"].(string)
	if !ok {
		t.Fatal("nonce not found or not string")
	}

	testutil.AssertNonceFormat(t, nonce)
	t.Logf("Challenge nonce format valid: %s...", nonce[:10])
}

// TestPaymentVerificationErrorHandling verifies error scenarios
func TestPaymentVerificationErrorHandling(t *testing.T) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	tests := []struct {
		name         string
		paymentProof string
		expectCode   int
		expectError  bool
	}{
		{
			name:         "EmptyPaymentProof",
			paymentProof: "",
			expectCode:   http.StatusPaymentRequired,
			expectError:  true,
		},
		{
			name:         "ValidPaymentProof",
			paymentProof: "0x1234abcd",
			expectCode:   http.StatusOK,
			expectError:  false,
		},
		{
			name:         "PaymentProofWithSignature",
			paymentProof: "0x5678efgh9012ijkl3456mnop",
			expectCode:   http.StatusOK,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.paymentProof != "" {
				req.Header.Set("X-PAYMENT", tt.paymentProof)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("expected %d, got %d", tt.expectCode, w.Code)
			}

			if tt.expectError && w.Code == http.StatusOK {
				t.Error("expected error response but got success")
			}
		})
	}
}

// TestPaymentVerificationHeaders verifies all required headers are set
func TestPaymentVerificationHeaders(t *testing.T) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	// Test 402 response headers
	t.Run("ChallengeHeaders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Content-Type should be application/json for challenge")
		}
	})

	// Test verified response headers
	t.Run("VerifiedHeaders", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-PAYMENT", "proof")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Header().Get("X-PAYMENT-RESPONSE") != "verified" {
			t.Error("X-PAYMENT-RESPONSE header should be 'verified'")
		}
	})
}

// BenchmarkPaymentVerification measures verification performance
func BenchmarkPaymentVerification(b *testing.B) {
	middleware := X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(next)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-PAYMENT", "test-proof")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
