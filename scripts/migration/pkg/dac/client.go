package dac

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// JSON-RPC request format
type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSON-RPC response format
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func GetDataFromDACByHash(ctx context.Context, dacURL string, hash common.Hash) ([]byte, error) {
	// Build request
	reqBody := rpcRequest{
		JSONRPC: "2.0",
		Method:  "sync_getOffChainData",
		Params:  []interface{}{hash},
		ID:      1,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	// Make HTTP call
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dacURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status %d: %s", resp.StatusCode, string(b))
	}

	// Decode response
	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Handle error or result
	if rpcResp.Error != nil {
		fmt.Printf("RPC Error: code=%d, msg=%s\n", rpcResp.Error.Code, rpcResp.Error.Message)
		return nil, fmt.Errorf("rpc error code=%d msg=%s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	resData := strings.Trim(string(rpcResp.Result), "\"")
	decoded, err := hex.DecodeString(resData[2:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	return decoded, nil
}
