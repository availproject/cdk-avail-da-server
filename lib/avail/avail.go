package avail

import (
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/vedhavyas/go-subkey/v2"

	"github.com/0xPolygon/cdk/log"

	"github.com/availproject/avail-go-sdk/primitives"
	avail_sdk "github.com/availproject/avail-go-sdk/sdk"
	"github.com/availproject/cdk-avail-da-server/lib/avail/availattestation"
	s3_storage_service "github.com/availproject/cdk-avail-da-server/lib/avail/s3StorageService"
)

const (
	AvailMessageHeaderFlag byte = 0x0a
	AvailNetworkID              = 42
	BridgeApiTimeout            = time.Duration(1200)
	AvailRPCTimeout             = time.Duration(120)
	BridgeApiWaitInterval       = time.Duration(420)
	BridgeApiRetryCount         = 10
	VectorXTimeout              = time.Duration(10000)
)

var (
	ErrAvailDAClientInit          = errors.New("unable to initialize to connect with AvailDA")
	ErrBatchSubmitToAvailDAFailed = errors.New("unable to submit batch to AvailDA")
	ErrWrongAvailDAPointer        = errors.New("unable to retrieve batch, wrong blobPointer")
)

type AvailBackend struct {
	logger *log.Logger

	sdk     avail_sdk.SDK
	acc     subkey.KeyPair
	address string
	appId   int

	httpApi string

	// AvailDA bridge
	bridgeEnabled       bool
	bridgeApi           string
	attestationContract *availattestation.Availattestation
	bridgeTimeout       int

	// S3 Fallback service
	fallbackS3Service *s3_storage_service.S3StorageService
}

func New(l1RPCURL string, attestationContractAddress common.Address, config Config, logger *log.Logger) (*AvailBackend, error) {

	logger.Info("AvailDAInfo: ✏️ Avail backend client is being initialized...")
	logger.Debug("AvailDADebug: AvailDA config",
		"ws-api-url", config.WsApiUrl,
		"http-api-url", config.HttpApiUrl,
		"app-id", config.AppID,
		"bridge-enabled", config.BridgeEnabled,
		"bridge-api-url", config.BridgeApiUrl,
		"bridge-timeout", config.BridgeTimeout,
	)
	logger.Debug("AvailDADebug: 📜 Attestation contract", "address", attestationContractAddress)

	ethClient, err := ethclient.Dial(l1RPCURL)
	if err != nil {
		logger.Error("AvailDAError: ⚠️ error connecting to %s: %+v", l1RPCURL, err)
		return nil, err
	}

	attestationContract, err := availattestation.NewAvailattestation(attestationContractAddress, ethClient)
	if err != nil {
		return nil, err
	}

	sdk, err := avail_sdk.NewSDK(config.HttpApiUrl)
	if err != nil {
		logger.Error("AvailDAError: ⚠️ error connecting to %s: %+v", config.HttpApiUrl, err)
		return nil, err
	}

	appId := 0

	// if app id is greater than 0 then it must be created before submitting data
	if config.AppID != 0 {
		appId = config.AppID
	}

	acc, err := avail_sdk.Account.NewKeyPair(config.Seed)
	if err != nil {
		logger.Error("AvailDAError: ⚠️ unable to generate keypair from given seed")
	}

	var fallbackS3Service *s3_storage_service.S3StorageService
	if config.FallbackS3ServiceConfig.Enable {
		logger.Debug("AvailDADebug:ℹ️ Fallback S3 config: s3-bucket: %+v, region: %+v, object-prefix: %+v, secret-key: %+v, access-key: %+v, ", config.FallbackS3ServiceConfig.Bucket, config.FallbackS3ServiceConfig.Region, config.FallbackS3ServiceConfig.ObjectPrefix, config.FallbackS3ServiceConfig.SecretKey, config.FallbackS3ServiceConfig.AccessKey)
		fallbackS3Service, err = s3_storage_service.NewS3StorageService(config.FallbackS3ServiceConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("AvailDAError: unable to intialize s3 storage service for fallback, %w. %w", err, ErrAvailDAClientInit)
		}
	}

	logger.Debug("AvailDADebug: 🔑 Using KeyringPair", "address", acc.SS58Address(AvailNetworkID))
	logger.Info("AvailDAInfo:✌️ Avail backend client is created successfully")

	return &AvailBackend{
		logger:  logger,
		sdk:     sdk,
		acc:     acc,
		address: acc.SS58Address(AvailNetworkID),
		appId:   appId,
		httpApi: config.HttpApiUrl,

		bridgeEnabled:       config.BridgeEnabled,
		attestationContract: attestationContract,
		bridgeApi:           config.BridgeApiUrl,
		bridgeTimeout:       config.BridgeTimeout,

		fallbackS3Service: fallbackS3Service,
	}, nil
}

