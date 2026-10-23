package config

import "testing"

func TestResolveAddress(t *testing.T) {
	c := Config{API: "x", Domain: "mailbust.com"}
	cases := map[string]string{
		"testing":              "testing@mailbust.com",
		"Testing":              "testing@mailbust.com",
		"  testing  ":          "testing@mailbust.com",
		"testing@mailbust.com": "testing@mailbust.com",
		"testing@other.com":    "testing@mailbust.com", // domain always forced to ours
		"TESTING@OTHER.COM":    "testing@mailbust.com",
	}
	for in, want := range cases {
		if got := c.ResolveAddress(in); got != want {
			t.Errorf("ResolveAddress(%q) = %q, want %q", in, got, want)
		}
	}
	if got := c.ResolveAddress(""); got != "" {
		t.Errorf("ResolveAddress(\"\") = %q, want empty", got)
	}
}
