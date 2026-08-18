// Package vault decrypts OpenSSL-encrypted secret files so they can be used
// as master passwords without storing them anywhere.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrFormat  = errors.New("vault is not in OpenSSL salted base64 format")
	ErrDecrypt = errors.New("wrong vault passphrase or corrupted vault")
)

// Decrypt decrypts a file produced by:
//
//	openssl enc -aes-256-cbc -md sha512 -a -pbkdf2 -iter 100000 -salt \
//	           -pass pass:... > vault
//
// File format: base64("Salted__" + 8-byte salt + AES-256-CBC ciphertext).
// Key/IV are derived with PBKDF2-HMAC-SHA512(passphrase, salt, 100000, 48):
// first 32 bytes are the AES key, the next 16 are the CBC IV.
// Trailing newlines are stripped, matching the $(...) substitution of the
// shell scripts this replaces.
func Decrypt(path, passphrase string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// The base64 body may contain newlines; strip all whitespace first.
	var b strings.Builder
	for _, c := range raw {
		switch c {
		case '\n', '\r', ' ', '\t':
		default:
			b.WriteByte(c)
		}
	}
	data, err := base64.StdEncoding.DecodeString(b.String())
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFormat, err)
	}
	if len(data) < aes.BlockSize*2 || string(data[:8]) != "Salted__" {
		return "", ErrFormat
	}
	salt := data[8:16]
	ciphertext := data[16:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", ErrFormat
	}

	keyIV, err := pbkdf2.Key(sha512.New, passphrase, salt, 100000, aes.BlockSize*3)
	if err != nil {
		return "", err
	}
	key, iv := keyIV[:32], keyIV[32:48]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)

	// Strip PKCS#7 padding; a bad pad almost always means a wrong passphrase.
	pad := int(plaintext[len(plaintext)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(plaintext) {
		return "", ErrDecrypt
	}
	for _, p := range plaintext[len(plaintext)-pad:] {
		if int(p) != pad {
			return "", ErrDecrypt
		}
	}
	plaintext = plaintext[:len(plaintext)-pad]

	secret := strings.TrimRight(string(plaintext), "\n")
	clear(plaintext) // best effort: wipe the decrypted bytes
	return secret, nil
}

func clear(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
