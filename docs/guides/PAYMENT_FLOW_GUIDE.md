# x402 Payment Flow Guide

## Overview

The x402 payment protocol implements a challenge-response payment flow where agents (AI clients) pay for API access through signed transactions.

## Complete Payment Flow

```
┌────────────────────────────────────────────────────────────────┐
│                    Payment Flow Diagram                         │
└────────────────────────────────────────────────────────────────┘

Agent                          Hub                      Blockchain
  │                             │                            │
  ├─── 1. Request (no auth) ──→ │                            │
  │                             │                            │
  │ ← 2. 402 Challenge (nonce)─ │                            │
  │                             │                            │
  ├─── 3. Sign Challenge ──→    │                            │
  │     (EIP-191)               │                            │
  │                             │                            │
  ├─── 4. Retry + X-PAYMENT ──→ │                            │
  │                             │                            │
  │                       5. Verify Signature              │
  │                             │                            │
  │                       6. Submit Settlement ──────────→  │
  │                             │         (USDC transfer)    │
  │                             │                            │
  │                       7. Wait for Confirmation←──────────│
  │                             │                            │
  │ ← ── 8. Return Results ──── │                            │
  │                             │                            │
```

## Step-by-Step Breakdown

### 1. Agent Requests Without Payment

```
GET /v1/search?q=rust HTTP/1.1
Host: hub.example.com
```

**Response**: HTTP 402 Payment Required
```json
{
  "x402Version": 1,
  "price": "0.001",
  "currency": "USDC",
  "networks": ["eip155:84532"],
  "payTo": "0x742d35Cc6634C0532925a3b844Bc9e7595f42521",
  "nonce": "0xaabbccdd..."  // Unique per request
}
```

### 2. Agent Creates Signer

```go
signer, _ := payment.NewEIP191Signer(os.Getenv("EVM_PRIVATE_KEY"))
agent := signer.AgentAddress()
// agent = "0x1234...abcd"
```

### 3. Agent Signs Challenge

EIP-191 personal_sign format:
```
Message: "x402-payment:{payTo}:{nonce}:{price}:{currency}"
Example: "x402-payment:0x742d...f42521:0xaabbcc...:0.001:USDC"

Signature: 0x{65 hex bytes (130 chars)}
           0x + recovery byte adjustment (v = 27 or 28)
```

```go
signature, _ := signer.SignChallenge(challenge)
// signature.Signature = "0xaabbccdd...1234" (66+ chars)
```

### 4. Agent Retries with X-PAYMENT Header

```
GET /v1/search?q=rust HTTP/1.1
Host: hub.example.com
X-PAYMENT: 0x1234...abcd:eip155:84532:0xaabbccdd...

Format: {agent}:{network}:{signature}
```

### 5. Hub Verifies Signature

```go
verified, _ := payment.VerifySignature(
    message,
    signature,
    expectedAgentAddress,
)
// verified = true ✓
```

### 6. Hub Submits Settlement to Smart Contract

```solidity
// SettlementContract.settlePayment()
// Transfers:
// - {price} USDC to provider
// - {fee} USDC to octogate
// Emits: PaymentSettled(agent, provider, nonce, ...)
```

### 7. Blockchain Listener Detects Settlement

EventListener subscribes to `PaymentSettled` events:
```
BlockchainListener → WebSocket RPC → PaymentSettled event
                  ↓
            Update pending_settlements
                  ↓
            Mark as "confirmed"
```

### 8. Hub Returns Results

```
HTTP/1.1 200 OK
Content-Type: application/json
X-PAYMENT-RESPONSE: verified

{
  "query": "rust",
  "results": [...],
  "paid": true,
  "tx_hash": "0x5678..."
}
```

---

## Testing the Payment Flow Locally

### Prerequisites

```bash
# 1. Private key (for testing only, not your real funds!)
export EVM_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb476caded732d8faf23d265da5a"

# 2. Hub running locally
export X402_HUB="http://localhost:8080"

# 3. Network (Base Sepolia for testnet)
export NETWORK="eip155:84532"
```

### Run CLI Search

```bash
# Terminal 1: Start Hub
make dev-hub

# Terminal 2: Run search
EVM_PRIVATE_KEY="0xac0974..." X402_HUB="http://localhost:8080" \
  make dev-cli search "rust patterns"
```

