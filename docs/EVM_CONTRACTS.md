# Deploying and Using Smart Contracts on Tachi with Go (Beginner Guide)

This guide shows you how to put a Solidity smart contract on Tachi and use it from Go with `github.com/tachibtc/tachi-sdk-go`. It assumes no blockchain experience. Follow the steps in order.

Every command and output here was run against regtest on 2026-11-05. The finished code is in [`examples/evm-contracts`](../examples/evm-contracts): run it as is, or copy it into your own project.

> ⚠️ **Testnet only.** Everything here uses **regtest** or **signet**. Never put a private key that holds real money into code or a `.env` file.

---

## 0. The ideas you need first (5-minute read)

| Term | What it means on Tachi |
|---|---|
| **Smart contract** | A small program stored on the chain. Anyone can call its functions. Written in **Solidity**. |
| **Deploy** | Uploading your compiled contract to the chain. You get back a **contract address** (`0x…`, 40 hex chars). |
| **Read (call)** | Asking the contract a question, like "what is `count`?". **Free.** No signature, nothing changes. |
| **Write (transaction)** | Changing something, like "`increment()` the counter". It **costs gas**, needs your **signature**, and only takes effect once it is in a block. |
| **Gas** | How much work a transaction does. You pay for gas in **sats** (the smallest unit of BTC). |
| **VTXO** | A "coin" you own on Tachi. Gas is paid by spending your VTXOs. You need some free VTXOs before you can deploy or write. |
| **Private key / public key** | Your private key signs transactions (keep it secret). Your public key (32 bytes, 64 hex chars) is your Tachi identity. |
| **EVM address** | The `0x…` address contracts see as `msg.sender`. It is derived from your public key. |
| **Nonce** | A counter that goes up by one with every transaction you send. The code fetches it for you. |
| **Receipt** | The result of a transaction once it is in a block: did it succeed (`Status` 1) or fail (`Status` 0), how much gas it used, and which events it emitted. |
| **ABI** | A JSON description of a contract's functions. Libraries use it to turn `increment()` into the bytes the chain understands. |

### How a write works, end to end

```
 write code ──► compile ──► simulate (free dry run) ──► build tx ──► sign ──► broadcast ──► wait for block ──► read receipt
                              │                                                                        │
                              └── tells you if it will fail and how much gas it needs                  └── Status 1 = success
```

**Always simulate first.** If a transaction fails on chain, you **still pay the gas**. The free simulation catches most failures before they cost you anything.

### Who does what

| Job | Who does it |
|---|---|
| Talk to the Tachi RPC: simulate (`c.EVM.Call`), receipts, nonce, VTXOs, broadcast | **`tachi-sdk-go`** |
| Encode function calls and decode results (ABI) | **go-ethereum**'s `accounts/abi` package |
| Derive your key from a recovery phrase, sign (BIP-340 Schnorr) | **btcd**'s `btcec/v2` and `hdkeychain` packages |
| Put the transaction into Tachi's byte format | **`evmtx.go`** in the example, about 100 lines (step 5). You copy it, you don't write it. |

---

## 1. Prerequisites

1. **Go 1.26 or newer.** Check with:
   ```bash
   go version   # must print go1.26 or higher
   ```
   If it's older, install the latest from https://go.dev/dl.
2. **The Solidity compiler `solc` 0.8.30 or newer**, to compile contracts:
   ```bash
   brew install solidity        # macOS
   # Linux: sudo snap install solc, or download from https://github.com/ethereum/solidity/releases
   solc --version
   ```
   The example already includes the compiled contract in `build/`, so you can skip compiling the first time.
3. **A code editor**, for example VS Code with the Go extension.
4. **A testnet wallet (12/24-word recovery phrase) that holds free VTXOs** (see step 3).
5. **A Tachi RPC endpoint.** These are the public test networks:

| Network | RPC URL | Explorer |
|---|---|---|
| regtest | `https://rpc-regtest.tachibtc.com` | https://regtest.tachibtcscan.com |
| signet | `https://rpc-signet.tachibtc.com` | https://signet.tachibtcscan.com |

> 💡 You can try any RPC endpoint in your browser at `https://rpc-regtest.tachibtc.com/swagger/index.html`.

---

## 2. Create the project

**Option A: run the example.**

```bash
git clone https://github.com/tachibtc/tachi-sdk-go.git
cd tachi-sdk-go/examples/evm-contracts
```

**Option B: your own project.** Copy `main.go`, `tachi.go`, `evmtx.go`, `contracts/`, `build/` and `.env.example` from [`examples/evm-contracts`](../examples/evm-contracts) into a new folder, then:

```bash
go mod init my-tachi-app
go get github.com/tachibtc/tachi-sdk-go@main
go get github.com/ethereum/go-ethereum github.com/btcsuite/btcd/btcec/v2 github.com/btcsuite/btcd/btcutil github.com/joho/godotenv
go mod tidy
```

> 💡 The EVM endpoints (`c.EVM`) are newer than SDK release v0.2.0, so use `@main` until the next release is tagged, then use that tag.

The layout:

```
evm-contracts/
├── contracts/Counter.sol   the Solidity contract
├── build/Counter.abi       compiler output: the ABI (JSON)
├── build/Counter.bin       compiler output: the bytecode (hex)
├── evmtx.go                Tachi transaction format + signing digest (copy as is)
├── tachi.go                shared helpers: client, key, send, wait for receipt
├── main.go                 the commands: whoami, deploy, read, write, set
└── .env                    your recovery phrase (never commit it)
```

---

## 3. Set up your wallet and funds

### 3a. Put your recovery phrase in `.env`

```bash
cp .env.example .env
```

Open `.env` and fill it in:

```bash
MNEMONIC="word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12"
TACHI_RPC=https://rpc-regtest.tachibtc.com
```

- The code uses the key at derivation path `m/84'/1'/0'/0/0`, the wallet's first account. To use a different account, add `KEY_PATH=m/84'/1'/0'/0/1`.
- If you have a raw private key (64 hex characters) instead of a phrase, use `PRIVATE_KEY=abcd…` in place of `MNEMONIC`.
- For signet, set `TACHI_RPC=https://rpc-signet.tachibtc.com`.

The example's `.gitignore` already lists `.env`. In your own project, add it:

```bash
echo ".env" >> .gitignore
```

### 3b. Make sure the wallet has free VTXOs

Deploying and writing spend VTXOs to pay gas. It's cheap: at the minimum gas price, deploying the example contract sets aside **255 sats** (it used 197 and refunded 58), and one `increment()` sets aside **37 sats**. But you need **some** free VTXOs. Ask the Tachi team for testnet funds, or deposit test BTC with the Taurus Vault wallet.

### 3c. Check your setup

```bash
go run . whoami
```

```
Tachi public key: d613…0e75
EVM address:      0x0DdB993574Fa741A7e9a09421C8Aa9630C16F99b
Free balance:     197163 sats in 17 VTXOs
```

- The **EVM address** should match what the Tachi extension shows for this account. If it doesn't, check the phrase and `KEY_PATH`.
- If the balance is 0, get funds (step 3b) before deploying.

---

## 4. Write and compile the contract

### 4a. The contract: `contracts/Counter.sol`

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.30;

contract Counter {
    uint256 public count;
    address public owner;

    event Incremented(address indexed by, uint256 newCount);

    constructor(uint256 start) {
        count = start;
        owner = msg.sender;
    }

    function increment() external {
        count += 1;
        emit Incremented(msg.sender, count);
    }

    function setCount(uint256 value) external {
        require(msg.sender == owner, "only owner");
        count = value;
    }
}
```

What it does:
- `count` and `owner` are `public`, so Solidity creates free read functions `count()` and `owner()` automatically.
- The `constructor` runs once, at deploy time. It takes a starting value and remembers who deployed it.
- Anyone can call `increment()`. It emits an **event** (a log entry you can read from the receipt).
- Only the owner can call `setCount()`. Anyone else gets the revert reason `"only owner"`.

### 4b. Compile it

```bash
solc --evm-version cancun --optimize --abi --bin -o build --overwrite contracts/Counter.sol
```

This writes `build/Counter.abi` (the ABI) and `build/Counter.bin` (the bytecode).

> ⚠️ **Always pass `--evm-version cancun`.** That's the EVM version Tachi runs. Code compiled for a newer version can use instructions Tachi doesn't have yet, and fail.

---

## 5. The transaction format: `evmtx.go`

Tachi transactions have their own byte format; they are not Ethereum transactions. `evmtx.go` builds it. **You don't need to change this file.** The parts to know:

```go
const (
	txDeploy    = 0x08 // deploys a contract
	txCall      = 0x09 // calls a contract
	minGasPrice = 1    // lowest gas price, in millisats per gas
)

type EVMTx struct {
	Type     byte     // txDeploy or txCall
	Inputs   [][]byte // VTXO IDs that pay for gas
	Outputs  []Output // change back to yourself
	PubKey   []byte   // your 32-byte public key
	Nonce    uint64
	GasLimit uint64
	GasPrice uint64
	To       [20]byte // the contract (calls only)
	Data     []byte   // bytecode (deploy) or encoded function call (call)
	Sig      []byte   // your signature
}

func (tx *EVMTx) sigHash() []byte // the 32 bytes you sign
func (tx *EVMTx) Hex() string     // what you broadcast
func (tx *EVMTx) Hash() string    // the tx hash, to look up the receipt
func feeSats(gasLimit, gasPrice uint64) uint64 // the most sats it can spend on gas
```

---

## 6. The shared helpers: `tachi.go`

`tachi.go` holds what every command shares. Open it alongside this section.

| Function | What it does |
|---|---|
| `newClient()` | Loads `.env` and connects: `tachi.NewClient(tachi.WithBaseURL(url))`. |
| `loadWallet()` | Turns your recovery phrase into a private key, your 32-byte public key, and your EVM address. |
| `gasLimitFor(gasUsed)` | Simulated gas + 30% headroom, kept inside 21,000 to 1,000,000. |
| `pickInputs(…)` | Picks free VTXOs, biggest first, until they cover the gas. |
| `sendEvmTx(…)` | Builds, signs and broadcasts a deploy or a call. |
| `waitForReceipt(…)` | Polls `c.EVM.Receipt` until the transaction is in a block. |

### What `sendEvmTx` does, step by step

```go
func (w *wallet) sendEvmTx(ctx context.Context, c *tachi.Client, to *common.Address, data []byte, gasLimit uint64) string {
	fee := feeSats(gasLimit, minGasPrice)    // 1. the most this tx can spend on gas
	inputs, sum := w.pickInputs(ctx, c, fee) //    and VTXOs that cover it

	// 2. The nonce the next EVM tx must use.
	acct, _, err := c.EVM.Account(ctx, hex.EncodeToString(w.PubKey))
	must(err)

	tx := &EVMTx{Type: txDeploy, Inputs: inputs, PubKey: w.PubKey, Nonce: acct.Nonce,
		GasLimit: gasLimit, GasPrice: minGasPrice, Data: data}
	if to != nil { // a call has a contract address
		tx.Type, tx.To = txCall, *to
	}
	if change := sum - fee; change > 0 { // the change goes back to you
		tx.Outputs = []Output{{Owner: w.PubKey, Amount: change}}
	}

	sig, err := schnorr.Sign(w.priv, tx.sigHash()) // 3. sign
	must(err)
	tx.Sig = sig.Serialize()

	res, _, err := c.Tx.BroadcastSync(ctx, tx.Hex()) // 4. broadcast
	must(err)
	// res.Result holds {"code": 0, ...}; any other code means the network refused it.
	...
	return tx.Hash()
}
```

1. **Fee**: the gas limit times the gas price, in sats. It is set aside; what isn't used comes back as a refund.
2. **Nonce**: `c.EVM.Account(pubkey).Nonce` is the number this transaction must use. Don't use `NextNonce` from `c.Address.Nonce` for EVM transactions: it is one too high.
3. **Sign**: a BIP-340 Schnorr signature over `tx.sigHash()`.
4. **Broadcast**: `code: 0` only means the network **accepted** it. Whether it **succeeded** is in the receipt.

---

## 7. Deploy the contract

The `deploy` command in `main.go`:

```go
bin, err := os.ReadFile("build/Counter.bin")
must(err)
code, err := hex.DecodeString(strings.TrimSpace(string(bin)))
must(err)
args, err := counter.Pack("", big.NewInt(10)) // constructor(uint256 start) = 10
must(err)
initCode := append(code, args...)             // bytecode + constructor arguments

// 1. Simulate: free, and tells you how much gas it needs. No To means a deploy.
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

// 2. Sign and broadcast, 3. wait for the receipt.
hash := w.sendEvmTx(ctx, c, nil, initCode, gasLimit)
r := waitForReceipt(ctx, c, hash)
if r.Status != 1 {
	fail("Deploy failed on chain: " + r.Error + " (gas was still charged)")
}
fmt.Println("Contract deployed at", r.ContractAddress)
```

Run it:

```bash
go run . deploy
```

```
Simulation OK: 196075 gas, using gasLimit 254897
Sent tx 8a88…3384 (sets aside up to 255 sats for gas, nonce 16)
Contract deployed at 0x99931a82ec9f6fec55c980ac6c071727820ad0e8 (fee 197 sats, refund 58 sats)
```

It takes about a minute, the time until the next block. **Copy the contract address**; the next commands need it.

> 💡 Constructor arguments are encoded with `counter.Pack("", …)` (an empty name means the constructor) and **appended** to the bytecode.

---

## 8. Read from the contract (free)

```go
// callView runs a read-only function for free and decodes its return values.
func callView(ctx context.Context, c *tachi.Client, a abi.ABI, to common.Address, fn string, args ...any) []any {
	data, err := a.Pack(fn, args...) // "count" → 0x06661abd
	must(err)
	res, _, err := c.EVM.Call(ctx, &tachi.EVMCallRequest{To: to.Hex(), Data: "0x" + hex.EncodeToString(data)})
	must(err)
	if res.Status != 1 {
		fail(fn + " reverted: " + revertReason(res.Error, res.ReturnData))
	}
	out, err := a.Unpack(fn, common.FromHex(res.ReturnData)) // bytes → Go values
	must(err)
	return out
}
```

```bash
go run . read 0x99931a82ec9f6fec55c980ac6c071727820ad0e8
```

```
count() = 10
owner() = 0x0DdB993574Fa741A7e9a09421C8Aa9630C16F99b
```

`count` is 10 (the constructor argument), and `owner` is your EVM address from `whoami`.

- `uint256` comes back as `*big.Int`, `address` as `common.Address`, `string` as `string`, `bool` as `bool`.
- Reads cost nothing and need no key. You can read any contract.

---

## 9. Write to the contract (costs gas)

```go
// sendWrite simulates a call as you, stops with the revert reason if it would fail, then sends it.
func (w *wallet) sendWrite(ctx context.Context, c *tachi.Client, to common.Address, data []byte) string {
	sim, _, err := c.EVM.Call(ctx, &tachi.EVMCallRequest{From: w.Me.Hex(), To: to.Hex(), Data: "0x" + hex.EncodeToString(data)})
	must(err)
	if sim.Status != 1 {
		fail("Simulation failed, nothing was sent: " + revertReason(sim.Error, sim.ReturnData))
	}
	return w.sendEvmTx(ctx, c, &to, data, gasLimitFor(sim.GasUsed))
}
```

The `write` command calls `increment()`, then reads its event from the receipt:

```go
data, err := counter.Pack("increment")
must(err)
r := waitForReceipt(ctx, c, w.sendWrite(ctx, c, addr, data))

for _, lg := range r.Logs {
	ev, err := counter.EventByID(common.HexToHash(lg.Topics[0])) // Topics[0] says which event
	if err != nil {
		continue // an event this ABI doesn't know
	}
	vals, err := ev.Inputs.NonIndexed().Unpack(common.FromHex(lg.Data))
	must(err)
	// indexed parameters (here `by`) are in Topics[1:], the others in Data
	fmt.Printf("Event %s: by=%s newCount=%v\n", ev.Name, common.HexToAddress(lg.Topics[1]), vals[0])
}
```

```bash
go run . write 0x99931a82ec9f6fec55c980ac6c071727820ad0e8
```

```
Sent tx 8c3a…f477 (sets aside up to 37 sats for gas, nonce 17)
Status 1, gas used 27866, fee 28 sats
Event Incremented: by=0x0DdB993574Fa741A7e9a09421C8Aa9630C16F99b newCount=11
count() is now 11
```

### Calling a function that takes arguments

Pass the arguments to `Pack`, in order, as Go values:

```go
data, err := counter.Pack("setCount", big.NewInt(42)) // uint256 → *big.Int
```

```bash
go run . set 0x99931a82ec9f6fec55c980ac6c071727820ad0e8 42
```

```
Sent tx a3e4…b97c (sets aside up to 38 sats for gas, nonce 18)
Status 1, count() is now 42
```

Go types for Solidity arguments: `uint256`/`int256` → `*big.Int`; `uint8`…`uint64` → `uint8`…`uint64`; `address` → `common.Address`; `bool` → `bool`; `string` → `string`; `bytes` → `[]byte`; `bytes32` → `[32]byte`.

### Getting the human-readable revert reason

When a `require(…, "only owner")` fails, the reason is ABI-encoded in `ReturnData`. go-ethereum decodes it:

```go
func revertReason(errMsg, returnData string) string {
	if reason, err := abi.UnpackRevert(common.FromHex(returnData)); err == nil {
		return errMsg + ": " + reason
	}
	return errMsg
}
```

Simulating `setCount` from an address that isn't the owner gives:

```
execution reverted: only owner
```

and **nothing is sent**, so it costs nothing.

---

## 10. Using an existing contract (for example an ERC-20 token)

You don't need to deploy anything to use someone else's contract. You only need its **address** and its **ABI**, and you can list just the functions you use:

```go
const erc20ABI = `[
 {"type":"function","name":"name","stateMutability":"view","inputs":[],"outputs":[{"type":"string"}]},
 {"type":"function","name":"symbol","stateMutability":"view","inputs":[],"outputs":[{"type":"string"}]},
 {"type":"function","name":"decimals","stateMutability":"view","inputs":[],"outputs":[{"type":"uint8"}]},
 {"type":"function","name":"balanceOf","stateMutability":"view","inputs":[{"name":"who","type":"address"}],"outputs":[{"type":"uint256"}]},
 {"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[{"name":"to","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"type":"bool"}]}
]`

