# Octogate: Non-Custodial Payment Infrastructure for AI Agents
## Technical Architecture & Design Specification

---

## 1. Executive Summary

**Octogate** is a non-custodial payment infrastructure layer that facilitates transactions between AI agents and API providers using the x402 HTTP payment protocol with USDC on blockchain (Base/Solana).

**Core Design Principles:**
- **Non-custodial**: Funds move directly from agent wallet → provider wallet via smart contract
- **Atomic settlements**: Payment + delivery happen in single transaction or fail together
- **Deterministic execution**: Smart contract enforces rules; no manual intervention
- **Fee-at-execution**: 2-5% fee taken during settlement, not separate
- **On-chain verifiability**: Complete audit trail on blockchain

---

## 2. System Architecture

### 2.1 High-Level Data Flow

```
┌─────────────┐
│  AI Agent   │
│  (wallet)   │
└──────┬──────┘
       │ 1. API request (no payment)
       ▼
┌──────────────────┐
│  Octogate Hub    │ ◄─── Non-custodial reverse proxy
│  (x402 server)   │
└──────┬───────────┘
       │ 2. Check request → route to provider
       ▼
┌──────────────────┐
│  Provider API    │ (third-party)
└──────┬───────────┘
       │ 3. HTTP 402 (Payment Required) + price
       ▼
┌──────────────────┐
│  Octogate Hub    │ ◄─── Returns 402 to agent
└──────┬───────────┘
       │ 4. Include payment challenge (provider addr, amount, nonce)
       ▼
┌─────────────┐
│  AI Agent   │
│  (wallet)   │ ◄─── Signs payment with private key
└──────┬──────┘
       │ 5. Sends signed payment proof
       ▼
┌──────────────────────────────────────┐
│  Smart Contract (EVM/Solana)         │
│  ┌──────────────────────────────┐    │
│  │ Executes atomically:         │    │
│  │ 1. Verify agent signature    │    │
│  │ 2. Transfer USDC to provider │    │
│  │ 3. Transfer fee to Octogate  │    │
│  │ 4. Emit settlement event     │    │
│  └──────────────────────────────┘    │
└──────┬───────────────────────────────┘
       │ 6. Settlement confirmed on-chain
       ▼
┌──────────────────┐
│  Octogate Hub    │
│  (listener)      │ ◄─── Monitors blockchain for settlement
└──────┬───────────┘
       │ 7. Confirms payment verified
       ▼
┌─────────────┐
│  Provider   │ ◄─── Delivers API response
│  API        │      (already cached)
└──────┬──────┘
       │ 8. Returns response to Octogate
       ▼
┌──────────────────┐
│  Octogate Hub    │
│  (proxy)         │
└──────┬───────────┘
       │ 9. Forwards response to agent
       ▼
┌─────────────┐
│  AI Agent   │ ◄─── Receives API result
└─────────────┘
```

### 2.2 Component Breakdown

#### **Octogate Hub (Server)**
- **Role**: Reverse proxy + payment coordinator
- **Responsibilities**:
  1. Accept incoming API requests from agents (no auth)
  2. Route to provider, catch HTTP 402 response
  3. Extract provider address + price from response
  4. Generate payment challenge (nonce, amount, deadline)
  5. Return 402 to agent with payment instructions
  6. Receive signed payment proof from agent
  7. Submit to blockchain (or agent submits directly)
  8. Monitor blockchain for settlement confirmation
  9. Forward API response once payment confirmed
  10. Record transaction in local database (no funds held)

**Key Architectural Constraint**: Hub never receives agent funds. Hub never receives provider funds. Hub is a **coordinator, not a custodian**.

#### **Agent (AI Model + Wallet)**
- **Role**: Consumer of APIs
- **Wallet**: Ethereum-compatible wallet (private key stored in agent environment)
- **Actions**:
  1. Detect HTTP 402 from Octogate
  2. Parse payment challenge (provider address, amount, nonce, deadline)
  3. Sign payment using private key (EIP-191 or x402 standard)
  4. Serialize payment proof
  5. Submit proof back to Octogate (or directly to blockchain)
  6. Wait for settlement confirmation on-chain
  7. Consume API response

