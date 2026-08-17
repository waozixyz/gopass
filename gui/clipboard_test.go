package main

import (
	"testing"
	"time"
)

func TestClipboardLeaseClearsOnlyItsOwnExpiredValue(t *testing.T) {
	now := time.Unix(100, 0)
	clipboard := ""
	lease := clipboardLease{
		read:  func() string { return clipboard },
		write: func(value string) { clipboard = value },
		now:   func() time.Time { return now },
	}
	lease.copy("generated", 20*time.Second)
	if clipboard != "generated" {
		t.Fatalf("clipboard = %q", clipboard)
	}
	now = now.Add(19 * time.Second)
	if lease.tick() || clipboard == "" {
		t.Fatal("lease cleared too early")
	}
	clipboard = "user replacement"
	now = now.Add(2 * time.Second)
	if !lease.tick() || clipboard != "user replacement" {
		t.Fatal("lease overwrote newer clipboard content")
	}
}

func TestClipboardLeaseClearsExpiredValue(t *testing.T) {
	now := time.Unix(100, 0)
	clipboard := ""
	lease := clipboardLease{
		read:  func() string { return clipboard },
		write: func(value string) { clipboard = value },
		now:   func() time.Time { return now },
	}
	lease.copy("generated", time.Second)
	now = now.Add(time.Second)
	if !lease.tick() || clipboard != "" {
		t.Fatal("expired generated password was not cleared")
	}
}
