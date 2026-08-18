//go:build linux

package main

import (
	"os"
	"testing"
	"time"
)

// Live X11 round trip: take ownership of the CLIPBOARD selection, then read
// it back as a separate X client.
func TestClipboardRoundTrip(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no X11 display available")
	}

	text := "TestPassword123"
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- serveClipboard(text, 10*time.Second, ready) }()

	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("clipboard server exited early: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for clipboard ownership")
	}

	// Reading immediately can race with clipboard managers stealing the
	// selection; poll briefly for the expected content.
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := readClipboardSelection()
		if err == nil && got == text {
			return // success
		}
		if time.Now().After(deadline) {
			t.Fatalf("clipboard round trip failed: last read = %q, err = %v", got, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
