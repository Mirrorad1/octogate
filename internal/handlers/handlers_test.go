package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octogate/octogate/internal/testutil"
)

// TestSearchHandlerWithQuery tests search handler returns results for valid query
func TestSearchHandlerWithQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantCode int
		validate func(t *testing.T, body string)
	}{
		{
			name:     "ValidQuery",
			query:    "rust patterns",
			wantCode: http.StatusOK,
			validate: func(t *testing.T, body string) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(body), &result); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}

				if result["query"] != "rust patterns" {
					t.Error("query not echoed back")
				}
				if _, ok := result["results"]; !ok {
					t.Error("missing results")
				}
			},
		},
		{
			name:     "EmptyQuery",
			query:    "",
			wantCode: http.StatusBadRequest,
			validate: func(t *testing.T, body string) {
				var result map[string]interface{}
				json.Unmarshal([]byte(body), &result)
				if _, ok := result["error"]; !ok {
					t.Error("error not present for empty query")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/search?q="+tt.query, nil)
			w := httptest.NewRecorder()

			SearchHandler(w, req)

			testutil.AssertStatusCode(t, w.Code, tt.wantCode)
			testutil.AssertContentType(t, w.Header().Get("Content-Type"), "application/json")

			if tt.validate != nil {
				tt.validate(t, w.Body.String())
			}
		})
	}
}

// TestHandlersReturnJSON verifies all handlers set correct Content-Type
func TestHandlersReturnJSON(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"Health":    HealthHandler,
		"Manifest":  ManifestHandler,
		"Tools":     ToolsHandler,
		"Search":    SearchHandler,
		"Crawl":     CrawlHandler,
		"Scrape":    ScrapeHandler,
		"Enrich":    EnrichHandler,
		"Balance":   BalanceHandler,
	}

	for name, handler := range handlers {
		t.Run(name+"ReturnsJSON", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" && contentType != "text/plain; charset=utf-8" {
				t.Errorf("%s: wrong Content-Type %q", name, contentType)
			}
		})
	}
}

// TestPaymentRequiredHandlers verify 402 endpoints return correct status
func TestPaymentRequiredHandlers(t *testing.T) {
	handlers := map[string]http.HandlerFunc{
		"Crawl":  CrawlHandler,
		"Scrape": ScrapeHandler,
		"Enrich": EnrichHandler,
	}

	for name, handler := range handlers {
		t.Run(name+"Returns402", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			testutil.AssertStatusCode(t, w.Code, http.StatusPaymentRequired)
		})
	}
}

// TestManifestHandlerContent verifies x402 manifest has required fields
func TestManifestHandlerContent(t *testing.T) {
	req := httptest.NewRequest("GET", "/.well-known/x402.json", nil)
	w := httptest.NewRecorder()

	ManifestHandler(w, req)

	var manifest map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&manifest); err != nil {
		t.Fatalf("failed to decode manifest: %v", err)
	}

	requiredFields := []string{"x402Version", "currency", "networks"}
	for _, field := range requiredFields {
		if _, ok := manifest[field]; !ok {
			t.Errorf("manifest missing field: %s", field)
		}
	}
}
