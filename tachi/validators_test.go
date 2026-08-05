package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestValidatorsService_List(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validators", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ValidatorsResponse{Count: 2, Validators: []ValidatorInfo{{PubKeyHex: "a"}, {PubKeyHex: "b"}}})
	})
	got, _, err := ts.client.Validators.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got.Count != 2 {
		t.Errorf("Count = %d, want 2", got.Count)
	}
}

func TestValidatorsService_Live(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validators/live", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, LiveValidatorsResponse{Count: 1, TotalKnown: 3})
	})
	got, _, err := ts.client.Validators.Live(context.Background())
	if err != nil {
		t.Fatalf("Live: %v", err)
	}
	if got.TotalKnown != 3 {
		t.Errorf("TotalKnown = %d, want 3", got.TotalKnown)
	}
}

func TestValidatorsService_Count(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validators/count", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ValidatorCountResponse{Count: 7})
	})
	got, _, err := ts.client.Validators.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if got.Count != 7 {
		t.Errorf("Count = %d, want 7", got.Count)
	}
}

func TestValidatorsService_Ready(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validators/ready", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "expected", "4")
		writeJSON(t, w, ReadyResponse{Ready: true, Count: 4})
	})
	got, _, err := ts.client.Validators.Ready(context.Background(), 4)
	if err != nil {
		t.Fatalf("Ready: %v", err)
	}
	if !got.Ready {
		t.Errorf("Ready = false, want true")
	}
}

func TestValidatorsService_Ready_DefaultOmitsParam(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validators/ready", func(w http.ResponseWriter, r *http.Request) {
		testQueryParamAbsent(t, r, "expected")
		writeJSON(t, w, ReadyResponse{})
	})
	if _, _, err := ts.client.Validators.Ready(context.Background(), 0); err != nil {
		t.Fatalf("Ready: %v", err)
	}
}

func TestValidatorsService_PeerInfo(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_peerInfo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, ValidatorInfo{PeerID: "self-peer", PubKeyHex: "03self"})
	})
	got, _, err := ts.client.Validators.PeerInfo(context.Background())
	if err != nil {
		t.Fatalf("PeerInfo: %v", err)
	}
	if got.PeerID != "self-peer" {
		t.Errorf("PeerID = %q, want self-peer", got.PeerID)
	}
}
