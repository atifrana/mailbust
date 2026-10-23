// Package api is the HTTP + WebSocket client for the MailBust backend.
// It is shared by the TUI, the scripting commands, and (later) the MCP server.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the MailBust REST/WS API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a client for baseURL (e.g. https://api.mailbust.com). token may be
// empty for public mailboxes; it is sent as a bearer token when set.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 0}, // per-request timeouts via context
	}
}

// APIError is a structured error returned by the backend.
type APIError struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("http %d", e.Status)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var wrap struct {
			Error APIError `json:"error"`
		}
		_ = json.Unmarshal(data, &wrap)
		wrap.Error.Status = resp.StatusCode
		return &wrap.Error
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// CreateMailbox creates a mailbox. An empty name yields a random private mailbox
// (the response includes a Token); a name yields a public named mailbox.
func (c *Client) CreateMailbox(ctx context.Context, name string, ttlSeconds int) (MailboxInfo, error) {
	req := map[string]any{}
	if name != "" {
		req["name"] = name
	}
	if ttlSeconds > 0 {
		req["ttl"] = ttlSeconds
	}
	var info MailboxInfo
	err := c.do(ctx, http.MethodPost, "/v1/mailboxes", req, &info)
	return info, err
}

// Info fetches mailbox metadata (also opens a named mailbox on demand).
func (c *Client) Info(ctx context.Context, address string) (MailboxInfo, error) {
	var info MailboxInfo
	err := c.do(ctx, http.MethodGet, "/v1/mailboxes/"+url.PathEscape(address), nil, &info)
	return info, err
}

// List returns the inbox summaries, newest first.
func (c *Client) List(ctx context.Context, address string) ([]Summary, error) {
	var out struct {
		Messages []Summary `json:"messages"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/mailboxes/"+url.PathEscape(address)+"/messages", nil, &out)
	return out.Messages, err
}

// Get fetches one full message.
func (c *Client) Get(ctx context.Context, address, id string) (Message, error) {
	var m Message
	err := c.do(ctx, http.MethodGet,
		"/v1/mailboxes/"+url.PathEscape(address)+"/messages/"+url.PathEscape(id), nil, &m)
	return m, err
}

// Delete removes one message.
func (c *Client) Delete(ctx context.Context, address, id string) error {
	return c.do(ctx, http.MethodDelete,
		"/v1/mailboxes/"+url.PathEscape(address)+"/messages/"+url.PathEscape(id), nil, nil)
}

// Extend pushes the mailbox expiry out by ttlSeconds (0 = server default).
func (c *Client) Extend(ctx context.Context, address string, ttlSeconds int) (MailboxInfo, error) {
	var info MailboxInfo
	err := c.do(ctx, http.MethodPost,
		"/v1/mailboxes/"+url.PathEscape(address)+"/extend",
		map[string]any{"ttl": ttlSeconds}, &info)
	return info, err
}

// Wait long-polls for the next message matching kind ("code", "link", or "any").
// Returns nil (no error) on timeout.
func (c *Client) Wait(ctx context.Context, address, kind string, timeout time.Duration) (*Summary, error) {
	secs := int(timeout.Seconds())
	if secs < 1 {
		secs = 1
	}
	q := fmt.Sprintf("?for=%s&timeout=%d", url.QueryEscape(kind), secs)
	var out struct {
		Message *Summary `json:"message"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/mailboxes/"+url.PathEscape(address)+"/wait"+q, nil, &out)
	if err != nil {
		return nil, err
	}
	return out.Message, nil
}

// WSURL returns the WebSocket URL for live updates on a mailbox.
func (c *Client) WSURL(address string) string {
	u := c.baseURL + "/v1/mailboxes/" + url.PathEscape(address) + "/ws"
	if strings.HasPrefix(u, "https://") {
		return "wss://" + strings.TrimPrefix(u, "https://")
	}
	return "ws://" + strings.TrimPrefix(u, "http://")
}

// Token returns the bearer token configured on this client (may be empty).
func (c *Client) Token() string { return c.token }
