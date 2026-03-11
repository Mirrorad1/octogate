package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	x402 "github.com/coinbase/x402/go"
	x402http "github.com/coinbase/x402/go/http"
	"github.com/coinbase/x402/go/mechanisms/evm/exact/server"
)

// X402Middleware creates a Chi-compatible middleware that enforces x402 payment.
// Uses the official Coinbase x402 library for spec-compliant payment verification.
func X402Middleware(payTo, network string, price float64, facilitatorURL string) func(http.Handler) http.Handler {
	// Create the EVM payment scheme server
	evmScheme := server.NewExactEvmScheme()

	// Configure payment requirements for this route
	paymentOptions := []x402http.PaymentOption{
		{
			Scheme:  "exact-evm",
			Network: x402.Network(network),
			PayTo:   payTo,
			Price:   price,
		},
	}

	// Set up routes configuration - using wildcard pattern for /v1/* endpoints
	routesConfig := x402http.RoutesConfig{
		"GET /v1/search": {
			Accepts:     paymentOptions,
			Resource:    "search",
			Description: "Search premium API results",
		},
	}

	// Create HTTP resource server with Coinbase library
	httpServer := x402http.Newx402HTTPResourceServer(routesConfig,
		x402.WithSchemeServer(x402.Network(network), evmScheme),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Create an HTTP adapter for the request
			adapter := &netHTTPAdapter{req: r}

			// Create HTTP request context
			reqCtx := x402http.HTTPRequestContext{
				Adapter: adapter,
				Path:    r.URL.Path,
				Method:  r.Method,
			}

			// Process the request using Coinbase x402 library
			result := httpServer.ProcessHTTPRequest(ctx, reqCtx, &x402http.PaywallConfig{})

			// Handle the processing result
			switch result.Type {
			case x402http.ResultNoPaymentRequired:
				// Payment not required, proceed to next handler
				next.ServeHTTP(w, r)

			case x402http.ResultPaymentVerified:
				// Payment verified, proceed to next handler
				next.ServeHTTP(w, r)

			case x402http.ResultPaymentError:
				// Payment error or 402 required
				if result.Response != nil {
					// Write the HTTP response from the library
					for key, value := range result.Response.Headers {
						w.Header().Set(key, value)
					}
					w.WriteHeader(result.Response.Status)

					// Write response body
					if result.Response.Body != nil {
						if err := json.NewEncoder(w).Encode(result.Response.Body); err != nil {
							log.Printf("Failed to encode response body: %v", err)
						}
					}
				} else {
					// Fallback if no response instructions provided
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusPaymentRequired)
					json.NewEncoder(w).Encode(map[string]string{"error": "payment required"})
				}
			}
		})
	}
}

// netHTTPAdapter implements x402http.HTTPAdapter for net/http
type netHTTPAdapter struct {
	req *http.Request
}

// GetHeader returns the value of the named HTTP request header
func (a *netHTTPAdapter) GetHeader(name string) string {
	return a.req.Header.Get(name)
}

// GetMethod returns the HTTP method (GET, POST, etc.)
func (a *netHTTPAdapter) GetMethod() string {
	return a.req.Method
}

// GetPath returns the request path (without query string)
func (a *netHTTPAdapter) GetPath() string {
	return a.req.URL.Path
}

// GetURL returns the full request URL
func (a *netHTTPAdapter) GetURL() string {
	return a.req.URL.String()
}

// GetAcceptHeader returns the Accept header value
func (a *netHTTPAdapter) GetAcceptHeader() string {
	return a.req.Header.Get("Accept")
}

// GetUserAgent returns the User-Agent header value
func (a *netHTTPAdapter) GetUserAgent() string {
	return a.req.Header.Get("User-Agent")
}

