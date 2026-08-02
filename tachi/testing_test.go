package tachi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testServer bundles an in-process HTTP server with a Client pointed at it,
// mirroring the harness most official Go API clients (e.g. go-github) use
// so every service method can be tested without touching the network.
type testServer struct {
	mux    *http.ServeMux
	server *httptest.Server
	client *Client
}

func setup(t *testing.T) *testServer {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := NewClient(WithBaseURL(server.URL + "/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return &testServer{mux: mux, server: server, client: client}
}

// setupServer starts an httptest.Server backed by mux and returns its
// trailing-slash URL, for tests that need to construct their own Client
// (e.g. to pass extra ClientOptions like WithAPIKey).
func setupServer(t *testing.T, mux *http.ServeMux) string {
	t.Helper()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL + "/"
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Errorf("request method = %s, want %s", r.Method, want)
	}
}

func testQueryParam(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != want {
		t.Errorf("query param %q = %q, want %q", key, got, want)
	}
}

func testQueryParamAbsent(t *testing.T, r *http.Request, key string) {
	t.Helper()
	if r.URL.Query().Has(key) {
		t.Errorf("query param %q present, want absent", key)
	}
}

func testHeader(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.Header.Get(key); got != want {
		t.Errorf("header %q = %q, want %q", key, got, want)
	}
}

// testBody decodes the request body as JSON into a map and compares it
// against want (also JSON-round-tripped so types line up, e.g. int vs
// float64).
func testBody(t *testing.T, r *http.Request, want interface{}) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	var got, wantDecoded interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode request body: %v (body: %s)", err, body)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	if err := json.Unmarshal(wantJSON, &wantDecoded); err != nil {
		t.Fatalf("decode want: %v", err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON2, _ := json.Marshal(wantDecoded)
	if string(gotJSON) != string(wantJSON2) {
		t.Errorf("request body = %s, want %s", gotJSON, wantJSON2)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("write json response: %v", err)
	}
}
