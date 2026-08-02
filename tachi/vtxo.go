package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// VTXOService groups VTXO-lookup endpoints.
type VTXOService service

// VTXOResponse is returned by VTXOService.Get and appears in
// LockedVTXOsResponse and ListVTXOsResponse.
type VTXOResponse struct {
	// ID is the hex-encoded 32-byte VTXO identifier.
	ID string `json:"id"`
	// Owner is the hex-encoded owner public key authorized to spend this VTXO.
	Owner string `json:"owner"`
	// Amount is the VTXO value in satoshis.
	Amount int64 `json:"amount"`
	// Spent is true once this VTXO has been consumed by a confirmed transaction.
	Spent bool `json:"spent"`
	// Height is the block height at which this VTXO was created.
	Height int64 `json:"height"`
	// Script is the hex-encoded locking script associated with this VTXO.
	Script string `json:"script"`
}

// Get returns a single VTXO by its 64-char hex ID.
func (s *VTXOService) Get(ctx context.Context, id string) (*VTXOResponse, *Response, error) {
	var out VTXOResponse
	resp, err := s.client.get(ctx, "tachi_vtxo", url.Values{"id": {id}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// LockedVTXOsResponse is returned by VTXOService.Locked.
type LockedVTXOsResponse struct {
	// Vault is the vault address (bech32m or hex) the locked VTXOs are bound to.
	Vault string `json:"vault"`
	// VTXOs is the list of VTXOs locked to Vault.
	VTXOs []VTXOResponse `json:"vtxos"`
	// Count is the number of entries returned in VTXOs.
	Count int `json:"count"`
}

// Locked returns all VTXOs locked for the given vault address.
func (s *VTXOService) Locked(ctx context.Context, vault string) (*LockedVTXOsResponse, *Response, error) {
	var out LockedVTXOsResponse
	resp, err := s.client.get(ctx, "tachi_vtxoLocked", url.Values{"vault": {vault}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ListVTXOsResponse is returned by VTXOService.List.
type ListVTXOsResponse struct {
	// VTXOs is the page of VTXOs returned, sorted by height descending.
	VTXOs []VTXOResponse `json:"vtxos"`
	// Total is the total number of VTXOs across all pages, as reported by the ABCI app.
	Total int `json:"total"`
	// Page is the 1-based page number the caller requested.
	Page int `json:"page"`
	// PageSize is the maximum number of VTXOs returned per page.
	PageSize int `json:"page_size"`
	// TotalPages is the total number of pages available for the current PageSize.
	TotalPages int `json:"total_pages"`
}

// List returns a paginated list of all VTXOs (unspent and spent), sorted
// by height descending. page defaults to 1; pageSize defaults to 50 (max
// 100). Pass 0 for either to use the default.
func (s *VTXOService) List(ctx context.Context, page, pageSize int) (*ListVTXOsResponse, *Response, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var out ListVTXOsResponse
	resp, err := s.client.get(ctx, "tachi_listVtxos", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
