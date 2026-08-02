package tachi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

// WSService opens live websocket subscriptions for chain alerts.
type WSService service

// WSEvent is the envelope delivered on WSConn.Events(). Event discriminates
// which of Tx/Block/Validator/Breach is populated, since one subscription
// can watch multiple filters at once.
type WSEvent struct {
	// Event is "tx", "block", "validator", or "breach".
	Event     string            `json:"event"`
	Tx        *WSTxAlert        `json:"tx,omitempty"`
	Block     *WSBlockAlert     `json:"block,omitempty"`
	Validator *WSValidatorAlert `json:"validator,omitempty"`
	Breach    *WSBreachAlert    `json:"breach,omitempty"`
}

// WSTxAlert is pushed when a tx (pending or committed) matches an
// address/vault subscription.
type WSTxAlert struct {
	TxHash string `json:"tx_hash"`
	Type   string `json:"type"`
	// State is "pending" (seen via CheckTx) or "committed".
	State  string `json:"state"`
	Height int64  `json:"height,omitempty"`
	// VaultAddress is set only when the alert matched on a vault subscription.
	VaultAddress string   `json:"vault_address,omitempty"`
	Vout         []TxVout `json:"vout"`
}

// WSValidatorAlert is pushed to "validators" subscribers when a new
// validator registers.
type WSValidatorAlert struct {
	PubKeyHex string `json:"pub_key_hex"`
	PeerID    string `json:"peer_id,omitempty"`
	Host      string `json:"host,omitempty"`
	RPCAddr   string `json:"rpc_addr,omitempty"`
	Total     int    `json:"total"`
}

// WSBreachAlert is pushed to "vaultId" subscribers when the watchtower
// observes an L1 spend of a vault's funding outpoint — every
// classification (legitimate/stale/anomalous) is delivered, same
// population as WatchtowerService.Receipts.
type WSBreachAlert struct {
	VaultID        string `json:"vault_id"`
	BroadcastState uint64 `json:"broadcast_state"`
	LatestState    uint64 `json:"latest_state"`
	Classification string `json:"classification"`
	SpendTxID      string `json:"spend_txid"`
	SpendVout      uint32 `json:"spend_vout"`
	DetectedHeight int64  `json:"detected_height"`
	DetectedAt     int64  `json:"detected_at"`
}

// WSBlockAlert is pushed to "blocks" subscribers for every
// durably-committed block.
type WSBlockAlert struct {
	Height      int64   `json:"height"`
	BlockHash   string  `json:"block_hash"`
	AppHash     string  `json:"app_hash"`
	TxCount     int     `json:"tx_count"`
	EpochClosed *uint32 `json:"epoch_closed,omitempty"`
}

// SubscribeOptions selects which alerts a websocket connection receives. At
// least one field must be set.
type SubscribeOptions struct {
	// Address watches for incoming vouts to this taproot address or pubkey hex.
	Address string
	// Vault watches for incoming locks into this vault address.
	Vault string
	// VaultID watches for watchtower breach receipts on this vault (64-hex).
	VaultID string
	// Blocks subscribes to an alert for every durably-committed block.
	Blocks bool
	// Validators subscribes to an alert for every new validator registration.
	Validators bool
}

func (o SubscribeOptions) empty() bool {
	return o.Address == "" && o.Vault == "" && o.VaultID == "" && !o.Blocks && !o.Validators
}

// WSConn is a live subscription opened by WSService.Subscribe. Events
// arrive on the channel returned by Events(); a transport error or server
// close terminates the read loop, closes the Events channel, and is
// reported once on Err(). Callers must call Close when done.
type WSConn struct {
	conn      *websocket.Conn
	events    chan WSEvent
	errc      chan error
	closeOnce sync.Once
}

// Events returns the channel of incoming alerts. It is closed when the
// connection ends (see Err for the reason, if any).
func (w *WSConn) Events() <-chan WSEvent { return w.events }

// Err returns a channel that receives at most one error when the read loop
// terminates abnormally (transport error, unexpected close). A clean close
// (via Close) delivers nothing.
func (w *WSConn) Err() <-chan error { return w.errc }

// Close closes the underlying websocket connection. Safe to call multiple
// times.
func (w *WSConn) Close() error {
	var err error
	w.closeOnce.Do(func() { err = w.conn.Close() })
	return err
}

func (w *WSConn) readLoop() {
	defer close(w.events)
	for {
		var evt WSEvent
		if err := w.conn.ReadJSON(&evt); err != nil {
			select {
			case w.errc <- err:
			default:
			}
			return
		}
		w.events <- evt
	}
}

// Subscribe upgrades to a websocket and streams tx, block, validator,
// and/or watchtower-breach alerts per opts. The connection is push-only —
// the server never expects client messages, and the SDK's read loop
// answers server pings transparently (handled by the underlying
// websocket.Conn).
func (s *WSService) Subscribe(ctx context.Context, opts SubscribeOptions) (*WSConn, error) {
	if opts.empty() {
		return nil, fmt.Errorf("tachi: at least one of Address, Vault, VaultID, Blocks, or Validators is required")
	}

	u := *s.client.BaseURL
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path += "tachi_ws"

	q := url.Values{}
	setIf(q, "address", opts.Address)
	setIf(q, "vault", opts.Vault)
	setIf(q, "vaultId", opts.VaultID)
	if opts.Blocks {
		q.Set("blocks", "true")
	}
	if opts.Validators {
		q.Set("validators", "true")
	}
	u.RawQuery = q.Encode()

	header := http.Header{}
	if s.client.UserAgent != "" {
		header.Set("User-Agent", s.client.UserAgent)
	}
	if s.client.apiKey != "" {
		header.Set("X-Api-Key", s.client.apiKey)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("tachi: dial websocket: %w", err)
	}

	wc := &WSConn{
		conn:   conn,
		events: make(chan WSEvent, 32),
		errc:   make(chan error, 1),
	}
	go wc.readLoop()
	return wc, nil
}
