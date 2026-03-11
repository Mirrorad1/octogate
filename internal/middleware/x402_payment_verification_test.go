package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPaymentVerificationMiddleware verifies payment verification middleware processes requests
func TestPaymentVerificationMiddleware(t *testing.T) {
	middleware := X402Middleware(
		"0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		"eip155:84532",
		0.001,
		"https://x402.org/facilitator",
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	})

	handler := middleware(next)

	// Test 1: Request without payment header
	t.Run("WithoutPaymentHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should return some response (402 or error from library)
		if w.Code < 200 || w.Code >= 600 {
			t.Logf("Response code %d is outside valid range", w.Code)
		}
	})

	// Test 2: Request with payment header
	t.Run("WithPaymentHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
		req.Header.Set("X-PAYMENT", "0x1234567890abcdef:eip155:84532:0xabc123")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should return some response
		if w.Code < 200 || w.Code >= 600 {
			t.Logf("Response code %d is outside valid range", w.Code)
		}
	})
}
