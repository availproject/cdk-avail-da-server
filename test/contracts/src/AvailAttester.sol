// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.29;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

// Mock interfaces for testing (replace with real ones in production)
interface IAvailBridge {
    struct MerkleProofInput {
        bytes32 leaf;
        bytes32 rangeHash;
        uint256 dataRootIndex;
        uint256 leafIndex;
        bytes32[] proof;
    }
    function verifyBlobLeaf(
        MerkleProofInput calldata input
    ) external view returns (bool);
    function vectorx() external view returns (address);
}

interface IAvailVectorx {
    function rangeStartBlocks(bytes32 rangeHash) external view returns (uint32);
}

contract AvailAttester is Ownable {
    struct AttestationData {
        uint32 blockNumber;
        uint128 leafIndex;
    }

    // ✅ Using immutable for gas optimization (set once in constructor)
    IAvailBridge public immutable bridge;
    IAvailVectorx public immutable vectorx;

    mapping(bytes32 => AttestationData) public attestations;

    error InvalidAttestationProof();

    // ✅ Regular constructor - runs automatically during deployment
    constructor(
        IAvailBridge _bridge,
        address initialOwner
    ) Ownable(initialOwner) {
        // Validate inputs
        require(
            address(_bridge) != address(0),
            "Bridge address cannot be zero"
        );
        require(initialOwner != address(0), "Owner address cannot be zero");

        // Set immutable variables
        bridge = _bridge;
        vectorx = IAvailVectorx(bridge.vectorx());
    }

    // ===== SETTER (RESTRICTED TO OWNER) =====
    function setAttestation(
        bytes32 leaf,
        uint32 blockNumber,
        uint128 leafIndex
    ) external onlyOwner {
        attestations[leaf] = AttestationData(blockNumber, leafIndex);
    }

    // ===== MAIN ATTESTATION LOGIC =====
    function attest(bytes calldata data) external {
        IAvailBridge.MerkleProofInput memory input = abi.decode(
            data,
            (IAvailBridge.MerkleProofInput)
        );

        if (!bridge.verifyBlobLeaf(input)) revert InvalidAttestationProof();

        attestations[input.leaf] = AttestationData(
            vectorx.rangeStartBlocks(input.rangeHash) +
                uint32(input.dataRootIndex) +
                1,
            uint128(input.leafIndex)
        );
    }

    // ===== VIEW FUNCTIONS =====
    function getAttestation(
        bytes32 leaf
    ) external view returns (AttestationData memory) {
        return attestations[leaf];
    }

    function getBridgeAddress() external view returns (address) {
        return address(bridge);
    }

    function getVectorxAddress() external view returns (address) {
        return address(vectorx);
    }
}
