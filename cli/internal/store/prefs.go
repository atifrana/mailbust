package store

import (
	"os"
	"path/filepath"
	"strings"
)

func themePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mailbust", "theme"), nil
}

// LoadTheme returns the saved accent-theme name, or "" if none is stored.
func LoadTheme() string {
	p, err := themePath()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// SaveTheme persists the chosen accent-theme name (best-effort).
func SaveTheme(name string) {
	p, err := themePath()
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
		_ = os.WriteFile(p, []byte(name), 0o644)
	}
}