func (a *AvailBackend) Init() error {
	return nil
}

func (a *AvailBackend) PostSequence(ctx context.Context, batchesData [][]byte) ([]byte, error) {
	// RLP Encode
	sequenceBlobData, err := rlp.EncodeToBytes(batchesData)
	if err != nil {
		return nil, fmt.Errorf("cannot RLP encode data:%w", err)
	}
	a.logger.Info("AvailDAInfo: ⚡️ Posting Sequence", "length", len(sequenceBlobData))

	// Submit the data to the Avail chain
	a.logger.Info("AvailDAInfo: 📤 Submitting data to Avail chain")
	txDetails, err := a.submitData(ctx, sequenceBlobData)
	if err != nil {
		return nil, fmt.Errorf("cannot submit data: %w", err)
	}
	a.logger.Info("AvailDAInfo: 📤 Data submitted to Avail chain")

	var dataAvailabilityMessage []byte
	if a.bridgeEnabled {
		a.logger.Info("AvailDAInfo: Bridge is enabled, getting merkle proof from the bridge")
		// Get the merkle proof from the Avail Bridge
		merkleProofInput, err := a.getMerkleProofFromAvailBridge(ctx, txDetails.BlockHash, txDetails.TxIndex)
		if err != nil {
			return nil, fmt.Errorf("cannot get merkle proof from bridge: %w", err)
		}
		payload, err := merkleProofInput.EnodeToBinary()
		if err != nil {
			return nil, fmt.Errorf("encode merkle proof failed:%w", err)
		}
		dataAvailabilityMessage, err = PackEnvelopeWithMsgType(DAM_TYPE_MERKLE_PROOF, payload)
		if err != nil {
			return nil, fmt.Errorf("pack envelope failed: %w", err)
		}
	} else {
		a.logger.Info("AvailDAInfo: Bridge is disabled, using blob pointer as data availability message")
		dataCommitment := crypto.Keccak256Hash(sequenceBlobData)
		blobPointer := NewBlobPointer(txDetails.BlockNumber, txDetails.TxIndex, dataCommitment)
		payload, err := blobPointer.MarshalToBinary()
		if err != nil {
			return nil, fmt.Errorf("encode blob pointer failed: %w", err)
		}
		dataAvailabilityMessage, err = PackEnvelopeWithMsgType(DAM_TYPE_BLOB_POINTER, payload)
		if err != nil {
			return nil, fmt.Errorf("pack envelope failed: %w", err)
		}
	}

	// fallback
	if a.fallbackS3Service != nil {
		a.logger.Info("AvailDAInfo: Fallback S3 storage service is enabled, putting data on s3 storage")
		// Put the data on the s3 storage service
		// Log error but don't fail the whole operation
		// as data is already submitted to Avail chain
		if err = a.fallbackS3Service.PutMultiple(ctx, batchesData); err != nil {
			a.logger.Error("AvailDAError: failed to put data on s3 storage service: %w", err)
		} else {
			a.logger.Info("AvailDAInfo: ✅  Succesfully posted data to S3 using fallbackS3Service")
		}
	}

	a.logger.Debug("AvailDADebug: ✅ Data availability message: %+v", dataAvailabilityMessage)
	a.logger.Info("AvailDAInfo: ⚡️ Sequence posted successfully", "length", len(sequenceBlobData))
	return dataAvailabilityMessage, nil
}

