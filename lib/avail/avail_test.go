package avail

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	avail_sdk "github.com/availproject/avail-go-sdk/sdk"
	"github.com/ethereum/go-ethereum/common"
)

func createAvailBackend(t *testing.T) AvailBackend {
	var config Config
	err := config.GetConfig("./avail-config.json")
	if err != nil {
		t.Fatalf("cannot get config: %+v", err)
	}

	t.Logf("AvailDAInfo: Config: %+v", config)

	sdk, err := avail_sdk.NewSDK(config.HttpApiUrl)
	if err != nil {
		t.Fatalf("AvailDAError: error connecting to %s: %+v", config.HttpApiUrl, err)
	}

	appId := 0
	if config.AppID != 0 {
		appId = config.AppID
	}

	acc, err := avail_sdk.Account.NewKeyPair(config.Seed)
	if err != nil {
		t.Fatalf("AvailDAError: unable to generate keypair from given seed")
	}

	t.Logf("AvailDAInfo: Using KeyringPair with address %s", acc.SS58Address(AvailNetworkID))
	t.Log("AvailDAInfo: Avail backend client created successfully")

	return AvailBackend{
		sdk, acc, acc.SS58Address(AvailNetworkID),
		appId, config.HttpApiUrl, false,
		config.BridgeApiUrl, nil, config.BridgeTimeout, nil,
	}
}

func TestPostAndGetSequence(t *testing.T) {
	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	message := "This is the power of Avail Data Availability layer"

	batchData := [][]byte{[]byte(message)}

	availBackend := createAvailBackend(t)

	dataAvailabilityMsgbytes, err := availBackend.PostSequence(ctx, batchData)
	if err != nil {
		t.Fatalf("Unable to post the msg over avail chain: %v", err)
	}

	msgBytes, err := availBackend.GetSequence(ctx, []common.Hash{}, dataAvailabilityMsgbytes)
	if err != nil {
		t.Fatalf("Unable to retrieve the msg from avail chain: %v", err)
	}

	t.Logf("Message from AvailDA: %s", string(msgBytes[0]))
}

func TestAvailDASubmitData(t *testing.T) {
	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	randomBytes, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("Error generating random bytes: %v", err)
	}

	availBackend := createAvailBackend(t)
	txDetails, err := availBackend.submitData(ctx, randomBytes)
	if err != nil {
		t.Fatalf("cannot submit data: %+v", err)
	}

	t.Logf("AvailDAInfo: Tx included in Avail chain. address=%s, appID=%d, block_number=%d, block_hash=%s, tx_index=%d",
		availBackend.address, availBackend.appId, txDetails.BlockNumber, txDetails.BlockHash, txDetails.TxIndex)
	t.Logf("/eth/proof/%s?index=%d", txDetails.BlockHash.String(), txDetails.TxIndex)
}

func TestAvailDAGetData(t *testing.T) {
	blockNumber := 1315421
	leafIndex := 3

	availBackend := createAvailBackend(t)
	data, err := availBackend.getData(uint32(blockNumber), uint32(leafIndex), LeafIndex)
	if err != nil {
		t.Fatalf("unable to get data: %+v", err)
	}
	t.Logf("AvailDAInfo: Data: %x", data)
}

