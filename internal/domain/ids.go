package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func NewID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-fallback", prefix)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
