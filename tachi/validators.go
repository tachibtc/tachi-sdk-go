package tachi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// ValidatorsService groups the bootstrap validator registry and this
// node's own peer info.
type ValidatorsService service

// registerSigDomain must match the daemon's domain-separation tag
// (rpc.registerSigDomain) — changing it here without a matching daemon
// change will make every signed registration fail verification.
const registerSigDomain = "tachi-register-v1"

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

// RegisterRequest is the ValidatorsService.Register body. Build one with
// SignRegisterRequest, which fills Timestamp and Signature.
type RegisterRequest struct {
	ValidatorInfo
	// Timestamp is the unix-seconds time at which the request was signed; checked against a freshness window.
	Timestamp int64 `json:"timestamp"`
	// Signature is the BIP-340 Schnorr signature (128-char hex) over the canonical register digest.
	Signature string `json:"signature"`
}

// RegisterResponse is returned by ValidatorsService.Register.
type RegisterResponse struct {
	// Status is a fixed acknowledgement string ("registered") confirming a successful registration.
	Status string `json:"status" example:"registered"`
	// Total is the new total number of validators in the registry after this registration.
	Total int `json:"total" example:"3"`
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

// canonicalRegisterDigest builds the 32-byte digest a /validators/register
// signature commits to, byte-for-byte matching the daemon's
// canonicalRegisterDigest (rpc.go). Any change to field order or the domain
// tag must be mirrored on the daemon side or every signature will fail
// verification.
func canonicalRegisterDigest(info ValidatorInfo, timestamp int64) [32]byte {
	var b strings.Builder
	b.WriteString(registerSigDomain)
	b.WriteByte('\n')
	b.WriteString(strings.ToLower(info.PubKeyHex))
	b.WriteByte('\n')
	b.WriteString(info.PeerID)
	b.WriteByte('\n')
	b.WriteString(info.Host)
	b.WriteByte('\n')
	b.WriteString(strconv.Itoa(info.P2PPort))
	b.WriteByte('\n')
	b.WriteString(info.RPCAddr)
	b.WriteByte('\n')
	b.WriteString(strconv.FormatInt(timestamp, 10))
	return sha256.Sum256([]byte(b.String()))
}

// SignRegisterRequest fills in Timestamp and Signature on req using
// privKey, producing a BIP-340 Schnorr signature over the canonical
// register digest the daemon verifies in ValidatorsService.Register.
// privKey must correspond to req.PubKeyHex (the compressed secp256k1
// pubkey hex).
func SignRegisterRequest(req *RegisterRequest, privKey *btcec.PrivateKey) error {
	if privKey == nil {
		return errors.New("tachi: privKey is required")
	}
	if req.PubKeyHex == "" {
		return errors.New("tachi: PubKeyHex is required")
	}
	req.Timestamp = time.Now().Unix()
	digest := canonicalRegisterDigest(req.ValidatorInfo, req.Timestamp)
	sig, err := schnorr.Sign(privKey, digest[:])
	if err != nil {
		return fmt.Errorf("tachi: sign register payload: %w", err)
	}
	req.Signature = hex.EncodeToString(sig.Serialize())
	return nil
}

// Register registers a validator with this bootstrap node. req must be
// signed first — see SignRegisterRequest.
func (s *ValidatorsService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, *Response, error) {
	var out RegisterResponse
	resp, err := s.client.post(ctx, "tachi_validators/register", req, &out)
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
