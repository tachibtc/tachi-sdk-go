package tachi

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// TxService groups transaction endpoints: lookup, decode/validate,
// broadcast, mempool, and fee estimation.
type TxService service

// GetTransactionResponse is returned by TxService.Get.
type GetTransactionResponse struct {
	// TxHash is the hex-encoded tmhash of the raw transaction bytes
	// (witness-included).
	TxHash string `json:"txHash"`
	// TxID is the segwit-style transaction id (witness stripped).
	TxID string `json:"txid"`
	// Type is transfer/deposit/withdraw/lock/unlock/vault_open/
	// vault_state_advance/vault_close/vault_breach/unknown.
	Type string `json:"type"`
	// State is committed/pending/failed.
	State string `json:"state"`
	// Status is the raw ABCI {code, log} pair describing tx execution.
	Status TxStatus `json:"status"`
	// Epoch is the epoch ID the committing block belongs to; nil for mempool/pending txs.
	Epoch *uint32 `json:"epoch"`
	// Vin lists the VTXO inputs being spent by this transaction.
	Vin []TxVin `json:"vin"`
	// Vout lists the VTXO outputs produced by this transaction.
	Vout []TxVout `json:"vout"`
	// Size is the byte length of the raw transaction bytes.
	Size int `json:"size"`
	// IsSegwit reports whether any input carries witness data (a non-empty SigScript).
	IsSegwit bool `json:"is_segwit"`
	// Weight is the Bitcoin-style segwit weight: 3*baseSize + totalSize.
	Weight int `json:"weight"`
	// VSize is the Bitcoin-style virtual size: ceil(Weight/4).
	VSize int `json:"vsize"`
	// Version is the TachiTx protocol version byte.
	Version uint8 `json:"version"`
	// Hex is the hex-encoded raw transaction bytes.
	Hex string `json:"hex"`
	// BlockHash is the hex-encoded committing block hash; nil for mempool/pending txs.
	BlockHash *string `json:"blockhash"`
	// Time is the unix-seconds timestamp of the committing block; nil for mempool/pending txs.
	Time *int64 `json:"time"`
	// HAT is the HAT commitment for the VTXO spent by this tx's first input; populated only via TxOptions.HAT.
	HAT *HATProofResponse `json:"hat,omitempty"`
	// RIP is the raw JSON-encoded recursive inclusion proof; populated only
	// when requested via TxOptions.RIP. Decode with json.Unmarshal into a
	// caller-defined struct.
	RIP json.RawMessage `json:"rip,omitempty"`
}

// ListTransactionsResponse is returned by TxService.List. Pagination is
// height-cursor based: pass NextBeforeHeight as the BeforeHeight of the
// next call; nil once genesis is reached.
type ListTransactionsResponse struct {
	// Transactions is the page of committed transactions, ordered newest-first by block height.
	Transactions []ListTransactionItem `json:"transactions"`
	// PageSize is the maximum number of transactions the caller asked the scanner to collect.
	PageSize int `json:"page_size"`
	// ScannedFromHeight is the highest block height included in this scan (inclusive).
	ScannedFromHeight int64 `json:"scanned_from_height"`
	// ScannedToHeight is the lowest block height included in this scan (inclusive).
	ScannedToHeight int64 `json:"scanned_to_height"`
	// NextBeforeHeight is the cursor to pass as BeforeHeight on the next call; nil once genesis is reached.
	NextBeforeHeight *int64 `json:"next_before_height,omitempty"`
}

// GetRawTransactionResponse is returned by TxService.Raw.
type GetRawTransactionResponse struct {
	// TxHash is the hex-encoded tmhash of the raw transaction bytes.
	TxHash string `json:"txHash"`
	// Hex is the hex-encoded raw transaction bytes.
	Hex string `json:"hex"`
}

