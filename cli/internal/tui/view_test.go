package tui

import (
	"strings"
	"testing"

	"github.com/atifrana/mailbust/cli/internal/api"
)

func TestInboxViewShowsKeybarAndHint(t *testing.T) {
	m := &Model{address: "abc@mailbust.com", width: 120, height: 30, state: inboxView}
	out := m.inboxView()
	for _, want := range []string{"navigate", "copy code", "copy address", "quit", "Waiting for mail"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty inbox view missing %q", want)
		}
	}
}

func TestInboxViewRendersRows(t *testing.T) {
	code := "123456"
	m := &Model{
		address: "abc@mailbust.com", width: 120, height: 30, state: inboxView,
		msgs: []api.Summary{{ID: "1", From: "a@b.com", Subject: "Hi", Code: &code, ReceivedAt: 1}},
	}
	out := m.inboxView()
	if !strings.Contains(out, "123456") || !strings.Contains(out, "Hi") {
		t.Error("populated inbox view missing message content")
	}
	if !strings.Contains(out, "quit") {
		t.Error("populated inbox view missing keybar")
	}
}
