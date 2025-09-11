package l1

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ABI fragment for sequenceBatchesValidium
const PolygonValidiumEtrogABI = `
[
  {
    "inputs": [
      {
        "components": [
          { "internalType": "bytes32", "name": "transactionsHash", "type": "bytes32" },
          { "internalType": "bytes32", "name": "forcedGlobalExitRoot", "type": "bytes32" },
          { "internalType": "uint64", "name": "forcedTimestamp", "type": "uint64" },
          { "internalType": "bytes32", "name": "forcedBlockHashL1", "type": "bytes32" }
        ],
        "internalType": "struct PolygonValidiumEtrog.ValidiumBatchData[]",
        "name": "batches",
        "type": "tuple[]"
      },
      { "internalType": "uint32", "name": "l1InfoTreeLeafCount", "type": "uint32" },
      { "internalType": "uint64", "name": "maxSequenceTimestamp", "type": "uint64" },
      { "internalType": "bytes32", "name": "expectedFinalAccInputHash", "type": "bytes32" },
      { "internalType": "address", "name": "l2Coinbase", "type": "address" },
      { "internalType": "bytes", "name": "dataAvailabilityMessage", "type": "bytes" }
    ],
    "name": "sequenceBatchesValidium",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`

type ValidiumBatchData struct {
	TransactionsHash     [32]byte
	ForcedGlobalExitRoot [32]byte
	ForcedTimestamp      uint64
	ForcedBlockHashL1    [32]byte
}

type SequenceBatchesValidiumArgs struct {
	Batches                   []ValidiumBatchData
	L1InfoTreeLeafCount       uint32
	MaxSequenceTimestamp      uint64
	ExpectedFinalAccInputHash [32]byte
	L2Coinbase                common.Address
	DataAvailabilityMessage   []byte
}

func QueryBatchHashesFromL1ByBlockNumber(ctx context.Context, client *ethclient.Client, contractAbi abi.ABI, contractAddr common.Address, block *big.Int) ([]common.Hash, error) {

	blk, err := client.BlockByNumber(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %v: %w", block, err)
	}

	res := make([]common.Hash, 0)
	for _, tx := range blk.Transactions() {
		if tx.To() != nil && *tx.To() == contractAddr {
			data := tx.Data()
			method, _ := contractAbi.MethodById(data[:4])
			if method != nil && method.Name == "sequenceBatchesValidium" {
				log.Printf("Tx: %s", tx.Hash().Hex())
				log.Printf("Method: %s", method.Name)
				inputs, err := method.Inputs.Unpack(data[4:])
				if err != nil {
					return nil, fmt.Errorf("failed to unpack inputs for tx %s: %w", tx.Hash().Hex(), err)
				}

				var args SequenceBatchesValidiumArgs
				err = method.Inputs.Copy(&args, inputs)
				if err != nil {
					log.Printf("Failed to copy inputs to struct for tx %s: %v", tx.Hash().Hex(), err)
					return nil, err
				}

				for _, batch := range args.Batches {
					res = append(res, common.BytesToHash(batch.TransactionsHash[:]))
				}
			}
		}
	}
	return res, nil
}