#### **Provider (Third-Party API)**
- **Role**: Service provider
- **No changes required**: Responds with HTTP 402 as per x402 spec
- **Receives payments**: USDC directly to provider wallet address
- **No integration**: No SDK, no account, no login needed
- **Incentive**: Earn USDC without merchant account, KYC, or infrastructure

#### **Smart Contract (Atomic Settlement)**
- **Chain**: Base (Ethereum Layer 2) or Solana
- **Function Signature**:
  ```solidity
  function settlePayment(
      address agent,           // payer
      address provider,        // payee
      uint256 amount,          // USDC amount (6 decimals)
      uint256 feePercent,      // 2-5% (e.g., 250 = 2.5%)
      bytes nonce,             // unique per request
      bytes signature          // agent's signature
  ) external {
      // 1. Verify signature (agent signed this exact call)
      require(isValidSignature(agent, signature, ...));

      // 2. Verify nonce not used before (prevent replay)
      require(!usedNonces[nonce]);
      usedNonces[nonce] = true;

      // 3. Calculate amounts
      uint256 fee = (amount * feePercent) / 10000;
      uint256 providerAmount = amount - fee;

      // 4. Transfer USDC from agent to provider
      usdc.transferFrom(agent, provider, providerAmount);

      // 5. Transfer fee to Octogate
      usdc.transferFrom(agent, octogate, fee);

      // 6. Emit event for Octogate to listen to
      emit PaymentSettled(agent, provider, amount, fee, nonce);
  }
  ```

**Key Properties**:
- **Deterministic**: Execution depends only on inputs, not external state (except deployed contract)
- **Atomic**: All transfers succeed or all fail; no partial execution
- **Non-reversible**: Once executed, settlement is final (blockchain immutability)
- **Signature verification**: Proves agent authorized this specific payment

---

## 3. Protocol Flow (Detailed)

### 3.1 First Request (No Payment History)

```
Agent → Octogate: GET /v1/search?q=rust+patterns
  (standard HTTP request, no auth)

Octogate → Provider API: GET /search?q=rust+patterns
  (proxied request)

Provider API → Octogate: HTTP 402 + x402 metadata
  Headers:
    Payment-Required: true
    Price: 0.001 (USDC)
    Provider-Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f42521
    Currency: USDC
    Networks: eip155:8453 (Base)

Octogate → Agent: HTTP 402 + payment challenge
  Headers:
    WWW-Authenticate: x402
    Price: 0.001 USDC
    Provider: 0x742d...f42521
    Nonce: <random 32 bytes>
    Fee-Percent: 250 (2.5%)
    Settlement-Contract: 0x123...abc (address on Base)
    Deadline: <unix timestamp + 5 minutes>

Agent → [off-chain wallet logic]:
  1. Parse challenge
  2. Sign: EIP-191 signature of (provider, amount, fee%, nonce, deadline, address(settlement_contract))
  3. Serialize as: {agent, provider, amount, feePercent, nonce, signature}

[Agent submits to blockchain or sends back to Octogate]

Option A: Agent submits directly to blockchain
Agent → Settlement Contract (Base): settlePayment(...)
  Contract verifies signature, transfers funds, emits event

Option B: Agent sends back to Octogate, Octogate relays
Agent → Octogate: POST /v1/settle (payment proof + signed data)
Octogate → Settlement Contract: settlePayment(...)
  Contract verifies signature, transfers funds, emits event

Octogate → [Event Listener]: Monitors blockchain
  Listens for PaymentSettled event matching this nonce

Octogate → Local DB: Records transaction
  {
    agent_address: "0x...",
    provider_address: "0x...",
    amount_usdc: 0.001,
    fee_usdc: 0.0000025,
    nonce: "0x...",
    tx_hash: "0x...",
    timestamp: <unix>,
    request_id: "...",
    provider_name: "...",
    status: "settled"
  }

Octogate → Provider API: GET /search?q=rust+patterns
  (fetch actual results, now that payment is confirmed)

Provider API → Octogate: 200 OK + results JSON

Octogate → Agent: 200 OK + results JSON
  Headers:
    X-Payment-Confirmed: 0x<tx_hash>
    X-Provider: <provider_address>
```

### 3.2 Repeated Requests (Reputation Building)

