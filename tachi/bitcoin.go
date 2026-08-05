package tachi

import (
	"context"
	"encoding/json"
	"fmt"
)

// BitcoinService wraps the daemon's pass-through Bitcoin Core JSON-RPC
// proxy (POST /).
type BitcoinService service

// BitcoinRPCError describes an error returned by bitcoind.
type BitcoinRPCError struct {
	// Code is the bitcoind JSON-RPC error code (negative integers per bitcoind convention).
	Code int `json:"code" example:"-8"`
	// Message is the human-readable error description from bitcoind.
	Message string `json:"message" example:"invalid parameter"`
}

func (e *BitcoinRPCError) Error() string {
	return fmt.Sprintf("code %d: %s", e.Code, e.Message)
}

// BitcoinRPCResponse is the JSON-RPC 1.0 envelope returned by POST / (the
// bitcoind proxy).
type BitcoinRPCResponse struct {
	// Result is the raw JSON result body from bitcoind; nil when an error occurred.
	Result json.RawMessage `json:"result" swaggertype:"object"`
	// Error carries the bitcoind error payload when the call failed; nil on success.
	Error *BitcoinRPCError `json:"error"`
	// ID echoes back the caller-supplied request identifier.
	ID string `json:"id" example:"1"`
}

// RPC forwards a JSON-RPC 1.0 request to the daemon's configured bitcoind
// (POST /). method is the bitcoind RPC method name (e.g.
// "getblockchaininfo"); params is marshalled as-is into the request's
// params array/object — pass nil for no parameters.
//
// Common read-only methods (chain/mempool/tx-decoding queries) are
// forwarded with no authentication. Any other method (wallet, admin,
// network-mutation) requires the Client to have been created with
// WithAPIKey using the daemon's master BTC_RPC_API_KEY — a missing or
// non-matching key on a non-common method returns a 403 *ErrorResponse.
//
// On a bitcoind-level error (non-nil BitcoinRPCResponse.Error), RPC
// returns that error wrapped so callers can still access the raw
// BitcoinRPCError via errors.As; on a transport/HTTP-level error it
// returns an *ErrorResponse as usual.
func (s *BitcoinService) RPC(ctx context.Context, method string, params interface{}) (json.RawMessage, *Response, error) {
	if params == nil {
		params = []interface{}{}
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, nil, fmt.Errorf("tachi: encode params: %w", err)
	}

	req := struct {
		// JSONRPC is the JSON-RPC protocol version, fixed at "1.0" for bitcoind compatibility.
		JSONRPC string `json:"jsonrpc"`
		// ID is the caller-chosen request identifier echoed back in the response.
		ID string `json:"id"`
		// Method is the bitcoind RPC method name to invoke.
		Method string `json:"method"`
		// Params is the raw JSON parameter array forwarded unchanged to bitcoind.
		Params json.RawMessage `json:"params"`
	}{
		JSONRPC: "1.0",
		ID:      "1",
		Method:  method,
		Params:  paramsJSON,
	}

	var out BitcoinRPCResponse
	httpResp, err := s.client.post(ctx, "", req, &out)
	if err != nil {
		return nil, httpResp, err
	}
	if out.Error != nil {
		return nil, httpResp, fmt.Errorf("tachi: bitcoind %s: %w", method, out.Error)
	}
	return out.Result, httpResp, nil
}
