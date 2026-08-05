# daemon-go-sdk

A Go client for the [Tachi daemon](../daemon) RPC API — validators, chain
data (blocks, epochs, transactions, VTXOs, vaults), account lookups,
dashboard/explorer endpoints, the bitcoind JSON-RPC proxy, cooperative-refund
signing, the watchtower, and live event subscriptions over WebSocket.

## Install

```sh
go get daemon-go-sdk
```

(Update the module path once this repo is published under its real import
path — see `go.mod`.)

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	"daemon-go-sdk/tachi"
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
daemon RPC endpoints.

| Field | Endpoints | Purpose |
|---|---|---|
| `c.Node` | `/health`, `/tachi_nodeInfo`, `/tachi_status`, `/tachi_netInfo`, `/tachi_consensusState`, `/tachi_validatorsPower`, `/tachi_peerInfo` | Node health, identity, consensus/network status. |
| `c.Validators` | `/tachi_validators*` | List, register, and wait-for-ready on the validator bootstrap registry. |
| `c.Block` | `/tachi_block`, `/tachi_listBlocks`, `/tachi_getBlock`, `/tachi_getBlockHash`, `/tachi_getBlockHeader` | Tachi-chain block lookups (not Bitcoin L1). |
| `c.Epoch` | `/tachi_epoch`, `/tachi_listEpochs` | Epoch (Verkle-root checkpoint) lookups. |
| `c.Tx` | `/tachi_tx`, `/tachi_txRaw`, `/tachi_listTransactions`, `/tachi_txBroadcastSync`, `/tachi_txBroadcastAsync`, `/tachi_txDecode`, `/tachi_txValidate`, `/tachi_feeEstimate`, `/tachi_mempool` | Transaction lookup, broadcast, decode/validate, mempool, fee estimation. |
| `c.VTXO` | `/tachi_vtxo`, `/tachi_vtxoLocked`, `/tachi_listVtxos` | VTXO lookup by ID, by vault lock, and paginated listing. |
| `c.Vault` | `/tachi_listVaults` | Vault listing (reconstruction params only with `WithAPIKey`). |
| `c.Address` | `/tachi_address`, `/tachi_addressVtxos`, `/tachi_addressTransactions`, `/tachi_balance`, `/tachi_nonce`, `/tachi_mempoolByAddress` | Account-scoped balance, nonce, VTXOs, history, pending activity. |
| `c.Dashboard` | `/tachi_stats`, `/tachi_supply`, `/tachi_search` | Explorer/dashboard aggregate stats and free-text search. |
| `c.Bitcoin` | `POST /` | Pass-through Bitcoin Core JSON-RPC 1.0 proxy. Common read-only methods need no auth; wallet/admin methods need `WithAPIKey`. |
| `c.Sign` | `/tachi_signTransaction` | Cooperative-refund threshold-signing ceremony. |
| `c.Watchtower` | `/tachi_watchtower/status`, `/tachi_watchtower/receipts` | Breach-detection status and receipts (only if the daemon has a watchtower enabled). |
| `c.WS` | `/tachi_ws` | Live subscription feed — see below. |

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
