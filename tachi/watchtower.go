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
	Mode              string `json:"mode"`
	LastScannedHeight int64  `json:"last_scanned_height"`
	ReceiptCount      int    `json:"receipt_count"`
	SweepThreshold    int    `json:"sweep_threshold"`
	BountyConfigured  bool   `json:"bounty_configured"`
}

// BreachReceipt is an entry (or single lookup result) from
// WatchtowerService.Receipts / Receipt.
type BreachReceipt struct {
	VaultID        string `json:"vault_id"`
	BroadcastState uint64 `json:"broadcast_state"`
	LatestState    uint64 `json:"latest_state"`
	// Classification is "legitimate", "stale", or "anomalous".
	Classification string `json:"classification"`
	SpendTxID      string `json:"spend_txid"`
	SpendVout      uint32 `json:"spend_vout"`
	DetectedHeight int64  `json:"detected_height"`
	DetectedAt     int64  `json:"detected_at"`
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
		Count    int             `json:"count"`
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
