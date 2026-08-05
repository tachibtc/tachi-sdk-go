package tachi

import (
	"context"
	"net/url"
	"strconv"
)

// ValidatorsService groups the bootstrap validator registry and this
// node's own peer info.
type ValidatorsService service

// ValidatorInfo represents a registered validator, and is also the shape
// returned by ValidatorsService.PeerInfo for the local node.
type ValidatorInfo struct {
	// PeerID is the libp2p peer identifier used by the discovery overlay.
	PeerID string `json:"peer_id"`
	// PubKeyHex is the compressed secp256k1 validator public key hex used for signed registration.
	PubKeyHex string `json:"pub_key_hex"`
	// Host is the CometBFT P2P address "host:port" the validator listens on.
	Host string `json:"host,omitempty"`
	// P2PPort is the CometBFT P2P listening port advertised by the validator.
	P2PPort int `json:"p2p_port,omitempty"`
	// RPCAddr is the daemon RPC listen address ("host:port") this validator serves.
	RPCAddr string `json:"rpc_addr,omitempty"`
}

// ValidatorsResponse is returned by ValidatorsService.List.
type ValidatorsResponse struct {
	// Validators is the merged list of bootstrap-registered and KDHT-discovered validators.
	Validators []ValidatorInfo `json:"validators"`
	// Count is the total number of validators in the response.
	Count int `json:"count" example:"3"`
}

// LiveValidatorsResponse is returned by ValidatorsService.Live.
type LiveValidatorsResponse struct {
	// Validators is the subset of validators whose peers are currently connected via the overlay network.
	Validators []ValidatorInfo `json:"validators"`
	// Count is the number of live validators returned.
	Count int `json:"count" example:"2"`
	// TotalKnown is the total number of validators known to the node (live plus offline).
	TotalKnown int `json:"total_known" example:"3"`
}

// ValidatorCountResponse is returned by ValidatorsService.Count.
type ValidatorCountResponse struct {
	// Count is the number of validators currently in the bootstrap registry.
	Count int `json:"count" example:"3"`
}

// ReadyResponse is returned by ValidatorsService.Ready.
type ReadyResponse struct {
	// Ready is true once the registry has reached the expected validator count before the long-poll deadline.
	Ready bool `json:"ready" example:"true"`
	// Validators is the current bootstrap registry snapshot at the time the long-poll returned.
	Validators []ValidatorInfo `json:"validators"`
	// Count is the number of validators in Validators.
	Count int `json:"count" example:"2"`
}

// List returns the merged list of bootstrap-registered and
// KDHT-discovered validators.
func (s *ValidatorsService) List(ctx context.Context) (*ValidatorsResponse, *Response, error) {
	var out ValidatorsResponse
	resp, err := s.client.get(ctx, "tachi_validators", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Live returns only validators whose peers are currently connected via the
// overlay network.
func (s *ValidatorsService) Live(ctx context.Context) (*LiveValidatorsResponse, *Response, error) {
	var out LiveValidatorsResponse
	resp, err := s.client.get(ctx, "tachi_validators/live", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Count returns the number of validators currently in the bootstrap
// registry.
func (s *ValidatorsService) Count(ctx context.Context) (*ValidatorCountResponse, *Response, error) {
	var out ValidatorCountResponse
	resp, err := s.client.get(ctx, "tachi_validators/count", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// Ready long-polls (up to 60s server-side) until expected validators have
// registered, then returns the current registry snapshot.
func (s *ValidatorsService) Ready(ctx context.Context, expected int) (*ReadyResponse, *Response, error) {
	q := url.Values{}
	if expected > 0 {
		q.Set("expected", strconv.Itoa(expected))
	}
	var out ReadyResponse
	resp, err := s.client.get(ctx, "tachi_validators/ready", q, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}

// PeerInfo returns this node's PeerID, public key, and network addresses.
func (s *ValidatorsService) PeerInfo(ctx context.Context) (*ValidatorInfo, *Response, error) {
	var out ValidatorInfo
	resp, err := s.client.get(ctx, "tachi_peerInfo", nil, &out)
	if err != nil {
		return nil, resp, err
	}
	return &out, resp, nil
}
