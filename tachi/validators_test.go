package tachi

import (
	"context"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
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

func TestValidatorsService_Register(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_validators/register", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPost)
		testBody(t, r, map[string]interface{}{
			"peer_id":     "peer1",
			"pub_key_hex": "03abc",
			"timestamp":   float64(12345),
			"signature":   "sig",
		})
		writeJSON(t, w, RegisterResponse{Status: "registered", Total: 4})
	})
	req := &RegisterRequest{
		ValidatorInfo: ValidatorInfo{PeerID: "peer1", PubKeyHex: "03abc"},
		Timestamp:     12345,
		Signature:     "sig",
	}
	got, _, err := ts.client.Validators.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if got.Total != 4 {
		t.Errorf("Total = %d, want 4", got.Total)
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

// TestSignRegisterRequest_ProducesVerifiableSignature is a pure-logic test
// with no HTTP involved: it signs a request with SignRegisterRequest and
// checks the signature verifies against the same canonical digest the
// daemon computes independently (rpc.canonicalRegisterDigest) — a
// regression guard against accidentally changing field order or the
// domain tag, which would silently break every deployed client.
func TestSignRegisterRequest_ProducesVerifiableSignature(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())

	req := &RegisterRequest{
		ValidatorInfo: ValidatorInfo{
			PeerID:    "16Uiu2HAmpeer",
			PubKeyHex: pubKeyHex,
			Host:      "127.0.0.1:26656",
			P2PPort:   26656,
			RPCAddr:   "127.0.0.1:26670",
		},
	}
	if err := SignRegisterRequest(req, privKey); err != nil {
		t.Fatalf("SignRegisterRequest: %v", err)
	}
	if req.Timestamp == 0 {
		t.Fatal("Timestamp was not set")
	}
	if req.Signature == "" {
		t.Fatal("Signature was not set")
	}

	sigBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		t.Fatalf("decode signature hex: %v", err)
	}
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		t.Fatalf("parse signature: %v", err)
	}
	digest := canonicalRegisterDigest(req.ValidatorInfo, req.Timestamp)
	if !sig.Verify(digest[:], privKey.PubKey()) {
		t.Fatal("signature does not verify against the canonical digest")
	}
}

func TestSignRegisterRequest_RequiresPubKeyAndPrivKey(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("NewPrivateKey: %v", err)
	}
	if err := SignRegisterRequest(&RegisterRequest{}, privKey); err == nil {
		t.Error("expected error when PubKeyHex is empty, got nil")
	}
	if err := SignRegisterRequest(&RegisterRequest{ValidatorInfo: ValidatorInfo{PubKeyHex: "abc"}}, nil); err == nil {
		t.Error("expected error when privKey is nil, got nil")
	}
}

func TestCanonicalRegisterDigest_ChangesWithFields(t *testing.T) {
	base := ValidatorInfo{PeerID: "p1", PubKeyHex: "03abc", Host: "h1", P2PPort: 1, RPCAddr: "r1"}
	d1 := canonicalRegisterDigest(base, 1000)

	variants := []ValidatorInfo{
		{PeerID: "p2", PubKeyHex: "03abc", Host: "h1", P2PPort: 1, RPCAddr: "r1"},
		{PeerID: "p1", PubKeyHex: "03abd", Host: "h1", P2PPort: 1, RPCAddr: "r1"},
		{PeerID: "p1", PubKeyHex: "03abc", Host: "h2", P2PPort: 1, RPCAddr: "r1"},
		{PeerID: "p1", PubKeyHex: "03abc", Host: "h1", P2PPort: 2, RPCAddr: "r1"},
		{PeerID: "p1", PubKeyHex: "03abc", Host: "h1", P2PPort: 1, RPCAddr: "r2"},
	}
	for i, v := range variants {
		if d := canonicalRegisterDigest(v, 1000); d == d1 {
			t.Errorf("variant %d produced same digest as base; field change not reflected", i)
		}
	}
	if d := canonicalRegisterDigest(base, 1001); d == d1 {
		t.Error("changing timestamp did not change digest")
	}

	// PubKeyHex must be case-normalized (lowercased) before hashing, so a
	// caller that stores it uppercase still signs the same digest the
	// daemon (which also lowercases) verifies against.
	upper := base
	upper.PubKeyHex = "03ABC"
	if d := canonicalRegisterDigest(upper, 1000); d != d1 {
		t.Error("digest is not case-insensitive on PubKeyHex")
	}
}
