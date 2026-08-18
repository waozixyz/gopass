package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Round-trip against the real openssl binary using the exact command line of
// the original pass shell script.
func TestDecryptAgainstOpenSSL(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, ".vault")

	encrypt := `printf 'my-top-secret' | openssl enc -aes-256-cbc -md sha512 -a -pbkdf2 -iter 100000 -salt -pass pass:masterpw > ` + file
	if out, err := exec.Command("sh", "-c", encrypt).CombinedOutput(); err != nil {
		t.Fatalf("openssl encrypt failed: %v: %s", err, out)
	}

	got, err := Decrypt(file, "masterpw")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "my-top-secret" {
		t.Errorf("got %q, want %q", got, "my-top-secret")
	}

	// Trailing newlines in the plaintext must be stripped ($(...) parity).
	encryptNL := `printf 'secret-with-newline\n\n' | openssl enc -aes-256-cbc -md sha512 -a -pbkdf2 -iter 100000 -salt -pass pass:masterpw > ` + file
	if out, err := exec.Command("sh", "-c", encryptNL).CombinedOutput(); err != nil {
		t.Fatalf("openssl encrypt failed: %v: %s", err, out)
	}
	got, err = Decrypt(file, "masterpw")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "secret-with-newline" {
		t.Errorf("got %q, want %q (trailing newlines stripped)", got, "secret-with-newline")
	}

	if _, err := Decrypt(file, "wrong-passphrase"); err == nil {
		t.Error("expected error with wrong passphrase")
	}

	if err := os.WriteFile(file, []byte("not-base64-$$$!!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(file, "masterpw"); err == nil {
		t.Error("expected error for non-OpenSSL file")
	}
}
