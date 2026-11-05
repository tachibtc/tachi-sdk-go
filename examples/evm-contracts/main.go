// go run . whoami | deploy | read <address> | write <address> | set <address> <value>
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tachibtc/tachi-sdk-go/tachi"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: go run . whoami | deploy | read <address> | write <address> | set <address> <value>")
	}
	ctx := context.Background()
	c := newClient()
	counter := loadABI()

	switch os.Args[1] {
	case "whoami":
		w := loadWallet()
		res, _, err := c.Address.VTXOs(ctx, hex.EncodeToString(w.PubKey), false)
		must(err)
		var free int64
		for _, v := range res.VTXOs {
			if !v.Locked {
				free += v.Amount
			}
		}
		fmt.Printf("Tachi public key: %x\nEVM address:      %s\nFree balance:     %d sats in %d VTXOs\n", w.PubKey, w.Me, free, len(res.VTXOs))

	case "deploy":
		w := loadWallet()
		bin, err := os.ReadFile("build/Counter.bin")
		must(err)
		code, err := hex.DecodeString(strings.TrimSpace(string(bin)))
		must(err)
		args, err := counter.Pack("", big.NewInt(10)) // constructor(uint256 start) = 10
		must(err)
		initCode := append(code, args...)

		// 1. Simulate first: free, and tells you the gas and the address it will get.
		acct, _, err := c.EVM.Account(ctx, hex.EncodeToString(w.PubKey))
		must(err)
		sim, _, err := c.EVM.Call(ctx, &tachi.EVMCallRequest{
			From: w.Me.Hex(), Data: "0x" + hex.EncodeToString(initCode), Nonce: &acct.Nonce,
		})
		must(err)
		if sim.Status != 1 {
			fail("Simulation failed, nothing was sent: " + sim.Error)
		}
		gasLimit := gasLimitFor(sim.GasUsed)
		fmt.Printf("Simulation OK: %d gas, using gasLimit %d\n", sim.GasUsed, gasLimit)

		// 2. Sign and broadcast, 3. wait for the receipt.
		hash := w.sendEvmTx(ctx, c, nil, initCode, gasLimit)
		r := waitForReceipt(ctx, c, hash)
		if r.Status != 1 {
			fail(fmt.Sprintf("Deploy failed on chain: %s (gas was still charged)", r.Error))
		}
		fmt.Printf("Contract deployed at %s (fee %d sats, refund %d sats)\n", r.ContractAddress, r.FeeSats, r.RefundSats)

	case "read":
		addr := argAddress(2)
		for _, fn := range []string{"count", "owner"} {
			fmt.Printf("%s() = %v\n", fn, callView(ctx, c, counter, addr, fn)[0])
		}

	case "write":
		w := loadWallet()
		addr := argAddress(2)
		data, err := counter.Pack("increment")
		must(err)
		r := waitForReceipt(ctx, c, w.sendWrite(ctx, c, addr, data))
		fmt.Printf("Status %d, gas used %d, fee %d sats\n", r.Status, r.GasUsed, r.FeeSats)
		for _, lg := range r.Logs {
			ev, err := counter.EventByID(common.HexToHash(lg.Topics[0]))
			if err != nil {
				continue // an event this ABI doesn't know
			}
			vals, err := ev.Inputs.NonIndexed().Unpack(common.FromHex(lg.Data))
			must(err)
			fmt.Printf("Event %s: by=%s newCount=%v\n", ev.Name, common.HexToAddress(lg.Topics[1]), vals[0])
		}
		fmt.Println("count() is now", callView(ctx, c, counter, addr, "count")[0])

	case "set":
		w := loadWallet()
		addr := argAddress(2)
		if len(os.Args) < 4 {
			fail("usage: go run . set <address> <value>")
		}
		v, ok := new(big.Int).SetString(os.Args[3], 10)
		if !ok {
			fail("value must be a whole number")
		}
		data, err := counter.Pack("setCount", v)
		must(err)
		r := waitForReceipt(ctx, c, w.sendWrite(ctx, c, addr, data))
		fmt.Printf("Status %d, count() is now %v\n", r.Status, callView(ctx, c, counter, addr, "count")[0])

	default:
		fail("unknown command " + os.Args[1])
	}
}

// sendWrite simulates a call as you, stops with the revert reason if it would fail, then sends it.
func (w *wallet) sendWrite(ctx context.Context, c *tachi.Client, to common.Address, data []byte) string {
	sim, _, err := c.EVM.Call(ctx, &tachi.EVMCallRequest{From: w.Me.Hex(), To: to.Hex(), Data: "0x" + hex.EncodeToString(data)})
	must(err)
	if sim.Status != 1 {
		fail("Simulation failed, nothing was sent: " + revertReason(sim.Error, sim.ReturnData))
	}
	return w.sendEvmTx(ctx, c, &to, data, gasLimitFor(sim.GasUsed))
}

// callView runs a read-only function for free and decodes its return values.
func callView(ctx context.Context, c *tachi.Client, a abi.ABI, to common.Address, fn string, args ...any) []any {
	data, err := a.Pack(fn, args...)
	must(err)
	res, _, err := c.EVM.Call(ctx, &tachi.EVMCallRequest{To: to.Hex(), Data: "0x" + hex.EncodeToString(data)})
	must(err)
	if res.Status != 1 {
		fail(fn + " reverted: " + revertReason(res.Error, res.ReturnData))
	}
	out, err := a.Unpack(fn, common.FromHex(res.ReturnData))
	must(err)
	return out
}

// revertReason turns `require(..., "only owner")` return data back into "only owner".
func revertReason(errMsg, returnData string) string {
	if reason, err := abi.UnpackRevert(common.FromHex(returnData)); err == nil {
		return errMsg + ": " + reason
	}
	return errMsg
}

func loadABI() abi.ABI {
	f, err := os.Open("build/Counter.abi")
	must(err)
	defer f.Close()
	a, err := abi.JSON(f)
	must(err)
	return a
}

func argAddress(i int) common.Address {
	if len(os.Args) <= i || !common.IsHexAddress(os.Args[i]) {
		fail("pass the contract address (0x…)")
	}
	return common.HexToAddress(os.Args[i])
}
