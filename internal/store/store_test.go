package store

import (
	"errors"
	"testing"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

func TestAddCaptureIsIdempotent(t *testing.T) {
	st := New(t.TempDir())
	first, existed, err := st.AddCapture(domain.Capture{
		ID:         "cap-1",
		UserID:     "user-1",
		DeviceID:   "device-1",
		Status:     domain.CapturePending,
		CapturedAt: time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if existed {
		t.Fatal("first insert should not be marked existing")
	}
	if err := st.UpdateCaptureStatus(first.ID, domain.CaptureSucceeded, ""); err != nil {
		t.Fatal(err)
	}

	second, existed, err := st.AddCapture(domain.Capture{
		ID:         "cap-1",
		UserID:     "user-1",
		DeviceID:   "device-1",
		Status:     domain.CapturePending,
		CapturedAt: first.CapturedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !existed {
		t.Fatal("duplicate insert should be marked existing")
	}
	if second.Status != domain.CaptureSucceeded || !second.CapturedAt.Equal(first.CapturedAt) {
		t.Fatalf("duplicate insert overwrote stored capture: %+v", second)
	}
}

func TestListCapturesByUserSortsAndPaginates(t *testing.T) {
	st := New(t.TempDir())
	base := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)
	for _, capture := range []domain.Capture{
		{ID: "oldest", UserID: "user-1", DeviceID: "device-1", CapturedAt: base},
		{ID: "middle", UserID: "user-1", DeviceID: "device-1", CapturedAt: base.Add(time.Hour)},
		{ID: "newest", UserID: "user-1", DeviceID: "device-1", CapturedAt: base.Add(2 * time.Hour)},
		{ID: "other-user", UserID: "user-2", DeviceID: "device-1", CapturedAt: base.Add(3 * time.Hour)},
	} {
		if _, _, err := st.AddCapture(capture); err != nil {
			t.Fatal(err)
		}
	}

	firstPage, cursor, err := st.ListCapturesByUser("user-1", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := captureIDs(firstPage); got != "newest,middle" {
		t.Fatalf("first page order = %s", got)
	}
	if cursor != "middle" {
		t.Fatalf("cursor = %q", cursor)
	}

	secondPage, cursor, err := st.ListCapturesByUser("user-1", 2, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if got := captureIDs(secondPage); got != "oldest" {
		t.Fatalf("second page order = %s", got)
	}
	if cursor != "" {
		t.Fatalf("final cursor = %q", cursor)
	}
}

func TestUpdateCaptureStatusMissingCapture(t *testing.T) {
	err := New(t.TempDir()).UpdateCaptureStatus("missing", domain.CaptureFailed, "boom")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func captureIDs(captures []domain.Capture) string {
	out := ""
	for i, capture := range captures {
		if i > 0 {
			out += ","
		}
		out += capture.ID
	}
	return out
}
