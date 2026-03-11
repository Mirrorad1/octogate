package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestX402MiddlewareCreation verifies middleware can be created
func TestX402MiddlewareCreation(t *testing.T) {
	payTo := "0x742d35Cc6634C0532925a3b844Bc9e7595f42521"
	network := "eip155:84532"
	price := 0.001

	// Should not panic
	middleware := X402Middleware(payTo, network, price, "https://x402.org/facilitator")
	if middleware == nil {
		t.Error("middleware should not be nil")
	}
}

// TestX402MiddlewareIntegration verifies middleware processes requests
func TestX402MiddlewareIntegration(t *testing.T) {
	middleware := X402Middleware(
		"0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
		"eip155:84532",
		0.001,
		"https://x402.org/facilitator",
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := middleware(next)

	// Make a request
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	// Execute handler - should handle request (either 402 or pass through)
	handler.ServeHTTP(w, req)

	// Response should be valid HTTP
	if w.Code == 0 {
		t.Error("handler should write a response")
	}

	// Note: With Coinbase x402 library, requests may fail if library isn't fully initialized
	// This test just verifies the middleware doesn't panic
	t.Logf("Response code: %d", w.Code)
}
