package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestSignService_Transaction(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_signTransaction", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testBody(t, r, map[string]interface{}{
			"version":  float64(2),
			"locktime": float64(0),
			"inputs": []interface{}{
				map[string]interface{}{
					"prevout":        map[string]interface{}{"hash": "abcd", "index": float64(0)},
					"sequence":       float64(0),
					"witnessUtxo":    map[string]interface{}{"value": float64(1000), "script": "51"},
					"tapLeafScript":  []interface{}{map[string]interface{}{"leafVersion": float64(0xc0), "script": "aa", "controlBlock": "bb"}},
					"tapInternalKey": "cc",
					"sighashType":    float64(0),
				},
			},
			"outputs": []interface{}{
				map[string]interface{}{"value": float64(900), "script": "52"},
			},
			"userSig": "deadbeef",
		})
		writeJSON(t, w, SignTransactionResponse{
			Signatures: 5,
			Refund: &RefundTx{
				Version: 2,
				Inputs: []RefundInput{{
					TapScriptSig: []RefundTapSig{{Pubkey: "pk1", LeafHash: "lh1", Signature: "sig1"}},
				}},
			},
		})
	})

	tx := &RefundTx{
		Version:  2,
		Locktime: 0,
		Inputs: []RefundInput{{
			Prevout:        RefundPrevout{Hash: "abcd", Index: 0},
			WitnessUtxo:    RefundWitnessUtxo{Value: 1000, Script: "51"},
			TapLeafScript:  []RefundTapLeaf{{LeafVersion: 0xc0, Script: "aa", ControlBlock: "bb"}},
			TapInternalKey: "cc",
		}},
		Outputs: []RefundOutput{{Value: 900, Script: "52"}},
		UserSig: "deadbeef",
	}

	got, _, err := ts.client.Sign.Transaction(context.Background(), tx)
	if err != nil {
		t.Fatalf("Transaction: %v", err)
	}
	if got.Signatures != 5 {
		t.Errorf("Signatures = %d, want 5", got.Signatures)
	}
	if len(got.Refund.Inputs[0].TapScriptSig) != 1 {
		t.Fatalf("TapScriptSig len = %d, want 1", len(got.Refund.Inputs[0].TapScriptSig))
	}
	if got.Refund.Inputs[0].TapScriptSig[0].Pubkey != "pk1" {
		t.Errorf("Pubkey = %q, want pk1", got.Refund.Inputs[0].TapScriptSig[0].Pubkey)
	}
}

func TestSignService_Transaction_ThresholdNotReached(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_signTransaction", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "threshold not reached", http.StatusGatewayTimeout)
	})
	_, _, err := ts.client.Sign.Transaction(context.Background(), &RefundTx{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errResp, ok := err.(*ErrorResponse)
	if !ok {
		t.Fatalf("error is not *ErrorResponse: %T", err)
	}
	if errResp.Response.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("StatusCode = %d, want 504", errResp.Response.StatusCode)
	}
}

func TestSignService_Transfer(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_signTransfer", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		writeJSON(t, w, TransferCosignResponse{
			Signatures: 5,
			Transfer:   &TransferTx{Version: 2, Inputs: []RefundInput{{TapScriptSig: []RefundTapSig{{Pubkey: "pk1"}}}}},
		})
	})
	got, _, err := ts.client.Sign.Transfer(context.Background(), &TransferTx{Version: 2, UserSig: "deadbeef"})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if got.Signatures != 5 || got.Transfer.Inputs[0].TapScriptSig[0].Pubkey != "pk1" {
		t.Errorf("got %+v", got)
	}
}
