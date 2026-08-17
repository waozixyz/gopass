package gopass

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

const derivationRounds = 100_000

func deriveKey(password, salt []byte) []byte {
	blockInput := make([]byte, len(salt)+4)
	copy(blockInput, salt)
	binary.BigEndian.PutUint32(blockInput[len(salt):], 1)

	mac := hmac.New(sha256.New, password)
	_, _ = mac.Write(blockInput)
	previous := mac.Sum(nil)
	key := append([]byte(nil), previous...)

	for round := 1; round < derivationRounds; round++ {
		mac.Reset()
		_, _ = mac.Write(previous)
		previous = mac.Sum(previous[:0])
		for i := range key {
			key[i] ^= previous[i]
		}
	}
	return key
}
