package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestVTXOService_Get(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_vtxo", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "id", "abcd1234")
		writeJSON(t, w, VTXOResponse{ID: "abcd1234", Amount: 100})
	})
	got, _, err := ts.client.VTXO.Get(context.Background(), "abcd1234")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Amount != 100 {
		t.Errorf("Amount = %d, want 100", got.Amount)
	}
}

func TestVTXOService_Locked(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_vtxoLocked", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "vault", "bcrt1pvault")
		writeJSON(t, w, LockedVTXOsResponse{Vault: "bcrt1pvault", Count: 1})
	})
	got, _, err := ts.client.VTXO.Locked(context.Background(), "bcrt1pvault")
	if err != nil {
		t.Fatalf("Locked: %v", err)
	}
	if got.Count != 1 {
		t.Errorf("Count = %d, want 1", got.Count)
	}
}

func TestVTXOService_List(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listVtxos", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "page", "1")
		testQueryParam(t, r, "page_size", "20")
		writeJSON(t, w, ListVTXOsResponse{Total: 15})
	})
	got, _, err := ts.client.VTXO.List(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Total != 15 {
		t.Errorf("Total = %d, want 15", got.Total)
	}
}
