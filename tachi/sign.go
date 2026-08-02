package tachi

import "context"

// SignService wraps the cooperative-refund threshold-signing ceremony.
type SignService service

// RefundTx is the PSBT-shaped refund transaction SignService.Transaction
// accepts and returns. Field naming mirrors the daemon's reference schema
// (camelCase).
type RefundTx struct {
	Version  uint32         `json:"version"`  // tx version (2 for CSV-aware refunds)
	Locktime uint32         `json:"locktime"` // nLockTime (0 for a cooperative refund)
	Inputs   []RefundInput  `json:"inputs"`   // exactly one (the funding UTXO)
	Outputs  []RefundOutput `json:"outputs"`  // outputs[0] = to_local, rest = extras
	// UserSig is the vault owner's 64-byte BIP-340 signature (hex) over the
	// cooperative-leaf sighash. Required.
	UserSig string `json:"userSig,omitempty"`
}

// RefundInput is the single funding input spent via the vault cooperative
// leaf. TapScriptSig is empty on the way in and carries the collected
// quorum partials on the way out.
type RefundInput struct {
	Prevout        RefundPrevout     `json:"prevout"`
	Sequence       uint32            `json:"sequence"`
	WitnessUtxo    RefundWitnessUtxo `json:"witnessUtxo"`
	TapLeafScript  []RefundTapLeaf   `json:"tapLeafScript"` // exactly one entry: the vault cooperative leaf
	TapInternalKey string            `json:"tapInternalKey"`
	SighashType    uint32            `json:"sighashType"` // must be SIGHASH_DEFAULT (0x00)
	TapScriptSig   []RefundTapSig    `json:"tapScriptSig,omitempty"`
}

// RefundTapSig is a PSBT tapscript partial signature.
type RefundTapSig struct {
	Pubkey    string `json:"pubkey"`    // x-only(32) hex
	LeafHash  string `json:"leafHash"`  // tapleaf hash hex
	Signature string `json:"signature"` // 64-byte BIP-340 hex
}

// RefundPrevout identifies the funding outpoint. Hash is the display/BE
// txid hex.
type RefundPrevout struct {
	Hash  string `json:"hash"`
	Index uint32 `json:"index"`
}

// RefundWitnessUtxo is the prevout the input sighash commits to (the vault
// P2TR output).
type RefundWitnessUtxo struct {
	Value  int64  `json:"value"`  // sats
	Script string `json:"script"` // hex
}

// RefundTapLeaf is the tapscript spend path (the vault cooperative leaf).
type RefundTapLeaf struct {
	LeafVersion  uint8  `json:"leafVersion"`
	Script       string `json:"script"`       // hex
	ControlBlock string `json:"controlBlock"` // hex
}

// RefundOutput is a refund output. Outputs[0] must be the canonical
// to_local P2TR.
type RefundOutput struct {
	Value  int64  `json:"value"`  // sats
	Script string `json:"script"` // hex
}

// SignTransactionResponse is returned by SignService.Transaction.
type SignTransactionResponse struct {
	Refund     *RefundTx `json:"refund"`
	Signatures int       `json:"signatures" example:"5"`
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