After first payment, agent has a history:
- **Agent reputation**: Number of successful transactions, payment reliability
- **Provider reputation**: Uptime, response quality, arbitration history

On subsequent requests:
- Agent may cache provider's address/price
- Octogate may offer "express" path (pre-approved, faster settlement)
- But each request still requires explicit payment (no subscriptions, no batching)

---

## 4. Custody & Control Analysis

### 4.1 Fund Flow at Each Stage

| Stage | Who Holds Funds | Octogate's Role | Custody? |
|-------|-----------------|-----------------|----------|
| Agent wallet | Agent | — | Agent is custodian |
| Signed proof | Network | Coordinator | No |
| Smart contract execution | Smart contract runtime | Enforcer | No (code is custodian) |
| Provider receives USDC | Provider | — | Provider is custodian |
| Octogate receives fee | Octogate account | Earner | **YES** — Octogate has custody of fees |
| Octogate fee account balance | Octogate | Holder | **YES** |

**Custody Claim**:
- ✅ Octogate is **NOT** custodian of transaction amounts (agent → provider)
- ❌ Octogate **IS** custodian of earned fees (they accumulate in Octogate wallet)

**Implication**: Non-custodial claim applies to the **transaction flow**, but Octogate operates a fee account (small custodial footprint). This strengthens the legal position relative to traditional escrow or payment processors.

### 4.2 Control Over Funds

| Action | Can Octogate Do It? | Can Smart Contract Prevent It? |
|--------|---------------------|--------------------------------|
| Redirect agent → provider payment | No | Yes (code enforces receiver) |
| Freeze agent funds | No | Yes (no freeze function) |
| Delay provider payment | No | Yes (immediate transfer) |
| Reverse transaction | No | No (blockchain immutability) |
| Arbitrate disputes | No | Code-based only (no discretion) |
| Intercept fee | Yes | Code specifies Octogate fee account |

**Control Analysis**: Octogate has **zero discretionary control** over payment execution. The smart contract is deterministic and immutable.

---

## 5. Data Model & Storage

### 5.1 Octogate Local Database

