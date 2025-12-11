package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type rpcRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error,omitempty"`
}

func callRPC(addr string, method string, params interface{}, out interface{}) error {
	reqBody, err := json.Marshal(rpcRequest{
		Method: method,
		Params: params,
	})
	if err != nil {
		return fmt.Errorf("failed to encode RPC request: %w", err)
	}

	resp, err := http.Post("http://"+addr+"/rpc", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("RPC call error: %w", err)
	}

	// Make Linter happy
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return fmt.Errorf("failed to decode RPC response: %w", err)
	}

	if rpcResp.Error != "" {
		return fmt.Errorf("RPC error: %s", rpcResp.Error)
	}

	if out != nil {
		if err := json.Unmarshal(rpcResp.Result, out); err != nil {
			return fmt.Errorf("failed to decode RPC result: %w", err)
		}
	}

	return nil
}
