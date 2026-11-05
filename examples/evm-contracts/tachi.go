// Shared setup: the client, your key, and the helpers every command uses.
package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
	"github.com/tachibtc/tachi-sdk-go/tachi"
)

type wallet struct {
	priv   *btcec.PrivateKey
	PubKey []byte         // 32-byte x-only key, your Tachi identity
	Me     common.Address // your EVM address: msg.sender inside contracts
}

func newClient() *tachi.Client {
	_ = godotenv.Load() // reads .env if present
	url := os.Getenv("TACHI_RPC")
	if url == "" {
		url = "https://rpc-regtest.tachibtc.com"
	}
	c, err := tachi.NewClient(tachi.WithBaseURL(url))
	must(err)
	return c
}

// loadWallet reads MNEMONIC (recovery phrase) or PRIVATE_KEY (64 hex) from the environment.
func loadWallet() *wallet {
	var priv *btcec.PrivateKey
	if phrase := strings.Join(strings.Fields(strings.ToLower(os.Getenv("MNEMONIC"))), " "); phrase != "" {
		if n := len(strings.Fields(phrase)); n != 12 && n != 24 {
			fail(fmt.Sprintf("MNEMONIC has %d words, want 12 or 24", n))
		}
		// BIP-39 seed: PBKDF2-HMAC-SHA512 of the phrase, salt "mnemonic", 2048 rounds.
		seed, err := pbkdf2.Key(sha512.New, phrase, []byte("mnemonic"), 2048, 64)
		must(err)
		key, err := hdkeychain.NewMaster(seed, &chaincfg.TestNet3Params)
		must(err)
		// m/84'/1'/0'/0/0 is the wallet's first account; set KEY_PATH for another.
		path := os.Getenv("KEY_PATH")
		if path == "" {
			path = "m/84'/1'/0'/0/0"
		}
		for _, part := range strings.Split(strings.TrimPrefix(path, "m/"), "/") {
			n, err := strconv.ParseUint(strings.TrimSuffix(part, "'"), 10, 31)
			if err != nil {
				fail("KEY_PATH is not a path like m/84'/1'/0'/0/0")
			}
			i := uint32(n)
			if strings.HasSuffix(part, "'") {
				i += hdkeychain.HardenedKeyStart
			}
			key, err = key.Derive(i)
			must(err)
		}
		priv, err = key.ECPrivKey()
		must(err)
	} else if k := strings.TrimPrefix(os.Getenv("PRIVATE_KEY"), "0x"); k != "" {
		b, err := hex.DecodeString(k)
		must(err)
		priv, _ = btcec.PrivKeyFromBytes(b)
	} else {
		fail("Set MNEMONIC (or PRIVATE_KEY) in .env. Testnet only!")
	}
	xonly := schnorr.SerializePubKey(priv.PubKey())
	// The EVM address of an x-only key is the Ethereum address of its even-Y point.
	even, err := schnorr.ParsePubKey(xonly)
	must(err)
	return &wallet{priv: priv, PubKey: xonly, Me: crypto.PubkeyToAddress(*even.ToECDSA())}
}

// gasLimitFor adds 30% headroom to simulated gas, kept inside 21,000..1,000,000.
func gasLimitFor(gasUsed uint64) uint64 {
	return min(max(gasUsed*13/10, 21_000), maxGas)
}

// pickInputs selects free VTXOs, biggest first, until they cover need sats.
func (w *wallet) pickInputs(ctx context.Context, c *tachi.Client, need uint64) ([][]byte, uint64) {
	res, _, err := c.Address.VTXOs(ctx, hex.EncodeToString(w.PubKey), false)
	must(err)
	var free []tachi.VTXOItem
	for _, v := range res.VTXOs {
		if !v.Spent && !v.Locked && v.Amount > 0 {
			free = append(free, v)
		}
	}
	sort.Slice(free, func(i, j int) bool { return free[i].Amount > free[j].Amount })
	var ids [][]byte
	var sum uint64
	for _, v := range free {
		if sum >= need {
			break
		}
		id, err := hex.DecodeString(v.ID)
		must(err)
		ids = append(ids, id)
		sum += uint64(v.Amount) // #nosec G115 -- only positive amounts are kept above
	}
	if sum < need {
		fail(fmt.Sprintf("Not enough funds: need %d sats, you have %d sats in free VTXOs", need, sum))
	}
	return ids, sum
}

// sendEvmTx builds, signs and broadcasts a deploy (to == nil) or a call (to = contract).
func (w *wallet) sendEvmTx(ctx context.Context, c *tachi.Client, to *common.Address, data []byte, gasLimit uint64) string {
	fee := feeSats(gasLimit, minGasPrice)
	inputs, sum := w.pickInputs(ctx, c, fee)

	// The nonce the next EVM tx must use. (Not tachi_nonce's next_nonce: that is one too high.)
	acct, _, err := c.EVM.Account(ctx, hex.EncodeToString(w.PubKey))
	must(err)

	tx := &EVMTx{Type: txDeploy, Inputs: inputs, PubKey: w.PubKey, Nonce: acct.Nonce,
		GasLimit: gasLimit, GasPrice: minGasPrice, Data: data}
	if to != nil {
		tx.Type, tx.To = txCall, *to
	}
	if change := sum - fee; change > 0 {
		tx.Outputs = []Output{{Owner: w.PubKey, Amount: change}}
	}

	must(tx.validate())
	sig, err := schnorr.Sign(w.priv, tx.sigHash())
	must(err)
	tx.Sig = sig.Serialize()

	res, _, err := c.Tx.BroadcastSync(ctx, tx.Hex())
	must(err)
	var r struct {
		Code int    `json:"code"`
		Log  string `json:"log"`
	}
	must(json.Unmarshal(res.Result, &r))
	if r.Code != 0 {
		fail(fmt.Sprintf("Broadcast rejected (code %d): %s", r.Code, r.Log))
	}
	fmt.Printf("Sent tx %s (sets aside up to %d sats for gas, nonce %d)\n", tx.Hash(), fee, tx.Nonce)
	return tx.Hash()
}

// waitForReceipt polls until the tx is in a block. Blocks can take a few minutes.
func waitForReceipt(ctx context.Context, c *tachi.Client, hash string) *tachi.EVMReceiptResponse {
	for range 150 {
		r, _, err := c.EVM.Receipt(ctx, hash)
		if err == nil {
			return r
		}
		var e *tachi.ErrorResponse
		if !errors.As(err, &e) || !strings.Contains(strings.ToLower(e.Message), "no evm receipt") {
			must(err)
		}
		time.Sleep(2 * time.Second)
	}
	fail("No receipt after 5 minutes for " + hash + "; check the explorer")
	return nil
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
