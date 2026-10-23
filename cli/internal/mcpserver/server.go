// Package mcpserver exposes MailBust over the Model Context Protocol so AI agents
// can create mailboxes and read verification codes. It reuses the same API client
// as the CLI, and remembers tokens for mailboxes it created during the session so
// private mailboxes stay readable without re-supplying the token.
package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/atifrana/mailbust/cli/internal/api"
	"github.com/atifrana/mailbust/cli/internal/config"
	"github.com/atifrana/mailbust/cli/internal/render"
	"github.com/atifrana/mailbust/cli/internal/store"
)

const maxWaitChunk = 110 * time.Second

type server struct {
	cfg    config.Config
	mu     sync.Mutex
	tokens map[string]string // address -> token, for mailboxes created this session
}

// Run starts the MCP server over stdio and blocks until the client disconnects.
func Run(ctx context.Context, version string) error {
	s := &server{cfg: config.Load(), tokens: map[string]string{}}

	impl := &mcp.Implementation{Name: "mailbust", Version: version}
	srv := mcp.NewServer(impl, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_mailbox",
		Description: "Create a disposable mailbox. Omit name for a random private mailbox; pass a name for a public shared one. Returns the address and expiry.",
	}, s.createMailbox)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "wait_for_code",
		Description: "Wait for the next verification code to arrive at a mailbox and return it (with the sender and subject).",
	}, s.waitForCode)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "wait_for_link",
		Description: "Wait for the next verification/magic link to arrive at a mailbox and return it (with the sender and subject).",
	}, s.waitForLink)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_messages",
		Description: "List the messages currently in a mailbox (sender, subject, extracted code/link).",
	}, s.listMessages)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "read_message",
		Description: "Read the plain-text body of one message by its id.",
	}, s.readMessage)

	return srv.Run(ctx, &mcp.StdioTransport{})
}

// ---- token handling --------------------------------------------------------

func (s *server) remember(address, token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	s.tokens[address] = token
	s.mu.Unlock()
}

func (s *server) clientFor(address string) *api.Client {
	s.mu.Lock()
	tok := s.tokens[address]
	s.mu.Unlock()
	if tok == "" {
		tok = os.Getenv("MAILBUST_TOKEN")
	}
	if tok == "" {
		tok = store.Load(address)
	}
	return api.New(s.cfg.API, tok)
}

// ---- tool argument shapes --------------------------------------------------

type createArgs struct {
	Name string `json:"name,omitempty" jsonschema:"optional name for a public shared mailbox; omit for a random private one"`
	TTL  string `json:"ttl,omitempty" jsonschema:"optional lifetime such as 30m, 1h, 24h (default 15m, max 24h)"`
}

type waitArgs struct {
	Address string `json:"address" jsonschema:"the mailbox address to watch"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"seconds to wait for the next message (default 120)"`
}

type listArgs struct {
	Address string `json:"address" jsonschema:"the mailbox address"`
}

type readArgs struct {
	Address   string `json:"address" jsonschema:"the mailbox address"`
	MessageID string `json:"message_id" jsonschema:"the id of the message to read"`
}

// ---- handlers --------------------------------------------------------------

func (s *server) createMailbox(ctx context.Context, _ *mcp.CallToolRequest, args createArgs) (*mcp.CallToolResult, any, error) {
	secs, err := parseTTL(args.TTL)
	if err != nil {
		return nil, nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	info, err := api.New(s.cfg.API, "").CreateMailbox(cctx, args.Name, secs)
	if err != nil {
		return nil, nil, err
	}
	s.remember(info.Address, info.Token)

	kind := "public"
	if !info.Public {
		kind = "private"
	}
	return text("Created %s mailbox %s (expires %s).", kind, info.Address,
		time.UnixMilli(info.ExpiresAt).Format(time.RFC3339)), nil, nil
}

func (s *server) waitForCode(ctx context.Context, _ *mcp.CallToolRequest, args waitArgs) (*mcp.CallToolResult, any, error) {
	return s.wait(ctx, args, "code")
}

func (s *server) waitForLink(ctx context.Context, _ *mcp.CallToolRequest, args waitArgs) (*mcp.CallToolResult, any, error) {
	return s.wait(ctx, args, "link")
}

func (s *server) wait(ctx context.Context, args waitArgs, kind string) (*mcp.CallToolResult, any, error) {
	address := s.cfg.ResolveAddress(args.Address)
	if address == "" {
		return nil, nil, fmt.Errorf("address is required")
	}
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 120
	}
	client := s.clientFor(address)
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return text("No %s arrived at %s within %ds.", kind, address, timeout), nil, nil
		}
		chunk := remaining
		if chunk > maxWaitChunk {
			chunk = maxWaitChunk
		}
		cctx, cancel := context.WithTimeout(ctx, chunk+10*time.Second)
		msg, err := client.Wait(cctx, address, kind, chunk)
		cancel()
		if err != nil {
			return nil, nil, err
		}
		if msg == nil {
			continue
		}
		value := ptr(msg.Code)
		if kind == "link" {
			value = ptr(msg.Link)
		}
		return text("%s: %s\nfrom: %s\nsubject: %s", kind, value, msg.From, msg.Subject), nil, nil
	}
}

func (s *server) listMessages(ctx context.Context, _ *mcp.CallToolRequest, args listArgs) (*mcp.CallToolResult, any, error) {
	address := s.cfg.ResolveAddress(args.Address)
	if address == "" {
		return nil, nil, fmt.Errorf("address is required")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	msgs, err := s.clientFor(address).List(cctx, address)
	if err != nil {
		return nil, nil, err
	}
	if len(msgs) == 0 {
		return text("Mailbox %s is empty.", address), nil, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d message(s) in %s:\n", len(msgs), address)
	for _, m := range msgs {
		extra := ""
		switch {
		case m.Code != nil:
			extra = "  code=" + *m.Code
		case m.Link != nil:
			extra = "  link=" + *m.Link
		}
		fmt.Fprintf(&b, "- [%s] %s - %q (%s)%s\n", m.ID, m.Sender(), m.Subject, render.Ago(m.ReceivedAt), extra)
	}
	return text("%s", b.String()), nil, nil
}

func (s *server) readMessage(ctx context.Context, _ *mcp.CallToolRequest, args readArgs) (*mcp.CallToolResult, any, error) {
	address := s.cfg.ResolveAddress(args.Address)
	if address == "" || args.MessageID == "" {
		return nil, nil, fmt.Errorf("address and message_id are required")
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	m, err := s.clientFor(address).Get(cctx, address, args.MessageID)
	if err != nil {
		return nil, nil, err
	}
	body := render.Body(m.Text, m.HTML)
	return text("From: %s\nSubject: %s\n\n%s", m.From, m.Subject, body), nil, nil
}

// ---- helpers ---------------------------------------------------------------

func text(format string, a ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, a...)}},
	}
}

func parseTTL(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid ttl %q (try 30m, 1h, 24h)", s)
	}
	return int(d.Seconds()), nil
}

func ptr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
