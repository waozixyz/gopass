// Package gopass derives site-specific passwords without storing them.
package gopass

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

const (
	LowercaseCharacters = "abcdefghijklmnopqrstuvwxyz"
	UppercaseCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DigitCharacters     = "0123456789"
	SymbolCharacters    = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
)

// Options controls the shape of a generated password.
type Options struct {
	Length    int
	Counter   uint64
	Lowercase bool
	Uppercase bool
	Digits    bool
	Symbols   bool
	Exclude   string
}

// DefaultOptions returns the standard generator settings.
func DefaultOptions() Options {
	return Options{
		Length:    16,
		Counter:   1,
		Lowercase: true,
		Uppercase: true,
		Digits:    true,
		Symbols:   true,
	}
}

// Generate derives a password from site, login, master password, and options.
func Generate(site, login, master string, options Options) (string, error) {
	classes, alphabet, err := availableCharacters(options)
	if err != nil {
		return "", err
	}
	if options.Length < len(classes) {
		return "", fmt.Errorf("password length %d is smaller than the %d enabled character classes", options.Length, len(classes))
	}

	salt := []byte(site + login + strconv.FormatUint(options.Counter, 10))
	entropy := new(big.Int).SetBytes(deriveKey([]byte(master), salt))

	password := make([]byte, 0, options.Length)
	for range options.Length - len(classes) {
		password = append(password, alphabet[takeRemainder(entropy, len(alphabet))])
	}

	required := make([]byte, len(classes))
	for i, class := range classes {
		required[i] = class[takeRemainder(entropy, len(class))]
	}

	for _, character := range required {
		position := takeRemainder(entropy, len(password)+1)
		password = append(password, 0)
		copy(password[position+1:], password[position:])
		password[position] = character
	}

	return string(password), nil
}

func availableCharacters(options Options) ([][]byte, []byte, error) {
	if options.Length < 1 {
		return nil, nil, errors.New("password length must be positive")
	}

	excluded := make(map[byte]bool, len(options.Exclude))
	for _, character := range []byte(options.Exclude) {
		excluded[character] = true
	}

	configured := []struct {
		enabled bool
		name    string
		value   string
	}{
		{options.Lowercase, "lowercase", LowercaseCharacters},
		{options.Uppercase, "uppercase", UppercaseCharacters},
		{options.Digits, "digits", DigitCharacters},
		{options.Symbols, "symbols", SymbolCharacters},
	}

	var classes [][]byte
	var alphabet []byte
	for _, candidate := range configured {
		if !candidate.enabled {
			continue
		}
		filtered := make([]byte, 0, len(candidate.value))
		for i := range len(candidate.value) {
			if !excluded[candidate.value[i]] {
				filtered = append(filtered, candidate.value[i])
			}
		}
		if len(filtered) == 0 {
			return nil, nil, fmt.Errorf("enabled %s character class is empty after exclusions", candidate.name)
		}
		classes = append(classes, filtered)
		alphabet = append(alphabet, filtered...)
	}
	if len(classes) == 0 {
		return nil, nil, errors.New("at least one character class must be enabled")
	}
	return classes, alphabet, nil
}

func takeRemainder(entropy *big.Int, divisor int) int {
	quotient := new(big.Int)
	remainder := new(big.Int)
	quotient.QuoRem(entropy, big.NewInt(int64(divisor)), remainder)
	entropy.Set(quotient)
	return int(remainder.Int64())
}
