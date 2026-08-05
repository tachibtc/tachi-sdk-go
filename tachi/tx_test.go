package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestTxService_Get_NoOptions(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_tx", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hash", "abcd")
		testQueryParamAbsent(t, r, "hat")
		testQueryParamAbsent(t, r, "rip")
		writeJSON(t, w, GetTransactionResponse{TxHash: "abcd", Type: "transfer"})
	})
	got, _, err := ts.client.Tx.Get(context.Background(), "abcd", nil)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TxHash != "abcd" {
		t.Errorf("TxHash = %q, want abcd", got.TxHash)
	}
}

func TestTxService_Get_WithHATAndRIP(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_tx", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hat", "true")
		testQueryParam(t, r, "rip", "true")
		testQueryParam(t, r, "origin_epoch", "3")
		testQueryParam(t, r, "final_epoch", "5")
		writeJSON(t, w, GetTransactionResponse{TxHash: "abcd"})
	})
	_, _, err := ts.client.Tx.Get(context.Background(), "abcd", &TxOptions{
		HAT: true, RIP: true, OriginEpoch: 3, FinalEpoch: 5,
	})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestTxService_Raw(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_txRaw", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hash", "abcd")
		writeJSON(t, w, GetRawTransactionResponse{TxHash: "abcd", Hex: "0102"})
	})
	got, _, err := ts.client.Tx.Raw(context.Background(), "abcd")
	if err != nil {
		t.Fatalf("Raw: %v", err)
	}
	if got.Hex != "0102" {
		t.Errorf("Hex = %q, want 0102", got.Hex)
	}
}

func TestTxService_List(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listTransactions", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "before_height", "500")
		testQueryParam(t, r, "page_size", "20")
		writeJSON(t, w, ListTransactionsResponse{ScannedFromHeight: 500})
	})
	got, _, err := ts.client.Tx.List(context.Background(), &ListTransactionsOptions{BeforeHeight: 500, PageSize: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.ScannedFromHeight != 500 {
		t.Errorf("ScannedFromHeight = %d, want 500", got.ScannedFromHeight)
	}
}

func TestTxService_List_NilOptions(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listTransactions", func(w http.ResponseWriter, r *http.Request) {
		testQueryParamAbsent(t, r, "before_height")
		testQueryParamAbsent(t, r, "page_size")
		writeJSON(t, w, ListTransactionsResponse{})
	})
	if _, _, err := ts.client.Tx.List(context.Background(), nil); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestTxService_Mempool(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_mempool", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		writeJSON(t, w, MempoolResponse{Count: 3})
	})
	got, _, err := ts.client.Tx.Mempool(context.Background())
	if err != nil {
		t.Fatalf("Mempool: %v", err)
	}
	if got.Count != 3 {
		t.Errorf("Count = %d, want 3", got.Count)
	}
}

func TestTxService_Decode(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_txDecode", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testBody(t, r, map[string]string{"hex": "deadbeef"})
		writeJSON(t, w, TxDecodeResponse{TxHash: "hash1", Type: "deposit"})
	})
	got, _, err := ts.client.Tx.Decode(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Type != "deposit" {
		t.Errorf("Type = %q, want deposit", got.Type)
	}
}

func TestTxService_Validate(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_txValidate", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testBody(t, r, map[string]string{"hex": "deadbeef"})
		writeJSON(t, w, TxValidateResponse{Valid: true, Code: 0})
	})
	got, _, err := ts.client.Tx.Validate(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !got.Valid {
		t.Errorf("Valid = false, want true")
	}
}

func TestTxService_BroadcastSync(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_txBroadcastSync", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testBody(t, r, map[string]string{"tx": "deadbeef"})
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0", ID: 1})
	})
	got, _, err := ts.client.Tx.BroadcastSync(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("BroadcastSync: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", got.JSONRPC)
	}
}

func TestTxService_BroadcastAsync(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_txBroadcastAsync", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testBody(t, r, map[string]string{"tx": "deadbeef"})
		writeJSON(t, w, CometRPCResponse{JSONRPC: "2.0", ID: 1})
	})
	if _, _, err := ts.client.Tx.BroadcastAsync(context.Background(), "deadbeef"); err != nil {
		t.Fatalf("BroadcastAsync: %v", err)
	}
}

func TestTxService_FeeEstimate(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_feeEstimate", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, FeeEstimateResponse{MinFeeSat: 1, AvgFeeSat: 2, RecommendedFeeSat: 2})
	})
	got, _, err := ts.client.Tx.FeeEstimate(context.Background())
	if err != nil {
		t.Fatalf("FeeEstimate: %v", err)
	}
	if got.RecommendedFeeSat != 2 {
		t.Errorf("RecommendedFeeSat = %d, want 2", got.RecommendedFeeSat)
	}
}