func (a *AvailBackend) GetSequence(ctx context.Context, batchHashes []common.Hash, dataAvailabilityMessage []byte) ([][]byte, error) {

	a.logger.Info("AvailDAInfo: 📤 Getting Sequence", "num_batches", len(batchHashes))

	msgType, payload, err := UnpackEnvelopeForMsgType(dataAvailabilityMessage)
	if err != nil {
		return nil, err
	}

	var blockNumber uint32
	var index uint32
	var indexType IndexType

	switch msgType {
	case DAM_TYPE_MERKLE_PROOF:
		a.logger.Debug("AvailDADebug: Data availability message is of type MerkleProofInput")
		merkleProofInput := &MerkleProofInput{}
		if err := merkleProofInput.DecodeFromBinary(payload); err != nil {
			return nil, fmt.Errorf("failed to decode MerkleProofInput: %w", err)
		}
		attestationData, err := a.attestationContract.Attestations(nil, merkleProofInput.Leaf)
		if err != nil {
			return nil, fmt.Errorf("cannot get attestation data: %w", err)
		}
		blockNumber = attestationData.BlockNumber
		index = uint32(attestationData.LeafIndex.Uint64())
		indexType = LeafIndex

	case DAM_TYPE_BLOB_POINTER:
		a.logger.Debug("AvailDADebug: Data availability message is of type BlobPointer")
		blobPointer := &BlobPointer{}
		if err := blobPointer.UnmarshalFromBinary(payload); err != nil {
			return nil, fmt.Errorf("failed to decode BlobPointer: %w", err)
		}
		blockNumber = blobPointer.BlockHeight
		index = blobPointer.ExtrinsicIndex
		indexType = TxIndex

	default:
		return nil, fmt.Errorf("unknown data availabilty message type: %d", msgType)
	}

	if a.fallbackS3Service != nil {
		a.logger.Info("AvailDAInfo: Fallback S3 storage service is enabled, trying to get data from s3 storage")
		var err error
		batchesData, err := a.fallbackS3Service.GetMultipleByHash(ctx, batchHashes)
		if err != nil {
			a.logger.Warn("AvailDAWarn: ❌  failed to read data from fallback s3 storage, err: %w", err)
		} else {
			a.logger.Info("AvailDAInfo: ✅  Succesfully fetched data from Avail S3 using fallbackS3Service")
			return batchesData, nil
		}
	}

	var blobData []byte
	blobDataCh := make(chan struct {
		data []byte
		err  error
	}, 1)
	a.logger.Info("AvailDAInfo: 📥 Retrieving data from AvailDA")
	go func() {
		data, err := a.getData(blockNumber, index, indexType)
		blobDataCh <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-blobDataCh:
		if res.err != nil {
			a.logger.Error("AvailDAError: unable to read data from AvailDA & Fallback s3 storage")
			return nil, fmt.Errorf("cannot get data from block:%w", res.err)
		}
		blobData = res.data
		a.logger.Info("AvailDAInfo: ✅ Successfully able to retreive the data from AvailDA")
	}

	var batchesData [][]byte
	if err := rlp.DecodeBytes(blobData, &batchesData); err != nil {
		return nil, fmt.Errorf("cannot RLP decode data:%w", err)
	}

	a.logger.Info("AvailDAInfo: 📥 Sequence retrieved successfully", "num_batches", len(batchesData))
	return batchesData, nil
}

func (a *AvailBackend) submitData(ctx context.Context, sequence []byte) (avail_sdk.TransactionDetails, error) {
	resultCh := make(chan struct {
		details avail_sdk.TransactionDetails
		err     error
	}, 1)

	// Run the blocking SDK call in a goroutine
	go func() {
		// Transaction will be signed, sent, and watched
		// If the transaction was dropped or never executed, the system will retry it
		// for 2 more times using the same nonce and app id.
		//
		// Waits for finalization to finalize the transaction.
		tx := a.sdk.Tx.DataAvailability.SubmitData(sequence)
		txDetails, err := tx.ExecuteAndWatchFinalization(
			a.acc,
			avail_sdk.NewTransactionOptions().WithAppId(uint32(a.appId)),
		)

		if err == nil {
			// Check success
			// Returns None if there was no way to determine the
			// success status of a transaction. Otherwise it returns
			// true or false.
			status := txDetails.IsSuccessful().UnsafeUnwrap()
			if !status {
				err = fmt.Errorf("⚠️ extrinsic failed on avail chain, status: %v", status)
			}
		}

		resultCh <- struct {
			details avail_sdk.TransactionDetails
			err     error
		}{txDetails, err}
	}()

	// Now wait for either SDK result or context cancellation
	select {
	case <-ctx.Done():
		return avail_sdk.TransactionDetails{}, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			return avail_sdk.TransactionDetails{}, fmt.Errorf("⚠️ extrinsic got rejected: %w", res.err)
		}

		a.logger.Debug("AvailDADebug: ✅ Data is included in Avail chain",
			"address", a.address,
			"appID", a.appId,
			"block_number", res.details.BlockNumber,
			"block_hash", res.details.BlockHash,
			"tx_index", res.details.TxIndex,
		)
		return res.details, nil
	}
}

