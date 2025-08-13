// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.29;

import {Initializable} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

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

contract AvailAttester is Initializable, OwnableUpgradeable {
    struct AttestationData {
        uint32 blockNumber;
        uint128 leafIndex;
    }

    IAvailBridge public bridge;
    IAvailVectorx public vectorx;

    mapping(bytes32 => AttestationData) public attestations;

    error InvalidAttestationProof();

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers(); // Lock implementation for proxies
    }

    function initialize(IAvailBridge _bridge) public initializer {
        __Ownable_init();
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
}
