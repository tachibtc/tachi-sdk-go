package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestWatchtowerService_Status(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_watchtower/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, WatchtowerStatus{Mode: "detection", ReceiptCount: 2, SweepThreshold: 5})
	})
	got, _, err := ts.client.Watchtower.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.Mode != "detection" {
		t.Errorf("Mode = %q, want detection", got.Mode)
	}
}

func TestWatchtowerService_Status_Disabled(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_watchtower/status", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "watchtower disabled", http.StatusServiceUnavailable)
	})
	_, _, err := ts.client.Watchtower.Status(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errResp, ok := err.(*ErrorResponse)
	if !ok || errResp.Response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 *ErrorResponse, got %v", err)
	}
}

func TestWatchtowerService_Receipts_All(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_watchtower/receipts", func(w http.ResponseWriter, r *http.Request) {
		testQueryParamAbsent(t, r, "vault")
		writeJSON(t, w, map[string]interface{}{
			"count":    2,
			"receipts": []BreachReceipt{{VaultID: "v1"}, {VaultID: "v2"}},
		})
	})
	got, _, err := ts.client.Watchtower.Receipts(context.Background(), "")
	if err != nil {
		t.Fatalf("Receipts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Receipts) = %d, want 2", len(got))
	}
}

func TestWatchtowerService_Receipts_FilteredByVault(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_watchtower/receipts", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "vault", "vault1")
		testQueryParamAbsent(t, r, "state")
		writeJSON(t, w, map[string]interface{}{
			"count":    1,
			"receipts": []BreachReceipt{{VaultID: "vault1"}},
		})
	})
	got, _, err := ts.client.Watchtower.Receipts(context.Background(), "vault1")
	if err != nil {
		t.Fatalf("Receipts: %v", err)
	}
	if len(got) != 1 || got[0].VaultID != "vault1" {
		t.Errorf("Receipts = %+v, unexpected", got)
	}
}

func TestWatchtowerService_Receipt_Single(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_watchtower/receipts", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "vault", "vault1")
		testQueryParam(t, r, "state", "3")
		writeJSON(t, w, BreachReceipt{VaultID: "vault1", BroadcastState: 3, Classification: "legitimate"})
	})
	got, _, err := ts.client.Watchtower.Receipt(context.Background(), "vault1", 3)
	if err != nil {
		t.Fatalf("Receipt: %v", err)
	}
	if got.Classification != "legitimate" {
		t.Errorf("Classification = %q, want legitimate", got.Classification)
	}
}

func TestWatchtowerService_Receipt_NotFound(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_watchtower/receipts", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "receipt not found", http.StatusNotFound)
	})
	_, _, err := ts.client.Watchtower.Receipt(context.Background(), "vault1", 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errResp, ok := err.(*ErrorResponse)
	if !ok || errResp.Response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 *ErrorResponse, got %v", err)
	}
}
