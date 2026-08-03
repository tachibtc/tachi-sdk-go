package tachi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSubscribeOptions_Empty(t *testing.T) {
	cases := []struct {
		name string
		opts SubscribeOptions
		want bool
	}{
		{"all zero", SubscribeOptions{}, true},
		{"address set", SubscribeOptions{Address: "abcd"}, false},
		{"vault set", SubscribeOptions{Vault: "abcd"}, false},
		{"vaultId set", SubscribeOptions{VaultID: "abcd"}, false},
		{"blocks set", SubscribeOptions{Blocks: true}, false},
		{"validators set", SubscribeOptions{Validators: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.opts.empty(); got != tc.want {
				t.Errorf("empty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWSService_Subscribe_RejectsEmptyOptions(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := c.WS.Subscribe(context.Background(), SubscribeOptions{}); err == nil {
		t.Fatal("expected error for empty SubscribeOptions, got nil")
	}
}

func TestWSService_Subscribe_RoundTrip(t *testing.T) {
	var upgrader = websocket.Upgrader{}

	mux := http.NewServeMux()
	mux.HandleFunc("/tachi_ws", func(w http.ResponseWriter, r *http.Request) {
		testQueryParam(t, r, "blocks", "true")
		testQueryParam(t, r, "vaultId", "deadbeef")
		testQueryParamAbsent(t, r, "address")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server upgrade: %v", err)
			return
		}
		defer conn.Close()
		height := int64(42)
		evt := WSEvent{Event: "block", Block: &WSBlockAlert{Height: height, BlockHash: "abc", TxCount: 1}}
		if err := conn.WriteJSON(evt); err != nil {
			t.Errorf("server write: %v", err)
		}
		// Keep the connection open briefly so the client's read loop has
		// time to deliver the message before the test server tears down.
		time.Sleep(200 * time.Millisecond)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c, err := NewClient(WithBaseURL(server.URL + "/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := c.WS.Subscribe(ctx, SubscribeOptions{Blocks: true, VaultID: "deadbeef"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer conn.Close()

	select {
	case evt := <-conn.Events():
		if evt.Event != "block" {
			t.Errorf("Event = %q, want block", evt.Event)
		}
		if evt.Block == nil || evt.Block.Height != 42 {
			t.Errorf("Block = %+v, want Height=42", evt.Block)
		}
	case err := <-conn.Err():
		t.Fatalf("read error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestWSService_Subscribe_SchemeTranslation(t *testing.T) {
	// Subscribe must translate http(s) -> ws(s); a bad scheme should
	// surface as a dial error rather than silently connecting over HTTP.
	c, err := NewClient(WithBaseURL("http://127.0.0.1:1/")) // nothing listens here
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = c.WS.Subscribe(ctx, SubscribeOptions{Blocks: true})
	if err == nil {
		t.Fatal("expected dial error against an unreachable address, got nil")
	}
}