token, err := abi.JSON(strings.NewReader(erc20ABI))
must(err)
tokenAddr := common.HexToAddress("0xa0fb3eae0bfc6b3c387edd938000380ba34eee65")

// Read: free
fmt.Println(callView(ctx, c, token, tokenAddr, "name")[0])            // Gopher Phoenix
fmt.Println(callView(ctx, c, token, tokenAddr, "symbol")[0])          // GP
fmt.Println(callView(ctx, c, token, tokenAddr, "balanceOf", w.Me)[0]) // 9999999999999999999900000

// Write: send 1 token (18 decimals) to a friend
one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 1 token = 10^18 units
data, err := token.Pack("transfer", common.HexToAddress("0xFRIEND_EVM_ADDRESS"), one)
must(err)
r := waitForReceipt(ctx, c, w.sendWrite(ctx, c, tokenAddr, data))
fmt.Println("status", r.Status)
```

> 💡 Token amounts are in the token's smallest unit. Read `decimals()` first: with 18 decimals, 1 token is 10^18 units.

The full ABI of any contract compiled with `solc` is in its `.abi` file. Load it with `abi.JSON(file)`, as `loadABI()` in `main.go` does.

---

## 11. No-code option: the Explorer

The Tachi Explorer does the same things in the browser:

- **Deploy:** https://regtest.tachibtcscan.com/tools/deploy-contract. Paste Solidity (or pick a template), click Compile, fill in the constructor arguments, then deploy with the **Tachi extension** or a testnet recovery phrase.
- **Read & write:** open `https://regtest.tachibtcscan.com/evm/contract/<address>`. **Read Contract** works without a wallet. **Write Contract** needs the Tachi extension connected and asks you to approve each transaction.
- **Check any transaction:** search its hash to see status, gas, events and the decoded method name.

For signet, use `signet.tachibtcscan.com`. The explorer is great for learning; use Go when you want to automate.

---

## 12. Rules and limits to remember

| Rule | Detail |
|---|---|
| **Failed transactions still cost gas** | `Status` 0 means it ran, failed, and you paid. Your nonce is still used up. Always simulate first. |
| **Broadcast `code: 0` ≠ success** | It only means the network accepted the transaction. Check the receipt's `Status`. |
| **One transaction at a time per wallet** | Wait for each receipt before sending the next. Two in flight can pick the same nonce or the same VTXO. |
| **Gas limit range** | 21,000 to 1,000,000 per transaction. |
| **Gas price** | In **millisats per gas**. Minimum and default is `1`. 1,000 gas at price 1 costs 1 sat. |
| **Contract size** | Init code is at most 49,152 bytes. Deployed code is at most 24,576 bytes. |
| **EVM version** | Compile with `--evm-version cancun`. |
| **No native BTC inside contracts (yet)** | `msg.value` must be 0. Any `payable` call that sends value fails. Tokens like ERC-20 work fine, because their balances live in contract storage. |
| **Nonce** | Use `c.EVM.Account(ctx, pubkeyHex).Nonce`. Don't use `NextNonce` from `c.Address.Nonce`; it is one too high for EVM transactions. |
| **Chain ID** | `0x54434849` ("TCHI"). You need it for EIP-712 signatures. |
| **Refund** | Unused gas comes back as a refund VTXO. The receipt's `RefundSats` and `RefundVTXOID` describe it. |
| **Rate limit** | `c.EVM.Call` is rate-limited per IP. Don't call it in a tight loop. |

