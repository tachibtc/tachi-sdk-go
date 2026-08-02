package tachi

// TxStatus mirrors the ABCI tx_result {code, log} pair. For pending txs the
// daemon reports code=0, log="pending"; for committed txs the values come
// from FinalizeBlock's ExecTxResult.
type TxStatus struct {
	Code uint32 `json:"code"`
	Log  string `json:"log"`
}

// TxVin is a hex-encoded view of a TachiTx VTXOInput.
type TxVin struct {
	VTXOID    string `json:"vtxo_id"` // 64-char hex
	Txid      string `json:"txid"`    // 64-char hex
	Vout      uint32 `json:"vout"`
	ValueSats int64  `json:"value_sats"`
	SigScript string `json:"sig_script"` // hex
}

// TxVout is a hex-encoded view of a TachiTx VTXOOutput.
type TxVout struct {
	Owner  string `json:"owner"` // hex pubkey
	Amount int64  `json:"amount"`
	Script string `json:"script"` // hex
}

// HATProofResponse is the hex-encoded view of a hat.HAT commitment.
type HATProofResponse struct {
	VTXOID       string `json:"vtxo_id"`
	BTCTimestamp uint32 `json:"btc_timestamp"`
	BTCHeight    uint32 `json:"btc_height"`
	Proof        string `json:"proof"`
}

// ListTransactionItem is a summary of a transaction in a list/block/address
// view — shared by TxService.List, BlockService.Get, AddressService.
// Transactions, and AddressService.Mempool.
type ListTransactionItem struct {
	TxHash    string   `json:"tx_hash"`
	Type      string   `json:"type"`
	State     string   `json:"state"`
	Epoch     *uint32  `json:"epoch,omitempty"`
	Height    int64    `json:"height"`
	BlockHash string   `json:"block_hash,omitempty"`
	Time      *int64   `json:"time,omitempty"`
	Size      int      `json:"size"`
	IsSegwit  bool     `json:"is_segwit"`
	Weight    int      `json:"weight"`
	VSize     int      `json:"vsize"`
	Fee       int64    `json:"fee"`
	Vin       []TxVin  `json:"vin"`
	Vout      []TxVout `json:"vout"`
	// Direction is "sent" or "received", populated only by address-scoped
	// endpoints (AddressService.Transactions, AddressService.Mempool).
	Direction string            `json:"direction,omitempty"`
	HAT       *HATProofResponse `json:"hat,omitempty"`
	HasRIP    bool              `json:"has_rip"`
}
