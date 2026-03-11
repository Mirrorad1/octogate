package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/octogate/octogate/internal/testutil"
)

// TestHealthHandlerWithStore verifies health check with database
func TestHealthHandlerWithStore(t *testing.T) {
	// Use mock store instead of real database
	mockStore := testutil.NewMockStore()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := HealthHandlerWithStore(mockStore)
	handler(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	testutil.AssertContentType(t, w.Header().Get("Content-Type"), "application/json")

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Error("status should be 'ok'")
	}
	if _, ok := resp["database"]; !ok {
		t.Error("database health info missing")
	}
}

// TestSearchHandlerWithStore demonstrates dependency injection pattern
func TestSearchHandlerWithStore(t *testing.T) {
	mockStore := testutil.NewMockStore()

	tests := []struct {
		name     string
		query    string
		nonce    string
		wantCode int
		setupDB  func(*testutil.MockStore)
	}{
		{
			name:     "ValidQueryWithoutNonce",
			query:    "rust patterns",
			wantCode: http.StatusOK,
			setupDB:  func(s *testutil.MockStore) {},
		},
		{
			name:     "ValidQueryWithNonce",
			query:    "go concurrency",
			nonce:    "0x" + "a" + "b"*63,
			wantCode: http.StatusOK,
			setupDB: func(s *testutil.MockStore) {
				// Pre-populate a pending settlement
				ctx := context.Background()
				s.RecordPendingSettlement(
					ctx,
					"0x" + "a" + "b"*63,
					"req-123",
					"0x" + "c"*40,
					"0x" + "d"*40,
					0.001,
					9999999999,
				)
			},
		},
		{
			name:     "EmptyQuery",
			query:    "",
			wantCode: http.StatusBadRequest,
			setupDB:  func(s *testutil.MockStore) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := testutil.NewMockStore()
			tt.setupDB(mockStore)

			handler := SearchHandlerWithStore(mockStore)

			url := "/v1/search?q=" + tt.query
			req := httptest.NewRequest("GET", url, nil)
			if tt.nonce != "" {
				req.Header.Set("X-NONCE", tt.nonce)
			}
			w := httptest.NewRecorder()

			handler(w, req)

			testutil.AssertStatusCode(t, w.Code, tt.wantCode)
			testutil.AssertContentType(t, w.Header().Get("Content-Type"), "application/json")

			if tt.wantCode == http.StatusOK {
				var result map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode: %v", err)
				}
				if result["query"] != tt.query {
					t.Error("query not echoed")
				}
			}
		})
	}
}

// TestHandlerWithMockStoreIntegration demonstrates full integration pattern
func TestHandlerWithMockStoreIntegration(t *testing.T) {
	// This test shows how handlers can be tested with mock stores
	// without needing a real database connection

	mockStore := testutil.NewMockStore()

	// Record a pending settlement
	ctx := context.Background()
	err := mockStore.RecordPendingSettlement(
		ctx,
		"0x" + "a"*64,
		"req-1",
		"0x" + "b"*40,
		"0x" + "c"*40,
		0.001,
		9999999999,
	)
	if err != nil {
		t.Fatalf("failed to record settlement: %v", err)
	}

	// Verify it was recorded
	settlement, err := mockStore.GetPendingSettlement(ctx, "0x"+"a"*64)
	if err != nil {
		t.Fatalf("failed to get settlement: %v", err)
	}
	if settlement == nil {
		t.Fatal("settlement not found")
	}
	if settlement["status"] != "awaiting_proof" {
		t.Errorf("wrong status: %v", settlement["status"])
	}

	// Now test that SearchHandlerWithStore can access this
	handler := SearchHandlerWithStore(mockStore)
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	req.Header.Set("X-NONCE", "0x"+"a"*64)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	t.Log("Integration test passed: handler can access mock store data")
}

// BenchmarkHandlerWithMockStore measures handler performance with mock store
func BenchmarkHandlerWithMockStore(b *testing.B) {
	mockStore := testutil.NewMockStore()
	handler := SearchHandlerWithStore(mockStore)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/search?q=benchmark", nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}
}
