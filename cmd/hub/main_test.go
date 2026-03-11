package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/octogate/octogate/internal/handlers"
	"github.com/octogate/octogate/internal/middleware"
	"github.com/octogate/octogate/internal/testutil"
)

// TestE2EPaymentFlow demonstrates the full payment challenge flow
func TestE2EPaymentFlow(t *testing.T) {
	// Setup router with middleware
	router := chi.NewRouter()

	payTo := "0x742d35Cc6634C0532925a3b844Bc9e7595f42521"
	network := "eip155:84532"
	price := 0.001

	// Register search endpoint with x402 middleware
	searchRoute := router.With(middleware.X402Middleware(payTo, network, price))
	searchRoute.Get("/v1/search", handlers.SearchHandler)

	// Create test server
	server := httptest.NewServer(router)
	defer server.Close()

	// Step 1: Client requests search without payment → expects 402
	t.Run("Step1_NoPayment_Returns402", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/v1/search?q=rust")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusPaymentRequired {
			t.Fatalf("expected 402, got %d", resp.StatusCode)
		}

		var challenge map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&challenge); err != nil {
			t.Fatalf("failed to decode challenge: %v", err)
		}

		// Verify challenge structure
		if challenge["x402Version"] == nil {
			t.Error("missing x402Version")
		}
		if challenge["price"] == nil {
			t.Error("missing price")
		}
		if challenge["payTo"] != payTo {
			t.Error("payTo mismatch")
		}

		// CRITICAL: Verify nonce format matches contract expectations
		nonce, ok := challenge["nonce"].(string)
		if !ok {
			t.Fatal("nonce not found")
		}
		testutil.AssertNonceFormat(t, nonce)

		t.Logf("Challenge nonce format valid: %s", nonce[:10]+"...")
	})

	// Step 2: Client submits with X-PAYMENT header → expects 200
	t.Run("Step2_WithPayment_ReturnsResults", func(t *testing.T) {
		client := &http.Client{}
		req, _ := http.NewRequest("GET", server.URL+"/v1/search?q=rust", nil)
		req.Header.Set("X-PAYMENT", "test-proof")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var results map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			t.Fatalf("failed to decode results: %v", err)
		}

		if results["query"] != "rust" {
			t.Error("query not found in results")
		}
		if _, ok := results["results"]; !ok {
			t.Error("results array missing")
		}

		t.Log("Search results returned successfully")
	})
}

// BenchmarkNonceGeneration measures nonce generation performance
func BenchmarkNonceGeneration(b *testing.B) {
	factory := testutil.NewPaymentChallengeFactory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.Build()
	}
}

// TestNonceUniqueness verifies each request gets a unique nonce
func TestNonceUniqueness(t *testing.T) {
	router := chi.NewRouter()
	searchRoute := router.With(middleware.X402Middleware("0x742d35Cc6634C0532925a3b844Bc9e7595f42521", "eip155:84532", 0.001))
	searchRoute.Get("/v1/search", handlers.SearchHandler)

	server := httptest.NewServer(router)
	defer server.Close()

	nonces := make(map[string]bool)
	for i := 0; i < 10; i++ {
		resp, _ := http.Get(server.URL + "/v1/search?q=test")
		var challenge map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&challenge)
		resp.Body.Close()

		nonce := challenge["nonce"].(string)
		if nonces[nonce] {
			t.Fatalf("duplicate nonce detected: %s", nonce)
		}
		nonces[nonce] = true
	}

	t.Logf("Generated %d unique nonces", len(nonces))
}
