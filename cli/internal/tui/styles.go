package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette. lipgloss degrades these to plain text automatically when NO_COLOR is
// set or the terminal lacks color support (via termenv).
// Colors adapt to the terminal background: the Dark shade on dark terminals,
// a darker, readable Light shade on white/light terminals.
var (
	cAccent lipgloss.TerminalColor = lipgloss.AdaptiveColor{Dark: "#4ade80", Light: "#16a34a"} // set by applyTheme
	cGreen                         = lipgloss.AdaptiveColor{Dark: "#6ee7a0", Light: "#047857"} // codes
	cBlue                          = lipgloss.AdaptiveColor{Dark: "#6cb6ff", Light: "#1d4ed8"} // links
	cDim                           = lipgloss.AdaptiveColor{Dark: "#8b93a3", Light: "#6b7280"} // muted text
	cFaint                         = lipgloss.AdaptiveColor{Dark: "#3c4350", Light: "#cbd1da"} // separators
	cWarn                          = lipgloss.AdaptiveColor{Dark: "#f2b45b", Light: "#b45309"} // timer warning
	cBg                            = lipgloss.Color("#0b0d10")                                 // dark text on colored badges

	addrStyle    = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	publicBadge  = lipgloss.NewStyle().Foreground(cBg).Background(cWarn).Bold(true).Padding(0, 1)
	privateBadge = lipgloss.NewStyle().Foreground(cBg).Background(cGreen).Bold(true).Padding(0, 1)

	timerStyle = lipgloss.NewStyle().Foreground(cDim)
	timerWarn  = lipgloss.NewStyle().Foreground(cWarn).Bold(true)

	selStyle  = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	codeStyle = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	linkStyle = lipgloss.NewStyle().Foreground(cBlue)
	dimStyle  = lipgloss.NewStyle().Foreground(cDim)
	sepStyle  = lipgloss.NewStyle().Foreground(cFaint)

	helpStyle   = lipgloss.NewStyle().Foreground(cDim)
	keyStyle    = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	warnStyle   = lipgloss.NewStyle().Foreground(cWarn).Bold(true)
	headStyle   = lipgloss.NewStyle().Bold(true)
)

// theme is a selectable accent color (codes stay green, links blue - those
// encode meaning). Cycle at runtime with the "t" key.
type theme struct {
	name   string
	accent lipgloss.AdaptiveColor
}

var themes = []theme{
	{"green", lipgloss.AdaptiveColor{Dark: "#4ade80", Light: "#16a34a"}},
	{"cyan", lipgloss.AdaptiveColor{Dark: "#22d3ee", Light: "#0891b2"}},
	{"amber", lipgloss.AdaptiveColor{Dark: "#f59e0b", Light: "#b45309"}},
	{"blue", lipgloss.AdaptiveColor{Dark: "#3b82f6", Light: "#2563eb"}},
	{"lavender", lipgloss.AdaptiveColor{Dark: "#c9a2ff", Light: "#7c3aed"}},
}

// themeIndexByName returns the index of a saved theme name, or 0 (green).
func themeIndexByName(name string) int {
	for i, t := range themes {
		if t.name == name {
			return i
		}
	}
	return 0
}

// applyTheme repaints the accent-driven styles for the chosen theme.
func applyTheme(i int) {
	cAccent = themes[i].accent
	addrStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	keyStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	selStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
}

// keybar renders "key label" pairs with the key highlighted, wrapping onto
// multiple lines so nothing is clipped when the terminal is narrow.
func keybar(width int, pairs ...[2]string) string {
	const gap = 3 // visible width of the separator between items
	sep := helpStyle.Render("   ")
	if width < 10 {
		width = 10
	}
	var lines []string
	cur := ""
	curW := 0
	for _, p := range pairs {
		item := keyStyle.Render(p[0]) + " " + helpStyle.Render(p[1])
		iw := lipgloss.Width(item)
		switch {
		case cur == "":
			cur, curW = item, iw
		case curW+gap+iw <= width-1:
			cur += sep + item
			curW += gap + iw
		default:
			lines = append(lines, " "+cur)
			cur, curW = item, iw
		}
	}
	if cur != "" {
		lines = append(lines, " "+cur)
	}
	return strings.Join(lines, "\n")
}
