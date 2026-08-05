package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestVaultService_List(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_listVaults", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "user", "bcrt1puser")
		testQueryParam(t, r, "page", "1")
		testQueryParam(t, r, "page_size", "10")
		writeJSON(t, w, ListVaultsResponse{User: "abcd", Total: 2})
	})
	got, _, err := ts.client.Vault.List(context.Background(), "bcrt1puser", 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total = %d, want 2", got.Total)
	}
}

func TestVaultService_List_APIKeyForwarded(t *testing.T) {
	mux := http.NewServeMux()
	server := setupServer(t, mux)
	c, err := NewClient(WithBaseURL(server), WithAPIKey("mastersecret"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	mux.HandleFunc("/tachi_listVaults", func(w http.ResponseWriter, r *http.Request) {
		testHeader(t, r, "X-Api-Key", "mastersecret")
		writeJSON(t, w, ListVaultsResponse{
			Vaults: []VaultListItem{{VaultID: "v1", CSVDelay: 144, Threshold: 5}},
		})
	})
	got, _, err := c.Vault.List(context.Background(), "bcrt1puser", 0, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Vaults) != 1 || got.Vaults[0].CSVDelay != 144 {
		t.Errorf("Vaults = %+v, unexpected", got.Vaults)
	}
}
