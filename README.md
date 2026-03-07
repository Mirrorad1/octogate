# Octogate: Non-Custodial x402 Payment Infrastructure

[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue)](go.mod)
[![Status](https://img.shields.io/badge/status-early%20alpha-red)](IMPLEMENTATION_ROADMAP.md)

Octogate is a non-custodial payment infrastructure layer for AI agents to transact autonomously. Built on the [x402 HTTP 402 Payment Protocol](https://github.com/coinbase/x402) with USDC settlement on blockchain (Base/Solana).

## Core Features

- **Non-custodial**: Funds move directly agent → provider via immutable smart contract
- **Atomic settlements**: Payment + API delivery in single transaction or fail together
- **Deterministic**: Smart contract execution; no manual intervention or discretion
- **Auditable**: Complete settlement trail on-chain, queryable forever
- **Fee-transparent**: Percentage-based fees taken at settlement time

## Architecture

```
Agent Request
    ↓
Octogate Hub (reverse proxy)
    ↓
Provider API (HTTP 402 response)
    ↓
Payment Challenge (price + nonce)
    ↓
Agent Signs (EIP-191 signature)
    ↓
Smart Contract Settlement (atomic)
    ├─ Transfer USDC to Provider
    ├─ Transfer Fee to Octogate
    └─ Emit PaymentSettled Event
    ↓
Hub Listener (confirms event)
    ↓
Return API Response
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design.

## Quick Start

### Prerequisites

- **Go 1.22+**
- **PostgreSQL 14+** (for transaction tracking)
- **Foundry** (for smart contract testing/deployment)
- **Base Sepolia testnet ETH + USDC** (for testing)

### Development Setup

```bash
# Clone and setup
git clone https://github.com/yourusername/octogate
cd octogate

# Configure environment
cp .env.example .env
nano .env  # Fill in RPC endpoints, database URL, etc.

# Build binaries
make build-cli build-hub

# Optionally: Install CLI globally
make install-cli

# Run tests
make test
```

### Local Testing

```bash
# Terminal 1: Start Hub
PAY_TO=0x742d35Cc6634C0532925a3b844Bc9e7595f42521 \
  NETWORK=eip155:84532 \
  SETTLEMENT_CONTRACT=0x... \
  DATABASE_URL=postgresql://... \
  make dev-hub

# Terminal 2: Test CLI
EVM_PRIVATE_KEY=0x... \
  X402_HUB=http://localhost:8080 \
  make dev-cli search "rust patterns"

# Terminal 3: Check status
EVM_PRIVATE_KEY=0x... make dev-cli status
```

## Project Structure

```
octogate/
├── cmd/
│   ├── x402/                    # CLI binary
│   │   ├── main.go              # Cobra CLI entry point
│   │   └── commands/            # Subcommands (search, crawl, balance, status, etc.)
│   └── hub/                     # Hub server
│       └── main.go              # Chi router + blockchain listener
├── internal/
│   ├── blockchain/
│   │   └── listener.go          # PaymentSettled event monitoring
│   ├── db/
│   │   └── client.go            # PostgreSQL client + queries
│   ├── handlers/
│   │   └── handlers.go          # HTTP endpoint implementations
│   └── middleware/
│       └── x402.go              # x402 payment protocol middleware
├── contracts/
│   ├── SettlementContract.sol   # Smart contract (immutable, no proxy)
│   ├── foundry.toml
│   └── test/                    # Foundry tests
├── migrations/
│   ├── 001_create_pending_settlements.sql
│   ├── 002_create_transactions.sql
│   ├── 003_create_transaction_events.sql
│   └── 004_create_reputation.sql
├── ARCHITECTURE.md              # System design (15 sections, 200+ lines)
├── IMPLEMENTATION_ROADMAP.md    # PoC plan (day-by-day with checklist)
├── ANALYSIS_SUMMARY.md          # Multi-agent analysis (security, compliance, ops)
├── .env.example                 # Configuration template
├── Makefile
└── README.md
```

## API Endpoints

### Public Endpoints

```bash
# Health check
curl http://localhost:8080/health

# x402 manifest (price, networks, contract address)
curl http://localhost:8080/.well-known/x402.json

# Available tools
curl http://localhost:8080/v1/tools

# LLM-friendly tool list (plain text)
curl http://localhost:8080/llms.txt
```

### Payment-Protected Endpoints

```bash
# Search (requires x402 payment flow)
curl http://localhost:8080/v1/search?q=rust+patterns

# Response: HTTP 402 with payment challenge
# Agent signs challenge and submits proof
# Hub verifies settlement on-chain
# Hub returns HTTP 200 with results
```

### Admin Endpoints

```bash
# Pause Hub (circuit breaker)
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/admin/pause

# Resume Hub
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/admin/resume

# Metrics (Prometheus format)
curl http://localhost:8080/metrics
```

## CLI Usage

```bash
# Search (requires EVM_PRIVATE_KEY to pay)
x402 search "hello world"

# Check system status
x402 status

# Show available tools
x402 tools

# Check wallet balance
x402 balance

# Display version
x402 --version
```

## Configuration

All configuration is environment-variable based. See [.env.example](.env.example) for complete list.

**Essential variables**:

| Variable | Purpose | Example |
|----------|---------|---------|
| `PAY_TO` | Fee wallet (where settlement fees go) | `0x742d35Cc6634C0532925a3b844Bc9e7595f42521` |
| `NETWORK` | Blockchain network | `eip155:84532` (Base Sepolia) |
| `SETTLEMENT_CONTRACT` | Smart contract address | `0x123456...` |
| `DATABASE_URL` | PostgreSQL connection | `postgresql://user:pass@host/db` |
| `RPC_PRIMARY` | WebSocket RPC | `wss://base-sepolia.g.alchemy.com/v2/KEY` |
| `RPC_FALLBACK` | Backup RPC | `wss://base-sepolia.infura.io/ws/v3/KEY` |

## Implementation Status

### ✅ Complete

- [x] Smart contract implementation (Solidity)
- [x] HTTP endpoint scaffolding
- [x] CLI command structure
- [x] Database schema design
- [x] Blockchain listener architecture

### 🔄 In Progress (Tier 1 Blockers)

- [ ] Smart contract deployment to Base Sepolia
- [ ] Payment verification wiring (Coinbase x402 SDK)
- [ ] Blockchain listener + database integration

### 📋 Planned (Before PoC Launch)

- [ ] Structured JSON logging
- [ ] Prometheus metrics
- [ ] Load testing on Base Sepolia
- [ ] Deployment runbook testing

**Timeline**: 2-4 weeks to operational PoC

## Testing

### Smart Contract

```bash
# Run Foundry tests
forge test

# Run with verbose output
forge test -vvv

# Fuzz testing (10,000 runs)
forge test --fuzz-runs 10000

# Coverage report
forge coverage
```

### Hub & CLI

```bash
# All tests
make test

# Specific package
go test ./internal/db -v

# With coverage
go test -cover ./...
```

## Deployment

### Local Development

```bash
# Start PostgreSQL
docker run --name octogate-postgres \
  -e POSTGRES_USER=octogate \
  -e POSTGRES_PASSWORD=octogate \
  -e POSTGRES_DB=octogate_sepolia \
  -p 5432:5432 \
  postgres:15

# Run migrations
psql octogate_sepolia < migrations/*.sql

# Start Hub
make dev-hub

# Run CLI
make dev-cli search "test query"
```

### Base Sepolia Testnet

See [IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md) for complete deployment runbook with:
- Smart contract deployment
- Database setup checklist
- Hub configuration
- Monitoring setup
- Load testing procedure
- Runbook validation

### Base Mainnet (Post-PoC)

Before mainnet deployment:
1. Smart contract audit by Tier 1 firm
2. Full e2e testing with real USDC
3. Database backup/restore validation
4. Monitoring + alerting configured
5. Legal review completed

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Detailed system design (15 sections)
- **[IMPLEMENTATION_ROADMAP.md](IMPLEMENTATION_ROADMAP.md)** - Day-by-day PoC plan with checklist
- **[ANALYSIS_SUMMARY.md](ANALYSIS_SUMMARY.md)** - Multi-agent analysis (security, compliance, operations)

## Regulatory & Compliance

### Non-Custodial Design

Octogate is designed to be a **technology platform**, not a financial services provider:

- ✅ **Non-custodial**: Funds flow directly agent → provider via smart contract
- ✅ **No discretionary control**: Contract logic is immutable, deterministic
- ✅ **Transparent fees**: Percentage-based, taken at settlement time
- ✅ **On-chain verifiable**: All settlements auditable on blockchain
- ❌ **Not a custodian**: Doesn't hold transaction amounts, only earned fees

See [ANALYSIS_SUMMARY.md](ANALYSIS_SUMMARY.md) for detailed regulatory analysis.

### Identity & KYC

- **Agent Registration**: Required at first transaction (deploying entity)
- **Provider Registration**: Self-service with OFAC screening
- **Reputation System**: Informational only (no monetary value, non-transferable)

### Data Retention

- **Transactions**: Immutable audit log, 5+ years retention
- **Events**: Append-only log for reconciliation
- **Reputation**: Derived scores (recalculated)

## Security

### Smart Contract

- Immutable design (no upgrade proxy)
- Uses OpenZeppelin audited libraries (ERC20, ECDSA)
- External audit required before mainnet
- Fuzz testing with 10,000+ runs

### Operational

- Private keys never in logs/config
- Database connections encrypted (TLS)
- Admin endpoints require bearer token
- RPC failover prevents single point of failure

### Bug Reports

Found a vulnerability? Email **security@octogate.dev** privately.

## License

MIT License - see [LICENSE](LICENSE)

## Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/octogate/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/octogate/discussions)
- **Documentation**: [Full docs](docs/)

---

**Building payment infrastructure for autonomous AI agents.**
# octogate
