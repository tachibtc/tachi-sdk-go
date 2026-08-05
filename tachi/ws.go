package tachi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// pongWait is how long the read loop waits for a server pong (or any
// message) before deciding the connection is dead. pingPeriod must stay
// well under pongWait so a ping always lands before the deadline expires.
const (
	pongWait   = 60 * time.Second
	pingPeriod = pongWait * 9 / 10
)

// WSService opens live websocket subscriptions for chain alerts.
type WSService service

// WSEvent is the envelope delivered on WSConn.Events(). Event discriminates
// which of Tx/Block/Validator/Breach is populated, since one subscription
// can watch multiple filters at once.
type WSEvent struct {
	// Event is "tx", "block", "validator", or "breach".
	Event string `json:"event"`
	// Tx is set when Event == "tx".
	Tx *WSTxAlert `json:"tx,omitempty"`
	// Block is set when Event == "block".
	Block *WSBlockAlert `json:"block,omitempty"`
	// Validator is set when Event == "validator".
	Validator *WSValidatorAlert `json:"validator,omitempty"`
	// Breach is set when Event == "breach".
	Breach *WSBreachAlert `json:"breach,omitempty"`
}

// WSTxAlert is pushed when a tx (pending or committed) matches an
// address/vault subscription.
type WSTxAlert struct {
	// TxHash is the hex-encoded hash of the matching transaction.
	TxHash string `json:"tx_hash"`
	// Type is transfer/deposit/withdraw/lock/unlock/vault_open/
	// vault_state_advance/vault_close/vault_breach/unknown.
	Type string `json:"type"`
	// State is "pending" (seen via CheckTx) or "committed".
	State string `json:"state"`
	// Height is the block height that committed the transaction; 0/omitted while State is "pending".
	Height int64 `json:"height,omitempty"`
	// VaultAddress is set only when the alert matched on a vault subscription.
	VaultAddress string `json:"vault_address,omitempty"`
	// Vout lists the transaction's outputs.
	Vout []TxVout `json:"vout"`
}

// WSValidatorAlert is pushed to "validators" subscribers when a new
// validator registers.
type WSValidatorAlert struct {
	// PubKeyHex is the compressed secp256k1 public key of the joining validator.
	PubKeyHex string `json:"pub_key_hex"`
	// PeerID is the joining validator's libp2p peer identifier.
	PeerID string `json:"peer_id,omitempty"`
	// Host is the joining validator's CometBFT P2P address "host:port".
	Host string `json:"host,omitempty"`
	// RPCAddr is the joining validator's daemon RPC listen address.
	RPCAddr string `json:"rpc_addr,omitempty"`
	// Total is the registry size immediately after this join.
	Total int `json:"total"`
}

// WSBreachAlert is pushed to "vaultId" subscribers when the watchtower
// observes an L1 spend of a vault's funding outpoint — every
// classification (legitimate/stale/anomalous) is delivered, same
// population as WatchtowerService.Receipts.
type WSBreachAlert struct {
	// VaultID is the hex VaultID whose funding outpoint was spent.
	VaultID string `json:"vault_id"`
	// BroadcastState is the state decoded from the spending tx's hint.
	BroadcastState uint64 `json:"broadcast_state"`
	// LatestState is the BFT-replicated latest state at detection.
	LatestState uint64 `json:"latest_state"`
	// Classification is "legitimate", "stale", or "anomalous".
	Classification string `json:"classification"`
	// SpendTxID is the L1 txid that spent the funding outpoint.
	SpendTxID string `json:"spend_txid"`
	// SpendVout is the funding output index that was consumed.
	SpendVout uint32 `json:"spend_vout"`
	// DetectedHeight is the L1 block height the spend was observed at.
	DetectedHeight int64 `json:"detected_height"`
	// DetectedAt is the node's wall-clock unix timestamp at detection.
	DetectedAt int64 `json:"detected_at"`
}

// WSBlockAlert is pushed to "blocks" subscribers for every
// durably-committed block.
type WSBlockAlert struct {
	// Height is the committed block height.
	Height int64 `json:"height"`
	// BlockHash is the hex-encoded committing block hash.
	BlockHash string `json:"block_hash"`
	// AppHash is the hex-encoded ABCI app hash for this block.
	AppHash string `json:"app_hash"`
	// TxCount is the number of transactions included in this block.
	TxCount int `json:"tx_count"`
	// EpochClosed is the epoch ID this block closed; omitted for blocks that didn't close an epoch.
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
	// Txs subscribes to every transaction (pending, then committed),
	// unfiltered by Address/Vault.
	Txs bool
}

func (o SubscribeOptions) empty() bool {
	return o.Address == "" && o.Vault == "" && o.VaultID == "" && !o.Blocks && !o.Validators && !o.Txs
}

// WSConn is a live subscription opened by WSService.Subscribe. Events
// arrive on the channel returned by Events(); a transport error or server
// close terminates the read loop, closes the Events channel, and is
// reported once on Err(). Callers must call Close when done.
type WSConn struct {
	// conn is the underlying websocket connection.
	conn *websocket.Conn
	// events delivers incoming alerts to the caller; closed when the read loop exits.
	events chan WSEvent
	// errc carries at most one error from an abnormal read-loop exit.
	errc chan error
	// done signals the ping keepalive goroutine to stop when Close is called.
	done chan struct{}
	// closeOnce ensures Close only closes conn once, even if called concurrently.
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
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.conn.Close()
	})
	return err
}

func (w *WSConn) readLoop() {
	defer close(w.events)
	_ = w.conn.SetReadDeadline(time.Now().Add(pongWait))
	w.conn.SetPongHandler(func(string) error {
		return w.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
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

// pingLoop periodically pings the server so a dead connection (e.g. a
// half-open TCP session) surfaces as a read error instead of hanging
// forever. Stops when Close is called.
func (w *WSConn) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := w.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		case <-w.done:
			return
		}
	}
}

// Subscribe upgrades to a websocket and streams tx, block, validator,
// and/or watchtower-breach alerts per opts. The connection is push-only —
// the server never expects client messages, and the SDK's read loop
// answers server pings transparently (handled by the underlying
// websocket.Conn).
func (s *WSService) Subscribe(ctx context.Context, opts SubscribeOptions) (*WSConn, error) {
	if opts.empty() {
		return nil, fmt.Errorf("tachi: at least one of Address, Vault, VaultID, Blocks, Validators, or Txs is required")
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
	if opts.Txs {
		q.Set("txs", "true")
	}
	u.RawQuery = q.Encode()

	header := http.Header{}
	if s.client.UserAgent != "" {
		header.Set("User-Agent", s.client.UserAgent)
	}
	if s.client.apiKey != "" {
		if u.Scheme != "wss" && !isLoopbackHost(u.Hostname()) {
			return nil, fmt.Errorf("tachi: refusing to use an API key over %s to non-loopback host %q; use wss", u.Scheme, u.Hostname())
		}
		header.Set("X-Api-Key", s.client.apiKey)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("tachi: dial websocket: %w", err)
	}
	conn.SetReadLimit(maxResponseBytes)

	wc := &WSConn{
		conn:   conn,
		events: make(chan WSEvent, 32),
		errc:   make(chan error, 1),
		done:   make(chan struct{}),
	}
	go wc.readLoop()
	go wc.pingLoop()
	return wc, nil
}
