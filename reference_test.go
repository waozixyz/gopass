package pass

import (
	"crypto/hmac"
	"crypto/sha256"
	"math/big"
	"strconv"
	"strings"
)

// referenceGenerate is intentionally self-contained so the examples exercise
// the public generator against a second implementation of the specification.
func referenceGenerate(site, login, master string, options Options) string {
	definitions := []struct {
		on  bool
		set string
	}{
		{options.Lowercase, "abcdefghijklmnopqrstuvwxyz"},
		{options.Uppercase, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{options.Digits, "0123456789"},
		{options.Symbols, "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"},
	}
	classes := make([]string, 0, 4)
	pool := ""
	for _, definition := range definitions {
		if !definition.on {
			continue
		}
		filtered := ""
		for _, character := range definition.set {
			if !strings.ContainsRune(options.Exclude, character) {
				filtered += string(character)
			}
		}
		classes = append(classes, filtered)
		pool += filtered
	}

	salt := []byte(site + login + strconv.FormatUint(options.Counter, 10))
	message := append(append([]byte(nil), salt...), 0, 0, 0, 1)
	mac := hmac.New(sha256.New, []byte(master))
	_, _ = mac.Write(message)
	last := mac.Sum(nil)
	derived := append([]byte(nil), last...)
	for iteration := 2; iteration <= 100_000; iteration++ {
		mac.Reset()
		_, _ = mac.Write(last)
		last = mac.Sum(nil)
		for i, value := range last {
			derived[i] ^= value
		}
	}

	value := new(big.Int).SetBytes(derived)
	draw := func(size int) int {
		base := big.NewInt(int64(size))
		remainder := new(big.Int)
		value.DivMod(value, base, remainder)
		return int(remainder.Int64())
	}

	result := make([]byte, options.Length-len(classes))
	for i := range result {
		result[i] = pool[draw(len(pool))]
	}
	required := make([]byte, len(classes))
	for i, class := range classes {
		required[i] = class[draw(len(class))]
	}
	for _, character := range required {
		at := draw(len(result) + 1)
		next := make([]byte, 0, len(result)+1)
		next = append(next, result[:at]...)
		next = append(next, character)
		next = append(next, result[at:]...)
		result = next
	}
	return string(result)
}
