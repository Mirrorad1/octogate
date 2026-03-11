package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/octogate/octogate/internal/db"
)

// HealthHandler returns the health status of the Hub.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HealthHandlerWithStore returns a health handler that checks the database
func HealthHandlerWithStore(store db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health, err := store.CheckHealth(r.Context())
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "database": health})
	}
}

// ManifestHandler returns the x402 manifest.
func ManifestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	manifest := map[string]interface{}{
		"x402Version": 1,
		"hub":         "https://octogate.dev",
		"currency":    "USDC",
		"networks":    []string{"base", "solana"},
		"tools":       []interface{}{},
	}
	_ = json.NewEncoder(w).Encode(manifest)
}

// LlmsTxtHandler returns available tools in plain text format.
func LlmsTxtHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(`# octogate Hub — Available Tools

> Pay-per-call API access for AI agents.

## search
- Description: Real-time web search
- Price: $0.001 per call
- Usage: x402 search "query" --json

## crawl
- Description: Full-page content extraction
- Price: $0.002 per call
- Usage: x402 crawl <url> --json
`))
}

// ToolsHandler returns available tools as JSON.
func ToolsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tools := []map[string]interface{}{
		{
			"name":        "search",
			"path":        "/v1/search",
			"price":       "0.001",
			"description": "Real-time web search",
		},
		{
			"name":        "crawl",
			"path":        "/v1/crawl",
			"price":       "0.002",
			"description": "Full-page content extraction",
		},
	}
	_ = json.NewEncoder(w).Encode(tools)
}

// SearchHandler returns search results for a given query.
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "q parameter required"})
		return
	}

	// Return mock search results
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	results := map[string]interface{}{
		"query": query,
		"results": []map[string]interface{}{
			{
				"title":       "Example Search Result 1",
				"url":         "https://example.com/1",
				"description": "This is a mock search result for: " + query,
			},
			{
				"title":       "Example Search Result 2",
				"url":         "https://example.com/2",
				"description": "Another mock result related to: " + query,
			},
		},
		"paid": true,
	}

	_ = json.NewEncoder(w).Encode(results)
}

// SearchHandlerWithStore returns a search handler with database access
func SearchHandlerWithStore(store db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "q parameter required"})
			return
		}

		// In production, you'd:
		// 1. Get nonce from X-PAYMENT header
		// 2. Look up pending settlement
		// 3. Wait for blockchain confirmation
		// 4. Return results

		ctx, cancel := context.WithTimeout(r.Context(), 5000000000) // 5 seconds
		defer cancel()

		// Example: Check if we have a pending settlement for this request
		nonce := r.Header.Get("X-NONCE")
		if nonce != "" {
			settlement, _ := store.GetPendingSettlement(ctx, nonce)
			_ = settlement // Use for validation
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		results := map[string]interface{}{
			"query": query,
			"results": []map[string]interface{}{
				{
					"title":       "Example Search Result 1",
					"url":         "https://example.com/1",
					"description": "This is a mock search result for: " + query,
				},
				{
					"title":       "Example Search Result 2",
					"url":         "https://example.com/2",
					"description": "Another mock result related to: " + query,
				},
			},
			"paid": true,
		}

		_ = json.NewEncoder(w).Encode(results)
	}
}

// CrawlHandler returns a 402 Payment Required response for crawl requests.
func CrawlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	_ = json.NewEncoder(w).Encode(map[string]string{"price": "0.002"})
}

// ScrapeHandler returns a 402 Payment Required response for scrape requests.
func ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	_ = json.NewEncoder(w).Encode(map[string]string{"price": "0.001"})
}

// EnrichHandler returns a 402 Payment Required response for enrich requests.
func EnrichHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	_ = json.NewEncoder(w).Encode(map[string]string{"price": "0.005"})
}

// BalanceHandler returns the balance for a given wallet address.
func BalanceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"balance": "TODO"})
}
