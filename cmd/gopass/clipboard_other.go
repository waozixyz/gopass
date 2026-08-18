//go:build !linux

package main

import (
	"errors"
	"time"
)

// The built-in X11 clipboard is Linux-only; other platforms always fall back
// to external clipboard tools.

func copyViaX11(text string, clearAfter time.Duration) (bool, error) {
	return false, nil
}

func runClipboardDaemon(timeout time.Duration) error {
	return errors.New("the built-in clipboard daemon is Linux-only")
}

func readClipboardSelection() (string, error) {
	return "", errors.New("reading the clipboard is Linux-only")
}
