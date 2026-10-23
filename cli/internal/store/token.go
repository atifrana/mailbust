// Package store persists per-mailbox secret tokens in the OS keychain so that
// private mailboxes can be reopened without re-sending the token each time.
package store

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "mailbust"

// Save stores the token for a mailbox address. A best-effort operation: if the
// OS keychain is unavailable, the error is returned for the caller to decide.
func Save(address, token string) error {
	return keyring.Set(service, address, token)
}

// Load returns the stored token for an address, or "" if none is stored.
func Load(address string) string {
	tok, err := keyring.Get(service, address)
	if err != nil {
		return ""
	}
	return tok
}

// Delete removes a stored token (ignored if absent).
func Delete(address string) error {
	err := keyring.Delete(service, address)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
