//go:build integration
// +build integration

package avail

import (
	"context"
	"testing"

	"github.com/0xPolygon/cdk/log"
	avail_sdk "github.com/availproject/avail-go-sdk/sdk"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
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
		log.GetDefaultLogger(),
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
	availBackend := createAvailBackend(t)
	batchesData := [][]byte{[]byte("This is the power of Avail Data Availability layer!")}
	dataAvailabilityMsg, err := availBackend.PostSequence(ctx, batchesData)
	require.NoError(t, err)

	retrieved, err := availBackend.GetSequence(ctx, []common.Hash{}, dataAvailabilityMsg)
	require.NoError(t, err)
	require.Equal(t, batchesData, retrieved)

	t.Logf("Message from AvailDA: %s", string(retrieved[0]))
}

func TestSubmitAndGetData(t *testing.T) {
	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	availBackend := createAvailBackend(t)

	data := []byte("This is the power of Avail Data Availability layer!")
	txDetails, err := availBackend.submitData(ctx, data)
	require.NoError(t, err)

	t.Logf("Tx included: block=%d, hash=%s, index=%d", txDetails.BlockNumber, txDetails.BlockHash, txDetails.TxIndex)

	data, err = availBackend.getData(txDetails.BlockNumber, txDetails.TxIndex, TxIndex)
	require.NoError(t, err)
	t.Logf("AvailDAInfo: Data: %x", data)
}
