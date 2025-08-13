package da

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/availproject/avail-go-sdk/primitives"
	avail_sdk "github.com/availproject/avail-go-sdk/sdk"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type AvailBackend struct {
	eth_client   *ethclient.Client
	avail_sdk    avail_sdk.SDK
	attestorAddr common.Address
}

func NewAvailBackend(attestorAddr string, l1RPCURL string, availRPCURL string) (*AvailBackend, error) {

	addr := common.HexToAddress(attestorAddr)

	client, err := ethclient.Dial(l1RPCURL)
	if err != nil {
		log.Printf("Failed to connect to Ethereum RPC, error:%v", err)
		return nil, err
	}

	sdk, err := avail_sdk.NewSDK(availRPCURL)
	if err != nil {
		log.Printf("AvailDAError: ⚠️ error connecting to %s: %+v", availRPCURL, err)
		return nil, err
	}

	return &AvailBackend{
		eth_client:   client,
		avail_sdk:    sdk,
		attestorAddr: addr,
	}, nil
}

func (a *AvailBackend) GetDataFromAvail(hash common.Hash) ([]byte, error) {
	start := time.Now()
	log.Printf("Fetching data from Avail, hash:%v", hash.Hex())

	blockNumber, leafIndex, err := a.getAttestation(hash)
	if blockNumber == 0 || leafIndex == 0 || err != nil {
		log.Printf("No attestation found, error:%v, duration:%v", err, time.Since(start))
		return nil, errors.New("no attestation found")
	}

	log.Printf("Attestation found, blockNumber:%d, leafIndex:%d",
		blockNumber,
		leafIndex,
	)

	data, err := a.getData(blockNumber, leafIndex)
	if err != nil {
		log.Printf("Failed to get data from Avail, error:%v, duration:%v", err, time.Since(start))
		return nil, err
	}

	log.Printf("Successfully retrieved data from Avail, duration:%v", time.Since(start))
	return data, nil
}

const attestationABI = `[{"inputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"name":"attestations","outputs":[{"internalType":"uint32","name":"blockNumber","type":"uint32"},{"internalType":"uint128","name":"leafIndex","type":"uint128"}],"stateMutability":"view","type":"function"}]`

func (a *AvailBackend) getData(blockNumber uint32, index int64) ([]byte, error) {
	blockHash, err := a.avail_sdk.Client.BlockHash(blockNumber)
	if err != nil {
		return nil, fmt.Errorf("❎ Cannot get block hash: %w", err)
	}

	block, err := avail_sdk.NewBlock(a.avail_sdk.Client, blockHash)
	if err != nil {
		return nil, fmt.Errorf("❎ Cannot get block: %w", err)
	}

	var blob avail_sdk.DataSubmission

	blobs := block.DataSubmissions(avail_sdk.Filter{})
	if int(index) >= len(blobs) {
		return nil, fmt.Errorf("❎ Unable to retrieve blob at index %d from block %d", index, blockNumber)
	}
	blob = blobs[index]

	signerAddress, err := primitives.NewAccountIdFromMultiAddress(blob.TxSigner)
	if err != nil {
		log.Printf("AvailDAWarn:‼️ Unable to extract the signer address for the blob")
	}

	log.Printf("AvailDAInfo: ✅ Tx batch retrieved from Avail chain, signer: %s, appID: %s, extrinsicHash: %s",
		signerAddress.ToHuman(),
		blob.AppId,
		blob.TxHash,
	)

	return blob.Data, nil
}

func (a *AvailBackend) getAttestation(hash common.Hash) (uint32, int64, error) {
	start := time.Now()
	log.Printf("Getting attestation, hash:%v", hash.Hex())

	parsedABI, err := abi.JSON(strings.NewReader(attestationABI))
	if err != nil {
		return 0, 0, err
	}

	data, err := parsedABI.Pack("attestations", hash)
	if err != nil {
		return 0, 0, err
	}

	res, err := a.eth_client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &a.attestorAddr,
		Data: data,
	}, nil)
	if err != nil {
		return 0, 0, err
	}

	var output struct {
		BlockNumber uint32
		LeafIndex   *big.Int
	}
	err = parsedABI.UnpackIntoInterface(&output, "attestations", res)
	if err != nil {
		log.Printf("Failed to unpack attestation result, error:%v, duration:%v", err, time.Since(start))
		return 0, 0, err
	}

	log.Printf("Successfully retrieved attestation, blockNumber%v,leafIndex:%d, duration:%v",
		output.BlockNumber,
		output.LeafIndex.Int64(),
		time.Since(start),
	)
	return output.BlockNumber, output.LeafIndex.Int64(), nil
}
