package tachi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestDashboardService_Stats(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, StatsResponse{Height: 100, CurrentEpoch: 5, NodeCount: 7})
	})
	got, _, err := ts.client.Dashboard.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if got.Height != 100 {
		t.Errorf("Height = %d, want 100", got.Height)
	}
}

func TestDashboardService_Supply(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_supply", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, SupplyResponse{TotalSupplySat: 500000, VTXOCount: 5})
	})
	got, _, err := ts.client.Dashboard.Supply(context.Background())
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}
	if got.VTXOCount != 5 {
		t.Errorf("VTXOCount = %d, want 5", got.VTXOCount)
	}
}

func TestDashboardService_Search(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_search", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "q", "42")
		writeJSON(t, w, SearchResponse{Type: "block", Result: json.RawMessage(`{"height":42}`)})
	})
	got, _, err := ts.client.Dashboard.Search(context.Background(), "42")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got.Type != "block" {
		t.Errorf("Type = %q, want block", got.Type)
	}
	var block struct {
		Height int `json:"height"`
	}
	if err := json.Unmarshal(got.Result, &block); err != nil {
		t.Fatalf("unmarshal Result: %v", err)
	}
	if block.Height != 42 {
		t.Errorf("Result.Height = %d, want 42", block.Height)
	}
}
