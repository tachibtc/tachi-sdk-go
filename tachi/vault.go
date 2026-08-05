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
	// VaultID is the hex VaultID = H(funding_txid || vout).
	VaultID string `json:"vault_id"`
	// Name is the display label set at open. Not unique — key on VaultID.
	// Untrusted: chosen by the vault opener and may contain HTML
	// metacharacters. Escape it for the render context before displaying.
	Name string `json:"name,omitempty"`
	// State is the vault lifecycle label (always "open" today).
	State string `json:"state"`
	// LatestStateNum is the vault's latest state number (always 0 today).
	LatestStateNum uint64 `json:"latest_state_num"`
	// FundingTxid is the hex L1 funding transaction id.
	FundingTxid string `json:"funding_txid"`
	// FundingVout is the L1 funding output index.
	FundingVout uint32 `json:"funding_vout"`
	// Address is the bech32m P2TR vault address; the ?vault= websocket filter key.
	Address string `json:"address"`
	// CSVDelay, Threshold, QuorumKeyset, and UserKey are the vault's
	// reconstruction parameters. Only populated when the Client was
	// created with WithAPIKey using a valid master key or a client key
	// registered with the vault scope; omitted otherwise.
	CSVDelay uint32 `json:"csv_delay,omitempty"`
	// Threshold is the number of quorum signatures required to authorize a spend.
	Threshold int `json:"threshold,omitempty"`
	// QuorumKeyset is the list of quorum member public keys.
	QuorumKeyset []string `json:"quorum_keyset,omitempty"`
	// UserKey is the vault owner's public key.
	UserKey string `json:"user_key,omitempty"`
}

// ListVaultsResponse is returned by VaultService.List.
type ListVaultsResponse struct {
	// User is the normalized 32-byte x-only pubkey (hex) the vaults are owned by.
	User string `json:"user"`
	// Vaults is the requested page of vaults owned by User.
	Vaults []VaultListItem `json:"vaults"`
	// Total is the total number of vaults User owns across all pages.
	Total int `json:"total"`
	// Page is the 1-based page number returned.
	Page int `json:"page"`
	// PageSize is the maximum entries per page.
	PageSize int `json:"page_size"`
	// TotalPages is the total number of pages available at the current PageSize.
	TotalPages int `json:"total_pages"`
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
