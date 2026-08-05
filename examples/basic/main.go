// Command smoketest exercises a handful of read-only SDK calls against a
// live Tachi daemon to sanity-check response decoding. Not part of the
// public SDK surface.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/tachibtc/daemon-go-sdk/tachi"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := tachi.NewClient()
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	health, _, err := c.Node.Health(ctx)
	must("Node.Health", health, err)

	stats, _, err := c.Dashboard.Stats(ctx)
	must("Dashboard.Stats", stats, err)

	supply, _, err := c.Dashboard.Supply(ctx)
	must("Dashboard.Supply", supply, err)

	nodeInfo, _, err := c.Node.Info(ctx)
	must("Node.Info", nodeInfo, err)

	feeEst, _, err := c.Tx.FeeEstimate(ctx)
	must("Tx.FeeEstimate", feeEst, err)

	mempool, _, err := c.Tx.Mempool(ctx)
	must("Tx.Mempool", mempool, err)

	blocks, _, err := c.Block.List(ctx, 1, 3)
	must("Block.List", blocks, err)

	if blocks != nil && len(blocks.Blocks) > 0 {
		h := blocks.Blocks[0].Height
		block, _, err := c.Block.Get(ctx, h)
		must(fmt.Sprintf("Block.Get(%d)", h), block, err)

		hdr, _, err := c.Block.HeaderByHeight(ctx, h)
		must(fmt.Sprintf("Block.HeaderByHeight(%d)", h), hdr, err)
	}

	epochs, _, err := c.Epoch.List(ctx, 1, 3)
	must("Epoch.List", epochs, err)

	vtxos, _, err := c.VTXO.List(ctx, 1, 3)
	must("VTXO.List", vtxos, err)

	txs, _, err := c.Tx.List(ctx, &tachi.ListTransactionsOptions{PageSize: 5})
	must("Tx.List", txs, err)

	validators, _, err := c.Validators.List(ctx)
	must("Validators.List", validators, err)

	status, _, err := c.Node.Status(ctx)
	must("Node.Status", status, err)

	if vtxos != nil && len(vtxos.VTXOs) > 0 {
		v := vtxos.VTXOs[0]
		got, _, err := c.VTXO.Get(ctx, v.ID)
		must(fmt.Sprintf("VTXO.Get(%s)", v.ID[:8]), got, err)

		addr, _, err := c.Address.Get(ctx, v.Owner)
		must(fmt.Sprintf("Address.Get(%s)", v.Owner[:8]), addr, err)

		bal, _, err := c.Address.Balance(ctx, v.Owner)
		must(fmt.Sprintf("Address.Balance(%s)", v.Owner[:8]), bal, err)

		nonce, _, err := c.Address.Nonce(ctx, v.Owner)
		must(fmt.Sprintf("Address.Nonce(%s)", v.Owner[:8]), nonce, err)

		addrVtxos, _, err := c.Address.VTXOs(ctx, v.Owner, true)
		must(fmt.Sprintf("Address.VTXOs(%s)", v.Owner[:8]), addrVtxos, err)

		search, _, err := c.Dashboard.Search(ctx, v.Owner)
		must(fmt.Sprintf("Dashboard.Search(%s)", v.Owner[:8]), search, err)
	}

	if txs != nil && len(txs.Transactions) > 0 {
		h := txs.Transactions[0].TxHash
		got, _, err := c.Tx.Get(ctx, h, &tachi.TxOptions{HAT: true})
		must(fmt.Sprintf("Tx.Get(%s)", h[:8]), got, err)

		raw, _, err := c.Tx.Raw(ctx, h)
		must(fmt.Sprintf("Tx.Raw(%s)", h[:8]), raw, err)

		if raw != nil {
			decoded, _, err := c.Tx.Decode(ctx, raw.Hex)
			must(fmt.Sprintf("Tx.Decode(%s)", h[:8]), decoded, err)
		}
	}

	wtStatus, _, err := c.Watchtower.Status(ctx)
	if err != nil {
		log.Printf("Watchtower.Status: %v (may be disabled on this daemon, non-fatal)", err)
	} else {
		must("Watchtower.Status", wtStatus, nil)
	}

	// Bitcoin JSON-RPC proxy read-only method.
	raw, _, err := c.Bitcoin.RPC(ctx, "getblockchaininfo", nil)
	if err != nil {
		log.Printf("Bitcoin.RPC(getblockchaininfo): %v (may be unconfigured on this daemon, non-fatal)", err)
	} else {
		fmt.Println("Bitcoin.RPC(getblockchaininfo) OK:", string(raw)[:min(120, len(raw))])
	}

	fmt.Println("\nsmoke test complete")
}

func must(label string, v interface{}, err error) {
	if err != nil {
		log.Fatalf("%s: %v", label, err)
	}
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	fmt.Printf("%-28s OK  %s\n", label, s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
