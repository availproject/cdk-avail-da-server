package rpc

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/availproject/cdk-avail-da-server/da"
	"github.com/availproject/cdk-avail-da-server/service"
)

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      int         `json:"id"`
}

func NewHandler(a *da.AvailBackend, s *da.S3Backend) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var req RPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Failed to decode request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var result interface{}
		var err error

		switch req.Method {
		case "sync_getOffChainData":
			if len(req.Params) != 1 {
				err = ErrInvalidParams
				break
			}
			hash, _ := req.Params[0].(string)
			result, err = service.GetOffChainData(a, s, hash)
		default:
			err = ErrMethodNotFound
		}

		resp := RPCResponse{JSONRPC: "2.0", ID: req.ID}
		if err != nil {
			log.Printf("RPC request failed [%s]: %v (duration %v)", req.Method, err, time.Since(start))
			resp.Error = err.Error()
		} else {
			log.Printf("RPC request succeeded [%s] (duration %v)", req.Method, time.Since(start))
			resp.Result = result
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	})
}

var (
	ErrInvalidParams  = &RPCError{Code: -32602, Message: "Invalid params"}
	ErrMethodNotFound = &RPCError{Code: -32601, Message: "Method not found"}
)

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }
