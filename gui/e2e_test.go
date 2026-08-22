package main

import (
	"testing"

	kryon "github.com/waozixyz/kryon/go/kryon"
	"github.com/waozixyz/kryon/go/kryui"
	password "github.com/waozixyz/pass"
)

func newHeadlessTestApp(t *testing.T) *app {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	kryon.SetRuntime(kryon.New(kryon.AppConfig{Title: "pass-test", Width: 720, Height: 740}))
	t.Cleanup(func() {
		kryon.SetRuntime(nil)
	})
	return newApp()
}

func drawTestFrame(a *app) {
	kryui.BeginDrawing()
	kryui.ClearBackground(kryui.GetThemeBackground())
	kryui.BeginUI(kryui.Key("pass/e2e"))
	a.draw(kryui.GetScreenWidth())
	kryui.EndUI()
	a.handleUIEvents()
	kryui.EndDrawing()
}

func tapTestFrame(a *app, x, y float32) {
	kryon.QueueTap(x, y)
	drawTestFrame(a)
}

func typeTestField(a *app, focusID int32, text string) {
	kryon.SetFocus(focusID)
	kryon.QueueText(text)
	drawTestFrame(a)
}

func TestHeadlessGenerateAndCopyWorkflow(t *testing.T) {
	a := newHeadlessTestApp(t)
	drawTestFrame(a)

	typeTestField(a, 101, "example.com")
	typeTestField(a, 102, "alice")
	typeTestField(a, 103, "correct horse battery staple")
	tapTestFrame(a, 110, 463)

	want, err := password.Generate("example.com", "alice", "correct horse battery staple", password.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got := a.generated; got != want {
		t.Fatalf("generated password = %q, want %q for site=%q login=%q master=%q", got, want, a.site.Text(), a.login.Text(), a.master.Text())
	}
	if got, want := a.message, "Password generated locally"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}

	tapTestFrame(a, 260, 463)
	if got := kryui.GetClipboardText(); got != want {
		t.Fatalf("clipboard = %q, want generated password", got)
	}
}

func TestHeadlessProfileSaveAndLoadWorkflow(t *testing.T) {
	a := newHeadlessTestApp(t)
	a.site.SetText("bank.example")
	a.login.SetText("me")
	a.exclude.SetText("abc")
	a.profileName.SetText("Bank")
	a.length = 20
	a.counter = 7
	a.lower = true
	a.upper = true
	a.digits = true
	a.symbols = false
	a.view = viewProfiles

	drawTestFrame(a)
	tapTestFrame(a, 120, 173)

	if got, want := a.message, "Profile saved"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
	if len(a.profiles) != 1 {
		t.Fatalf("profiles length = %d, want 1", len(a.profiles))
	}
	if got, want := a.profiles[0].Name, "Bank"; got != want {
		t.Fatalf("profile name = %q, want %q", got, want)
	}

	a.site.Clear()
	a.login.Clear()
	a.exclude.Clear()
	a.length = 16
	a.counter = 1
	a.symbols = true
	drawTestFrame(a)
	tapTestFrame(a, 120, 263)

	if a.view != viewGenerate {
		t.Fatalf("view = %d, want generate view", a.view)
	}
	if got, want := a.site.Text(), "bank.example"; got != want {
		t.Fatalf("site = %q, want %q", got, want)
	}
	if got, want := a.login.Text(), "me"; got != want {
		t.Fatalf("login = %q, want %q", got, want)
	}
	if got, want := a.exclude.Text(), "abc"; got != want {
		t.Fatalf("exclude = %q, want %q", got, want)
	}
	if got, want := a.length, int32(20); got != want {
		t.Fatalf("length = %d, want %d", got, want)
	}
	if got, want := a.counter, int32(7); got != want {
		t.Fatalf("counter = %d, want %d", got, want)
	}
	if a.symbols {
		t.Fatal("symbols = true, want false")
	}
	if got, want := a.message, "Profile loaded"; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}
