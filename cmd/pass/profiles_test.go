package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	password "github.com/waozixyz/pass"
)

// withConfig points the config loader at a temporary file for one test.
func withConfig(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	original := configPath
	configPath = func() string { return path }
	t.Cleanup(func() { configPath = original })
}

func withVaultPassphrase(t *testing.T, passphrase string) {
	t.Helper()
	original := readVaultPassphrase
	readVaultPassphrase = func() (string, error) { return passphrase, nil }
	t.Cleanup(func() { readVaultPassphrase = original })
}

func TestRunWithProfileSettings(t *testing.T) {
	withConfig(t, `{
		"vault": "/nonexistent/vault",
		"copy": true,
		"clear_after": "90s",
		"profiles": {
			"w": {"login": "alice", "counter": 9, "length": 23, "symbols": false}
		}
	}`)
	t.Setenv(environmentVariable, "correct horse battery staple")
	var output, diagnostics bytes.Buffer
	err := run([]string{"-P", "w", "--no-copy", "example.com"}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	want, err := password.Generate("example.com", "alice", "correct horse battery staple", password.Options{
		Length: 23, Counter: 9, Lowercase: true, Uppercase: true, Digits: true, Symbols: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestProfileSettingsDoNotOverrideExplicitFlags(t *testing.T) {
	withConfig(t, `{"profiles": {"w": {"login": "alice", "length": 23, "symbols": false}}}`)
	t.Setenv(environmentVariable, "correct horse battery staple")
	var output, diagnostics bytes.Buffer
	err := run([]string{"-P", "w", "-L", "10", "example.com"}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); len(got) != 10 {
		t.Fatalf("output length = %d, want 10 (flag must beat profile)", len(got))
	}
}

func TestRunRejectsBadProfileNames(t *testing.T) {
	withConfig(t, `{"profiles": {"w": {"login": "alice"}, "n": {}}}`)
	t.Setenv(environmentVariable, "x")

	var output, diagnostics bytes.Buffer
	if err := run([]string{"-P", "x", "example.com"}, &output, &diagnostics); err == nil ||
		!strings.Contains(err.Error(), "unknown profile") {
		t.Fatalf("error = %v, want unknown profile", err)
	}
	if err := run([]string{"-P", "n", "example.com"}, &output, &diagnostics); err == nil ||
		!strings.Contains(err.Error(), "has no login") {
		t.Fatalf("error = %v, want missing login", err)
	}
	// A profile without a configuration file at all.
	original := configPath
	configPath = func() string { return filepath.Join(t.TempDir(), "absent.json") }
	defer func() { configPath = original }()
	if err := run([]string{"-P", "w", "example.com"}, &output, &diagnostics); err == nil ||
		!strings.Contains(err.Error(), "no configuration file") {
		t.Fatalf("error = %v, want missing configuration", err)
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	withConfig(t, `{"profiles":`)
	var output, diagnostics bytes.Buffer
	if err := run([]string{"example.com", "alice", "master"}, &output, &diagnostics); err == nil {
		t.Fatal("expected error for malformed configuration")
	}
}

func TestRunRejectsInvalidClearAfter(t *testing.T) {
	withConfig(t, `{"clear_after": "not-a-duration", "profiles": {"w": {"login": "alice"}}}`)
	t.Setenv(environmentVariable, "x")
	var output, diagnostics bytes.Buffer
	err := run([]string{"-P", "w", "--no-copy", "example.com"}, &output, &diagnostics)
	if err == nil || !strings.Contains(err.Error(), "clear_after") {
		t.Fatalf("error = %v, want clear_after parse failure", err)
	}
}

func TestVaultMasterPasswordBeatsEnvironmentAndPositional(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	vaultFile := filepath.Join(t.TempDir(), ".vault")
	encrypt := `printf 'my-top-secret' | openssl enc -aes-256-cbc -md sha512 -a -pbkdf2 -iter 100000 -salt -pass pass:masterpw > ` + vaultFile
	if out, err := exec.Command("sh", "-c", encrypt).CombinedOutput(); err != nil {
		t.Fatalf("openssl encrypt failed: %v: %s", err, out)
	}

	withVaultPassphrase(t, "masterpw")
	t.Setenv(environmentVariable, "environment-must-be-ignored")

	var output, diagnostics bytes.Buffer
	err := run([]string{"--vault", vaultFile, "example.com", "alice"}, &output, &diagnostics)
	if err != nil {
		t.Fatal(err)
	}
	want, err := password.Generate("example.com", "alice", "my-top-secret", password.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	if err := run([]string{"--vault", vaultFile, "example.com", "alice", "master"}, &output, &diagnostics); err == nil ||
		!strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("error = %v, want source conflict with MASTER_PASSWORD", err)
	}
	if err := run([]string{"--vault", vaultFile, "--prompt", "example.com", "alice"}, &output, &diagnostics); err == nil ||
		!strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("error = %v, want source conflict with --prompt", err)
	}
}

func TestConfigVaultIsUsedForProfile(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	vaultFile := filepath.Join(t.TempDir(), ".vault")
	encrypt := `printf 'cfg-secret' | openssl enc -aes-256-cbc -md sha512 -a -pbkdf2 -iter 100000 -salt -pass pass:masterpw > ` + vaultFile
	if out, err := exec.Command("sh", "-c", encrypt).CombinedOutput(); err != nil {
		t.Fatalf("openssl encrypt failed: %v: %s", err, out)
	}

	withConfig(t, `{"vault": "`+vaultFile+`", "profiles": {"w": {"login": "alice"}}}`)
	withVaultPassphrase(t, "masterpw")

	var output, diagnostics bytes.Buffer
	if err := run([]string{"-P", "w", "--no-copy", "example.com"}, &output, &diagnostics); err != nil {
		t.Fatal(err)
	}
	want, err := password.Generate("example.com", "alice", "cfg-secret", password.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	for _, test := range []struct{ in, want string }{
		{"~", home},
		{"~/bin/.vault", filepath.Join(home, "bin", ".vault")},
		{"/abs/path", "/abs/path"},
		{"relative", "relative"},
	} {
		if got := expandHome(test.in); got != test.want {
			t.Errorf("expandHome(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestVaultPassphraseFromPipedStdin(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	vaultFile := filepath.Join(t.TempDir(), ".vault")
	encrypt := `printf 'piped-secret' | openssl enc -aes-256-cbc -md sha512 -a -pbkdf2 -iter 100000 -salt -pass pass:masterpw > ` + vaultFile
	if out, err := exec.Command("sh", "-c", encrypt).CombinedOutput(); err != nil {
		t.Fatalf("openssl encrypt failed: %v: %s", err, out)
	}

	for _, test := range []struct {
		name     string
		contents string
	}{
		{"newline", "masterpw\n"},
		{"crlf", "masterpw\r\n"},
		{"no trailing newline", "masterpw"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, err := os.CreateTemp(t.TempDir(), "stdin")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := input.WriteString(test.contents); err != nil {
				t.Fatal(err)
			}
			if _, err := input.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			previous := os.Stdin
			os.Stdin = input
			defer func() { os.Stdin = previous }()

			got, err := promptVaultPassphrase()
			if err != nil {
				t.Fatal(err)
			}
			if got != "masterpw" {
				t.Fatalf("passphrase = %q, want %q", got, "masterpw")
			}
		})
	}
}
