package tachi

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL.String() != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), defaultBaseURL)
	}
	if c.UserAgent != defaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, defaultUserAgent)
	}
	if c.apiKey != "" {
		t.Errorf("apiKey = %q, want empty", c.apiKey)
	}
	// Every service field must be wired to the shared common struct.
	svcs := []interface{}{c.Address, c.Block, c.Epoch, c.Tx, c.VTXO, c.Vault, c.Validators, c.Node, c.Dashboard, c.Bitcoin, c.Sign, c.Watchtower, c.WS}
	for i, s := range svcs {
		if s == nil {
			t.Errorf("service field %d is nil", i)
		}
	}
	if (*AddressService)(&c.common) != c.Address {
		t.Errorf("Address service not wired to common")
	}
}

func TestNewClient_BaseURLTrailingSlash(t *testing.T) {
	c, err := NewClient(WithBaseURL("http://127.0.0.1:26670"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL.String() != "http://127.0.0.1:26670/" {
		t.Errorf("BaseURL = %q, want trailing slash appended", c.BaseURL.String())
	}
}

func TestNewClient_InvalidBaseURL(t *testing.T) {
	_, err := NewClient(WithBaseURL("://not-a-url"))
	if err == nil {
		t.Fatal("expected error for invalid base URL, got nil")
	}
}

func TestNewClient_RejectsNonHTTPScheme(t *testing.T) {
	_, err := NewClient(WithBaseURL("ftp://example.com/"))
	if err == nil {
		t.Fatal("expected error for non-http(s) base URL scheme, got nil")
	}
}

func TestNewClient_RejectsAPIKeyOverPlaintextRemote(t *testing.T) {
	_, err := NewClient(WithBaseURL("http://example.com/"), WithAPIKey("secret"))
	if err == nil {
		t.Fatal("expected error for API key over http to a non-loopback host, got nil")
	}
}

func TestNewClient_AllowsAPIKeyOverPlaintextLoopback(t *testing.T) {
	c, err := NewClient(WithBaseURL("http://127.0.0.1:26670/"), WithAPIKey("secret"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.apiKey != "secret" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "secret")
	}
}

func TestNewClient_Options(t *testing.T) {
	hc := &http.Client{}
	c, err := NewClient(
		WithAPIKey("secret"),
		WithUserAgent("custom-agent/1.0"),
		WithHTTPClient(hc),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.apiKey != "secret" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "secret")
	}
	if c.UserAgent != "custom-agent/1.0" {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, "custom-agent/1.0")
	}
	if c.client != hc {
		t.Errorf("client not set to provided http.Client")
	}
}

func TestClient_NewRequest_PathAndQuery(t *testing.T) {
	c, err := NewClient(WithBaseURL("http://127.0.0.1:26670/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	req, err := c.NewRequest(http.MethodGet, "tachi_stats", url.Values{"foo": {"bar"}}, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	want := "http://127.0.0.1:26670/tachi_stats?foo=bar"
	if req.URL.String() != want {
		t.Errorf("URL = %q, want %q", req.URL.String(), want)
	}
}

func TestClient_NewRequest_RootPath(t *testing.T) {
	// The bitcoin proxy posts to "" (relative to BaseURL), which must
	// resolve to exactly the base URL (the daemon's POST / route).
	c, err := NewClient(WithBaseURL("http://127.0.0.1:26670/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	req, err := c.NewRequest(http.MethodPost, "", nil, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if req.URL.String() != "http://127.0.0.1:26670/" {
		t.Errorf("URL = %q, want %q", req.URL.String(), "http://127.0.0.1:26670/")
	}
}

func TestClient_NewRequest_Headers(t *testing.T) {
	c, err := NewClient(WithAPIKey("mykey"), WithUserAgent("ua/1"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	req, err := c.NewRequest(http.MethodGet, "tachi_stats", nil, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got := req.Header.Get("X-Api-Key"); got != "mykey" {
		t.Errorf("X-Api-Key = %q, want %q", got, "mykey")
	}
	if got := req.Header.Get("User-Agent"); got != "ua/1" {
		t.Errorf("User-Agent = %q, want %q", got, "ua/1")
	}
	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}
	if got := req.Header.Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want empty for GET with no body", got)
	}
}

func TestClient_NewRequest_JSONBody(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	body := struct {
		Hex string `json:"hex"`
	}{Hex: "deadbeef"}
	req, err := c.NewRequest(http.MethodPost, "tachi_txDecode", nil, body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	testBody(t, req, map[string]string{"hex": "deadbeef"})
}

func TestClient_Do_DecodesJSON(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]string{"status": "ok"})
	})
	req, err := ts.client.NewRequest(http.MethodGet, "ok", nil, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	var out map[string]string
	resp, err := ts.client.Do(context.Background(), req, &out)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if out["status"] != "ok" {
		t.Errorf("decoded body = %v, want status=ok", out)
	}
}

func TestClient_Do_NilOutDiscardsBody(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]string{"status": "ok"})
	})
	req, err := ts.client.NewRequest(http.MethodGet, "ok", nil, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := ts.client.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestClient_Do_ErrorResponse(t *testing.T) {
	ts := setup(t)
	ts.mux.HandleFunc("/bad", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "address query parameter is required", http.StatusBadRequest)
	})
	req, err := ts.client.NewRequest(http.MethodGet, "bad", nil, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = ts.client.Do(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errResp, ok := err.(*ErrorResponse)
	if !ok {
		t.Fatalf("error is not *ErrorResponse: %v (%T)", err, err)
	}
	if errResp.Response.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", errResp.Response.StatusCode)
	}
	if !strings.Contains(errResp.Message, "address query parameter is required") {
		t.Errorf("Message = %q, missing expected text", errResp.Message)
	}
	if !strings.Contains(errResp.Error(), "400") {
		t.Errorf("Error() = %q, missing status code", errResp.Error())
	}
}
