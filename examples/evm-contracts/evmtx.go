// Building, encoding and signing Tachi EVM transactions (TxDeploy, TxCall).
// The wire format and signing digest match the daemon's types.EncodeTx and
// types.SigHash, so the bytes here are exactly what the daemon checks.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
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

// validate checks the limits the daemon enforces, so encode's length fields can't overflow.
func (tx *EVMTx) validate() error {
	switch {
	case len(tx.PubKey) != 32:
		return errors.New("pubkey must be a 32-byte x-only key")
	case len(tx.Inputs) == 0 || len(tx.Inputs) > math.MaxUint16:
		return errors.New("need 1 to 65535 inputs")
	case len(tx.Outputs) > math.MaxUint16:
		return errors.New("too many outputs")
	case len(tx.Data) > 0xffffff:
		return errors.New("data exceeds 16 MiB")
	case tx.GasLimit == 0 || tx.GasLimit > maxGas:
		return errors.New("gas limit must be 1 to 1,000,000")
	}
	for _, id := range tx.Inputs {
		if len(id) != 32 {
			return errors.New("input VTXO IDs must be 32 bytes")
		}
	}
	for _, o := range tx.Outputs {
		if len(o.Owner) != 32 || o.Amount == 0 || o.Amount > math.MaxInt64 {
			return errors.New("outputs need a 32-byte owner and a positive amount")
		}
	}
	return nil
}

// encode serializes the tx; call validate first. With forSigning the
// signature is left out: that is the preimage the signature commits to.
func (tx *EVMTx) encode(forSigning bool) []byte {
	be := binary.BigEndian
	// Lengths below are bounded by validate (and a signature is 64 bytes).
	u16 := func(b []byte, n int) []byte { return be.AppendUint16(b, uint16(n)) } // #nosec G115 -- bounded by validate
	u32 := func(b []byte, n int) []byte { return be.AppendUint32(b, uint32(n)) } // #nosec G115 -- bounded by validate

	b := []byte{txVersion, tx.Type}
	b = u16(b, len(tx.Inputs))
	for _, id := range tx.Inputs {
		b = append(b, id...)
		b = append(b, make([]byte, 32)...) // L1 txid: unused by EVM txs
		b = be.AppendUint32(b, 0)          // L1 vout
		b = be.AppendUint64(b, 0)          // L1 value
		b = u16(b, 0)                      // empty sigScript
	}
	b = u16(b, len(tx.Outputs))
	for _, o := range tx.Outputs {
		b = u16(b, len(o.Owner))
		b = append(b, o.Owner...)
		b = be.AppendUint64(b, o.Amount)
		b = u16(b, 0) // empty script
	}
	sig := tx.Sig
	if forSigning {
		sig = nil
	}
	b = be.AppendUint64(b, 0) // Fee: always 0, gas pays instead
	b = be.AppendUint64(b, tx.Nonce)
	b = u16(b, len(tx.PubKey))
	b = append(b, tx.PubKey...)
	b = u16(b, len(sig))
	b = append(b, sig...)
	b = u32(b, 0) // empty PSBT payload
	b = be.AppendUint64(b, tx.GasLimit)
	b = be.AppendUint64(b, tx.GasPrice)
	if len(tx.Data) > 0 || tx.To != [20]byte{} {
		b = append(b, tx.To[:]...)
		b = u32(b, len(tx.Data))
		b = append(b, tx.Data...)
	}
	return b
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
