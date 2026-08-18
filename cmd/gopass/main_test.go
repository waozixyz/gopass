package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	password "github.com/waozixyz/gopass"
)

// TestMain isolates every test in this package from the developer's real
// profiles file: without this, a machine with ~/.config/gopass/profiles.json
// would feed private configuration into the tests.
func TestMain(m *testing.M) {
	configPath = func() string {
		return filepath.Join(os.TempDir(), "gopass-tests-absent-profiles.json")
	}
	os.Exit(m.Run())
}

func TestRunWithArgument(t *testing.T) {
	var output, diagnostics bytes.Buffer
	err := run([]string{"example.com", "alice", "correct horse battery staple"}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "&vLf44D'/cSkP-_8\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunUsesEnvironmentAndFlags(t *testing.T) {
	t.Setenv(environmentVariable, "correct horse battery staple")
	var output, diagnostics bytes.Buffer
	err := run([]string{"--length", "10", "--no-symbols", "--exclude", "aA0", "example.com", "alice"}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(output.String())
	if len(got) != 10 {
		t.Fatalf("output length = %d, want 10", len(got))
	}
	if strings.ContainsAny(got, "aA0"+"!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~") {
		t.Fatalf("output %q contains a disabled character", got)
	}
}

func TestRunAcceptsFlagsAfterSiteAndLogin(t *testing.T) {
	t.Setenv(environmentVariable, "correct horse battery staple")
	var output, diagnostics bytes.Buffer
	err := run([]string{"example.com", "alice", "-C", "9", "-L", "23", "--no-symbols"}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); len(got) != 23 {
		t.Fatalf("output length = %d, want 23", len(got))
	}
}

func TestRunInformationOptions(t *testing.T) {
	for _, test := range []struct {
		argument string
		want     string
	}{
		{"--version", "gopass " + password.Version},
		{"--help", "Usage: gopass"},
	} {
		var output, diagnostics bytes.Buffer
		if err := run([]string{test.argument}, &output, &diagnostics); err != nil {
			t.Errorf("run(%s): %v", test.argument, err)
		}
		combined := output.String() + diagnostics.String()
		if !strings.Contains(combined, test.want) {
			t.Errorf("run(%s) output %q does not contain %q", test.argument, combined, test.want)
		}
	}
}

func TestRunRejectsConflictingPasswordSources(t *testing.T) {
	var output, diagnostics bytes.Buffer
	err := run([]string{"--prompt", "example.com", "alice", "master"}, &output, &diagnostics)
	if err == nil || !strings.Contains(err.Error(), "cannot be supplied") {
		t.Fatalf("error = %v, want source conflict", err)
	}
}

func TestClipboardCommandsDoNotCarryPasswordData(t *testing.T) {
	secret := "this must travel only over standard input"
	for _, command := range clipboardCommands() {
		if command.program == secret {
			t.Fatal("secret appeared as clipboard executable")
		}
		for _, argument := range command.args {
			if strings.Contains(argument, secret) {
				t.Fatal("secret appeared in clipboard command arguments")
			}
		}
	}
}