### Expected Output

```
→ Requesting: rust patterns
✓ Challenge received (402)
✓ Signing with EVM_PRIVATE_KEY
✓ Submitting payment: 0.001 USDC
✓ Payment verified
✓ Results received

{
  "query": "rust patterns",
  "results": [
    {
      "title": "Example Search Result 1",
      "url": "https://example.com/1",
      "description": "..."
    }
  ],
  "paid": true
}
```

---

## Signature Format Details

### EIP-191 Message Structure

```
Prefix: "\x19Ethereum Signed Message:\n"
Length: "{length of message}"
Message: "x402-payment:{payTo}:{nonce}:{price}:{currency}"

Example:
"\x19Ethereum Signed Message:\n89x402-payment:0x742d...f42521:0xaabbcc...:0.001:USDC"
```

### Signature Recovery

```
Signature: [0-63] bytes of r
           [64-127] bytes of s
           [128-129] recovery byte (v)

Recovery: v ∈ {27, 28}  // Adjusted from {0, 1}
```

### Verification

```go
1. Hash EIP-191 message: keccak256("{prefix}{length}{message}")
2. Recover public key from signature
3. Derive address from public key
4. Compare with expectedAddress
```

---

## Error Scenarios

### Invalid Private Key

```
Error: invalid private key format
Solution: Ensure EVM_PRIVATE_KEY is 32 bytes (64 hex chars)
```

### Signature Verification Failed

```
Error: signature did not verify
Causes:
  - Message format mismatch (typo in payTo/nonce)
  - Signature corrupted or truncated
  - Wrong agent address
```

### Nonce Mismatch

```
Error: nonce not found in pending_settlements
Cause: Signature signed different nonce than submitted
Solution: Ensure challenge nonce matches signed nonce
```

### Payment Settlement Failed

```
Error: insufficient USDC balance
Solution: Fund test wallet with Base Sepolia USDC
         Get faucet at https://www.alchemy.com/faucets/base-sepolia
```

---

## File Structure

```
Payment signing logic:
├── internal/payment/
│   ├── signer.go              # EIP-191 signing
│   └── signer_test.go         # Signature tests
│
CLI command:
├── cmd/x402/commands/
│   └── search.go              # 402 challenge-response flow
│
Hub middleware:
├── internal/middleware/
│   └── x402.go                # Challenge generation + verification
│
Smart contract (future):
└── contracts/
    └── SettlementContract.sol  # USDC settlement
```

---

## Next Steps

1. **Deploy Smart Contract**
   ```bash
   forge build
   forge create --rpc-url $RPC_URL
   ```

2. **Fund Test Wallets**
   ```bash
   # Base Sepolia USDC faucet
   https://www.alchemy.com/faucets/base-sepolia
   ```

3. **Run End-to-End Test**
   ```bash
   make dev-hub &
   EVM_PRIVATE_KEY="..." make dev-cli search "test"
   ```

4. **Monitor Settlement**
   ```bash
   # View transaction on Basescan
   https://sepolia.basescan.org/tx/{tx_hash}
   ```

---

## Security Considerations

### Do NOT

- ❌ Commit private keys to git
- ❌ Expose EVM_PRIVATE_KEY in logs
- ❌ Use test keys for real funds
- ❌ Trust unsigned payment headers

### Do

- ✅ Use environment variables for secrets
- ✅ Verify all signatures server-side
- ✅ Log payment attempts for audit
- ✅ Monitor for replay attacks (nonce validation)
- ✅ Set appropriate price limits per agent

---

## Testing Payment Signing

```bash
# Run payment tests
go test ./internal/payment -v

# Run full flow test
go test ./internal/payment -run TestCompletePaymentFlow -v

# Benchmark signature creation
go test ./internal/payment -bench BenchmarkSignature -benchmem
```

---

## References

- [EIP-191: Signed Data Standard](https://eips.ethereum.org/EIPS/eip-191)
- [go-ethereum crypto package](https://pkg.go.dev/github.com/ethereum/go-ethereum/crypto)
- [Ethereum Signature Standard](https://docs.metamask.io/guide/signing-data.html)
