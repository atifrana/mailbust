package tui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atifrana/mailbust/cli/internal/render"
)

// View renders the whole screen.
func (m *Model) View() string {
	if m.expired {
		return m.expiredView()
	}
	if m.state == messageView && m.current != nil {
		return m.messageView()
	}
	return m.inboxView()
}

func (m *Model) width80() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

func (m *Model) header() string {
	badge := privateBadge.Render("PRIVATE")
	if m.public {
		badge = publicBadge.Render("PUBLIC")
	}

	timer := render.Countdown(m.remaining())
	ts := timerStyle
	if m.remaining() <= time.Minute {
		ts = timerWarn
	}
	right := ts.Render("⏱ " + timer + " left")

	left := headStyle.Render(" mailbust ") + sepStyle.Render("── ") +
		addrStyle.Render(m.address) + " " + badge

	gap := m.width80() - utf8.RuneCountInString(stripANSI(left)) - utf8.RuneCountInString(stripANSI(right)) - 1
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *Model) statusLine() string {
	if m.status != "" && time.Now().Before(m.statusUntil) {
		if strings.HasPrefix(m.status, "⚠") || strings.HasPrefix(m.status, "error") {
			return warnStyle.Render(m.status)
		}
		return statusStyle.Render(m.status)
	}
	return ""
}

func (m *Model) inboxView() string {
	w := m.width80()
	rule := sepStyle.Render(strings.Repeat("─", w))
	top := m.header() + "\n" + rule

	// Body: either the message rows or a friendly empty hint.
	var body []string
	if len(m.msgs) == 0 {
		body = append(body,
			"",
			dimStyle.Render("   Waiting for mail…"),
			dimStyle.Render("   Your address is on the clipboard - paste it into any signup,"),
			dimStyle.Render("   and the message (with its code) shows up here instantly."),
		)
	} else {
		senderW, codeW, timeW := 16, 10, 10
		subjectW := w - senderW - codeW - timeW - 6
		if subjectW < 10 {
			subjectW = 10
		}
		for i, s := range m.msgs {
			marker := "  "
			sender := pad(truncate(s.Sender(), senderW), senderW)
			subject := pad(truncate(s.Subject, subjectW), subjectW)
			var badge string
			switch {
			case s.Code != nil:
				badge = codeStyle.Render(pad(*s.Code, codeW))
			case s.Link != nil:
				badge = linkStyle.Render(pad("[link]", codeW))
			default:
				badge = pad("", codeW)
			}
			at := dimStyle.Render(render.Ago(s.ReceivedAt))
			if i == m.cursor {
				body = append(body, selStyle.Render("▸ ")+headStyle.Render(sender)+" "+
					headStyle.Render(subject)+" "+badge+" "+at)
			} else {
				body = append(body, marker+sender+" "+subject+" "+badge+" "+at)
			}
		}
	}

	footer := rule + "\n" + keybar(w,
		[2]string{"↑↓", "navigate"}, [2]string{"⏎", "read email"}, [2]string{"c", "copy code"},
		[2]string{"l", "copy link"}, [2]string{"o", "open link"}, [2]string{"a", "copy address"},
		[2]string{"d", "delete"}, [2]string{"e", "add time"}, [2]string{"t", "theme"}, [2]string{"q", "quit"},
	)
	if s := m.statusLine(); s != "" {
		footer += "\n " + s
	} else {
		footer += "\n"
	}

	return m.fill(top, strings.Join(body, "\n"), footer)
}

// fill stacks top, body, and footer, padding with blank lines so the footer sits
// at the bottom of the terminal - a stable full-screen layout.
func (m *Model) fill(top, body, footer string) string {
	content := top + "\n" + body
	if m.height > 0 {
		used := countLines(content) + countLines(footer)
		for pad := m.height - used; pad > 0; pad-- {
			content += "\n"
		}
	}
	return content + "\n" + footer
}

func countLines(s string) int { return strings.Count(s, "\n") + 1 }

func (m *Model) messageView() string {
	msg := m.current
	rule := sepStyle.Render(strings.Repeat("─", m.width80()))

	from := msg.From
	if msg.FromName != nil && *msg.FromName != "" {
		from = *msg.FromName + " <" + msg.From + ">"
	}
	head := []string{
		m.header(), rule,
		headStyle.Render("From:    ") + from,
		headStyle.Render("Subject: ") + msg.Subject,
		headStyle.Render("Date:    ") + time.UnixMilli(msg.ReceivedAt).Format("Mon 15:04:05"),
	}
	if msg.Code != nil {
		head = append(head, headStyle.Render("Code:    ")+codeStyle.Render(*msg.Code))
	}
	if msg.Link != nil {
		head = append(head, headStyle.Render("Link:    ")+linkStyle.Render(*msg.Link))
	}
	if len(msg.Attachments) > 0 {
		var names []string
		for _, a := range msg.Attachments {
			names = append(names, a.Filename)
		}
		head = append(head, headStyle.Render("Files:   ")+dimStyle.Render(strings.Join(names, ", ")))
	}
	head = append(head, rule)

	footer := rule + "\n" + keybar(m.width80(),
		[2]string{"↑↓", "scroll"}, [2]string{"c", "copy code"}, [2]string{"l", "copy link"},
		[2]string{"o", "open link"}, [2]string{"a", "copy address"}, [2]string{"t", "theme"}, [2]string{"esc", "back"}, [2]string{"q", "quit"},
	)
	status := ""
	if s := m.statusLine(); s != "" {
		status = " " + s
	}

	// Size the scrollable body to exactly fill the gap.
	if m.height > 0 {
		h := m.height - len(head) - countLines(footer)
		if status != "" {
			h--
		}
		if h < 3 {
			h = 3
		}
		m.vp.Height = h
	}

	out := strings.Join(head, "\n") + "\n" + m.vp.View() + "\n" + footer
	if status != "" {
		out += "\n" + status
	}
	return out
}

func (m *Model) expiredView() string {
	msg := "\n\n  " + warnStyle.Render("Mailbox expired.") +
		dimStyle.Render("  "+m.address+" and all its messages have been deleted.") +
		"\n\n  " + helpStyle.Render("Press q to quit.")
	return m.header() + "\n" + msg
}

// ---- text helpers ----------------------------------------------------------

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	if n <= 1 {
		return string([]rune(s)[:n])
	}
	return string([]rune(s)[:n-1]) + "…"
}

func pad(s string, n int) string {
	gap := n - utf8.RuneCountInString(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// stripANSI removes SGR escape sequences for width calculation.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
