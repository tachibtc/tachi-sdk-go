package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// VaultService groups vault-lookup endpoints.
type VaultService service

// VaultListItem is a compact per-vault view returned by VaultService.List.
type VaultListItem struct {
	VaultID        string `json:"vault_id"` // hex VaultID = H(funding_txid || vout)
	State          string `json:"state"`
	LatestStateNum uint64 `json:"latest_state_num"`
	FundingTxid    string `json:"funding_txid"`
	FundingVout    uint32 `json:"funding_vout"`
	Address        string `json:"address"` // bech32m P2TR vault address
	// CSVDelay, Threshold, QuorumKeyset, and UserKey are the vault's
	// reconstruction parameters. Only populated when the Client was
	// created with WithAPIKey using a valid master key or a client key
	// registered with the vault scope; omitted otherwise.
	CSVDelay     uint32   `json:"csv_delay,omitempty"`
	Threshold    int      `json:"threshold,omitempty"`
	QuorumKeyset []string `json:"quorum_keyset,omitempty"`
	UserKey      string   `json:"user_key,omitempty"`
}

// ListVaultsResponse is returned by VaultService.List.
type ListVaultsResponse struct {
	User       string          `json:"user"`
	Vaults     []VaultListItem `json:"vaults"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// List returns a paginated list of vaults owned by user (a public key or
// taproot address). csv_delay/threshold/quorum_keyset/user_key are only
// populated when the Client was created with WithAPIKey using a valid
// master key or a vault-scoped client key. page defaults to 1; pageSize
// defaults to 50 (max 100). Pass 0 for either to use the default.
func (s *VaultService) List(ctx context.Context, user string, page, pageSize int) (*ListVaultsResponse, *Response, error) {
	q := url.Values{"user": {user}}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var out ListVaultsResponse
	resp, err := s.client.get(ctx, "tachi_listVaults", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
