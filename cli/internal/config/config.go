// Package config resolves runtime settings from environment variables and
// normalizes mailbox names the same way the backend does.
package config

import (
	"os"
	"strings"
)

const (
	defaultAPI    = "https://api.mailbust.com"
	defaultDomain = "mailbust.com"
)

// Config holds resolved runtime settings.
type Config struct {
	API    string // base URL of the backend
	Domain string // mail domain for bare names
}

// Load reads MAILBUST_API and MAILBUST_DOMAIN, falling back to defaults.
func Load() Config {
	return Config{
		API:    envOr("MAILBUST_API", defaultAPI),
		Domain: envOr("MAILBUST_DOMAIN", defaultDomain),
	}
}

// ResolveAddress turns a user argument ("testing" or "testing@mailbust.com")
// into a full, lowercased address on the configured domain.
func (c Config) ResolveAddress(arg string) string {
	a := strings.ToLower(strings.TrimSpace(arg))
	if a == "" {
		return ""
	}
	if i := strings.IndexByte(a, '@'); i >= 0 {
		a = a[:i]
	}
	return a + "@" + c.Domain
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
