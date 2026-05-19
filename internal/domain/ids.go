package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

func NewID(prefix string) (string, error) {
	return newID(prefix, rand.Reader)
}

func newID(prefix string, reader io.Reader) (string, error) {
	var b [12]byte
	if _, err := io.ReadFull(reader, b[:]); err != nil {
		return "", fmt.Errorf("generate %s id: %w", prefix, err)
	}
	return prefix + "_" + hex.EncodeToString(b[:]), nil
}