func (a *AvailBackend) getMerkleProofFromAvailBridge(ctx context.Context, blockHash primitives.H256, txIndex uint32) (*MerkleProofInput, error) {

	a.logger.Info("AvailDAInfo: ℹ️ Querying merkle proof of data submitted from Avail Bridge for attesting on settlement layer", "blockHash", blockHash, "txIndex", txIndex)
	var input *BridgeAPIResponse
	waitTime := time.Duration(a.bridgeTimeout) * time.Second
	retryCount := BridgeApiRetryCount
	for retryCount > 0 {
		url := fmt.Sprintf("%s/eth/proof/%s?index=%d", a.bridgeApi, blockHash.String(), txIndex)
		a.logger.Debug("AvailDAInfo: ℹ️ Querying Bridge for merkle proof", "URL", url)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("new request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			a.logger.Info("AvailDAInfo: ✅ Attestation proof received")
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("cannot read body:%w", err)
			}
			input = &BridgeAPIResponse{}
			err = json.Unmarshal(data, input)
			if err != nil {
				return nil, fmt.Errorf("cannot unmarshal data:%w", err)
			}
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
		a.logger.Debug("AvailDAWarn: ⏳ Attestation proof RPC errored, response code: %v, retry count left: %v, retrying in %v", resp.StatusCode, (retryCount - 1), waitTime)

		timer := time.NewTimer(waitTime)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			retryCount--
		}
	}

	if input == nil {
		return nil, fmt.Errorf("didn't get any proof from bridge api")
	}

	a.logger.Debug("AvailDAInfo: 🔗 Attestation proof: %+v", input)

	merkleProofInput := NewMerkleProofInput(input)

	return merkleProofInput, nil
}

type IndexType string

const (
	LeafIndex IndexType = "leaf"
	TxIndex   IndexType = "tx"
)

func (a *AvailBackend) getData(blockNumber uint32, index uint32, indexType IndexType) ([]byte, error) {
	blockHash, err := a.sdk.Client.BlockHash(blockNumber)
	if err != nil {
		return nil, fmt.Errorf("❎ Cannot get block hash: %w", err)
	}

	block, err := avail_sdk.NewBlock(a.sdk.Client, blockHash)
	if err != nil {
		return nil, fmt.Errorf("❎ Cannot get block: %w", err)
	}

	var blob avail_sdk.DataSubmission

	switch indexType {
	case LeafIndex:
		blobs := block.DataSubmissions(avail_sdk.Filter{})
		if int(index) >= len(blobs) {
			return nil, fmt.Errorf("❎ Unable to retrieve blob at index %d from block %d", index, blockNumber)
		}
		blob = blobs[index]

	case TxIndex:
		blobs := block.DataSubmissions(avail_sdk.Filter{}.WTxIndex(index))
		if len(blobs) == 0 {
			return nil, fmt.Errorf("❎ No blobs found for transaction index %d in block %d", index, blockNumber)
		}
		blob = blobs[0]

	default:
		return nil, fmt.Errorf("❎ Invalid index type: %v", indexType)
	}

	signerAddress, err := primitives.NewAccountIdFromMultiAddress(blob.TxSigner)
	if err != nil {
		a.logger.Warn("AvailDAWarn:‼️ Unable to extract the signer address for the blob")
	}

	a.logger.Debug("AvailDADebug: ✅ Data retrieved from Avail chain",
		" signer: ", signerAddress.ToHuman(),
		", appID: ", blob.AppId,
		", extrinsicHash: ", blob.TxHash,
	)

	return blob.Data, nil
}
