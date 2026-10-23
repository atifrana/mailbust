package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// Event is a live update pushed over the WebSocket.
type Event struct {
	Type    string  `json:"type"`
	Message Summary `json:"message"`
}

// Subscribe opens a live WebSocket to the mailbox and sends each new message to
// the returned channel until ctx is cancelled. It reconnects automatically on
// failure with a short backoff, so transient drops are invisible to the caller.
// The channel is closed when ctx is done.
func (c *Client) Subscribe(ctx context.Context, address string) <-chan Summary {
	out := make(chan Summary, 16)
	go func() {
		defer close(out)
		backoff := time.Second
		for ctx.Err() == nil {
			if c.readLoop(ctx, address, out) {
				backoff = time.Second // clean session; reset backoff
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 15*time.Second {
				backoff *= 2
			}
		}
	}()
	return out
}

// readLoop holds one WebSocket session. Returns true if it connected at all.
func (c *Client) readLoop(ctx context.Context, address string, out chan<- Summary) bool {
	header := http.Header{}
	if c.token != "" {
		header.Set("Authorization", "Bearer "+c.token)
	}
	conn, _, err := websocket.Dial(ctx, c.WSURL(address), &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		return false
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(1 << 20)

	// Keepalive pings so idle connections aren't culled.
	pingCtx, cancelPing := context.WithCancel(ctx)
	defer cancelPing()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-t.C:
				_ = conn.Ping(pingCtx)
			}
		}
	}()

	for ctx.Err() == nil {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return true
		}
		var ev Event
		if json.Unmarshal(data, &ev) == nil && ev.Type == "message" {
			select {
			case out <- ev.Message:
			case <-ctx.Done():
				return true
			}
		}
	}
	return true
}
