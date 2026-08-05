package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// WatchtowerService groups the vault watchtower's status and breach-receipt
// endpoints.
type WatchtowerService service

// WatchtowerStatus is returned by WatchtowerService.Status.
type WatchtowerStatus struct {
	// Mode is "detection" (watch + receipt only), "responder" (also co-signs
	// peers' sweeps), or "initiator" (also originates and broadcasts sweeps).
	Mode string `json:"mode"`
	// LastScannedHeight is the L1 height the subscriber has scanned through.
	LastScannedHeight int64 `json:"last_scanned_height"`
	// ReceiptCount is the number of breach receipts on record.
	ReceiptCount int `json:"receipt_count"`
	// SweepThreshold is the partial-signature threshold the collector requires.
	SweepThreshold int `json:"sweep_threshold"`
	// BountyConfigured reports whether a sweep-bounty payout script is set
	// (required to initiate sweeps).
	BountyConfigured bool `json:"bounty_configured"`
}

// BreachReceipt is an entry (or single lookup result) from
// WatchtowerService.Receipts / Receipt.
type BreachReceipt struct {
	// VaultID is the vault whose funding outpoint was spent.
	VaultID string `json:"vault_id"`
	// BroadcastState is the state decoded from the spending tx's hint.
	BroadcastState uint64 `json:"broadcast_state"`
	// LatestState is the BFT-replicated latest state at detection.
	LatestState uint64 `json:"latest_state"`
	// Classification is "legitimate", "stale", or "anomalous".
	Classification string `json:"classification"`
	// SpendTxID is the displayed txid of the transaction that spent the funding outpoint.
	SpendTxID string `json:"spend_txid"`
	// SpendVout is the funding output index the spend consumed.
	SpendVout uint32 `json:"spend_vout"`
	// DetectedHeight is the L1 block height the spend was observed at.
	DetectedHeight int64 `json:"detected_height"`
	// DetectedAt is the node's wall-clock unix timestamp at detection.
	DetectedAt int64 `json:"detected_at"`
}

// Status returns the watchtower mode, last-scanned L1 height, and receipt
// count. Returns a 503 *ErrorResponse if the watchtower is disabled.
func (s *WatchtowerService) Status(ctx context.Context) (*WatchtowerStatus, *Response, error) {
	var out WatchtowerStatus
	resp, err := s.client.get(ctx, "tachi_watchtower/status", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Receipts lists breach receipts, optionally filtered to one vault (pass
// "" for vaultID to list every receipt). Returns a 503 *ErrorResponse if
// the watchtower is disabled.
func (s *WatchtowerService) Receipts(ctx context.Context, vaultID string) ([]BreachReceipt, *Response, error) {
	q := url.Values{}
	setIf(q, "vault", vaultID)
	var out struct {
		// Count is the number of entries in Receipts.
		Count int `json:"count"`
		// Receipts is the (optionally vault-filtered) list of breach receipts.
		Receipts []BreachReceipt `json:"receipts"`
	}
	resp, err := s.client.get(ctx, "tachi_watchtower/receipts", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return out.Receipts, resp, nil
}

// Receipt returns a single breach receipt for vaultID at the given
// broadcast state number. Returns a 404 *ErrorResponse if no receipt
// exists at that state.
func (s *WatchtowerService) Receipt(ctx context.Context, vaultID string, state uint64) (*BreachReceipt, *Response, error) {
	q := url.Values{
		"vault": {vaultID},
		"state": {strconv.FormatUint(state, 10)},
	}
	var out BreachReceipt
	resp, err := s.client.get(ctx, "tachi_watchtower/receipts", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
