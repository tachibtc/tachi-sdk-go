package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// AddressService groups account-scoped endpoints: balance, nonce, VTXOs,
// transaction history, and pending mempool activity for a public key or
// taproot address.
type AddressService service

// AddressResponse is returned by AddressService.Get.
type AddressResponse struct {
	Pubkey     string `json:"pubkey"`
	BalanceSat int64  `json:"balance_sat"`
	Nonce      uint64 `json:"nonce"`
	VTXOCount  int    `json:"vtxo_count"`
}

// Get returns account detail (balance, nonce, VTXO count) for a public key
// or taproot address (bc1p/tb1p/bcrt1p, or 32/33-byte hex).
func (s *AddressService) Get(ctx context.Context, address string) (*AddressResponse, *Response, error) {
	var out AddressResponse
	resp, err := s.client.get(ctx, "tachi_address", url.Values{"address": {address}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// VTXOItem is a single VTXO in an AddressService.VTXOs response.
type VTXOItem struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	Amount       int64  `json:"amount"`
	Spent        bool   `json:"spent"`
	Height       int64  `json:"height"`
	Script       string `json:"script"`
	Locked       bool   `json:"locked"`
	VaultAddress string `json:"vault_address,omitempty"`
}

// AddressVTXOsResponse is returned by AddressService.VTXOs.
type AddressVTXOsResponse struct {
	Pubkey string     `json:"pubkey"`
	VTXOs  []VTXOItem `json:"vtxos"`
	Count  int        `json:"count"`
}

// VTXOs returns VTXOs owned by address. By default only unspent VTXOs are
// returned; pass includeSpent=true for full history.
func (s *AddressService) VTXOs(ctx context.Context, address string, includeSpent bool) (*AddressVTXOsResponse, *Response, error) {
	q := url.Values{"address": {address}}
	if includeSpent {
		q.Set("include_spent", "true")
	}
	var out AddressVTXOsResponse
	resp, err := s.client.get(ctx, "tachi_addressVtxos", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// AddressTransactionsResponse is returned by AddressService.Transactions.
// Pagination is height-cursor based, same semantics as
// TxService.List's ListTransactionsResponse.
type AddressTransactionsResponse struct {
	Pubkey            string                `json:"pubkey"`
	Transactions      []ListTransactionItem `json:"transactions"`
	PageSize          int                   `json:"page_size"`
	ScannedFromHeight int64                 `json:"scanned_from_height"`
	ScannedToHeight   int64                 `json:"scanned_to_height"`
	NextBeforeHeight  *int64                `json:"next_before_height,omitempty"`
}

// AddressTransactionsOptions configures AddressService.Transactions
// pagination.
type AddressTransactionsOptions struct {
	// BeforeHeight is the exclusive upper-bound block height; zero means
	// "start from the chain tip".
	BeforeHeight int64
	// PageSize is the max transactions to return (default 50, max 100).
	PageSize int
}

// Transactions returns transactions involving address in descending height
// order, height-cursor paginated (chain opts.BeforeHeight on the previous
// response's NextBeforeHeight). The scan is bounded by a server-side time
// budget rather than a block cap, so a sparse address can return zero
// transactions along with a cursor to keep walking older blocks.
func (s *AddressService) Transactions(ctx context.Context, address string, opts *AddressTransactionsOptions) (*AddressTransactionsResponse, *Response, error) {
	q := url.Values{"address": {address}}
	if opts != nil {
		if opts.BeforeHeight > 0 {
			q.Set("before_height", strconv.FormatInt(opts.BeforeHeight, 10))
		}
		if opts.PageSize > 0 {
			q.Set("page_size", strconv.Itoa(opts.PageSize))
		}
	}
	var out AddressTransactionsResponse
	resp, err := s.client.get(ctx, "tachi_addressTransactions", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// BalanceResponse is returned by AddressService.Balance.
type BalanceResponse struct {
	Pubkey     string `json:"pubkey"`
	BalanceSat int64  `json:"balance_sat"`
}

// Balance returns the total unspent balance for address in satoshis.
func (s *AddressService) Balance(ctx context.Context, address string) (*BalanceResponse, *Response, error) {
	var out BalanceResponse
	resp, err := s.client.get(ctx, "tachi_balance", url.Values{"address": {address}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// NonceResponse is returned by AddressService.Nonce.
type NonceResponse struct {
	Address   string `json:"address"`
	Nonce     uint64 `json:"nonce"`
	NextNonce uint64 `json:"next_nonce"`
}

// Nonce returns the last used nonce for address; the next transaction
// should use NextNonce.
func (s *AddressService) Nonce(ctx context.Context, address string) (*NonceResponse, *Response, error) {
	var out NonceResponse
	resp, err := s.client.get(ctx, "tachi_nonce", url.Values{"address": {address}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// MempoolByAddressResponse is returned by AddressService.Mempool.
type MempoolByAddressResponse struct {
	Pubkey       string                `json:"pubkey"`
	Transactions []ListTransactionItem `json:"transactions"`
	Count        int                   `json:"count"`
}

// Mempool returns pending (unconfirmed) transactions crediting address or
// spending one of its currently-unspent VTXOs. Poll this once for current
// pending state, then use WSService.Subscribe for new arrivals instead of
// re-polling.
func (s *AddressService) Mempool(ctx context.Context, address string) (*MempoolByAddressResponse, *Response, error) {
	var out MempoolByAddressResponse
	resp, err := s.client.get(ctx, "tachi_mempoolByAddress", url.Values{"address": {address}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
