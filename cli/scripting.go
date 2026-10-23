package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"

	"github.com/atifrana/mailbust/cli/internal/api"
	"github.com/atifrana/mailbust/cli/internal/config"
	"github.com/atifrana/mailbust/cli/internal/render"
	"github.com/atifrana/mailbust/cli/internal/store"
	"github.com/atifrana/mailbust/cli/internal/tui"
)

// The server caps a single wait call; we chunk longer waits across several.
const maxWaitChunk = 110 * time.Second

func isTTY() bool {
	fd := os.Stdout.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// clientFor builds a client for an address, taking the token from MAILBUST_TOKEN
// if set (useful in CI, where there's no OS keychain), else the keychain.
func clientFor(cfg config.Config, address string) *api.Client {
	tok := os.Getenv("MAILBUST_TOKEN")
	if tok == "" {
		tok = store.Load(address)
	}
	return api.New(cfg.API, tok)
}

// ---- scripting output shapes (snake_case for shell/jq friendliness) --------

type newOut struct {
	Address   string `json:"address"`
	Token     string `json:"token,omitempty"`
	ExpiresAt int64  `json:"expires_at"`
}

type waitOut struct {
	Code       string `json:"code,omitempty"`
	Link       string `json:"link,omitempty"`
	From       string `json:"from"`
	Subject    string `json:"subject"`
	ID         string `json:"id"`
	ReceivedAt int64  `json:"received_at"`
}

type listItem struct {
	ID             string `json:"id"`
	From           string `json:"from"`
	Subject        string `json:"subject"`
	Code           string `json:"code,omitempty"`
	Link           string `json:"link,omitempty"`
	ReceivedAt     int64  `json:"received_at"`
	HasAttachments bool   `json:"has_attachments"`
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fail(1, "encoding json: %v", err)
	}
	fmt.Println(string(b))
	return nil
}

// ---- modes -----------------------------------------------------------------

func runNew(cfg config.Config, o *opts) error {
	secs, err := parseTTL(o.ttl)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	info, err := api.New(cfg.API, "").CreateMailbox(ctx, "", secs)
	if err != nil {
		return fail(1, "creating mailbox: %v", err)
	}
	if info.Token != "" {
		if err := store.Save(info.Address, info.Token); err != nil {
			fmt.Fprintln(os.Stderr, "mailbust: warning: could not save token to keychain:", err)
		}
	}
	if o.jsonOut {
		return printJSON(newOut{Address: info.Address, Token: info.Token, ExpiresAt: info.ExpiresAt})
	}
	fmt.Println(info.Address)
	return nil
}

func runWait(cfg config.Config, name, kind string, o *opts) error {
	if o.timeout <= 0 {
		return fail(3, "--timeout must be positive")
	}
	address := cfg.ResolveAddress(name)
	client := clientFor(cfg, address)
	deadline := time.Now().Add(time.Duration(o.timeout) * time.Second)

	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return exitError{code: 2, msg: fmt.Sprintf("timed out waiting for %s", kind)}
		}
		chunk := remaining
		if chunk > maxWaitChunk {
			chunk = maxWaitChunk
		}
		ctx, cancel := context.WithTimeout(context.Background(), chunk+10*time.Second)
		s, err := client.Wait(ctx, address, kind, chunk)
		cancel()
		if err != nil {
			return fail(1, "waiting for %s: %v", kind, err)
		}
		if s == nil {
			continue // chunk elapsed with no match; keep waiting
		}

		value := ptrVal(s.Code)
		if kind == "link" {
			value = ptrVal(s.Link)
		}
		if o.jsonOut {
			return printJSON(waitOut{
				Code:       ptrVal(s.Code),
				Link:       ptrVal(s.Link),
				From:       s.From,
				Subject:    s.Subject,
				ID:         s.ID,
				ReceivedAt: s.ReceivedAt,
			})
		}
		fmt.Println(value)
		return nil
	}
}

func runList(cfg config.Config, name string, o *opts) error {
	address := cfg.ResolveAddress(name)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgs, err := clientFor(cfg, address).List(ctx, address)
	if err != nil {
		return fail(1, "listing messages: %v", err)
	}

	if o.jsonOut {
		items := make([]listItem, 0, len(msgs))
		for _, s := range msgs {
			items = append(items, listItem{
				ID:             s.ID,
				From:           s.From,
				Subject:        s.Subject,
				Code:           ptrVal(s.Code),
				Link:           ptrVal(s.Link),
				ReceivedAt:     s.ReceivedAt,
				HasAttachments: s.HasAttachments,
			})
		}
		return printJSON(items)
	}

	for _, s := range msgs {
		extra := ""
		switch {
		case s.Code != nil:
			extra = *s.Code
		case s.Link != nil:
			extra = *s.Link
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", render.Ago(s.ReceivedAt), s.Sender(), s.Subject, extra)
	}
	return nil
}

func runTUI(cfg config.Config, name, ttl string) error {
	secs, err := parseTTL(ttl)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var client *api.Client
	var info api.MailboxInfo

	if name == "" {
		created, err := api.New(cfg.API, "").CreateMailbox(ctx, "", secs)
		if err != nil {
			return fail(1, "creating mailbox: %v", err)
		}
		if created.Token != "" {
			if err := store.Save(created.Address, created.Token); err != nil {
				fmt.Fprintln(os.Stderr, "mailbust: warning: could not save token to keychain:", err)
			}
		}
		client = api.New(cfg.API, created.Token)
		info = created
	} else {
		address := cfg.ResolveAddress(name)
		client = clientFor(cfg, address)
		opened, err := client.Info(ctx, address)
		if err != nil {
			return fail(1, "opening %s: %v", address, err)
		}
		info = opened
	}

	_, err = tea.NewProgram(tui.New(cfg, client, info), tea.WithAltScreen()).Run()
	return err
}

func parseTTL(s string) (int, error) {
	if s == "" {
		return 0, nil // server default
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fail(3, "invalid --time %q (try 30m, 1h, 24h)", s)
	}
	if d <= 0 {
		return 0, fail(3, "--time must be positive")
	}
	return int(d.Seconds()), nil
}

func ptrVal(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
