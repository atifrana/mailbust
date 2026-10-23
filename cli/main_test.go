package main

import "testing"

func TestParseTTL(t *testing.T) {
	ok := map[string]int{
		"":    0,
		"30m": 1800,
		"1h":  3600,
		"24h": 86400,
		"90s": 90,
	}
	for in, want := range ok {
		got, err := parseTTL(in)
		if err != nil {
			t.Errorf("parseTTL(%q) unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseTTL(%q) = %d, want %d", in, got, want)
		}
	}

	for _, bad := range []string{"abc", "-5m", "0", "10"} {
		if _, err := parseTTL(bad); err == nil {
			t.Errorf("parseTTL(%q) expected error, got nil", bad)
		}
	}
}
