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
	State     string            `json:"state"`
	Status    TxStatus          `json:"status"`
	Epoch     *uint32           `json:"epoch"`
	Vin       []TxVin           `json:"vin"`
	Vout      []TxVout          `json:"vout"`
	Size      int               `json:"size"`
	IsSegwit  bool              `json:"is_segwit"`
	Weight    int               `json:"weight"`
	VSize     int               `json:"vsize"`
	Version   uint8             `json:"version"`
	Hex       string            `json:"hex"`
	BlockHash *string           `json:"blockhash"`
	Time      *int64            `json:"time"`
	HAT       *HATProofResponse `json:"hat,omitempty"`
	// RIP is the raw JSON-encoded recursive inclusion proof; populated only
	// when requested via TxOptions.RIP. Decode with json.Unmarshal into a
	// caller-defined struct.
	RIP json.RawMessage `json:"rip,omitempty"`
}

// ListTransactionsResponse is returned by TxService.List. Pagination is
// height-cursor based: pass NextBeforeHeight as the BeforeHeight of the
// next call; nil once genesis is reached.
type ListTransactionsResponse struct {
	Transactions      []ListTransactionItem `json:"transactions"`
	PageSize          int                   `json:"page_size"`
	ScannedFromHeight int64                 `json:"scanned_from_height"`
	ScannedToHeight   int64                 `json:"scanned_to_height"`
	NextBeforeHeight  *int64                `json:"next_before_height,omitempty"`
}

// GetRawTransactionResponse is returned by TxService.Raw.
type GetRawTransactionResponse struct {
	TxHash string `json:"txHash"`
	Hex    string `json:"hex"`
}

// TxDecodeResponse is returned by TxService.Decode.
type TxDecodeResponse struct {
	TxHash   string   `json:"tx_hash"`
	Type     string   `json:"type"`
	Version  uint8    `json:"version"`
	Fee      int64    `json:"fee"`
	Nonce    uint64   `json:"nonce"`
	Size     int      `json:"size"`
	IsSegwit bool     `json:"is_segwit"`
	Weight   int      `json:"weight"`
	VSize    int      `json:"vsize"`
	Vin      []TxVin  `json:"vin"`
	Vout     []TxVout `json:"vout"`
	PubKey   string   `json:"pubkey"`
}

// TxValidateResponse is returned by TxService.Validate.
type TxValidateResponse struct {
	Valid bool   `json:"valid"`
	Code  uint32 `json:"code"`
	Log   string `json:"log"`
}

// MempoolResponse is returned by TxService.Mempool.
type MempoolResponse struct {
	Transactions []ListTransactionItem `json:"transactions"`
	Count        int                   `json:"count"`
}

// FeeEstimateResponse is returned by TxService.FeeEstimate.
type FeeEstimateResponse struct {
	MinFeeSat         int64 `json:"min_fee_sat"`
	AvgFeeSat         int64 `json:"avg_fee_sat"`
	RecommendedFeeSat int64 `json:"recommended_fee_sat"`
}

// TxOptions configures TxService.Get's optional proof attachments.
type TxOptions struct {
	// HAT attaches the HAT commitment for the tx's first spent VTXO.
	HAT bool
	// RIP attaches a RIP inclusion proof; requires OriginEpoch and FinalEpoch.
	RIP         bool
	OriginEpoch uint32
	FinalEpoch  uint32
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