func TestAvailDAMerkleProofInputEncodeToBinary(t *testing.T) {
	merkleProofInput := MerkleProofInput{
		DataRootProof: [][32]byte{
			{173, 50, 40, 182, 118, 247, 211, 205, 66, 132, 165, 68, 63, 23, 241, 150, 43, 54, 228, 145, 179, 10, 64, 178, 64, 88, 73, 229, 151, 186, 95, 181},
			{81, 248, 78, 114, 121, 205, 246, 172, 184, 26, 247, 122, 236, 100, 246, 24, 247, 16, 41, 183, 217, 198, 211, 124, 3, 92, 55, 19, 78, 81, 122, 242},
			{105, 200, 69, 141, 214, 45, 39, 234, 154, 189, 64, 88, 108, 229, 62, 82, 32, 212, 59, 98, 108, 39, 247, 100, 104, 165, 126, 148, 52, 127, 13, 107},
			{90, 2, 30, 101, 234, 92, 107, 118, 70, 155, 104, 219, 40, 199, 163, 144, 131, 110, 34, 194, 28, 111, 149, 205, 239, 77, 52, 8, 235, 107, 128, 80},
			{161, 66, 157, 58, 135, 235, 114, 86, 237, 149, 23, 84, 136, 117, 99, 160, 214, 72, 96, 196, 136, 84, 26, 110, 157, 79, 171, 38, 135, 227, 6, 88},
			{153, 192, 154, 157, 133, 237, 44, 209, 222, 10, 185, 2, 105, 111, 216, 21, 5, 111, 194, 157, 75, 78, 12, 35, 201, 13, 85, 176, 215, 158, 165, 212},
			{252, 225, 122, 81, 66, 13, 91, 136, 167, 217, 26, 113, 105, 115, 101, 74, 156, 70, 52, 87, 251, 31, 177, 227, 246, 41, 119, 71, 58, 220, 71, 230},
			{233, 226, 152, 144, 241, 188, 123, 237, 78, 185, 34, 25, 84, 239, 42, 78, 171, 75, 105, 167, 49, 1, 130, 135, 124, 242, 184, 189, 178, 200, 245, 28},
			{88, 91, 133, 61, 102, 164, 187, 200, 166, 83, 185, 7, 241, 98, 72, 83, 162, 54, 165, 33, 100, 138, 34, 86, 171, 109, 167, 239, 210, 171, 67, 97},
			{80, 109, 134, 88, 45, 37, 36, 5, 184, 64, 1, 135, 146, 202, 210, 191, 18, 89, 241, 239, 90, 165, 248, 135, 225, 60, 178, 240, 9, 79, 81, 225},
			{255, 255, 10, 215, 230, 89, 119, 47, 149, 52, 193, 149, 200, 21, 239, 196, 1, 78, 241, 225, 218, 237, 68, 4, 192, 99, 133, 209, 17, 146, 233, 43},
		},
		LeafProof:     [][32]byte{}, // Empty
		RangeHash:     [32]byte{218, 51, 52, 143, 216, 175, 43, 27, 235, 186, 222, 186, 213, 219, 94, 198, 195, 197, 45, 79, 57, 244, 190, 143, 134, 173, 120, 171, 103, 106, 125, 226},
		DataRootIndex: big.NewInt(143),
		BlobRoot:      [32]byte{63, 109, 5, 38, 27, 218, 88, 55, 255, 136, 45, 122, 132, 99, 67, 15, 186, 205, 124, 227, 52, 227, 9, 212, 73, 236, 182, 45, 72, 172, 209, 67},
		BridgeRoot:    [32]byte{}, // Empty
		Leaf:          [32]byte{142, 204, 27, 154, 88, 162, 236, 187, 225, 40, 188, 34, 193, 120, 141, 171, 109, 118, 85, 107, 113, 166, 157, 42, 221, 12, 250, 89, 34, 159, 50, 172},
		LeafIndex:     big.NewInt(0),
	}

	encodedProof, err := merkleProofInput.EnodeToBinary()
	if err != nil {
		t.Fatalf("Failed to encode MerkleProofInput: %v", err)
	}

	dataAvailabilityMsg, err := PackEnvelopeWithMsgType(DAM_TYPE_MERKLE_PROOF, encodedProof)
	if err != nil {
		t.Fatalf("Failed to pack envelope with message type: %v", err)
	}

	t.Logf("Encoded MerkleProofInput with DAM type flag: %x", dataAvailabilityMsg)
}

// GenerateRandomBytes returns a random byte slice of the given size
func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
