package tachi

// TxStatus mirrors the ABCI tx_result {code, log} pair. For pending txs the
// daemon reports code=0, log="pending"; for committed txs the values come
// from FinalizeBlock's ExecTxResult.
type TxStatus struct {
	// Code is the ABCI execution code; 0 means success, non-zero means failure.
	Code uint32 `json:"code"`
	// Log is the ABCI execution log message; "pending" for mempool txs.
	Log string `json:"log"`
}

// TxVin is a hex-encoded view of a TachiTx VTXOInput.
type TxVin struct {
	// VTXOID is the hex-encoded 32-byte identifier of the VTXO being spent.
	VTXOID string `json:"vtxo_id"` // 64-char hex
	// Txid is the hex-encoded 32-byte parent transaction hash that created the VTXO.
	Txid string `json:"txid"` // 64-char hex
	// Vout is the zero-based output index within the parent transaction.
	Vout uint32 `json:"vout"`
	// ValueSats is the satoshi value committed by the input from the spent VTXO.
	ValueSats int64 `json:"value_sats"`
	// SigScript is the hex-encoded signature script authorizing the spend.
	SigScript string `json:"sig_script"` // hex
}

// TxVout is a hex-encoded view of a TachiTx VTXOOutput.
type TxVout struct {
	// Owner is the hex-encoded public key authorized to spend this output's VTXO.
	Owner string `json:"owner"` // hex pubkey
	// Amount is the value of this output in satoshis.
	Amount int64 `json:"amount"`
	// Script is the hex-encoded locking script attached to the output.
	Script string `json:"script"` // hex
}

// HATProofResponse is the hex-encoded view of a hat.HAT commitment.
type HATProofResponse struct {
	// VTXOID is the hex-encoded 32-byte VTXO identifier the HAT commits to.
	VTXOID string `json:"vtxo_id"`
	// BTCTimestamp is the Bitcoin L1 block timestamp bound into the proof preimage.
	BTCTimestamp uint32 `json:"btc_timestamp"`
	// BTCHeight is the Bitcoin L1 block height bound into the proof preimage.
	BTCHeight uint32 `json:"btc_height"`
	// Proof is the hex-encoded SHA256d commitment over the raw finalized PSBT payload.
	Proof string `json:"proof"`
}

// ListTransactionItem is a summary of a transaction in a list/block/address
// view — shared by TxService.List, BlockService.Get, AddressService.
// Transactions, and AddressService.Mempool.
type ListTransactionItem struct {
	// TxHash is the hex-encoded tmhash of the raw transaction bytes.
	TxHash string `json:"tx_hash"`
	// Type is transfer/deposit/withdraw/lock/unlock/vault_open/vault_state_advance/vault_close/vault_breach/unknown.
	Type string `json:"type"`
	// State is "committed" for confirmed txs, "pending" for mempool entries.
	State string `json:"state"`
	// Epoch is the epoch ID the committing block belongs to; nil for mempool/pending txs.
	Epoch *uint32 `json:"epoch,omitempty"`
	// Height is the committing block height, or 0 for mempool entries.
	Height int64 `json:"height"`
	// BlockHash is the hex-encoded committing block hash; empty for mempool entries.
	BlockHash string `json:"block_hash,omitempty"`
	// Time is the unix-seconds timestamp of the committing block; nil for mempool entries.
	Time *int64 `json:"time,omitempty"`
	// Size is the byte length of the raw transaction bytes.
	Size int `json:"size"`
	// IsSegwit reports whether any input carries witness data (a non-empty SigScript).
	IsSegwit bool `json:"is_segwit"`
	// Weight is the Bitcoin-style segwit weight: 3*baseSize + totalSize.
	Weight int `json:"weight"`
	// VSize is the Bitcoin-style virtual size: ceil(Weight/4).
	VSize int `json:"vsize"`
	// Fee is the fee paid by the transaction in satoshis.
	Fee int64 `json:"fee"`
	// Vin lists the VTXO inputs being spent by this transaction.
	Vin []TxVin `json:"vin"`
	// Vout lists the VTXO outputs produced by this transaction.
	Vout []TxVout `json:"vout"`
	// Direction is "sent" or "received", populated only by address-scoped
	// endpoints (AddressService.Transactions, AddressService.Mempool).
	Direction string `json:"direction,omitempty"`
	// HAT is the HAT commitment for the VTXO spent by this tx's first input; omitted when there is none or the proof fetch failed.
	HAT *HATProofResponse `json:"hat,omitempty"`
	// HasRIP reports whether a RIP inclusion proof is available for this tx's HAT without generating it.
	HasRIP bool `json:"has_rip"`
}
