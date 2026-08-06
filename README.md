# tachi-sdk-go

A Go client for the [Tachi daemon](../daemon) RPC API — validators, chain
data (blocks, epochs, transactions, VTXOs, vaults), account lookups,
dashboard/explorer endpoints, the bitcoind JSON-RPC proxy, cooperative-refund
signing, the watchtower, and live event subscriptions over WebSocket.

## Install

```sh
go get github.com/tachibtc/tachi-sdk-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tachibtc/tachi-sdk-go/tachi"
)

func main() {
	c, err := tachi.NewClient(tachi.WithBaseURL("http://127.0.0.1:26670/"))
	if err != nil {
		log.Fatal(err)
	}

	status, _, err := c.Node.Status(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", status)
}
```

`tachi.NewClient()` with no options defaults to the public Tachi regtest
daemon (`https://rpc-regtest.tachibtc.com/`). Pass `WithBaseURL` to target a
local or private node, e.g. `http://127.0.0.1:26670/`.

See [`docs/INTEGRATION.md`](./docs/INTEGRATION.md) for a fuller integration
guide (auth model, error handling, pagination, every service with request/
response shapes) written to hand to external consumers of this SDK.

## Client options

| Option | Purpose |
|---|---|
| `WithBaseURL(url)` | Point at a non-default daemon RPC address. |
| `WithAPIKey(key)` | Send `X-Api-Key` on every request. Required for privileged bitcoind methods, vault reconstruction params, and the `vaults_by_user` ABCI passthrough. Over a non-loopback host the base URL must be `https`/`wss` or the client refuses to send the key. |
| `WithHTTPClient(*http.Client)` | Use a custom HTTP client (custom TLS config, proxy, timeouts). |
| `WithUserAgent(ua)` | Override the default `User-Agent` header. |

Every call returns `(result, *tachi.Response, error)`. `*tachi.Response`
wraps the raw `*http.Response` — check it for status code, rate-limit
headers, etc. On a non-2xx response, `err` is a `*tachi.ErrorResponse`.

## Services

