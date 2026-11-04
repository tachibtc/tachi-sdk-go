package tachi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEVMService_Call(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_evmCall", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		b, _ := io.ReadAll(r.Body)
		if want := `{"to":"0xabc","data":"0x18160ddd","nonce":0}`; strings.TrimSpace(string(b)) != want {
			t.Errorf("body = %s, want %s", b, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":1,"gas_used":2400,"return_data":"0x01","logs":[]}`)
	})
	zero := uint64(0)
	got, _, err := ts.client.EVM.Call(context.Background(), &EVMCallRequest{To: "0xabc", Data: "0x18160ddd", Nonce: &zero})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got.Status != 1 || got.GasUsed != 2400 || got.ReturnData != "0x01" {
		t.Errorf("got %+v", got)
	}
}

func TestEVMService_Lookups(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_evmCode", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "0xc0de")
		writeJSON(t, w, EVMCodeResponse{Address: "0xc0de", Code: "0x6080"})
	})
	ts.mux.HandleFunc("/tachi_evmAccount", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "d61318bb")
		writeJSON(t, w, EVMAccountResponse{Address: "0xa", Pubkey: "d61318bb", Nonce: 3})
	})
	ts.mux.HandleFunc("/tachi_evmStorage", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "0xc0de")
		testQueryParam(t, r, "slot", "0x2")
		writeJSON(t, w, EVMStorageResponse{Value: "0x01"})
	})
	ctx := context.Background()
	if c, _, err := ts.client.EVM.Code(ctx, "0xc0de"); err != nil || c.Code != "0x6080" {
		t.Errorf("Code = %+v, %v", c, err)
	}
	if a, _, err := ts.client.EVM.Account(ctx, "d61318bb"); err != nil || a.Nonce != 3 {
		t.Errorf("Account = %+v, %v", a, err)
	}
	if st, _, err := ts.client.EVM.Storage(ctx, "0xc0de", "0x2"); err != nil || st.Value != "0x01" {
		t.Errorf("Storage = %+v, %v", st, err)
	}
}

// The daemon embeds the execution fields flat in the receipt and in tx.evm;
// these decode from the daemon's literal wire shape.
func TestEVMService_Receipt(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_evmReceipt", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hash", "aa")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tx_hash":"aa","height":1501600,"from":"0xf","gas_limit":400000,"gas_price":1,
			"status":1,"gas_used":51234,"fee_sats":52,"refund_sats":348,"refund_vtxo_id":"9f",
			"logs":[{"log_index":1,"address":"0xc","topics":["0xdd"],"data":"0x"}]}`)
	})
	got, _, err := ts.client.EVM.Receipt(context.Background(), "aa")
	if err != nil {
		t.Fatalf("Receipt: %v", err)
	}
	if got.Height != 1501600 || got.GasUsed != 51234 || got.RefundVTXOID != "9f" || len(got.Logs) != 1 || got.Logs[0].LogIndex != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestGetTransactionResponse_EVM(t *testing.T) {
	var pending, done GetTransactionResponse
	if err := json.Unmarshal([]byte(`{"type":"evm_call","evm":{"from":"0xf","to":"0xc","nonce":2,"executed":false}}`), &pending); err != nil {
		t.Fatal(err)
	}
	if pending.EVM == nil || pending.EVM.To != "0xc" || pending.EVM.EVMExecution != nil {
		t.Errorf("pending evm = %+v", pending.EVM)
	}
	if err := json.Unmarshal([]byte(`{"type":"evm_deploy","vout":[{"owner":"ab","amount":348,"script":"","refund":true}],
		"evm":{"from":"0xf","contract_address":"0xc","executed":true,"status":1,"gas_used":7,"fee_sats":1,"logs":[]}}`), &done); err != nil {
		t.Fatal(err)
	}
	if done.EVM.EVMExecution == nil || done.EVM.GasUsed != 7 || !done.Vout[0].Refund {
		t.Errorf("done = %+v / %+v", done.EVM, done.Vout)
	}
}
