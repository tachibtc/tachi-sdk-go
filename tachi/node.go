package tachi

import (
	"context"
	"encoding/json"
	"net/url"
)

// NodeService groups liveness, node-info, and raw CometBFT-proxy endpoints.
type NodeService service

// HealthResponse is returned by NodeService.Health.
type HealthResponse struct {
	Status     string `json:"status" example:"ok"`
	Validators int    `json:"validators" example:"3"`
}

// NodeInfoResponse is returned by NodeService.Info.
type NodeInfoResponse struct {
	Version           string `json:"version"`
	ChainID           string `json:"chain_id"`
	NodeID            string `json:"node_id"`
	Network           string `json:"network"`
	Moniker           string `json:"moniker"`
	SyncStatus        string `json:"sync_status"`
	LatestBlockHeight int64  `json:"latest_block_height"`
	LatestBlockTime   *int64 `json:"latest_block_time,omitempty"`
	EpochBlocks       int64  `json:"epoch_blocks"`
	Peers             int    `json:"peers"`
}

// CometRPCResponse is the raw JSON-RPC 2.0 envelope CometBFT wraps every
// forwarded result in. Shape of Result varies by endpoint.
type CometRPCResponse struct {
	JSONRPC string          `json:"jsonrpc" example:"2.0"`
	ID      int             `json:"id" example:"1"`
	Result  json.RawMessage `json:"result" swaggertype:"object"`
}

// Health is a liveness probe that returns the daemon status and validator
// count.
func (s *NodeService) Health(ctx context.Context) (*HealthResponse, *Response, error) {
	var out HealthResponse
	resp, err := s.client.get(ctx, "health", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Info returns extended node information: version, chain ID, sync status,
// epoch config, and peer count.
func (s *NodeService) Info(ctx context.Context) (*NodeInfoResponse, *Response, error) {
	var out NodeInfoResponse
	resp, err := s.client.get(ctx, "tachi_nodeInfo", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Status forwards to CometBFT's status endpoint: node info, sync status,
// and the latest block height.
func (s *NodeService) Status(ctx context.Context) (*CometRPCResponse, *Response, error) {
	var out CometRPCResponse
	resp, err := s.client.get(ctx, "tachi_status", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// NetInfo forwards to CometBFT's net_info endpoint: listening addresses,
// connected peer count, and per-peer connection info.
func (s *NodeService) NetInfo(ctx context.Context) (*CometRPCResponse, *Response, error) {
	var out CometRPCResponse
	resp, err := s.client.get(ctx, "tachi_netInfo", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ConsensusState forwards to CometBFT's consensus_state endpoint: current
// round, step, and proposer info.
func (s *NodeService) ConsensusState(ctx context.Context) (*CometRPCResponse, *Response, error) {
	var out CometRPCResponse
	resp, err := s.client.get(ctx, "tachi_consensusState", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// ValidatorsPower forwards to CometBFT's validators endpoint: the current
// validator set with voting power.
func (s *NodeService) ValidatorsPower(ctx context.Context) (*CometRPCResponse, *Response, error) {
	var out CometRPCResponse
	resp, err := s.client.get(ctx, "tachi_validatorsPower", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// QueryOptions configures NodeService.Query's optional parameters.
type QueryOptions struct {
	// DataHex is hex-encoded query data; its meaning depends on Path (see
	// the daemon's RPC.md "ABCI query paths" table, e.g. "vtxo", "utxos",
	// "epoch", "rip").
	DataHex string
	// Height is the block height to query at ("" or "0" for latest).
	Height string
}

// Query forwards a raw query to CometBFT's abci_query endpoint; the ABCI
// application interprets path and the optional data/height. See the
// daemon's RPC.md for the full set of supported query paths (height,
// nonce, supply, vtxo, utxos, all_vtxos, locked_vtxos, current_epoch,
// epoch_root, epoch, epochs, rip, vaults_by_user).
func (s *NodeService) Query(ctx context.Context, path string, opts *QueryOptions) (*CometRPCResponse, *Response, error) {
	q := url.Values{"path": {path}}
	if opts != nil {
		setIf(q, "data", opts.DataHex)
		setIf(q, "height", opts.Height)
	}
	var out CometRPCResponse
	resp, err := s.client.get(ctx, "tachi_query", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
