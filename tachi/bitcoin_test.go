package tachi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestBitcoinService_RPC_Success(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      string          `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.JSONRPC != "1.0" {
			t.Errorf("JSONRPC = %q, want 1.0", req.JSONRPC)
		}
		if req.Method != "getblockchaininfo" {
			t.Errorf("Method = %q, want getblockchaininfo", req.Method)
		}
		if string(req.Params) != "[]" {
			t.Errorf("Params = %s, want [] (nil params should encode as empty array)", req.Params)
		}
		writeJSON(t, w, BitcoinRPCResponse{
			Result: json.RawMessage(`{"chain":"regtest"}`),
			ID:     "1",
		})
	})

	result, _, err := ts.client.Bitcoin.RPC(context.Background(), "getblockchaininfo", nil)
	if err != nil {
		t.Fatalf("RPC: %v", err)
	}
	if !strings.Contains(string(result), "regtest") {
		t.Errorf("result = %s, missing expected content", result)
	}
}

func TestBitcoinService_RPC_ParamsEncoded(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if string(req.Params) != `["abc",1]` {
			t.Errorf("Params = %s, want [\"abc\",1]", req.Params)
		}
		writeJSON(t, w, BitcoinRPCResponse{Result: json.RawMessage(`null`)})
	})
	if _, _, err := ts.client.Bitcoin.RPC(context.Background(), "somemethod", []interface{}{"abc", 1}); err != nil {
		t.Fatalf("RPC: %v", err)
	}
}

func TestBitcoinService_RPC_BitcoindError(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, BitcoinRPCResponse{
			Error: &BitcoinRPCError{Code: -8, Message: "invalid parameter"},
		})
	})
	_, _, err := ts.client.Bitcoin.RPC(context.Background(), "getblock", []interface{}{"badhash"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid parameter") {
		t.Errorf("error = %q, missing bitcoind message", err.Error())
	}
}

func TestBitcoinRPCError_Error(t *testing.T) {
	e := &BitcoinRPCError{Code: -8, Message: "invalid parameter"}
	got := e.Error()
	if !strings.Contains(got, "-8") || !strings.Contains(got, "invalid parameter") {
		t.Errorf("Error() = %q, missing code or message", got)
	}
}

func TestBitcoinService_RPC_TransportError(t *testing.T) {
	// A 403 (missing X-Api-Key on a privileged method) is a transport/HTTP
	// level failure, distinct from a bitcoind-level JSON-RPC error, and
	// must surface as *ErrorResponse.
	ts := setup(t)
	ts.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	_, _, err := ts.client.Bitcoin.RPC(context.Background(), "getwalletinfo", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ErrorResponse); !ok {
		t.Fatalf("error is not *ErrorResponse: %T", err)
	}
}
