package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestBlockService_Get(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_block", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		testQueryParam(t, r, "height", "42")
		writeJSON(t, w, BlockResponse{Height: 42, Hash: "abc", TxCount: 0})
	})
	got, _, err := ts.client.Block.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Height != 42 || got.Hash != "abc" {
		t.Errorf("Get() = %+v, unexpected", got)
	}
}

func TestBlockService_List_Defaults(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listBlocks", func(w http.ResponseWriter, r *http.Request) {
		testQueryParamAbsent(t, r, "page")
		testQueryParamAbsent(t, r, "page_size")
		writeJSON(t, w, ListBlocksResponse{Total: 100})
	})
	got, _, err := ts.client.Block.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Total != 100 {
		t.Errorf("Total = %d, want 100", got.Total)
	}
}

func TestBlockService_List_Explicit(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listBlocks", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "page", "2")
		testQueryParam(t, r, "page_size", "10")
		writeJSON(t, w, ListBlocksResponse{Page: 2, PageSize: 10})
	})
	if _, _, err := ts.client.Block.List(context.Background(), 2, 10); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestBlockService_Hash(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_getBlockHash", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "height", "7")
		writeJSON(t, w, GetBlockHashResponse{Height: 7, Hash: "hhh"})
	})
	got, _, err := ts.client.Block.Hash(context.Background(), 7)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if got.Hash != "hhh" {
		t.Errorf("Hash = %q, want hhh", got.Hash)
	}
}

func TestBlockService_HeaderByHeight(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_getBlockHeader", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "height", "7")
		testQueryParamAbsent(t, r, "hash")
		writeJSON(t, w, GetBlockHeaderResponse{Height: 7})
	})
	if _, _, err := ts.client.Block.HeaderByHeight(context.Background(), 7); err != nil {
		t.Fatalf("HeaderByHeight: %v", err)
	}
}

func TestBlockService_HeaderByHash(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_getBlockHeader", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hash", "deadbeef")
		testQueryParamAbsent(t, r, "height")
		writeJSON(t, w, GetBlockHeaderResponse{Hash: "deadbeef"})
	})
	got, _, err := ts.client.Block.HeaderByHash(context.Background(), "deadbeef")
	if err != nil {
		t.Fatalf("HeaderByHash: %v", err)
	}
	if got.Hash != "deadbeef" {
		t.Errorf("Hash = %q, want deadbeef", got.Hash)
	}
}

func TestBlockService_ByHeight(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_getBlock", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "height", "9")
		writeJSON(t, w, BlockResponse{Height: 9})
	})
	got, _, err := ts.client.Block.ByHeight(context.Background(), 9)
	if err != nil {
		t.Fatalf("ByHeight: %v", err)
	}
	if got.Height != 9 {
		t.Errorf("Height = %d, want 9", got.Height)
	}
}

func TestBlockService_ByHash(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_getBlock", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "hash", "cafe")
		writeJSON(t, w, BlockResponse{Hash: "cafe"})
	})
	got, _, err := ts.client.Block.ByHash(context.Background(), "cafe")
	if err != nil {
		t.Fatalf("ByHash: %v", err)
	}
	if got.Hash != "cafe" {
		t.Errorf("Hash = %q, want cafe", got.Hash)
	}
}
