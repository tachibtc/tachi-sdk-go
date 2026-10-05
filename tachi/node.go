package tachi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

// NodeService groups liveness, node-info, and raw CometBFT-proxy endpoints.
type NodeService service

// HealthResponse is returned by NodeService.Health.
type HealthResponse struct {
	// Status is the liveness label; always "ok" when the daemon is serving requests.
	Status string `json:"status" example:"ok"`
	// Validators is the count of validators currently in the bootstrap registry.
	Validators int `json:"validators" example:"3"`
}

// ChainHealthResponse is returned by NodeService.ChainHealth.
type ChainHealthResponse struct {
	// Status is "ok", or "unhealthy" when any problem is listed.
	Status string `json:"status" example:"ok"`
	// Problems lists why the node is unhealthy; empty when healthy.
	Problems []string `json:"problems"`
	// Height is the last block this node committed (0 before the first).
	Height int64 `json:"height" example:"798295"`
	// LastBlockAgeSeconds is the time since this node last committed a block.
	LastBlockAgeSeconds float64 `json:"last_block_age_seconds" example:"4.2"`
	// MaxBlockAgeSeconds is the threshold for LastBlockAgeSeconds.
	MaxBlockAgeSeconds float64 `json:"max_block_age_seconds" example:"300"`
	// MemoryBytes is the memory the daemon's Go runtime has mapped from the OS.
	MemoryBytes uint64 `json:"memory_bytes" example:"3900000000"`
	// MaxMemoryBytes is the threshold for MemoryBytes; 0 = not checked.
	MaxMemoryBytes uint64 `json:"max_memory_bytes" example:"8000000000"`
}

// NodeInfoResponse is returned by NodeService.Info.
type NodeInfoResponse struct {
	// Version is the CometBFT node software version reported by /status.
	Version string `json:"version"`
	// ChainID identifies the chain the node is participating in.
	ChainID string `json:"chain_id"`
	// NodeID is the CometBFT p2p node identifier.
	NodeID string `json:"node_id"`
	// Network is the network name advertised by the node (mirrors ChainID for CometBFT).
	Network string `json:"network"`
	// Moniker is the operator-chosen human-readable name for the node.
	Moniker string `json:"moniker"`
	// SyncStatus is "catching_up" when the node is replaying blocks, "synced" otherwise.
	SyncStatus string `json:"sync_status"`
	// LatestBlockHeight is the height of the most recently seen block.
	LatestBlockHeight int64 `json:"latest_block_height"`
	// LatestBlockTime is the Unix-seconds timestamp of the latest block, omitted if unknown.
	LatestBlockTime *int64 `json:"latest_block_time,omitempty"`
	// EpochBlocks is the configured number of blocks per Tachi epoch.
	EpochBlocks int64 `json:"epoch_blocks"`
	// Peers is the count of currently connected p2p peers.
	Peers int `json:"peers"`
}

// CometRPCResponse is the raw JSON-RPC 2.0 envelope CometBFT wraps every
// forwarded result in. Shape of Result varies by endpoint.
type CometRPCResponse struct {
	// JSONRPC is the JSON-RPC version reported by CometBFT, always "2.0".
	JSONRPC string `json:"jsonrpc" example:"2.0"`
	// ID is the request identifier echoed back by CometBFT.
	ID int `json:"id" example:"1"`
	// Result is the endpoint-specific result body forwarded verbatim from CometBFT.
	Result json.RawMessage `json:"result" swaggertype:"object"`
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

// ChainHealth reports whether the node is committing blocks and within its
// memory threshold. Meant for alerting, not liveness: an unhealthy node
// answers 503, in which case the decoded body is still returned alongside
// the *ErrorResponse so callers can read Problems.
func (s *NodeService) ChainHealth(ctx context.Context) (*ChainHealthResponse, *Response, error) {
	var out ChainHealthResponse
	resp, err := s.client.get(ctx, "health/chain", nil, &out)
	var errResp *ErrorResponse
	if errors.As(err, &errResp) && errResp.Response.StatusCode == http.StatusServiceUnavailable &&
		json.Unmarshal([]byte(errResp.Message), &out) == nil {
		return &out, resp, err
	}
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
