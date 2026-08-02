package tachi

import (
	"context"
	"net/http"
	"testing"
)

func TestAddressService_Get(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_address", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		testQueryParam(t, r, "address", "bcrt1pxyz")
		writeJSON(t, w, AddressResponse{Pubkey: "abcd", BalanceSat: 1000, Nonce: 5, VTXOCount: 2})
	})

	got, resp, err := ts.client.Address.Get(context.Background(), "bcrt1pxyz")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	want := &AddressResponse{Pubkey: "abcd", BalanceSat: 1000, Nonce: 5, VTXOCount: 2}
	if *got != *want {
		t.Errorf("Get() = %+v, want %+v", got, want)
	}
}

func TestAddressService_Get_Error(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_address", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "address query parameter is required", http.StatusBadRequest)
	})
	_, _, err := ts.client.Address.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ErrorResponse); !ok {
		t.Fatalf("error is not *ErrorResponse: %T", err)
	}
}

func TestAddressService_VTXOs(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_addressVtxos", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		testQueryParam(t, r, "address", "abcd")
		testQueryParam(t, r, "include_spent", "true")
		writeJSON(t, w, AddressVTXOsResponse{
			Pubkey: "abcd",
			VTXOs:  []VTXOItem{{ID: "id1", Owner: "abcd", Amount: 500}},
			Count:  1,
		})
	})

	got, _, err := ts.client.Address.VTXOs(context.Background(), "abcd", true)
	if err != nil {
		t.Fatalf("VTXOs: %v", err)
	}
	if got.Count != 1 || got.VTXOs[0].ID != "id1" {
		t.Errorf("VTXOs() = %+v, unexpected shape", got)
	}
}

func TestAddressService_VTXOs_ExcludeSpentOmitsParam(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_addressVtxos", func(w http.ResponseWriter, r *http.Request) {
		testQueryParamAbsent(t, r, "include_spent")
		writeJSON(t, w, AddressVTXOsResponse{})
	})
	if _, _, err := ts.client.Address.VTXOs(context.Background(), "abcd", false); err != nil {
		t.Fatalf("VTXOs: %v", err)
	}
}

func TestAddressService_Transactions(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_addressTransactions", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "abcd")
		testQueryParam(t, r, "before_height", "100")
		testQueryParam(t, r, "page_size", "10")
		next := int64(50)
		writeJSON(t, w, AddressTransactionsResponse{
			Pubkey:            "abcd",
			ScannedFromHeight: 100,
			ScannedToHeight:   50,
			NextBeforeHeight:  &next,
		})
	})

	got, _, err := ts.client.Address.Transactions(context.Background(), "abcd", &AddressTransactionsOptions{
		BeforeHeight: 100,
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("Transactions: %v", err)
	}
	if got.NextBeforeHeight == nil || *got.NextBeforeHeight != 50 {
		t.Errorf("NextBeforeHeight = %v, want 50", got.NextBeforeHeight)
	}
}

func TestAddressService_Transactions_NilOptions(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_addressTransactions", func(w http.ResponseWriter, r *http.Request) {
		testQueryParamAbsent(t, r, "before_height")
		testQueryParamAbsent(t, r, "page_size")
		writeJSON(t, w, AddressTransactionsResponse{})
	})
	if _, _, err := ts.client.Address.Transactions(context.Background(), "abcd", nil); err != nil {
		t.Fatalf("Transactions: %v", err)
	}
}

func TestAddressService_Balance(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_balance", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "abcd")
		writeJSON(t, w, BalanceResponse{Pubkey: "abcd", BalanceSat: 42})
	})
	got, _, err := ts.client.Address.Balance(context.Background(), "abcd")
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if got.BalanceSat != 42 {
		t.Errorf("BalanceSat = %d, want 42", got.BalanceSat)
	}
}

func TestAddressService_Nonce(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_nonce", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "abcd")
		writeJSON(t, w, NonceResponse{Address: "abcd", Nonce: 5, NextNonce: 6})
	})
	got, _, err := ts.client.Address.Nonce(context.Background(), "abcd")
	if err != nil {
		t.Fatalf("Nonce: %v", err)
	}
	if got.NextNonce != 6 {
		t.Errorf("NextNonce = %d, want 6", got.NextNonce)
	}
}

func TestAddressService_Mempool(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/tachi_mempoolByAddress", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "address", "abcd")
		writeJSON(t, w, MempoolByAddressResponse{Pubkey: "abcd", Count: 0})
	})
	got, _, err := ts.client.Address.Mempool(context.Background(), "abcd")
	if err != nil {
		t.Fatalf("Mempool: %v", err)
	}
	if got.Pubkey != "abcd" {
		t.Errorf("Pubkey = %q, want abcd", got.Pubkey)
	}
}
