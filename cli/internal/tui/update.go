package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atifrana/mailbust/cli/internal/api"
	"github.com/atifrana/mailbust/cli/internal/render"
	"github.com/atifrana/mailbust/cli/internal/store"
)

// Init starts the live subscription, timers, and first inbox load.
func (m *Model) Init() tea.Cmd {
	m.events = m.client.Subscribe(m.ctx, m.address)
	return tea.Batch(
		tick(),
		pollTick(),
		m.loadList(),
		m.waitWS(),
	)
}

// Update handles incoming messages and key presses.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeViewport()
		return m, nil

	case tickMsg:
		if m.remaining() <= 0 {
			m.expired = true
			return m, nil
		}
		if m.remaining() <= time.Minute && !m.warned {
			m.warned = true
			m.setStatus("⚠ less than 1 minute left - press e to extend")
		}
		return m, tick()

	case pollMsg:
		return m, tea.Batch(m.loadList(), pollTick())

	case loadedMsg:
		m.mergeList(msg)
		return m, nil

	case wsMsg:
		before := len(m.msgs)
		m.upsert(api.Summary(msg))
		if len(m.msgs) > before {
			m.setStatus("✉ new mail")
		}
		return m, m.waitWS()

	case openedMsg:
		full := api.Message(msg)
		m.current = &full
		m.state = messageView
		m.setupMessageViewport()
		return m, nil

	case deletedMsg:
		m.removeByID(string(msg))
		m.setStatus("message deleted")
		return m, nil

	case extendedMsg:
		m.expiresAt = time.Time(msg)
		m.warned = false
		m.setStatus("lifetime extended")
		return m, nil

	case statusMsg:
		m.setStatus(string(msg))
		return m, nil

	case errMsg:
		m.setStatus("error: " + msg.err.Error())
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.cancel()
		return m, tea.Quit
	case "t":
		m.themeIdx = (m.themeIdx + 1) % len(themes)
		applyTheme(m.themeIdx)
		store.SaveTheme(themes[m.themeIdx].name) // remember across runs
		m.setStatus("theme: " + themes[m.themeIdx].name + " (press t to cycle)")
		return m, nil
	}

	if m.expired {
		return m, nil // only quit works once expired
	}

	if m.state == messageView {
		return m.handleMessageKey(msg)
	}
	return m.handleInboxKey(msg)
}

func (m *Model) handleInboxKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sel := m.selected()
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.msgs)-1 {
			m.cursor++
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(0, len(m.msgs)-1)
	case "enter":
		if sel != nil {
			return m, m.openMessage(sel.ID)
		}
	case "c":
		return m, copyCmd(derefCode(sel), "code")
	case "l":
		return m, copyCmd(derefLink(sel), "link")
	case "o":
		return m, openURLCmd(derefLink(sel))
	case "a", "y":
		return m, copyCmd(m.address, "address")
	case "d":
		if sel != nil {
			id := sel.ID
			m.removeByID(id) // optimistic
			return m, m.deleteMessage(id)
		}
	case "e":
		return m, m.extend()
	case "r":
		return m, m.loadList()
	}
	return m, nil
}

func (m *Model) handleMessageKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace", "left", "h":
		m.state = inboxView
		m.current = nil
		return m, nil
	case "c":
		return m, copyCmd(ptr(m.current.Code), "code")
	case "l":
		return m, copyCmd(ptr(m.current.Link), "link")
	case "o":
		return m, openURLCmd(ptr(m.current.Link))
	case "a", "y":
		return m, copyCmd(m.address, "address")
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// ---- view-state helpers ----------------------------------------------------

func (m *Model) remaining() time.Duration {
	return time.Until(m.expiresAt)
}

func (m *Model) mergeList(list []api.Summary) {
	// Replace wholesale but keep selection stable by ID.
	var selID string
	if s := m.selected(); s != nil {
		selID = s.ID
	}
	m.msgs = list
	m.sortMsgs()
	m.cursor = 0
	for i := range m.msgs {
		if m.msgs[i].ID == selID {
			m.cursor = i
			break
		}
	}
}

func (m *Model) removeByID(id string) {
	out := m.msgs[:0]
	for _, s := range m.msgs {
		if s.ID != id {
			out = append(out, s)
		}
	}
	m.msgs = out
	if m.cursor >= len(m.msgs) {
		m.cursor = max(0, len(m.msgs)-1)
	}
}

func (m *Model) resizeViewport() {
	h := m.height - 4
	if h < 3 {
		h = 3
	}
	if !m.vpReady {
		m.vp = viewport.New(m.width, h)
		m.vpReady = true
	} else {
		m.vp.Width = m.width
		m.vp.Height = h
	}
	if m.current != nil {
		m.setupMessageViewport()
	}
}

func (m *Model) setupMessageViewport() {
	if !m.vpReady {
		m.vp = viewport.New(m.width, max(3, m.height-6))
		m.vpReady = true
	}
	m.vp.SetContent(render.Body(m.current.Text, m.current.HTML))
	m.vp.GotoTop()
}

func ptr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
