package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestNodeService_Health(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, HealthResponse{Status: "ok", Validators: 3})
	})
	got, _, err := ts.client.Node.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if got.Status != "ok" {
		t.Errorf("Status = %q, want ok", got.Status)
	}
}

func TestNodeService_Info(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_nodeInfo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, NodeInfoResponse{Version: "0.39.0", Peers: 6})
	})
	got, _, err := ts.client.Node.Info(context.Background())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got.Peers != 6 {
		t.Errorf("Peers = %d, want 6", got.Peers)
	}
}

func TestNodeService_Status(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0", ID: 1})
	})
	got, _, err := ts.client.Node.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", got.JSONRPC)
	}
}

func TestNodeService_NetInfo(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_netInfo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0"})
	})
	if _, _, err := ts.client.Node.NetInfo(context.Background()); err != nil {
		t.Fatalf("NetInfo: %v", err)
	}
}

func TestNodeService_ConsensusState(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_consensusState", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0"})
	})
	if _, _, err := ts.client.Node.ConsensusState(context.Background()); err != nil {
		t.Fatalf("ConsensusState: %v", err)
	}
}

func TestNodeService_ValidatorsPower(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validatorsPower", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0"})
	})
	if _, _, err := ts.client.Node.ValidatorsPower(context.Background()); err != nil {
		t.Fatalf("ValidatorsPower: %v", err)
	}
}

func TestNodeService_Query(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_query", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "path", "vtxo")
		testQueryParam(t, r, "data", "deadbeef")
		testQueryParam(t, r, "height", "100")
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0"})
	})
	_, _, err := ts.client.Node.Query(context.Background(), "vtxo", &QueryOptions{DataHex: "deadbeef", Height: "100"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
}

func TestNodeService_Query_NilOptions(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_query", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "path", "supply")
		testQueryParamAbsent(t, r, "data")
		testQueryParamAbsent(t, r, "height")
		writeJSON(t, w, CometRPCResponse{})
	})
	if _, _, err := ts.client.Node.Query(context.Background(), "supply", nil); err != nil {
		t.Fatalf("Query: %v", err)
	}
}

func TestNodeService_ChainHealth_Unhealthy(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/health/chain", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(t, w, ChainHealthResponse{Status: "unhealthy", Problems: []string{"no block committed"}, Height: 42})
	})
	got, _, err := ts.client.Node.ChainHealth(context.Background())
	if _, ok := err.(*ErrorResponse); !ok {
		t.Fatalf("err = %v, want *ErrorResponse", err)
	}
	if got == nil || got.Status != "unhealthy" || got.Height != 42 || len(got.Problems) != 1 {
		t.Errorf("got %+v, want decoded unhealthy body", got)
	}
}
