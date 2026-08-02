package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// BlockService groups block-lookup endpoints, including the Tachi-chain
// analogues of bitcoind's getblockhash/getblockheader/getblock.
type BlockService service

// BlockResponse is returned by BlockService.Get, BlockService.ByHeight, and
// BlockService.ByHash.
type BlockResponse struct {
	Height       int64                 `json:"height"`
	Hash         string                `json:"hash"`
	Time         *int64                `json:"time,omitempty"`
	Epoch        uint32                `json:"epoch"`
	TxCount      int                   `json:"tx_count"`
	Transactions []ListTransactionItem `json:"transactions"`
}

// Get returns block metadata and all decoded transactions at height.
func (s *BlockService) Get(ctx context.Context, height int64) (*BlockResponse, *Response, error) {
	var out BlockResponse
	q := url.Values{"height": {strconv.FormatInt(height, 10)}}
	resp, err := s.client.get(ctx, "tachi_block", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// BlockSummary is a lightweight block for the BlockService.List view.
type BlockSummary struct {
	Height  int64  `json:"height"`
	Hash    string `json:"hash"`
	Time    *int64 `json:"time,omitempty"`
	TxCount int    `json:"tx_count"`
	Epoch   uint32 `json:"epoch"`
}

// ListBlocksResponse is returned by BlockService.List.
type ListBlocksResponse struct {
	Blocks     []BlockSummary `json:"blocks"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// List returns a paginated list of blocks from latest to oldest with tx
// counts. page defaults to 1; pageSize defaults to 50 (max 100). Pass 0 for
// either to use the default.
func (s *BlockService) List(ctx context.Context, page, pageSize int) (*ListBlocksResponse, *Response, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	var out ListBlocksResponse
	resp, err := s.client.get(ctx, "tachi_listBlocks", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// GetBlockHashResponse is returned by BlockService.Hash.
type GetBlockHashResponse struct {
	Height int64  `json:"height"`
	Hash   string `json:"hash"`
}

// Hash returns the Tachi chain's block hash at height — the Tachi-chain
// analogue of bitcoind's getblockhash (not a bitcoin RPC).
func (s *BlockService) Hash(ctx context.Context, height int64) (*GetBlockHashResponse, *Response, error) {
	var out GetBlockHashResponse
	q := url.Values{"height": {strconv.FormatInt(height, 10)}}
	resp, err := s.client.get(ctx, "tachi_getBlockHash", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// GetBlockHeaderResponse is returned by BlockService.HeaderByHeight and
// BlockService.HeaderByHash.
type GetBlockHeaderResponse struct {
	Height   int64  `json:"height"`
	Hash     string `json:"hash"`
	PrevHash string `json:"prev_hash"`
	Time     int64  `json:"time"`
	Epoch    uint32 `json:"epoch"`
}

// HeaderByHeight returns Tachi chain block header metadata (no transaction
// list) for height — the Tachi-chain analogue of bitcoind's getblockheader.
func (s *BlockService) HeaderByHeight(ctx context.Context, height int64) (*GetBlockHeaderResponse, *Response, error) {
	var out GetBlockHeaderResponse
	q := url.Values{"height": {strconv.FormatInt(height, 10)}}
	resp, err := s.client.get(ctx, "tachi_getBlockHeader", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// HeaderByHash returns Tachi chain block header metadata (no transaction
// list) for the block with the given hex hash.
func (s *BlockService) HeaderByHash(ctx context.Context, hash string) (*GetBlockHeaderResponse, *Response, error) {
	var out GetBlockHeaderResponse
	q := url.Values{"hash": {hash}}
	resp, err := s.client.get(ctx, "tachi_getBlockHeader", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ByHeight returns full Tachi chain block detail (header + decoded
// transactions) for height — the Tachi-chain analogue of bitcoind's
// getblock. Same response shape as Get.
func (s *BlockService) ByHeight(ctx context.Context, height int64) (*BlockResponse, *Response, error) {
	var out BlockResponse
	q := url.Values{"height": {strconv.FormatInt(height, 10)}}
	resp, err := s.client.get(ctx, "tachi_getBlock", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ByHash returns full Tachi chain block detail (header + decoded
// transactions) for the block with the given hex hash.
func (s *BlockService) ByHash(ctx context.Context, hash string) (*BlockResponse, *Response, error) {
	var out BlockResponse
	q := url.Values{"hash": {hash}}
	resp, err := s.client.get(ctx, "tachi_getBlock", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
