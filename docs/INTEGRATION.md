# Integration guide

This is a fuller reference for external consumers of `daemon-go-sdk`. The
[README](../README.md) covers install/quickstart; this doc covers the
patterns you need to build a real integration: auth, errors, pagination,
and every service's request/response shapes.

## Client construction

```go
c, err := tachi.NewClient(
	tachi.WithBaseURL("http://127.0.0.1:26670/"),
	tachi.WithAPIKey(os.Getenv("TACHI_API_KEY")),
)
```

`NewClient` validates the base URL eagerly:
- scheme must be `http` or `https`
- if an API key is set, the URL must be `https` unless the host is
  loopback (`localhost`, `127.0.0.1`, `::1`) — this prevents accidentally
  sending a privileged key in cleartext over a network link

## Auth model

- No key: every read-only endpoint works (node status, blocks, epochs, txs,
  VTXOs, vaults, address lookups, dashboard stats, most `Bitcoin.RPC` methods).
- `WithAPIKey(key)`: sent as `X-Api-Key` on every request. Required for:
  - privileged `Bitcoin.RPC` methods (wallet, admin, network-mutation)
  - `Vault.List` reconstruction parameters
  - the `vaults_by_user` ABCI passthrough
  - it also raises the daemon's per-caller rate limit

There is no per-request auth override — the key is fixed at client
construction. Build a second `*Client` if you need both an authenticated and
an anonymous view.

## Error handling

Every service method returns `(result, *tachi.Response, error)`.

- Network/transport errors (DNS, TLS, timeout, context cancellation) come
  back as a plain `error` (or `ctx.Err()` if the context was already done).
- A non-2xx HTTP response comes back as `*tachi.ErrorResponse`, which wraps
  the raw `*http.Response` and the daemon's plain-text error body:

```go
epoch, resp, err := c.Epoch.Get(ctx, uint32(42))
if err != nil {
	var apiErr *tachi.ErrorResponse
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.Response.StatusCode, apiErr.Message)
	}
	return err
}
_ = resp // *tachi.Response wraps *http.Response for headers/status on success too
```

`*tachi.Response` is non-nil on any response the daemon actually sent (even
error responses) — nil only on request-construction or transport failure.

## Pagination

`Tx.List`, `Address.Transactions`, and similar history endpoints use
height-cursor pagination, newest-first:

```go
var before *int64
for {
	page, _, err := c.Tx.List(ctx, &tachi.ListTransactionsOptions{BeforeHeight: before})
	if err != nil {
		log.Fatal(err)
	}
	handle(page.Transactions)
	if page.NextBeforeHeight == nil {
		break // reached genesis
	}
	before = page.NextBeforeHeight
}
```

Pass `nil`/zero-value on the first call to start from the chain tip. Each
response reports `ScannedFromHeight`/`ScannedToHeight` for the range actually
covered by that page.

## Services

| Field | Purpose |
|---|---|
| `c.Node` | Health, node identity, consensus/network status. |
| `c.Validators` | Validator bootstrap registry: list, register, wait-for-ready. |
| `c.Block` | Tachi-chain block lookups by height, hash, or range. |
| `c.Epoch` | Verkle-root checkpoint lookups. |
| `c.Tx` | Transaction lookup, broadcast (sync/async), decode/validate, mempool, fee estimation. |
| `c.VTXO` | VTXO lookup by ID, by vault lock, paginated listing. |
| `c.Vault` | Vault listing (reconstruction params gated behind `WithAPIKey`). |
| `c.Address` | Account-scoped balance, nonce, VTXOs, transaction history, pending mempool activity. |
| `c.Dashboard` | Explorer aggregate stats, circulating supply, free-text search. |
| `c.Bitcoin` | Pass-through Bitcoin Core JSON-RPC 1.0 proxy — see below. |
| `c.Sign` | Cooperative-refund threshold-signing ceremony. |
| `c.Watchtower` | Breach-detection status and receipts. |
| `c.WS` | Live event subscriptions over WebSocket — see below. |

Every method takes a `context.Context` first and accepts cancellation
mid-request.

### Escaping untrusted fields

A handful of response fields are set by other chain participants and are
not sanitized by the daemon — most notably `VaultListItem.Name` (the display
label a vault opener chooses at open time). Treat these as untrusted input:
HTML-escape before rendering in a browser, and don't interpolate them into
shell commands or SQL.

## Bitcoin JSON-RPC proxy

```go
raw, _, err := c.Bitcoin.RPC(ctx, "getblockchaininfo", nil)
```

`params` is marshalled as-is into the JSON-RPC 1.0 request; pass `nil` for
none. `raw` is the raw `json.RawMessage` result — decode it into whatever
shape the specific bitcoind method returns. Read-only chain/mempool/decode
methods need no auth; wallet, admin, and network-mutation methods require
`WithAPIKey` set to the daemon's master `BTC_RPC_API_KEY`.

## Live subscriptions (WebSocket)

```go
conn, err := c.WS.Subscribe(ctx, tachi.SubscribeOptions{
	Address: ownerPubkeyHex,
	Blocks:  true,
})
if err != nil {
	log.Fatal(err)
}
defer conn.Close()

for {
	select {
	case evt, ok := <-conn.Events():
		if !ok {
			// channel closed: check conn.Err() for why
			return
		}
		handle(evt)
	case err := <-conn.Err():
		log.Println("ws error:", err)
		return
	case <-ctx.Done():
		return
	}
}
```

`SubscribeOptions` — at least one of `Address`, `Vault`, `VaultID`,
`Blocks`, `Validators`, `Txs` is required, or `Subscribe` returns an error
before opening a connection:

- `Address` — incoming vouts to this address
- `Vault` / `VaultID` — watchtower breach alerts for a specific vault
- `Blocks` — every durably-committed block
- `Validators` — every new validator registration
- `Txs` — every transaction, pending then committed, unfiltered

Each delivered `WSEvent` sets `Event` to `"tx"`, `"block"`, `"validator"`,
or `"breach"`, with the corresponding field populated. The connection sends
periodic pings internally; `conn.Close()` is safe to call more than once.

## Examples

Runnable end-to-end examples live in [`examples/`](../examples):
- `examples/basic` exercises read-only calls across most services.
- `examples/websocket` opens a block-alert subscription and prints one event.