---

## 13. Troubleshooting

| Error / symptom | Cause | Fix |
|---|---|---|
| `Set MNEMONIC (or PRIVATE_KEY) in .env` | `.env` is missing, or you ran from another folder | Create `.env` (step 3a), and run from the project folder. |
| `MNEMONIC has N words, want 12 or 24` | A missing word, or missing quotes | Wrap the phrase in `"…"` and check every word. |
| `whoami` shows a different EVM address than the extension | A typo in a word (it still makes *a* key, just not yours), or another account | Check every word, and set `KEY_PATH` to the extension's account. |
| `Not enough funds: need N sats, you have 0 sats` | No free VTXOs on this key | Get testnet funds (step 3b). Try a different `KEY_PATH`. |
| `Simulation failed, nothing was sent: execution reverted: …` | The contract would revert, for example because of a `require` or wrong arguments | Read the reason after the colon, and fix the arguments or the caller. |
| `Broadcast rejected (code N): … nonce …` | Two transactions were sent at once | Send one at a time, and wait for each receipt. |
| `Broadcast rejected (code N): …` (other) | The network refused it, for example because an input VTXO was already spent | Wait a few seconds and run again. The code picks fresh VTXOs each time. |
| `No receipt after 5 minutes` | The block is slow, or the transaction was dropped | Search the hash in the explorer. If it's not there, run again. |
| `429 Too Many Requests` | Too many `c.EVM.Call`s in a short time | Slow down, or add a short delay between calls. |
| `c.EVM undefined (type *tachi.Client has no field or method EVM)` | Your SDK version is older than the EVM endpoints | `go get github.com/tachibtc/tachi-sdk-go@main` (or the newest release). |
| `abi: attempting to unmarshal an empty string` | No contract at that address, or the wrong network | Check the address in the explorer, and check `TACHI_RPC`. |
| Contract works in Remix but fails on Tachi | It was compiled for a newer EVM, or it sends BTC value | Recompile with `--evm-version cancun`, and remove `msg.value` / `payable` transfers. |

---

## 14. Quick reference

### Tachi SDK (`github.com/tachibtc/tachi-sdk-go/tachi`)

| Call | What it does |
|---|---|
| `tachi.NewClient(tachi.WithBaseURL(url))` | Connects to a Tachi RPC. |
| `c.EVM.Call(ctx, &tachi.EVMCallRequest{From, To, Data, Nonce})` | Free simulation. Without `To`, it simulates a deploy. Returns `Status`, `GasUsed`, `ReturnData`, `Logs`, `Error`. |
| `c.EVM.Account(ctx, pubkeyOrAddress)` | `Nonce` (the one your next tx must use), `IsContract`, `CodeSize`. |
| `c.EVM.Receipt(ctx, hash)` | The result once it's in a block: `Status`, `GasUsed`, `FeeSats`, `RefundSats`, `ContractAddress`, `Logs`. |
| `c.EVM.Code(ctx, address)` | Deployed bytecode (`"0x"` means no contract there). |
| `c.EVM.Storage(ctx, address, slot)` | One raw 32-byte storage slot. |
| `c.Address.VTXOs(ctx, pubkey, false)` | Lists your VTXOs (your coins). |
| `c.Tx.BroadcastSync(ctx, hex)` | Broadcasts a signed transaction. |
| `c.Tx.Get(ctx, hash, nil)` | Any transaction. For EVM ones, `.EVM` has from/to, gas and, once executed, the result. |

### go-ethereum (`accounts/abi`, `common`)

| Call | What it does |
|---|---|
| `abi.JSON(reader)` | Loads an ABI. |
| `a.Pack("fn", args…)` | Encodes a function call. `a.Pack("", args…)` encodes constructor arguments. |
| `a.Unpack("fn", data)` | Decodes return data into Go values. |
| `a.EventByID(topic0)` + `ev.Inputs.NonIndexed().Unpack(data)` | Decodes an event log. |
| `abi.UnpackRevert(data)` | The revert reason string. |
| `common.HexToAddress(s)` / `common.FromHex(s)` | Hex to address / bytes. |

Every EVM endpoint and field: [INTEGRATION.md](./INTEGRATION.md#evm).
