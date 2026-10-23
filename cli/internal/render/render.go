// Package render contains presentation helpers: safe HTML-to-text conversion and
// human-friendly time formatting. We never render raw HTML.
package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/k3a/html2text"
)

// Body returns the best plain-text rendering of a message: the text part if
// present, otherwise a flattened version of the HTML part.
func Body(text string, html *string) string {
	if strings.TrimSpace(text) != "" {
		return text
	}
	if html != nil && strings.TrimSpace(*html) != "" {
		return html2text.HTML2Text(*html)
	}
	return "(no content)"
}

// Ago formats an epoch-millis timestamp as a short relative time.
func Ago(epochMillis int64) string {
	d := time.Since(time.UnixMilli(epochMillis))
	switch {
	case d < 5*time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// Countdown formats a remaining duration as MM:SS (or H:MM:SS past an hour).
func Countdown(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	h, m, s := total/3600, (total%3600)/60, total%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
