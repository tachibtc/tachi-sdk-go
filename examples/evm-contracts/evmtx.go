// Building, encoding and signing Tachi EVM transactions (TxDeploy, TxCall).
// The wire format and signing digest match the daemon's types.EncodeTx and
// types.SigHash, so the bytes here are exactly what the daemon checks.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

const (
	txVersion    = 1
	txDeploy     = 0x08 // deploys a contract
	txCall       = 0x09 // calls a contract
	gasPriceUnit = 1000 // gas price is in millisats: 1000 make one sat
	minGasPrice  = 1    // lowest gas price the daemon accepts (millisats per gas)
	maxGas       = 1_000_000
)

// Output pays Amount sats to the owner of the 32-byte x-only key Owner.
type Output struct {
	Owner  []byte
	Amount uint64
}

// EVMTx is a TxDeploy (To all zero, Data = init code) or a TxCall (To = contract, Data = calldata).
type EVMTx struct {
	Type     byte
	Inputs   [][]byte // 32-byte VTXO IDs that pay for gas
	Outputs  []Output // change back to yourself
	PubKey   []byte   // your 32-byte x-only public key
	Nonce    uint64
	GasLimit uint64
	GasPrice uint64
	To       [20]byte
	Data     []byte
	Sig      []byte // 64-byte BIP-340 signature
}

// encode serializes the tx. With forSigning the signature is left out:
// that is the preimage the signature commits to.
func (tx *EVMTx) encode(forSigning bool) []byte {
	var b bytes.Buffer
	u16 := func(v int) { binary.Write(&b, binary.BigEndian, uint16(v)) }
	u32 := func(v int) { binary.Write(&b, binary.BigEndian, uint32(v)) }
	u64 := func(v uint64) { binary.Write(&b, binary.BigEndian, v) }

	b.WriteByte(txVersion)
	b.WriteByte(tx.Type)
	u16(len(tx.Inputs))
	for _, id := range tx.Inputs {
		b.Write(id)
		b.Write(make([]byte, 32)) // L1 txid: unused by EVM txs
		u32(0)                    // L1 vout
		u64(0)                    // L1 value
		u16(0)                    // empty sigScript
	}
	u16(len(tx.Outputs))
	for _, o := range tx.Outputs {
		u16(len(o.Owner))
		b.Write(o.Owner)
		u64(o.Amount)
		u16(0) // empty script
	}
	sig := tx.Sig
	if forSigning {
		sig = nil
	}
	u64(0) // Fee: always 0, gas pays instead
	u64(tx.Nonce)
	u16(len(tx.PubKey))
	b.Write(tx.PubKey)
	u16(len(sig))
	b.Write(sig)
	u32(0) // empty PSBT payload
	u64(tx.GasLimit)
	u64(tx.GasPrice)
	if len(tx.Data) > 0 || tx.To != [20]byte{} {
		b.Write(tx.To[:])
		u32(len(tx.Data))
		b.Write(tx.Data)
	}
	return b.Bytes()
}

// sigHash is the 32-byte digest you sign.
func (tx *EVMTx) sigHash() []byte {
	h := sha256.Sum256(tx.encode(true))
	return h[:]
}

// Hex is what you broadcast; Hash is the tx hash receipts and the explorer use.
func (tx *EVMTx) Hex() string { return hex.EncodeToString(tx.encode(false)) }
func (tx *EVMTx) Hash() string {
	h := sha256.Sum256(tx.encode(false))
	return hex.EncodeToString(h[:])
}

// feeSats is the most this tx can spend on gas, rounded up like the daemon.
func feeSats(gasLimit, gasPrice uint64) uint64 {
	return (gasLimit*gasPrice + gasPriceUnit - 1) / gasPriceUnit
}
