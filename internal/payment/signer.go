package payment

import (
	"crypto/ecdsa"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// EIP191Signer signs payment challenges using EIP-191 message signing
// This uses the standard Ethereum signing format compatible with the x402 protocol
type EIP191Signer struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}


// PaymentChallenge represents a payment challenge from the Hub
type PaymentChallenge struct {
	X402Version int
	Price       string
	Currency    string
	Networks    []string
	PayTo       string
	Nonce       string
}

// PaymentSignature represents a signed payment proof
type PaymentSignature struct {
	Agent     string // Agent address
	Challenge PaymentChallenge
	Signature string // EIP-191 signature (0x + hex)
	Timestamp int64
}

// NewEIP191Signer creates a new signer from a private key
// This is the original custom implementation, kept for backward compatibility
func NewEIP191Signer(privKeyHex string) (*EIP191Signer, error) {
	// Remove 0x prefix if present
	if len(privKeyHex) > 2 && privKeyHex[:2] == "0x" {
		privKeyHex = privKeyHex[2:]
	}

	// Parse hex private key
	privKeyBytes, err := hexutil.Decode("0x" + privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %w", err)
	}

	// Create ECDSA private key
	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Get public address
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key type")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &EIP191Signer{
		privateKey: privKey,
		address:    address,
	}, nil
}

// AgentAddress returns the agent's Ethereum address
func (s *EIP191Signer) AgentAddress() string {
	return s.address.Hex()
}

// SignChallenge signs a payment challenge and returns a payment signature
// Uses EIP-191 personal_sign format: "\x19Ethereum Signed Message:\n{length}{message}"
func (s *EIP191Signer) SignChallenge(challenge *PaymentChallenge) (*PaymentSignature, error) {
	// Create message to sign
	// Format: "x402-payment:{payTo}:{nonce}:{price}:{currency}"
	message := fmt.Sprintf("x402-payment:%s:%s:%s:%s",
		challenge.PayTo,
		challenge.Nonce,
		challenge.Price,
		challenge.Currency,
	)

	// Sign using EIP-191 personal_sign format
	signature, err := s.signEIP191Message(message)
	if err != nil {
		return nil, fmt.Errorf("failed to sign challenge: %w", err)
	}

	return &PaymentSignature{
		Agent:     s.address.Hex(),
		Challenge: *challenge,
		Signature: signature,
	}, nil
}

// signEIP191Message signs a message using EIP-191 personal_sign
// This is the format used by most wallets (e.g., MetaMask)
func (s *EIP191Signer) signEIP191Message(message string) (string, error) {
	// EIP-191 prefix
	prefix := "\x19Ethereum Signed Message:\n"
	fullMessage := prefix + fmt.Sprintf("%d", len(message)) + message

	// Hash the message
	hash := crypto.Keccak256Hash([]byte(fullMessage))

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	// Signature recovery ID (v value) is the last byte
	// Adjust for Ethereum standard (27 or 28)
	signature[64] += 27

	return hexutil.Encode(signature), nil
}

// VerifySignature verifies a signature (for server-side validation)
// Not typically used in CLI, but useful for testing
func VerifySignature(message, signature string, expectedAddress string) (bool, error) {
	// Decode signature
	sig, err := hexutil.Decode(signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature format: %w", err)
	}

	if len(sig) != 65 {
		return false, fmt.Errorf("signature must be 65 bytes")
	}

	// EIP-191 prefix
	prefix := "\x19Ethereum Signed Message:\n"
	fullMessage := prefix + fmt.Sprintf("%d", len(message)) + message

	// Hash the message
	hash := crypto.Keccak256Hash([]byte(fullMessage))

	// Recover public key
	// Adjust v value back from Ethereum standard (27/28 → 0/1)
	sig[64] -= 27

	publicKey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Get address from public key
	recoveredAddress := crypto.PubkeyToAddress(*publicKey)

	// Parse expected address
	expectedAddr := common.HexToAddress(expectedAddress)

	// Compare
	return recoveredAddress == expectedAddr, nil
}

// LogSignatureInfo logs signature details for debugging
func (ps *PaymentSignature) LogSignatureInfo() {
	log.Printf("Payment Signature:")
	log.Printf("  Agent: %s", ps.Agent)
	log.Printf("  Nonce: %s", ps.Challenge.Nonce[:10]+"...")
	log.Printf("  Price: %s %s", ps.Challenge.Price, ps.Challenge.Currency)
	log.Printf("  PayTo: %s", ps.Challenge.PayTo)
	log.Printf("  Signature: %s...", ps.Signature[:10])
}

// ToX402Header formats the signature as an X-PAYMENT header
// Format: "{agent}:{network}:{signature}"
func (ps *PaymentSignature) ToX402Header(network string) string {
	return fmt.Sprintf("%s:%s:%s", ps.Agent, network, ps.Signature)
}

// ChallengeHash returns the hash that was signed (for verification)
func (challenge *PaymentChallenge) Hash() string {
	message := fmt.Sprintf("x402-payment:%s:%s:%s:%s",
		challenge.PayTo,
		challenge.Nonce,
		challenge.Price,
		challenge.Currency,
	)
	return message
}
