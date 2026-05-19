package domain

import (
	"errors"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestNewIDReportsEntropyFailure(t *testing.T) {
	id, err := newID("capture", failingReader{})
	if err == nil {
		t.Fatal("expected entropy failure")
	}
	if id != "" {
		t.Fatalf("id = %q, want empty id on failure", id)
	}
	if !strings.Contains(err.Error(), "capture") {
		t.Fatalf("error %q does not include id prefix", err)
	}
}

func TestNewIDIncludesPrefix(t *testing.T) {
	id, err := newID("capture", strings.NewReader("abcdefghijkl"))
	if err != nil {
		t.Fatalf("newID failed: %v", err)
	}
	if !strings.HasPrefix(id, "capture_") {
		t.Fatalf("id = %q, want capture_ prefix", id)
	}
}