// TxDecodeResponse is returned by TxService.Decode.
type TxDecodeResponse struct {
	// TxHash is the hex-encoded tmhash of the raw transaction bytes.
	TxHash string `json:"tx_hash"`
	// Type is transfer/deposit/withdraw/lock/unlock/vault_open/
	// vault_state_advance/vault_close/vault_breach/unknown.
	Type string `json:"type"`
	// Version is the TachiTx protocol version byte.
	Version uint8 `json:"version"`
	// Fee is the fee paid by the transaction in satoshis.
	Fee int64 `json:"fee"`
	// Nonce is the sender's transaction nonce as committed by the signer.
	Nonce uint64 `json:"nonce"`
	// Size is the byte length of the decoded raw transaction.
	Size int `json:"size"`
	// IsSegwit reports whether any input carries witness data (a non-empty SigScript).
	IsSegwit bool `json:"is_segwit"`
	// Weight is the Bitcoin-style segwit weight: 3*baseSize + totalSize.
	Weight int `json:"weight"`
	// VSize is the Bitcoin-style virtual size: ceil(Weight/4).
	VSize int `json:"vsize"`
	// Vin lists the VTXO inputs being spent by this transaction.
	Vin []TxVin `json:"vin"`
	// Vout lists the VTXO outputs produced by this transaction.
	Vout []TxVout `json:"vout"`
	// PubKey is the hex-encoded signer public key authorizing the transaction.
	PubKey string `json:"pubkey"`
}

// TxValidateResponse is returned by TxService.Validate.
type TxValidateResponse struct {
	// Valid is true when CometBFT CheckTx returned code 0 (transaction is acceptable).
	Valid bool `json:"valid"`
	// Code is the raw ABCI CheckTx response code; non-zero indicates rejection.
	Code uint32 `json:"code"`
	// Log is the human-readable CheckTx log message explaining acceptance or rejection.
	Log string `json:"log"`
}

// MempoolResponse is returned by TxService.Mempool.
type MempoolResponse struct {
	// Transactions is the list of decoded pending transactions currently in the CometBFT unconfirmed mempool.
	Transactions []ListTransactionItem `json:"transactions"`
	// Count is the number of pending transactions returned.
	Count int `json:"count"`
}

// FeeEstimateResponse is returned by TxService.FeeEstimate.
type FeeEstimateResponse struct {
	// MinFeeSat is the minimum acceptable fee in satoshis (currently a flat floor of 1).
	MinFeeSat int64 `json:"min_fee_sat"`
	// AvgFeeSat is the mean fee in satoshis across recently scanned blocks.
	AvgFeeSat int64 `json:"avg_fee_sat"`
	// RecommendedFeeSat is the suggested fee to use for a new transaction, derived from recent activity.
	RecommendedFeeSat int64 `json:"recommended_fee_sat"`
}

// TxOptions configures TxService.Get's optional proof attachments.
type TxOptions struct {
	// HAT attaches the HAT commitment for the tx's first spent VTXO.
	HAT bool
	// RIP attaches a RIP inclusion proof; requires OriginEpoch and FinalEpoch.
	RIP bool
	// OriginEpoch is the epoch the HAT was inserted into (required when RIP is true).
	OriginEpoch uint32
	// FinalEpoch is the L1-settled epoch to chain the proof up to (required when RIP is true).
	FinalEpoch uint32
}

