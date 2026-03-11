# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**octogate** is a pay-per-call API gateway with two main components:

1. **x402 CLI** - Command-line tool for AI agents to access premium APIs without API keys or signups
2. **octogate Hub** - HTTP server implementing the x402 payment protocol (HTTP 402 Payment Required)

The x402 protocol enables clients to pay per API call using blockchain payments (EIP-3009 on Base, Solana). Payment is sent to the Hub, which verifies it and returns the requested data.

## Build & Development Commands

```bash
# Build binaries
make build-cli          # Builds ./bin/x402
make build-hub          # Builds ./bin/octogate-hub

# Development mode (rebuilds and runs)
make dev-cli search "query"    # Run CLI command
make dev-hub                   # Start Hub server (port 8080)

# Other commands
make test              # Run all tests
make clean             # Remove ./bin directory
make install-cli       # Install x402 to ~/bin/x402
make help              # Show all available commands
```

## Architecture & Code Organization

### Directory Structure

```
cmd/
├── x402/              # CLI binary entry point (main.go)
│   └── commands/      # Cobra subcommands (search, crawl, balance, tools, etc.)
├── hub/               # HTTP server entry point (main.go)

internal/
└── handlers/          # HTTP endpoint handlers for the Hub
    └── handlers.go    # All handler implementations
```

### CLI Architecture (cmd/x402/)

- **main.go**: Sets up Cobra root command with version info and registers all subcommands
- **commands/*.go**: Each file defines a subcommand (NewSearchCmd, NewCrawlCmd, etc.)
  - Currently most commands return placeholder responses (marked as TODO)
  - Uses Cobra's RunE pattern for error handling
  - Commands support `--json` flag for output formatting

### Hub Architecture (cmd/hub/)

- **main.go**: Sets up Chi router with middleware (Logger, Recoverer)
- Routes defined:
  - **Public endpoints**: `/health`, `/.well-known/x402.json`, `/llms.txt`, `/v1/tools`
  - **API endpoints** (trigger 402 flow): `/v1/search`, `/v1/crawl`, `/v1/scrape`, `/v1/enrich`
  - **Admin**: `/v1/balance/{address}`

### Handlers (internal/handlers/)

- **handlers.go**: Contains all HTTP endpoint implementations
  - Metadata endpoints return x402 manifest (version 1, currency, networks, tools)
  - API endpoints return HTTP 402 Payment Required with payment challenge
  - Payment challenge includes: x402Version, price, currency, networks, payTo address, nonce

## Key Dependencies

- **cobra**: CLI framework for x402 commands
- **chi/v5**: HTTP router for Hub
- **go-ethereum**: Ethereum client for EIP-3009 payment signing
- **go-redis/v9**: Redis client for state management (planned: nonce dedup)
- **log15**: Structured logging

## Implementation Status

The project is in early alpha (v0.1.0-alpha). Core infrastructure is in place:
- ✅ CLI and Hub structure
- ✅ HTTP endpoint routing and 402 flow
- ❌ Actual command implementations (still placeholders)
- ❌ Payment verification and execution
- ❌ Redis integration for nonce management
- ❌ Config file support

## Common Development Tasks

### Adding a New CLI Command

1. Create `cmd/x402/commands/newcommand.go`
2. Implement `NewNewcommandCmd()` returning `*cobra.Command`
3. Add the command to `cmd/x402/main.go` root.AddCommand()

### Adding a New Hub Endpoint

1. Add handler function to `internal/handlers/handlers.go`
2. Register route in `cmd/hub/main.go` router
3. Return 402 Payment Required for paid endpoints

### Testing

- Run all tests: `make test`
- Run single package tests: `go test ./internal/handlers`
- Run single test: `go test -run TestName ./path`

## Go Version

Go 1.22 (defined in go.mod)
