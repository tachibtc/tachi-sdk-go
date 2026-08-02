// Package tachi is a Go client for the Tachi daemon HTTP RPC API (tachid).
//
// It covers every route in the daemon's swagger spec: dashboard/stats,
// address & balance, blocks, transactions, epochs, VTXOs, vaults,
// validators & peers, node operations, the cooperative-refund signing
// ceremony, watchtower status/receipts, live websocket alerts, and the
// pass-through Bitcoin Core JSON-RPC proxy.
//
// The Client follows the same shape as most official Go API clients
// (e.g. google/go-github): a root Client exposes one field per resource
// group, and each group's methods return (result, *Response, error).
//
//	c, err := tachi.NewClient()
//	stats, _, err := c.Dashboard.Stats(ctx)
//	addr, _, err := c.Address.Get(ctx, "bcrt1p...")
//
// See https://rpc-regtest.tachibtc.com/swagger/index.html for the reference
// spec this client was generated against.
package tachi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Version is the SDK's semantic version, sent as part of the default
// User-Agent header.
const Version = "0.1.0"

// defaultBaseURL is the public Tachi regtest daemon endpoint.
const defaultBaseURL = "https://rpc-regtest.tachibtc.com/"

var defaultUserAgent = "daemon-go-sdk/" + Version

// Client manages communication with the Tachi daemon RPC API. Create one
// with NewClient; the zero value is not usable. A Client is safe for
// concurrent use.
type Client struct {
	clientMu sync.Mutex
	client   *http.Client

	// BaseURL is the root of every request, e.g.
	// "https://rpc-regtest.tachibtc.com/" or "http://127.0.0.1:26670/".
	// Always has a trailing slash.
	BaseURL *url.URL

	// UserAgent is sent on every request. Defaults to
	// "daemon-go-sdk/<Version>".
	UserAgent string

	// apiKey, when set, is sent as X-Api-Key on every request. Unlocks
	// privileged bitcoind methods, vault reconstruction parameters on
	// Vault.List, the vaults_by_user query path, and bypasses rate limits.
	apiKey string

	// rawBaseURL holds a pending WithBaseURL value until NewClient parses
	// and validates it into BaseURL.
	rawBaseURL string

	// common is reused across every service instead of allocating one
	// struct per service — each exported service field below is just a
	// type-converted pointer to this same struct.
	common service

	Address    *AddressService
	Block      *BlockService
	Epoch      *EpochService
	Tx         *TxService
	VTXO       *VTXOService
	Vault      *VaultService
	Validators *ValidatorsService
	Node       *NodeService
	Dashboard  *DashboardService
	Bitcoin    *BitcoinService
	Sign       *SignService
	Watchtower *WatchtowerService
	WS         *WSService
}

// service is the shared base type every *Service is aliased from; it gives
// each service a back-reference to the owning Client.
type service struct {
	client *Client
}

// ClientOption configures a Client constructed by NewClient.
type ClientOption func(*Client)

// WithHTTPClient overrides the default http.Client (30s timeout). Pass a
// client wrapping a retry/backoff RoundTripper (e.g. hashicorp's
// go-retryablehttp) to add retry behavior — the SDK does not retry on its
// own, matching the convention of most official Go API clients.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.client = hc }
}

// WithAPIKey attaches an X-Api-Key header to every request. Pass the
// daemon's master BTC_RPC_API_KEY to unlock privileged bitcoind methods,
// vault reconstruction parameters on Vault.List, and the vaults_by_user
// query path, or a client key registered with the relevant scope.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) { c.apiKey = key }
}

// WithUserAgent overrides the default "daemon-go-sdk/<Version>" User-Agent.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) { c.UserAgent = ua }
}

// WithBaseURL points the client at a different daemon, e.g.
// "http://127.0.0.1:26670/" for a local node. Defaults to the public
// regtest endpoint.
func WithBaseURL(rawURL string) ClientOption {
	return func(c *Client) { c.rawBaseURL = rawURL }
}

// NewClient creates a Client. With no options it targets the public Tachi
// regtest daemon; use WithBaseURL to target a local or private node.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		client:     &http.Client{Timeout: 30 * time.Second},
		UserAgent:  defaultUserAgent,
		rawBaseURL: defaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}

	u, err := url.Parse(c.rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("tachi: parse base URL %q: %w", c.rawBaseURL, err)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	c.BaseURL = u

	c.common.client = c
	c.Address = (*AddressService)(&c.common)
	c.Block = (*BlockService)(&c.common)
	c.Epoch = (*EpochService)(&c.common)
	c.Tx = (*TxService)(&c.common)
	c.VTXO = (*VTXOService)(&c.common)
	c.Vault = (*VaultService)(&c.common)
	c.Validators = (*ValidatorsService)(&c.common)
	c.Node = (*NodeService)(&c.common)
	c.Dashboard = (*DashboardService)(&c.common)
	c.Bitcoin = (*BitcoinService)(&c.common)
	c.Sign = (*SignService)(&c.common)
	c.Watchtower = (*WatchtowerService)(&c.common)
	c.WS = (*WSService)(&c.common)
	return c, nil
}

// NewRequest builds an API request against path (relative to c.BaseURL,
// no leading slash — e.g. "tachi_stats", not "/tachi_stats") with an
// optional query string and an optional JSON-encoded body. Exposed so
// callers can reach an endpoint the SDK hasn't wrapped yet; every generated
// service method is built on top of it.
func (c *Client) NewRequest(method, path string, query url.Values, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("tachi: parse path %q: %w", path, err)
	}
	u := c.BaseURL.ResolveReference(rel)
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, fmt.Errorf("tachi: encode request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	return req, nil
}

// Response wraps the raw *http.Response returned by every service method,
// mirroring the shape used by most official Go API clients so callers can
// inspect status/headers alongside the decoded result.
type Response struct {
	*http.Response
}

// Do sends req and, on a 2xx response, JSON-decodes the body into v (which
// may be nil to discard it). On a non-2xx response it returns an
// *ErrorResponse. Exposed alongside NewRequest for reaching
// not-yet-wrapped endpoints.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}
	defer resp.Body.Close()

	response := &Response{Response: resp}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("tachi: read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, &ErrorResponse{Response: resp, Message: strings.TrimSpace(string(body))}
	}
	if v != nil && len(body) > 0 {
		if err := json.Unmarshal(body, v); err != nil {
			return response, fmt.Errorf("tachi: decode response: %w", err)
		}
	}
	return response, nil
}

// get is a convenience wrapper around NewRequest+Do for the common
// GET-with-query-params case used by nearly every read endpoint.
func (c *Client) get(ctx context.Context, path string, query url.Values, out interface{}) (*Response, error) {
	req, err := c.NewRequest(http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req, out)
}

// post is a convenience wrapper around NewRequest+Do for the common
// POST-with-JSON-body case.
func (c *Client) post(ctx context.Context, path string, body, out interface{}) (*Response, error) {
	req, err := c.NewRequest(http.MethodPost, path, nil, body)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req, out)
}

// ErrorResponse is returned when the daemon responds with a non-2xx
// status. The daemon's error bodies are plain text (http.Error), not
// JSON, so Message carries the raw response body.
type ErrorResponse struct {
	Response *http.Response
	Message  string
}

func (r *ErrorResponse) Error() string {
	req := r.Response.Request
	return fmt.Sprintf("tachi: %v %v: %d %s", req.Method, req.URL, r.Response.StatusCode, r.Message)
}

// setIf sets query[key] = value in q when value is non-empty.
func setIf(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}
