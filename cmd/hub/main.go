package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/octogate/octogate/internal/blockchain"
	"github.com/octogate/octogate/internal/db"
	"github.com/octogate/octogate/internal/handlers"
	mw "github.com/octogate/octogate/internal/middleware"
)

//go:embed web
var webFS embed.FS

func main() {
	port := flag.String("port", "8080", "HTTP port for the Hub")
	flag.Parse()

	// ========== Database Setup ==========
	database, err := db.New()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Printf("Database connected: %v", database.Stats())

	// ========== x402 Configuration ==========
	payTo := os.Getenv("PAY_TO")
	if payTo == "" {
		log.Fatal("PAY_TO environment variable not set")
	}

	network := os.Getenv("NETWORK")
	if network == "" {
		network = "eip155:84532" // Base Sepolia testnet default
	}

	priceStr := os.Getenv("PRICE")
	if priceStr == "" {
		priceStr = "0.001"
	}
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		log.Fatalf("Invalid PRICE: %v", err)
	}

	facilitatorURL := os.Getenv("X402_FACILITATOR_URL")
	if facilitatorURL == "" {
		facilitatorURL = "https://x402.org/facilitator"
	}

	// ========== Blockchain Listener Setup ==========
	rpcURL := os.Getenv("RPC_PRIMARY")
	if rpcURL == "" {
		rpcURL = "wss://base-sepolia.g.alchemy.com/v2/demo" // Demo endpoint
	}

	fallbackRPCURL := os.Getenv("RPC_FALLBACK")
	if fallbackRPCURL == "" {
		fallbackRPCURL = "wss://base-sepolia.infura.io/ws/v3/demo" // Demo endpoint
	}

	settlementContractStr := os.Getenv("SETTLEMENT_CONTRACT")
	if settlementContractStr == "" {
		log.Fatalf("SETTLEMENT_CONTRACT environment variable not set")
	}

	settlementContract := common.HexToAddress(settlementContractStr)

	listener, err := blockchain.NewEventListener(rpcURL, fallbackRPCURL, settlementContract, database.DB())
	if err != nil {
		log.Fatalf("Failed to initialize blockchain listener: %v", err)
	}

	err = listener.Start()
	if err != nil {
		log.Fatalf("Failed to start blockchain listener: %v", err)
	}

	log.Printf("x402 Hub configured: payTo=%s, network=%s, price=%.6f", payTo, network, price)
	log.Printf("x402 Facilitator URL: %s", facilitatorURL)
	log.Printf("Settlement contract: %s", settlementContractStr)
	log.Printf("Blockchain listener started (RPC: %s)", rpcURL)

	// ========== HTTP Router Setup ==========
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// ========== Web UI ==========
	fsys, _ := fs.Sub(webFS, "web")
	router.Handle("/hub", http.FileServer(http.FS(fsys)))
	router.Handle("/hub/*", http.FileServer(http.FS(fsys)))

	// Public endpoints
	router.Get("/health", handlers.HealthHandlerWithStore(database))

	router.Get("/.well-known/x402.json", handlers.ManifestHandler)
	router.Get("/llms.txt", handlers.LlmsTxtHandler)
	router.Get("/v1/tools", handlers.ToolsHandler)

	// API endpoints (will trigger 402 flow)
	// Wire x402 middleware on /v1/search
	searchRoute := router.With(mw.X402Middleware(payTo, network, price, facilitatorURL))
	searchRoute.Get("/v1/search", handlers.SearchHandlerWithStore(database))

	// Other API endpoints (placeholder)
	router.Get("/v1/crawl", handlers.CrawlHandler)
	router.Get("/v1/scrape", handlers.ScrapeHandler)
	router.Get("/v1/enrich", handlers.EnrichHandler)

	// Admin
	router.Get("/v1/balance/{address}", handlers.BalanceHandler)

	// ========== Metrics Endpoint ==========
	router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Wire Prometheus metrics
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# octogate metrics (not yet implemented)\n")
	})

	// ========== Start Server ==========
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("octogate Hub listening on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
