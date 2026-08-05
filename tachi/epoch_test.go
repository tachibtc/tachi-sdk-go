package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestEpochService_Get(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_epoch", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "id", "42")
		testQueryParamAbsent(t, r, "hash")
		writeJSON(t, w, GetEpochResponse{Height: 42, Status: "closed"})
	})
	got, _, err := ts.client.Epoch.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Height != 42 || got.Status != "closed" {
		t.Errorf("Get() = %+v, unexpected", got)
	}
}

func TestEpochService_ByHash(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_epoch", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hash", "abcd1234")
		testQueryParamAbsent(t, r, "id")
		writeJSON(t, w, GetEpochResponse{Hash: "abcd1234"})
	})
	got, _, err := ts.client.Epoch.ByHash(context.Background(), "abcd1234")
	if err != nil {
		t.Fatalf("ByHash: %v", err)
	}
	if got.Hash != "abcd1234" {
		t.Errorf("Hash = %q, want abcd1234", got.Hash)
	}
}

func TestEpochService_List(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listEpochs", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "page", "1")
		testQueryParam(t, r, "page_size", "5")
		writeJSON(t, w, ListEpochsResponse{Total: 19, Page: 1, PageSize: 5})
	})
	got, _, err := ts.client.Epoch.List(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Total != 19 {
		t.Errorf("Total = %d, want 19", got.Total)
	}
}