The `Client` exposes one field per functional area; each maps to a group of
daemon RPC endpoints. Every method takes `context.Context` first and
returns `(result, *tachi.Response, error)`. Full descriptions, response
field docs, and a usage example for every method are in
[`docs/INTEGRATION.md`](./docs/INTEGRATION.md#services) — this is the
compact signature index.

### `c.Node` — health, identity, consensus/network status

```go
Health(ctx) (*HealthResponse, *Response, error)                          // GET /health
Info(ctx) (*NodeInfoResponse, *Response, error)                          // GET /tachi_nodeInfo
Status(ctx) (*CometRPCResponse, *Response, error)                        // GET /tachi_status
NetInfo(ctx) (*CometRPCResponse, *Response, error)                       // GET /tachi_netInfo
ConsensusState(ctx) (*CometRPCResponse, *Response, error)                // GET /tachi_consensusState
ValidatorsPower(ctx) (*CometRPCResponse, *Response, error)               // GET /tachi_validatorsPower
Query(ctx, path string, opts *QueryOptions) (*CometRPCResponse, *Response, error) // GET /tachi_query
```

### `c.Validators` — bootstrap registry, this node's peer info

```go
List(ctx) (*ValidatorsResponse, *Response, error)          // GET /tachi_validators
Live(ctx) (*LiveValidatorsResponse, *Response, error)      // GET /tachi_validators/live
Count(ctx) (*ValidatorCountResponse, *Response, error)     // GET /tachi_validators/count
Ready(ctx, expected int) (*ReadyResponse, *Response, error) // GET /tachi_validators/ready
PeerInfo(ctx) (*ValidatorInfo, *Response, error)            // GET /tachi_peerInfo
```

`/tachi_validators/register` is intentionally not wrapped by this SDK.

### `c.Block` — Tachi-chain block lookups (not Bitcoin L1)

```go
Get(ctx, height int64) (*BlockResponse, *Response, error)                    // GET /tachi_block
List(ctx, page, pageSize int) (*ListBlocksResponse, *Response, error)        // GET /tachi_listBlocks
Hash(ctx, height int64) (*GetBlockHashResponse, *Response, error)            // GET /tachi_getBlockHash
HeaderByHeight(ctx, height int64) (*GetBlockHeaderResponse, *Response, error) // GET /tachi_getBlockHeader
HeaderByHash(ctx, hash string) (*GetBlockHeaderResponse, *Response, error)   // GET /tachi_getBlockHeader
ByHeight(ctx, height int64) (*BlockResponse, *Response, error)                // GET /tachi_getBlock
ByHash(ctx, hash string) (*BlockResponse, *Response, error)                  // GET /tachi_getBlock
```

### `c.Epoch` — Verkle-root checkpoint lookups

```go
Get(ctx, id uint32) (*GetEpochResponse, *Response, error)             // GET /tachi_epoch?id=
ByHash(ctx, hash string) (*GetEpochResponse, *Response, error)         // GET /tachi_epoch?hash=
List(ctx, page, pageSize int) (*ListEpochsResponse, *Response, error) // GET /tachi_listEpochs
```

### `c.Tx` — transaction lookup, decode/validate, broadcast, mempool, fees

```go
Get(ctx, hash string, opts *TxOptions) (*GetTransactionResponse, *Response, error) // GET /tachi_tx
Raw(ctx, hash string) (*GetRawTransactionResponse, *Response, error)               // GET /tachi_txRaw
List(ctx, opts *ListTransactionsOptions) (*ListTransactionsResponse, *Response, error) // GET /tachi_listTransactions
Mempool(ctx) (*MempoolResponse, *Response, error)                                  // GET /tachi_mempool
Decode(ctx, hexTx string) (*TxDecodeResponse, *Response, error)                    // POST /tachi_txDecode
Validate(ctx, hexTx string) (*TxValidateResponse, *Response, error)                // POST /tachi_txValidate
BroadcastSync(ctx, hexTx string) (*CometRPCResponse, *Response, error)             // POST /tachi_txBroadcastSync
BroadcastAsync(ctx, hexTx string) (*CometRPCResponse, *Response, error)            // POST /tachi_txBroadcastAsync
FeeEstimate(ctx) (*FeeEstimateResponse, *Response, error)                          // GET /tachi_feeEstimate
```

### `c.VTXO` — VTXO lookup and listing

```go
Get(ctx, id string) (*VTXOResponse, *Response, error)                    // GET /tachi_vtxo
Locked(ctx, vault string) (*LockedVTXOsResponse, *Response, error)       // GET /tachi_vtxoLocked
List(ctx, page, pageSize int) (*ListVTXOsResponse, *Response, error)     // GET /tachi_listVtxos
```

### `c.Vault` — vault listing

```go
List(ctx, user string, page, pageSize int) (*ListVaultsResponse, *Response, error) // GET /tachi_listVaults
```

Reconstruction fields (`CSVDelay`, `Threshold`, `QuorumKeyset`, `UserKey`)
are only populated with `WithAPIKey`.

### `c.Address` — account-scoped balance, nonce, VTXOs, history

```go
Get(ctx, address string) (*AddressResponse, *Response, error)                                    // GET /tachi_address
VTXOs(ctx, address string, includeSpent bool) (*AddressVTXOsResponse, *Response, error)          // GET /tachi_addressVtxos
Transactions(ctx, address string, opts *AddressTransactionsOptions) (*AddressTransactionsResponse, *Response, error) // GET /tachi_addressTransactions
Balance(ctx, address string) (*BalanceResponse, *Response, error)                                 // GET /tachi_balance
Nonce(ctx, address string) (*NonceResponse, *Response, error)                                     // GET /tachi_nonce
Mempool(ctx, address string) (*MempoolByAddressResponse, *Response, error)                        // GET /tachi_mempoolByAddress
```

### `c.Dashboard` — explorer aggregate stats and search

```go
Stats(ctx) (*StatsResponse, *Response, error)          // GET /tachi_stats
Supply(ctx) (*SupplyResponse, *Response, error)        // GET /tachi_supply
Search(ctx, q string) (*SearchResponse, *Response, error) // GET /tachi_search
```

### `c.Bitcoin` — pass-through Bitcoin Core JSON-RPC proxy

```go
RPC(ctx, method string, params interface{}) (json.RawMessage, *Response, error) // POST /
```

Common read-only methods need no auth; wallet/admin/network-mutation
methods need `WithAPIKey` set to the daemon's master `BTC_RPC_API_KEY`. See
[Bitcoin JSON-RPC proxy](#bitcoin-json-rpc-proxy) below.

### `c.Sign` — cooperative-refund threshold-signing ceremony

```go
Transaction(ctx, tx *RefundTx) (*SignTransactionResponse, *Response, error) // POST /tachi_signTransaction
```

### `c.Watchtower` — vault breach detection status and receipts

```go
Status(ctx) (*WatchtowerStatus, *Response, error)                            // GET /tachi_watchtower/status
Receipts(ctx, vaultID string) ([]BreachReceipt, *Response, error)             // GET /tachi_watchtower/receipts
Receipt(ctx, vaultID string, state uint64) (*BreachReceipt, *Response, error) // GET /tachi_watchtower/receipts?state=
```

Returns a 503 `*ErrorResponse` if the daemon has no watchtower configured.

### `c.WS` — live subscription feed

```go
Subscribe(ctx, opts SubscribeOptions) (*WSConn, error) // GET /tachi_ws (upgraded to ws/wss)
```

See [Live subscriptions](#live-subscriptions-websocket) below.

## Live subscriptions (WebSocket)

```go
conn, err := c.WS.Subscribe(ctx, tachi.SubscribeOptions{
	Address: ownerPubkeyHex, // incoming vouts to this address
	Blocks:  true,           // every durably-committed block
})
if err != nil {
	log.Fatal(err)
}
defer conn.Close()

for {
	select {
	case evt, ok := <-conn.Events():
		if !ok {
			return // connection closed; check conn.Err()
		}
		fmt.Printf("%+v\n", evt)
	case err := <-conn.Err():
		log.Println("ws error:", err)
		return
	}
}
```

`SubscribeOptions` fields (at least one required): `Address`, `Vault`,
`VaultID` (watchtower breach alerts), `Blocks`, `Validators`, and `Txs`
(every transaction, unfiltered). Each delivered `WSEvent` has `Event` set to
`"tx"`, `"block"`, `"validator"`, or `"breach"`, with the matching field
populated.

## Bitcoin JSON-RPC proxy

```go
raw, _, err := c.Bitcoin.RPC(ctx, "getblockchaininfo", nil)
```

`params` is marshalled as-is into the request; pass `nil` for none. Common
read-only bitcoind methods (chain/mempool/tx-decoding) need no auth. Wallet,
admin, and network-mutation methods require `WithAPIKey` set to the
daemon's master `BTC_RPC_API_KEY`.

## Examples

- [`examples/basic`](./examples/basic) — smoke-tests read-only calls across
  most services against a live daemon.
- [`examples/websocket`](./examples/websocket) — opens a block-alert
  subscription and prints the first event.

Run either with a daemon listening on `127.0.0.1:26670`:

```sh
go run ./examples/basic
go run ./examples/websocket
```

## Development

```sh
go build ./...
go vet ./...
go test ./...
```
