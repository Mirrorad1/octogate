# Dependency Injection Pattern in Octogate

## Overview

The codebase uses Go's interface-based dependency injection to enable testability, flexibility, and clean separation of concerns.

## Core Interfaces

### 1. Store Interface (internal/db/store.go)

Defines all database operations required by the Hub:

```go
type Store interface {
    RecordPendingSettlement(ctx context.Context, ...) error
    UpdateSettlementConfirmed(ctx context.Context, ...) error
    GetPendingSettlement(ctx context.Context, ...) (map[string]interface{}, error)
    RecordTransaction(ctx context.Context, ...) error
    RecordEvent(ctx context.Context, ...) error
    CheckHealth(ctx context.Context) (map[string]interface{}, error)
    Close() error
}
```

**Implementations:**
- `db.Client` - Real PostgreSQL implementation
- `testutil.MockStore` - In-memory mock for testing

### 2. Listener Interface (internal/blockchain/listener_interface.go)

Defines blockchain event monitoring:

```go
type Listener interface {
    Start() error
    Stop()
    SettlementEvents() <-chan PaymentSettledEvent
    Errors() <-chan error
    LastProcessedBlock() uint64
    GetLastProcessedBlock(ctx context.Context) (uint64, error)
}
```

**Implementations:**
- `blockchain.EventListener` - Real WebSocket + polling listener
- (Future) `testutil.MockListener` - Fake events for testing

## Handler Pattern

### Without Dependency Injection (Old)

```go
func SearchHandler(w http.ResponseWriter, r *http.Request) {
    // Can't access database
    // Can't be tested without complex setup
}
```

### With Dependency Injection (New)

```go
// Constructor that captures dependencies
func SearchHandlerWithStore(store Store) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // store is available and can be mocked
        settlement, _ := store.GetPendingSettlement(r.Context(), nonce)
        // ... use settlement ...
    }
}
```

**Usage in Hub:**

```go
// Wire with real database
router.Get("/v1/search", handlers.SearchHandlerWithStore(database))

// Wire with mock in tests
mockStore := testutil.NewMockStore()
handler := handlers.SearchHandlerWithStore(mockStore)
```

## Testing Examples

### Before (Hard to Test)

```go
// Had to set up real database
// Had to worry about test data isolation
// Had to clean up after each test
func TestSearchHandler(t *testing.T) {
    // Complex setup...
}
```

### After (Simple Mocking)

```go
func TestSearchHandlerWithStore(t *testing.T) {
    // Just create a mock
    mockStore := testutil.NewMockStore()

    // Pre-populate test data
    mockStore.RecordPendingSettlement(ctx, nonce, ...)

    // Test with mock
    handler := handlers.SearchHandlerWithStore(mockStore)
    req := httptest.NewRequest("GET", "/v1/search", nil)
    w := httptest.NewRecorder()
    handler(w, req)

    // Assertions
    if w.Code != http.StatusOK { ... }
}
```

## Benefits

1. **Testability**: Tests don't need real database or blockchain
2. **Flexibility**: Easy to swap implementations (PostgreSQL → MongoDB, etc.)
3. **Isolation**: Unit tests are fast and don't have side effects
4. **Documentation**: Interfaces clearly show what dependencies handlers need
5. **Maintainability**: Changes to Store don't break handlers if interface stays same

## Extending the Pattern

### Adding a New Handler with Dependencies

```go
// 1. Define the handler constructor
func MyNewHandler(store Store, listener Listener) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Both dependencies available
        settlement, _ := store.GetPendingSettlement(r.Context(), ...)
        events := listener.SettlementEvents()
        // ...
    }
}

// 2. Wire in Hub
router.Get("/v1/mynew", handlers.MyNewHandler(database, listener))

// 3. Test with mocks
mockStore := testutil.NewMockStore()
mockListener := testutil.NewMockListener()
handler := handlers.MyNewHandler(mockStore, mockListener)
// Test...
```

### Creating a New Mock

```go
// 1. Create a new mock type
type MockService interface {
    DoSomething(ctx context.Context) error
}

type MockServiceImpl struct {
    // ...
}

func (m *MockServiceImpl) DoSomething(ctx context.Context) error {
    // Return test data or errors
}

// 2. Verify it implements the interface
var _ MockService = (*MockServiceImpl)(nil)

// 3. Use in tests
mock := &MockServiceImpl{}
handler := handlers.MyHandler(mock)
```

## Backward Compatibility

Old handlers (without dependency injection) are kept for compatibility:

```go
// Old style - still works, no dependencies
func SearchHandler(w http.ResponseWriter, r *http.Request) { ... }

// New style - with dependencies for testability
func SearchHandlerWithStore(store Store) http.HandlerFunc { ... }
```

This allows gradual migration without breaking existing code.

## Pattern Summary

| Aspect | Pattern |
|--------|---------|
| **Injection Method** | Constructor closures |
| **Interface Location** | `internal/{package}/{interface_name}.go` or `internal/{package}/store.go` |
| **Mock Location** | `internal/testutil/mocks.go` |
| **Handler Style** | `HandlerNameWithDep(dep Dep) http.HandlerFunc` |
| **Verification** | `var _ Interface = (*Implementation)(nil)` at end of file |

## Related Files

- `internal/db/store.go` - Store interface definition
- `internal/blockchain/listener_interface.go` - Listener interface definition
- `internal/testutil/mocks.go` - Mock implementations
- `internal/handlers/handlers_with_store_test.go` - Example tests using DI
- `cmd/hub/main.go` - Wiring handlers with real dependencies
