package tachi

import "context"

// SignService wraps the cooperative-refund threshold-signing ceremony.
type SignService service

// RefundTx is the PSBT-shaped refund transaction SignService.Transaction
// accepts and returns. Field naming mirrors the daemon's reference schema
// (camelCase).
type RefundTx struct {
	// Version is the tx version (2 for CSV-aware refunds).
	Version uint32 `json:"version"`
	// Locktime is nLockTime (0 for a cooperative refund).
	Locktime uint32 `json:"locktime"`
	// Inputs holds exactly one entry: the funding UTXO.
	Inputs []RefundInput `json:"inputs"`
	// Outputs holds the refund outputs; Outputs[0] is to_local, the rest are extras.
	Outputs []RefundOutput `json:"outputs"`
	// UserSig is the vault owner's 64-byte BIP-340 signature (hex) over the
	// cooperative-leaf sighash. Required.
	UserSig string `json:"userSig,omitempty"`
}

// RefundInput is the single funding input spent via the vault cooperative
// leaf. TapScriptSig is empty on the way in and carries the collected
// quorum partials on the way out.
type RefundInput struct {
	// Prevout is the funding outpoint being spent.
	Prevout RefundPrevout `json:"prevout"`
	// Sequence is nSequence (0 for a cooperative refund; no CSV).
	Sequence uint32 `json:"sequence"`
	// WitnessUtxo is the vault P2TR output this input commits to.
	WitnessUtxo RefundWitnessUtxo `json:"witnessUtxo"`
	// TapLeafScript holds exactly one entry: the vault cooperative leaf.
	TapLeafScript []RefundTapLeaf `json:"tapLeafScript"`
	// TapInternalKey is the x-only(32) hex internal key.
	TapInternalKey string `json:"tapInternalKey"`
	// SighashType must be SIGHASH_DEFAULT (0x00).
	SighashType uint32 `json:"sighashType"`
	// TapScriptSig is empty in; the collected quorum partials out.
	TapScriptSig []RefundTapSig `json:"tapScriptSig,omitempty"`
}

// RefundTapSig is a PSBT tapscript partial signature.
type RefundTapSig struct {
	// Pubkey is the signer's x-only(32) hex public key.
	Pubkey string `json:"pubkey"`
	// LeafHash is the tapleaf hash hex the signature is bound to.
	LeafHash string `json:"leafHash"`
	// Signature is the 64-byte BIP-340 signature hex.
	Signature string `json:"signature"`
}

// RefundPrevout identifies the funding outpoint. Hash is the display/BE
// txid hex.
type RefundPrevout struct {
	// Hash is the display-order (big-endian) txid hex.
	Hash string `json:"hash"`
	// Index is the funding output index.
	Index uint32 `json:"index"`
}

// RefundWitnessUtxo is the prevout the input sighash commits to (the vault
// P2TR output).
type RefundWitnessUtxo struct {
	// Value is the prevout amount in satoshis.
	Value int64 `json:"value"`
	// Script is the hex-encoded prevout locking script.
	Script string `json:"script"`
}

// RefundTapLeaf is the tapscript spend path (the vault cooperative leaf).
type RefundTapLeaf struct {
	// LeafVersion is the tapscript leaf version (0xc0 for the vault cooperative leaf).
	LeafVersion uint8 `json:"leafVersion"`
	// Script is the hex-encoded tapleaf script.
	Script string `json:"script"`
	// ControlBlock is the hex-encoded control block proving the leaf's inclusion in the taproot tree.
	ControlBlock string `json:"controlBlock"`
}

// RefundOutput is a refund output. Outputs[0] must be the canonical
// to_local P2TR.
type RefundOutput struct {
	// Value is the output amount in satoshis.
	Value int64 `json:"value"`
	// Script is the hex-encoded output locking script.
	Script string `json:"script"`
}

// SignTransactionResponse is returned by SignService.Transaction.
type SignTransactionResponse struct {
	// Refund is the refund transaction with the quorum partial signatures attached.
	Refund *RefundTx `json:"refund"`
	// Signatures is the number of quorum partial signatures collected.
	Signatures int `json:"signatures" example:"5"`
}

// Transaction fans a PSBT-shaped refund transaction (which must carry the
// owner's UserSig over the cooperative-leaf sighash) out to the vault's
// signing quorum and returns it with the collected tapScriptSig partials
// attached. The caller adds its own user signature and finalizes.
//
// Returns a 503 *ErrorResponse if refund signing is disabled on the
// daemon, or a 504 *ErrorResponse if the signing threshold wasn't reached
// in time.
func (s *SignService) Transaction(ctx context.Context, tx *RefundTx) (*SignTransactionResponse, *Response, error) {
	var out SignTransactionResponse
	resp, err := s.client.post(ctx, "tachi_signTransaction", tx, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