Octogate maintains minimal local state for **coordination & reputation only**. No transaction amounts stored in plaintext at rest (they're on-chain).

**Transactions Table** (audited, immutable log):
```sql
CREATE TABLE transactions (
  id UUID PRIMARY KEY,
  request_id VARCHAR,           -- generated by Octogate
  timestamp BIGINT,              -- unix timestamp
  agent_address VARCHAR(42),     -- 0x... (hashed for privacy)
  provider_address VARCHAR(42),  -- 0x... (public)
  amount_usdc DECIMAL(20, 6),    -- from blockchain event
  fee_usdc DECIMAL(20, 6),       -- calculated from smart contract
  tx_hash VARCHAR(66),           -- blockchain tx hash
  network VARCHAR,               -- "eip155:8453", "solana:mainnet"
  status VARCHAR,                -- "pending", "settled", "failed"
  provider_name VARCHAR,         -- human-readable for logs
  notes TEXT,                    -- error messages if failed
  created_at TIMESTAMP,
  settled_at TIMESTAMP
);
```

**Reputation Table** (derived from transactions):
```sql
CREATE TABLE reputation (
  entity_address VARCHAR(42) PRIMARY KEY,
  entity_type VARCHAR,      -- "agent" or "provider"
  transaction_count INT,
  successful_count INT,
  failed_count INT,
  avg_settlement_time_ms INT,
  uptime_percent DECIMAL,
  last_updated TIMESTAMP
);
```

**API Metadata Table** (provider info):
```sql
CREATE TABLE providers (
  provider_address VARCHAR(42) PRIMARY KEY,
  name VARCHAR,
  endpoint_url VARCHAR,
  price_usdc DECIMAL(20, 6),
  network VARCHAR,
  status VARCHAR,               -- "active", "paused", "blacklisted"
  reputation_score INT,         -- derived from transactions
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### 5.2 Off-Chain State (Agent Wallets)

Agents manage their own:
- Private keys (secured in agent environment)
- USDC balance (in agent's wallet, verified on-chain)
- Transaction history cache (optional, for optimization)
- Nonce tracking (to avoid replay attacks in their signatures)

---

## 6. Network & Blockchain Integration

### 6.1 Supported Networks (PoC)

**Phase 1 (MVP)**:
- **Base Sepolia** (testnet): EVM-compatible L2, low gas fees, Circle's USDC available

**Phase 2 (Mainnet)**:
- **Base Mainnet** (EVM): Production network, fastest block time, cheapest gas
- **Solana Mainnet** (Optional): Different VM model, atomic transactions, parallel execution

### 6.2 Smart Contract Deployment Strategy

**Base Sepolia (PoC)**:
```
Settlement Contract Address: 0x... (TBD)
USDC Token Address: 0x... (Circle's USDC on Base Sepolia)
Octogate Fee Wallet: 0x... (sole recipient of fee transfers)
```

**Verification**:
- Contract code is publicly verifiable on Basescan
- No upgradeable proxy (immutable logic)
- No admin keys (no pause/freeze functions)
- No arbitrary owner transfers (deterministic)

### 6.3 Event Monitoring

Octogate runs a blockchain listener:
```go
// Pseudocode
listener.On("PaymentSettled", func(event) {
    // Verify nonce matches an in-flight request
    // Update transaction record to "settled"
    // Unblock API response delivery
    // Update reputation scores
})
```

**Fallback**: If blockchain listener fails, agent can provide tx_hash in response and Octogate verifies on-demand.

---

## 7. Security Considerations

### 7.1 Signature Verification

**Standard**: EIP-191 (Ethereum message signing)
```
message = abi.encodePacked(
    "\x19Ethereum Signed Message:\n" + len(data),
    keccak256(provider, amount, fee%, nonce, deadline)
)
signature = agent.sign(message)
```

**Contract Verification**:
```solidity
address recoveredSigner = ecrecover(msgHash, v, r, s);
require(recoveredSigner == agent, "Invalid signature");
```

### 7.2 Replay Attack Prevention

**Nonce**: Unique per request, burned after first use
```solidity
require(!usedNonces[nonce], "Nonce already used");
usedNonces[nonce] = true;
```

**Deadline**: Signature includes expiration (5 min default)
```solidity
require(block.timestamp <= deadline, "Payment expired");
```

### 7.3 Double-Spend Prevention

**Atomic Execution**: Smart contract checks balance before transfer
```solidity
require(usdc.balanceOf(agent) >= amount, "Insufficient funds");
usdc.transferFrom(agent, provider, providerAmount);
```

### 7.4 Reentrancy & Contract Vulnerabilities

**Protections**:
- USDC uses OpenZeppelin's ERC20 (audited)
- Settlement contract uses `transferFrom`, not `call` (safe)
- No callbacks to untrusted addresses (provider/agent)
- Check-Effects-Interactions pattern followed

### 7.5 Octogate Hub Security

**API Security**:
- CORS: Allow any origin (agents can be from anywhere)
- Rate limiting: Per agent address, not per IP (IP spoofing possible)
- DDoS: Relay on CDN (e.g., Cloudflare) in production
- Payment replay: Blockchain handles nonce verification

**Database Security**:
- No private keys stored
- No plaintext passwords
- Encrypted connections to blockchain RPC
- Regular backups (non-custodial, so lower risk)

---

## 8. Fee Mechanism

### 8.1 Fee Collection

**Model**: Percentage-based, taken at settlement time
- 2-5% range (configurable per provider partnership)
- Default: 2.5% (250 basis points)

**Example**:
```
Agent wants to pay Provider $1.00 USDC
Octogate fee: 2.5% = $0.025
Provider receives: $0.975
Octogate receives: $0.025
```

**On-Chain Execution**:
```solidity
uint256 feeAmount = (amount * feePercent) / 10000;
uint256 providerAmount = amount - feeAmount;

usdc.transferFrom(agent, provider, providerAmount);
usdc.transferFrom(agent, octogate, feeAmount);
```

### 8.2 Fee Account Management

**Octogate fee account**:
- Receives USDC directly (small custodial footprint)
- Accumulates fees from all transactions
- Can be swept to cold storage periodically
- Auditable on-chain (all transfers visible)

**Withdrawal**: Octogate can transfer from fee account at any time (funds earned, not held in escrow)

---

## 9. Identity & Accountability

### 9.1 Agent Identity

**Required for Compliance**:
- Agent address (public, on-chain)
- Deploying entity (company/individual, off-chain)
  - ⚠️ **TODO**: Define how agents register their deploying entity
  - Proposal: Agent sends signed message at first transaction, claims deploying entity
  - Alternative: Require deploying entity to register agent beforehand

**For Reputation**:
- Transaction history is public (blockchain)
- Reputation score is derived from transaction history
- No KYC required to transact, but identity linkage is enforced for accountability

### 9.2 Provider Identity

**Required**:
- Provider wallet address (on-chain)
- API endpoint (off-chain)
- Human name/contact (optional, for reputation display)

**Registration** (TODO):
- Proposal: Providers self-register via Octogate dashboard
- Verify endpoint returns valid x402 response
- Publish to reputation system

---

## 10. Error Handling & Edge Cases

### 10.1 Payment Failures

**Case: Agent signature is invalid**
```
Agent submits invalid signature →
Smart contract rejects (ecrecover fails) →
Octogate watches blockchain (no event) →
Timeout (5 min) →
Octogate returns error to agent
```

**Case: Agent insufficient balance**
```
Agent balance < payment amount →
Smart contract fails transferFrom →
Octogate detects failed tx →
Octogate returns "Insufficient funds" to agent
```

**Case: Provider wallet address is invalid**
```
Smart contract still executes (addresses are just values) →
USDC transfer to invalid address succeeds (Ethereum quirk) →
Funds effectively burned (provider never receives)
```

⚠️ **Mitigation**: Octogate validates provider address format before including in challenge.

### 10.2 Timeout Handling

**Standard flow**: 5-minute timeout for payment
```
1. Octogate returns 402 with deadline = now + 5min
2. Agent signs & submits payment
3. Octogate waits for blockchain confirmation (12-30s on Base)
4. If no event by deadline, return error
5. Agent can retry (new nonce)
```

### 10.3 Provider Downtime

**Case: Provider API is unreachable**
```
Agent → Octogate → Provider (timeout) →
Octogate returns HTTP 504 (or 400) to agent before 402
(No payment required if provider is down)
```

**Case: Provider returns HTTP 402, then becomes unreachable**
```
Agent pays → Octogate fetches results (timeout) →
Octogate has already taken fee, cannot refund (blockchain is final)
```

⚠️ **Mitigation**: Octogate caches provider responses. If provider is down after payment, serve cached result if available. Consider reputation penalty for provider.

---

## 11. Reputation System

### 11.1 Agent Reputation

**Metrics**:
- `successful_transactions`: Count of settled payments
- `failed_transactions`: Count of failed payments
- `avg_settlement_time`: How long agent waits after payment before collecting result
- `reliability_score`: (successful / (successful + failed)) * 100

**Usage**:
- Informational only (no monetary value)
- Non-transferable (tied to agent address)
- Cannot be staked or pledged as collateral
- Used for display in Octogate dashboard
- Agents with low scores may see higher fees in future (opt-in by providers)

### 11.2 Provider Reputation

**Metrics**:
- `uptime`: % of time API responds (excluding planned maintenance)
- `response_quality`: Derived from agent feedback (future feature)
- `settlement_reliability`: Ability to deliver after payment
- `reputation_score`: Composite of above

**Usage**:
- Informational (no monetary value)
- Non-transferable
- Agents may prioritize high-reputation providers
- May affect fee discounts (future feature)

### 11.3 Reputation Immutability

**Design**: Reputation scores are derived from immutable transaction history
- Cannot be manipulated by Octogate
- Agents cannot erase transaction history
- Providers cannot claim false reputation
- Scores are auditable on-chain

---

## 12. Scalability Considerations

### 12.1 Performance Targets (PoC)

- **Latency**: Agent gets 402 in <100ms, total roundtrip ~10-30s (blockchain confirmation)
- **Throughput**: 1 request/sec per agent (sufficient for AI testing), 10-100 requests/sec total (lab scale)
- **Cost per transaction**: $0.01-0.10 USD in gas fees (Base L2 is cheap)

### 12.2 Optimizations (Phase 2+)

**Batching** (not in PoC):
- Accumulate multiple agent → provider payments
- Submit to smart contract in single tx (lower gas)
- Requires escrow logic (violates non-custodial claim)
- ⚠️ Deferred to Phase 2

**Payment Channels** (not in PoC):
- Agent opens state channel with Octogate
- Multiple requests settle on-chain later
- Complex implementation, deferred

**Caching** (in PoC):
- Cache provider responses after first settlement
- Subsequent requests to same provider may use cache if hot
- Reduces blockchain writes

---

## 13. Deployment & Testing

### 13.1 Local Development

```bash
# 1. Deploy smart contract to Base Sepolia testnet
forge deploy SettlementContract.sol --network base-sepolia

# 2. Start Octogate Hub locally
PAY_TO=0x<local_address> NETWORK=eip155:84532 make dev-hub

# 3. Deploy mock provider
python mock_provider.py --port 9000

# 4. Test agent payment flow
EVM_PRIVATE_KEY=0x<test_key> X402_HUB=http://localhost:8080 make dev-cli search "test query"
```

### 13.2 Testnet (Base Sepolia)

- Deploy Settlement contract
- Test with faucet-funded wallets
- Verify signature validation
- Confirm event emission
- Load test with multiple agents

### 13.3 Mainnet (Future)

- Deploy to Base Mainnet (not L1 Ethereum, too expensive)
- Real USDC value at stake
- May attract regulatory attention (scale dependent)

---

## 14. Assumptions & Risks

### 14.1 Assumptions

1. **Agents are programmatic wallets**: They can sign transactions autonomously
2. **Blockchain is trust-anchored**: Settlement is final once on-chain
3. **USDC is regulatory safe**: Circle's stablecoin is issued by regulated entity
4. **Providers accept HTTP 402**: API ecosystem adopts x402 standard
5. **Agents can pay gas fees**: Or sponsors cover gas on their behalf
6. **No censorship**: Blockchain RPC is available, not censored

### 14.2 Risks

**Technical**:
- Smart contract bugs (mitigated by audit)
- Blockchain RPC downtime (mitigated by fallback RPC)
- Agent private key compromise (agent responsibility)

**Operational**:
- Provider reputation fraud (reputation is informational, not enforced)
- DDoS on Octogate hub (mitigated by CDN)
- Mass adoption causes gas fees to spike

**Regulatory** (see separate legal analysis):
- FinCEN deems this money transmission
- State-level requirements differ
- Enforcement against small operator
- Reputation system deemed "security"

---

## 15. Future Extensions

### 15.1 Solana Support

Add Solana as alternative blockchain:
- Different signature scheme (Ed25519)
- Different transfer model (not ERC20)
- Parallel transaction execution
- Lower latency (sub-second blocks)

### 15.2 Multi-Chain Settlement

Allow agent on Chain A to pay provider on Chain B:
- Requires bridge or liquidity pool
- Introduces counterparty risk

### 15.3 Subscription Model

Allow agents to "subscribe" to providers:
- Pre-authorize payment for N requests
- Provider receives aggregate payment
- Breaks "atomic per-request" model (TODO: evaluate)

### 15.4 Dispute Resolution

Enable agents/providers to dispute transactions:
- Requires governance (multi-sig, DAO, courts)
- Introduces discretionary decision-making (complexity)
- May trigger money transmission license requirement

### 15.5 Advanced Reputation

- Weighted scoring based on transaction volume
- Delegation of reputation (A trusts B's assessment)
- Predictive scoring using ML

---

## Appendix A: Glossary

- **Agent**: AI model or system that makes API requests
- **Provider**: Third-party API service that responds to agent requests
- **Octogate Hub**: Central proxy & payment coordinator (this project)
- **Settlement**: Blockchain transaction that completes payment
- **Nonce**: Unique identifier to prevent replay attacks
- **x402 Protocol**: HTTP 402 Payment Required standard (Coinbase open source)
- **USDC**: Circle's regulated stablecoin, ERC20 token
- **Non-custodial**: Octogate does not hold, freeze, or control transaction funds
- **Custody**: Legal ownership/control of assets

---

## Appendix B: Reference Documents

- Coinbase x402 spec: https://github.com/coinbase/x402
- HTTP 402 Payment Required: https://tools.ietf.org/html/rfc7231#section-6.5.2
- EIP-191 Signing: https://eips.ethereum.org/EIPS/eip-191
- Circle USDC Docs: https://developers.circle.com/
