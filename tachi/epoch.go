package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// EpochService groups epoch-lookup endpoints.
type EpochService service

// GetEpochResponse is returned by EpochService.Get and EpochService.ByHash.
type GetEpochResponse struct {
	// Hash is the hex-encoded 32-byte Verkle root committed by this epoch.
	Hash string `json:"hash"`
	// Height is the sequential EpochID (uint32) of this epoch.
	Height uint32 `json:"height"`
	// BitcoinBlockHeight is nil until L1-settled or bitcoin client unavailable.
	BitcoinBlockHeight *int64 `json:"bitcoin_block_height"`
	// Status is "open" or "closed".
	Status string `json:"status"`
	// Timestamp is unix seconds at epoch close; nil for open epochs.
	Timestamp *int64 `json:"timestamp"`
	// TxCount is the number of transactions committed within this epoch.
	TxCount int `json:"tx_count"`
	// TxHashes is the list of 64-char hex transaction hashes included in this epoch.
	TxHashes []string `json:"tx_hashes"`
	// HATCount is the number of HAT (hash anchor tree) commitments recorded
	// in this epoch's Verkle tree.
	HATCount int `json:"hat_count"`
	// L1SettlementTxID is the hex-encoded Bitcoin L1 transaction that settled
	// this epoch's Verkle root; empty until the epoch is L1-settled.
	L1SettlementTxID string `json:"l1_settlement_txid,omitempty"`
}

// Get returns a Tachi-decoded view of the epoch with the given sequential
// EpochID.
func (s *EpochService) Get(ctx context.Context, id uint32) (*GetEpochResponse, *Response, error) {
	var out GetEpochResponse
	q := url.Values{"id": {strconv.FormatUint(uint64(id), 10)}}
	resp, err := s.client.get(ctx, "tachi_epoch", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ByHash returns a Tachi-decoded view of the epoch whose Verkle root
// matches the given 64-char hex hash.
func (s *EpochService) ByHash(ctx context.Context, hash string) (*GetEpochResponse, *Response, error) {
	var out GetEpochResponse
	q := url.Values{"hash": {hash}}
	resp, err := s.client.get(ctx, "tachi_epoch", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ListEpochsResponse is returned by EpochService.List.
type ListEpochsResponse struct {
	// Epochs is the page of epochs returned, ordered latest-first by EpochID.
	Epochs []GetEpochResponse `json:"epochs"`
	// Total is the total number of epochs known to the chain (current_epoch + 1).
	Total int `json:"total"`
	// Page is the 1-based page number the caller requested.
	Page int `json:"page"`
	// PageSize is the maximum number of epochs returned per page (capped at 100).
	PageSize int `json:"page_size"`
	// TotalPages is the total number of pages available for the current PageSize.
	TotalPages int `json:"total_pages"`
}

// List returns a paginated list of all epochs from latest to oldest with
// full metadata. page defaults to 1; pageSize defaults to 50 (max 100).
// Pass 0 for either to use the default.
func (s *EpochService) List(ctx context.Context, page, pageSize int) (*ListEpochsResponse, *Response, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var out ListEpochsResponse
	resp, err := s.client.get(ctx, "tachi_listEpochs", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
