// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

/**
 * @title SettlementContract
 * @dev Non-custodial atomic payment settlement for x402 protocol.
 *
 * Payment flow:
 * 1. Agent wants to pay Provider via Octogate Hub
 * 2. Hub returns 402 with challenge: (provider, amount, fee%, nonce, deadline, contract address)
 * 3. Agent signs the challenge with their private key
 * 4. Agent or Hub submits signed payment to this contract
 * 5. Contract verifies signature, transfers USDC atomically
 * 6. Emits PaymentSettled event for Hub to detect
 *
 * Key invariants:
 * - No admin keys, no pause functions, no upgrade proxy
 * - All parameters immutable after deployment
 * - Nonces are burned after first use (replay prevention)
 * - Transfers are atomic: provider + fee settle together or not at all
 * - USDC token is deterministically deployed at construction time
 */

contract SettlementContract {
    using SafeERC20 for IERC20;
    using ECDSA for bytes32;
    using MessageHashUtils for bytes32;

    // ============ Events ============

    /// @dev Emitted when a payment is successfully settled
    /// @param agent The agent's wallet address (payer)
    /// @param provider The provider's wallet address (payee)
    /// @param amount The USDC amount transferred to provider (6 decimals)
    /// @param fee The USDC amount transferred to Octogate (6 decimals)
    /// @param nonce Unique identifier for this payment (prevents replay)
    /// @param txHash Transaction hash for off-chain reconciliation (optional)
    event PaymentSettled(
        indexed address agent,
        indexed address provider,
        uint256 amount,
        uint256 fee,
        bytes32 indexed nonce,
        bytes32 txHash
    );

    /// @dev Emitted when a payment fails validation (for debugging)
    event PaymentFailed(address indexed agent, bytes32 indexed nonce, string reason);

    // ============ Immutable State ============

    /// @dev USDC ERC20 token contract (Circle's regulated stablecoin)
    IERC20 public immutable usdcToken;

    /// @dev Octogate's fee wallet address (receives percentage-based fees)
    address public immutable octogateWallet;

    /// @dev Fee percentage in basis points (e.g., 250 = 2.5%)
    /// Stored as immutable to prevent runtime modification
    uint256 public immutable feePercentBp;

    // ============ Mutable State ============

    /// @dev Tracks used nonces to prevent replay attacks
    /// mapping[nonce] -> true if already used
    mapping(bytes32 => bool) public usedNonces;

    // ============ Constructor ============

    /**
     * @dev Initialize the settlement contract with immutable parameters.
     *
     * @param _usdc Address of USDC token on this chain
     * @param _octogateWallet Wallet that receives settlement fees
     * @param _feePercentBp Fee percentage in basis points (e.g., 250 for 2.5%)
     *
     * Invariants enforced:
     * - All parameters are immutable after construction
     * - No admin key is created
     * - No upgrade proxy is used
     */
    constructor(
        address _usdc,
        address _octogateWallet,
        uint256 _feePercentBp
    ) {
        require(_usdc != address(0), "Invalid USDC address");
        require(_octogateWallet != address(0), "Invalid Octogate wallet");
        require(_feePercentBp > 0 && _feePercentBp <= 10000, "Invalid fee percentage");

        usdcToken = IERC20(_usdc);
        octogateWallet = _octogateWallet;
        feePercentBp = _feePercentBp;
    }

    // ============ External Functions ============

    /**
     * @dev Settle a payment atomically.
     *
     * This is the main entry point for payment settlement. The caller (Hub or agent)
     * provides a signed payment proof from the agent. The contract verifies the signature,
     * prevents replay attacks, and transfers USDC atomically to both provider and Octogate.
     *
     * @param agent Agent's wallet address (the payer, signer of the message)
     * @param provider Provider's wallet address (the payee)
     * @param amountUsdc Amount of USDC to pay provider (6 decimals, e.g., 1000000 = 1 USDC)
     * @param nonce Unique identifier for this payment (issued by Hub)
     * @param deadline Unix timestamp after which this signature is invalid
     * @param signature Agent's signature of the settlement details (EIP-191 format)
     *
     * Execution flow:
     * 1. Verify the signature (agent signed this exact payment)
     * 2. Check deadline has not passed
     * 3. Ensure nonce has not been used before (burn it)
     * 4. Calculate fee amount based on fee percentage
     * 5. Transfer USDC from agent to provider
     * 6. Transfer USDC from agent to Octogate (fee)
     * 7. Emit PaymentSettled event
     *
     * Reverts if:
     * - Signature is invalid or agent didn't sign
     * - Deadline has passed
     * - Nonce was already used
     * - Agent has insufficient USDC balance
     * - Agent has not approved contract to transfer USDC
     *
     * Gas cost: ~80,000 gas (two transferFrom calls)
     */
    function settlePayment(
        address agent,
        address provider,
        uint256 amountUsdc,
        bytes32 nonce,
        uint256 deadline,
        bytes calldata signature
    ) external {
        // ========== Validations ==========

        // 1. Check deadline (signature expires after N minutes)
        require(block.timestamp <= deadline, "Payment signature expired");

        // 2. Check nonce hasn't been used (replay attack prevention)
        require(!usedNonces[nonce], "Nonce already used");

        // 3. Verify agent signature
        // Message format: abi.encodePacked(
        //   provider, amountUsdc, feePercentBp, nonce, deadline, address(this), block.chainid
        // )
        bytes32 messageHash = keccak256(
            abi.encodePacked(provider, amountUsdc, feePercentBp, nonce, deadline, address(this), block.chainid)
        );

        // EIP-191 signed message: "\x19Ethereum Signed Message:\n{len(message)}"
        bytes32 digest = messageHash.toEthSignedMessageHash();
        address recoveredSigner = digest.recover(signature);

        require(recoveredSigner == agent, "Invalid signature or wrong agent");

        // 4. Validate amounts (prevent zero-fee dust, avoid overflow)
        require(amountUsdc > 0, "Amount must be greater than 0");
        require(amountUsdc <= type(uint256).max / 10000, "Amount too large");

        // ========== State Mutations ==========

        // Burn nonce to prevent replay
        usedNonces[nonce] = true;

        // Calculate fee and provider payment
        uint256 feeAmount = (amountUsdc * feePercentBp) / 10000;
        uint256 providerAmount = amountUsdc - feeAmount;

        require(providerAmount > 0, "Payment amount too small after fee");

        // ========== Fund Transfers ==========

        // Transfer USDC from agent to provider
        // SafeERC20 reverts if transfer fails (balance, approval, token paused, etc.)
        usdcToken.safeTransferFrom(agent, provider, providerAmount);

        // Transfer fee to Octogate
        usdcToken.safeTransferFrom(agent, octogateWallet, feeAmount);

        // ========== Event Emission ==========

        // Emit settled event with nonce indexed for off-chain listener
        bytes32 txHash = keccak256(abi.encodePacked(agent, provider, amountUsdc, block.timestamp));
        emit PaymentSettled(agent, provider, providerAmount, feeAmount, nonce, txHash);
    }

    // ============ Public Read Functions ============

    /**
     * @dev Check if a nonce has been used (and thus cannot be used again).
     *
     * @param nonce The nonce to check
     * @return true if nonce has been used, false otherwise
     */
    function isNonceUsed(bytes32 nonce) external view returns (bool) {
        return usedNonces[nonce];
    }

    /**
     * @dev Recover the signer address from a payment signature.
     *
     * This is a helper function for off-chain validation (e.g., agents checking
     * that their signature is valid before submitting to chain).
     *
     * @param agent Agent's wallet address
     * @param provider Provider's wallet address
     * @param amountUsdc USDC amount
     * @param nonce Payment nonce
     * @param deadline Signature expiration
     * @param signature Signature bytes
     * @return The recovered signer address (should match agent if valid)
     */
    function recoverSigner(
        address agent,
        address provider,
        uint256 amountUsdc,
        bytes32 nonce,
        uint256 deadline,
        bytes calldata signature
    ) external view returns (address) {
        bytes32 messageHash = keccak256(
            abi.encodePacked(provider, amountUsdc, feePercentBp, nonce, deadline, address(this), block.chainid)
        );

        bytes32 digest = messageHash.toEthSignedMessageHash();
        return digest.recover(signature);
    }

    // ============ View Functions for Testing =============

    /**
     * @dev Calculate the fee for a given amount.
     *
     * @param amountUsdc The USDC amount
     * @return feeAmount The fee portion (in USDC)
     * @return providerAmount The amount delivered to provider (in USDC)
     */
    function calculateFee(uint256 amountUsdc) external view returns (uint256 feeAmount, uint256 providerAmount) {
        feeAmount = (amountUsdc * feePercentBp) / 10000;
        providerAmount = amountUsdc - feeAmount;
    }
}
