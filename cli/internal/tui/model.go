// Package tui implements the interactive MailBust inbox.
package tui

import (
	"context"
	"os/exec"
	"runtime"
	"sort"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atifrana/mailbust/cli/internal/api"
	"github.com/atifrana/mailbust/cli/internal/config"
	"github.com/atifrana/mailbust/cli/internal/store"
)

type viewState int

const (
	inboxView viewState = iota
	messageView
)

// Model is the Bubble Tea model for the inbox TUI.
type Model struct {
	cfg     config.Config
	client  *api.Client
	address string
	public  bool

	expiresAt time.Time
	msgs      []api.Summary
	cursor    int

	state   viewState
	current *api.Message
	vp      viewport.Model
	vpReady bool

	width, height int
	status        string
	statusUntil   time.Time
	warned        bool
	expired       bool
	themeIdx      int

	events <-chan api.Summary
	ctx    context.Context
	cancel context.CancelFunc
}

// New builds a model for an already-created/opened mailbox.
func New(cfg config.Config, client *api.Client, info api.MailboxInfo) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	themeIdx := themeIndexByName(store.LoadTheme()) // remembered accent, else green
	applyTheme(themeIdx)
	return &Model{
		themeIdx:  themeIdx,
		cfg:       cfg,
		client:    client,
		address:   info.Address,
		public:    info.Public,
		expiresAt: time.UnixMilli(info.ExpiresAt),
		state:     inboxView,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// ---- messages --------------------------------------------------------------

type tickMsg time.Time
type pollMsg time.Time
type loadedMsg []api.Summary
type wsMsg api.Summary
type openedMsg api.Message
type deletedMsg string
type extendedMsg time.Time
type statusMsg string
type errMsg struct{ err error }

// ---- commands --------------------------------------------------------------

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func pollTick() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg { return pollMsg(t) })
}

func (m *Model) loadList() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()
		list, err := m.client.List(ctx, m.address)
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg(list)
	}
}

func (m *Model) waitWS() tea.Cmd {
	return func() tea.Msg {
		s, ok := <-m.events
		if !ok {
			return nil
		}
		return wsMsg(s)
	}
}

func (m *Model) openMessage(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()
		msg, err := m.client.Get(ctx, m.address, id)
		if err != nil {
			return errMsg{err}
		}
		return openedMsg(msg)
	}
}

func (m *Model) deleteMessage(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()
		if err := m.client.Delete(ctx, m.address, id); err != nil {
			return errMsg{err}
		}
		return deletedMsg(id)
	}
}

func (m *Model) extend() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
		defer cancel()
		info, err := m.client.Extend(ctx, m.address, 0)
		if err != nil {
			return errMsg{err}
		}
		return extendedMsg(time.UnixMilli(info.ExpiresAt))
	}
}

func copyCmd(text, label string) tea.Cmd {
	return func() tea.Msg {
		if text == "" {
			return statusMsg("nothing to copy")
		}
		if err := clipboard.WriteAll(text); err != nil {
			return statusMsg("clipboard unavailable")
		}
		return statusMsg(label + " copied")
	}
}

func openURLCmd(u string) tea.Cmd {
	return func() tea.Msg {
		if u == "" {
			return statusMsg("no link")
		}
		if err := openURL(u); err != nil {
			return statusMsg("could not open link")
		}
		return statusMsg("opening link…")
	}
}

func openURL(u string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

// ---- helpers ---------------------------------------------------------------

func (m *Model) setStatus(s string) {
	m.status = s
	m.statusUntil = time.Now().Add(4 * time.Second)
}

func (m *Model) selected() *api.Summary {
	if m.cursor >= 0 && m.cursor < len(m.msgs) {
		return &m.msgs[m.cursor]
	}
	return nil
}

func (m *Model) sortMsgs() {
	sort.SliceStable(m.msgs, func(i, j int) bool {
		return m.msgs[i].ReceivedAt > m.msgs[j].ReceivedAt
	})
}

func (m *Model) upsert(s api.Summary) {
	for i := range m.msgs {
		if m.msgs[i].ID == s.ID {
			m.msgs[i] = s
			return
		}
	}
	m.msgs = append(m.msgs, s)
	m.sortMsgs()
}

func derefCode(s *api.Summary) string {
	if s != nil && s.Code != nil {
		return *s.Code
	}
	return ""
}

func derefLink(s *api.Summary) string {
	if s != nil && s.Link != nil {
		return *s.Link
	}
	return ""
}
