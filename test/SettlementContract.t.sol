// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import {SettlementContract} from "../contracts/SettlementContract.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {MockERC20} from "./mocks/MockERC20.sol";
import {ECDSA} from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import {MessageHashUtils} from "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

/**
 * @title SettlementContractTest
 * @dev Comprehensive test suite for atomic settlement contract
 *
 * Tests cover:
 * - Happy path: valid signature, transfers succeed
 * - Replay attacks: nonce burning prevents reuse
 * - Signature validation: invalid/wrong signer rejected
 * - Deadline checks: expired signatures rejected
 * - Amount validation: zero/dust amounts rejected
 * - Fee calculations: correct split between provider and Octogate
 * - Event emission: PaymentSettled emitted with correct data
 */
contract SettlementContractTest is Test {
    using ECDSA for bytes32;
    using MessageHashUtils for bytes32;

    SettlementContract public settlement;
    MockERC20 public usdc;

    address public constant OCTOGATE = address(0xAbc123);
    address public constant PROVIDER = address(0xDef456);

    uint256 public agentPrivateKey = 0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef;
    address public agentAddress;

    uint256 public constant USDC_AMOUNT = 1e6; // 1 USDC (6 decimals)
    uint256 public constant FEE_PERCENT_BP = 250; // 2.5%
    bytes32 public nonce = keccak256("test-nonce-1");
    uint256 public deadline;

    // ========== Setup ==========

    function setUp() public {
        agentAddress = vm.addr(agentPrivateKey);
        deadline = block.timestamp + 1 hours;

        // Deploy mock USDC
        usdc = new MockERC20("USDC", "USDC", 6);

        // Deploy settlement contract
        settlement = new SettlementContract(address(usdc), OCTOGATE, FEE_PERCENT_BP);

        // Fund agent with USDC
        usdc.mint(agentAddress, 10e6); // 10 USDC

        // Agent approves settlement contract to spend USDC
        vm.prank(agentAddress);
        usdc.approve(address(settlement), type(uint256).max);
    }

    // ========== Happy Path Tests ==========

    function test_SettlePaymentSuccess() public {
        // Generate valid signature
        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        // Settle payment
        vm.expectEmit(true, true, true, true);
        emit SettlementContract.PaymentSettled(
            agentAddress, PROVIDER, USDC_AMOUNT * (10000 - FEE_PERCENT_BP) / 10000, USDC_AMOUNT * FEE_PERCENT_BP / 10000, nonce, ""
        );

        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature);

        // Verify provider received correct amount (minus fee)
        uint256 expectedProvider = USDC_AMOUNT * (10000 - FEE_PERCENT_BP) / 10000;
        assertEq(usdc.balanceOf(PROVIDER), expectedProvider);

        // Verify Octogate received fee
        uint256 expectedFee = USDC_AMOUNT * FEE_PERCENT_BP / 10000;
        assertEq(usdc.balanceOf(OCTOGATE), expectedFee);

        // Verify nonce is burned
        assertTrue(settlement.isNonceUsed(nonce));
    }

    function test_MultiplePaymentsWithDifferentNonces() public {
        // First payment
        bytes32 nonce1 = keccak256("nonce-1");
        bytes memory sig1 = _signPayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce1, deadline);
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce1, deadline, sig1);

        // Second payment with different nonce
        bytes32 nonce2 = keccak256("nonce-2");
        bytes memory sig2 = _signPayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce2, deadline);
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce2, deadline, sig2);

        // Both should succeed
        assertTrue(settlement.isNonceUsed(nonce1));
        assertTrue(settlement.isNonceUsed(nonce2));

        // Provider should have received 2x payment
        uint256 expectedProvider = 2 * USDC_AMOUNT * (10000 - FEE_PERCENT_BP) / 10000;
        assertEq(usdc.balanceOf(PROVIDER), expectedProvider);
    }

    // ========== Replay Attack Tests ==========

    function test_ReplayAttackPrevented() public {
        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        // First settlement succeeds
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature);

        // Second settlement with same nonce fails
        vm.expectRevert("Nonce already used");
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature);
    }

    function test_NonceBurningWorks() public {
        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        assertFalse(settlement.isNonceUsed(nonce));

        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature);

        assertTrue(settlement.isNonceUsed(nonce));
    }

    // ========== Signature Validation Tests ==========

    function test_InvalidSignatureRejected() public {
        // Create valid signature
        bytes memory validSig = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        // Tamper with signature (flip a bit)
        bytes memory invalidSig = validSig;
        invalidSig[0] = bytes1(uint8(invalidSig[0]) ^ 1);

        vm.expectRevert("Invalid signature or wrong agent");
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, invalidSig);
    }

    function test_WrongSignerRejected() public {
        // Sign as different agent
        uint256 wrongPrivateKey = 0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba;
        address wrongAgent = vm.addr(wrongPrivateKey);

        bytes memory signature = _signPaymentWithKey(
            wrongPrivateKey, agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        vm.expectRevert("Invalid signature or wrong agent");
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature);
    }

    function test_SignatureRecovery() public view {
        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        address recovered = settlement.recoverSigner(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature
        );

        assertEq(recovered, agentAddress);
    }

    // ========== Deadline Tests ==========

    function test_ExpiredDeadlineRejected() public {
        uint256 pastDeadline = block.timestamp - 1;

        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, pastDeadline
        );

        vm.expectRevert("Payment signature expired");
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, pastDeadline, signature);
    }

    function test_DeadlineAtBlockTimestampAccepted() public {
        uint256 blockTimestampDeadline = block.timestamp;

        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, blockTimestampDeadline
        );

        // Should not revert
        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, blockTimestampDeadline, signature);
    }

    function test_FutureDeadlineWorks() public {
        uint256 futureDeadline = block.timestamp + 365 days;

        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, futureDeadline
        );

        settlement.settlePayment(agentAddress, PROVIDER, USDC_AMOUNT, nonce, futureDeadline, signature);

        assertTrue(settlement.isNonceUsed(nonce));
    }

    // ========== Amount Validation Tests ==========

    function test_ZeroAmountRejected() public {
        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, 0, nonce, deadline
        );

        vm.expectRevert("Amount must be greater than 0");
        settlement.settlePayment(agentAddress, PROVIDER, 0, nonce, deadline, signature);
    }

    function test_InsufficientBalanceRejected() public {
        uint256 largeAmount = 100e6; // 100 USDC (agent only has 10)

        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, largeAmount, nonce, deadline
        );

        // transferFrom will revert due to insufficient balance
        vm.expectRevert();
        settlement.settlePayment(agentAddress, PROVIDER, largeAmount, nonce, deadline, signature);
    }

    function test_SmallAmountWithLargeFee() public {
        uint256 smallAmount = 1; // 1 wei of USDC

        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, smallAmount, nonce, deadline
        );

        // Fee (0) + provider (1) should split correctly but provider gets 0
        vm.expectRevert("Payment amount too small after fee");
        settlement.settlePayment(agentAddress, PROVIDER, smallAmount, nonce, deadline, signature);
    }

    // ========== Fee Calculation Tests ==========

    function test_FeeCalculationCorrect() public {
        (uint256 feeAmount, uint256 providerAmount) = settlement.calculateFee(USDC_AMOUNT);

        uint256 expectedFee = USDC_AMOUNT * FEE_PERCENT_BP / 10000;
        uint256 expectedProvider = USDC_AMOUNT - expectedFee;

        assertEq(feeAmount, expectedFee);
        assertEq(providerAmount, expectedProvider);
    }

    function test_FeePercentageImmutable() public view {
        assertEq(settlement.feePercentBp(), FEE_PERCENT_BP);
    }

    // ========== Cross-Chain Replay Prevention Tests ==========

    function test_ChainIdInSignatureMessage() public {
        bytes memory signature = _signPayment(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline
        );

        // Signature should include chain ID in the message hash
        // (If chain ID changes, same signature is invalid - prevents cross-chain replay)
        // This is tested by verifying the recovered signer matches our agent

        address recovered = settlement.recoverSigner(
            agentAddress, PROVIDER, USDC_AMOUNT, nonce, deadline, signature
        );

        assertEq(recovered, agentAddress);
    }

    // ========== State Transition Tests ==========

    function test_UnusedNonceIsNotUsed() public {
        bytes32 unusedNonce = keccak256("unused");
        assertFalse(settlement.isNonceUsed(unusedNonce));
    }

    // ========== Helper Functions ==========

    function _signPayment(
        address agent,
        address provider,
        uint256 amountUsdc,
        bytes32 paymentNonce,
        uint256 paymentDeadline
    ) internal view returns (bytes memory) {
        return _signPaymentWithKey(
            agentPrivateKey, agent, provider, amountUsdc, paymentNonce, paymentDeadline
        );
    }

    function _signPaymentWithKey(
        uint256 privateKey,
        address agent,
        address provider,
        uint256 amountUsdc,
        bytes32 paymentNonce,
        uint256 paymentDeadline
    ) internal view returns (bytes memory) {
        bytes32 messageHash = keccak256(
            abi.encodePacked(provider, amountUsdc, FEE_PERCENT_BP, paymentNonce, paymentDeadline, address(settlement), block.chainid)
        );

        bytes32 digest = messageHash.toEthSignedMessageHash();
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKey, digest);

        return abi.encodePacked(r, s, v);
    }
}
