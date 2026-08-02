package tachi

import (
	"context"
	"encoding/json"
	"net/url"
)

// DashboardService groups network-overview endpoints.
type DashboardService service

// StatsResponse is returned by DashboardService.Stats.
type StatsResponse struct {
	Height            int64  `json:"height"`
	TotalTransactions int    `json:"total_transactions"`
	TotalAccounts     int    `json:"total_accounts"`
	CurrentEpoch      uint32 `json:"current_epoch"`
	NodeCount         int    `json:"node_count"`
	ChainID           string `json:"chain_id"`
	LatestBlockTime   *int64 `json:"latest_block_time,omitempty"`
	TotalSupplySat    int64  `json:"total_supply_sat"`
	VTXOCount         int    `json:"vtxo_count"`
}

// SupplyResponse is returned by DashboardService.Supply.
type SupplyResponse struct {
	TotalSupplySat int64 `json:"total_supply_sat"`
	VTXOCount      int   `json:"vtxo_count"`
}

// SearchResponse is returned by DashboardService.Search. Type is one of
// "block", "epoch", "tx", "vtxo", or "address"; Result's shape depends on
// Type — the SDK leaves it as raw JSON since each type has a different
// schema. Use json.Unmarshal(resp.Result, &dst) with the appropriate
// target type (a map for "block"/"address", GetEpochResponse,
// GetTransactionResponse, etc).
type SearchResponse struct {
	Type   string          `json:"type"`
	Result json.RawMessage `json:"result"`
}

// Stats returns a single-call network dashboard overview: chain height,
// total transactions, accounts, current epoch, node count, and supply.
func (s *DashboardService) Stats(ctx context.Context) (*StatsResponse, *Response, error) {
	var out StatsResponse
	resp, err := s.client.get(ctx, "tachi_stats", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Supply returns the total circulating supply (sum of all unspent VTXO
// amounts) and unspent VTXO count.
func (s *DashboardService) Supply(ctx context.Context) (*SupplyResponse, *Response, error) {
	var out SupplyResponse
	resp, err := s.client.get(ctx, "tachi_supply", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Search auto-detects whether q is a tx hash, block height, epoch ID, VTXO
// ID, or pubkey and returns the matching result. Inspect resp.Type
// ("block", "epoch", "tx", "vtxo", or "address") and json.Unmarshal
// resp.Result into the matching type.
func (s *DashboardService) Search(ctx context.Context, q string) (*SearchResponse, *Response, error) {
	var out SearchResponse
	resp, err := s.client.get(ctx, "tachi_search", url.Values{"q": {q}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