// Get returns a Tachi-decoded view of a transaction by its 40-char hex
// CometBFT hash. Looks up committed txs first, then falls back to the
// unconfirmed mempool.
func (s *TxService) Get(ctx context.Context, hash string, opts *TxOptions) (*GetTransactionResponse, *Response, error) {
	q := url.Values{"hash": {hash}}
	if opts != nil {
		if opts.HAT {
			q.Set("hat", "true")
		}
		if opts.RIP {
			q.Set("rip", "true")
			q.Set("origin_epoch", strconv.FormatUint(uint64(opts.OriginEpoch), 10))
			q.Set("final_epoch", strconv.FormatUint(uint64(opts.FinalEpoch), 10))
		}
	}
	var out GetTransactionResponse
	resp, err := s.client.get(ctx, "tachi_tx", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Raw returns just the raw hex-encoded bytes of a transaction by its
// 40-char hex CometBFT hash.
func (s *TxService) Raw(ctx context.Context, hash string) (*GetRawTransactionResponse, *Response, error) {
	var out GetRawTransactionResponse
	resp, err := s.client.get(ctx, "tachi_txRaw", url.Values{"hash": {hash}}, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ListTransactionsOptions configures TxService.List pagination.
type ListTransactionsOptions struct {
	// BeforeHeight is the exclusive upper-bound block height; zero means
	// "start from the chain tip".
	BeforeHeight int64
	// PageSize is the max transactions to return (default 50, max 100).
	PageSize int
}

// List returns committed transactions in descending height order,
// height-cursor paginated. Pass the previous response's NextBeforeHeight
// as opts.BeforeHeight to fetch the next page; when NextBeforeHeight is
// nil the scan has reached genesis.
func (s *TxService) List(ctx context.Context, opts *ListTransactionsOptions) (*ListTransactionsResponse, *Response, error) {
	q := url.Values{}
	if opts != nil {
		if opts.BeforeHeight > 0 {
			q.Set("before_height", strconv.FormatInt(opts.BeforeHeight, 10))
		}
		if opts.PageSize > 0 {
			q.Set("page_size", strconv.Itoa(opts.PageSize))
		}
	}
	var out ListTransactionsResponse
	resp, err := s.client.get(ctx, "tachi_listTransactions", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Mempool returns decoded pending transactions from the CometBFT
// unconfirmed mempool.
func (s *TxService) Mempool(ctx context.Context) (*MempoolResponse, *Response, error) {
	var out MempoolResponse
	resp, err := s.client.get(ctx, "tachi_mempool", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Decode decodes a raw hex-encoded Tachi transaction without broadcasting
// it.
func (s *TxService) Decode(ctx context.Context, hexTx string) (*TxDecodeResponse, *Response, error) {
	var out TxDecodeResponse
	body := struct {
		Hex string `json:"hex"`
	}{Hex: hexTx}
	resp, err := s.client.post(ctx, "tachi_txDecode", body, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Validate validates a hex-encoded transaction against current chain state
// via CometBFT CheckTx, without broadcasting.
func (s *TxService) Validate(ctx context.Context, hexTx string) (*TxValidateResponse, *Response, error) {
	var out TxValidateResponse
	body := struct {
		Hex string `json:"hex"`
	}{Hex: hexTx}
	resp, err := s.client.post(ctx, "tachi_txValidate", body, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// BroadcastSync forwards a hex-encoded transaction to CometBFT's
// broadcast_tx_sync, waiting for CheckTx to complete before returning.
func (s *TxService) BroadcastSync(ctx context.Context, hexTx string) (*CometRPCResponse, *Response, error) {
	var out CometRPCResponse
	body := struct {
		Tx string `json:"tx"`
	}{Tx: hexTx}
	resp, err := s.client.post(ctx, "tachi_txBroadcastSync", body, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// BroadcastAsync forwards a hex-encoded transaction to CometBFT's
// broadcast_tx_async, returning immediately without waiting for CheckTx.
func (s *TxService) BroadcastAsync(ctx context.Context, hexTx string) (*CometRPCResponse, *Response, error) {
	var out CometRPCResponse
	body := struct {
		Tx string `json:"tx"`
	}{Tx: hexTx}
	resp, err := s.client.post(ctx, "tachi_txBroadcastAsync", body, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// FeeEstimate returns the minimum, average, and recommended transaction
// fee based on recent confirmed transactions.
func (s *TxService) FeeEstimate(ctx context.Context) (*FeeEstimateResponse, *Response, error) {
	var out FeeEstimateResponse
	resp, err := s.client.get(ctx, "tachi_feeEstimate", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
