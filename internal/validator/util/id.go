package util

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewRunID returns a unique run identifier using timestamp and random bytes.
func NewRunID() string {
	return time.Now().UTC().Format("20060102-150405") + "-" + randomHex(4)
}

// RandomName generates a deterministic-looking suffix for resources.
func RandomName(prefix string) string {
	return prefix + "-" + randomHex(3)
}

func randomHex(nBytes int) string {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "rnd"
	}
	return hex.EncodeToString(buf)
}
