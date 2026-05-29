package processor

import (
	"bytes"
	"context"
	"testing"

	"github.com/jibbscript/throne-backend-poc/internal/inference"
	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func TestDrainCreatesFindingAndDerivedArtifact(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	raw := localaws.NewBlobStore(dir, ingest.DefaultRawBucket)
	q := localaws.NewQueue(dir, "jobs")
	ing := ingest.New(st, raw, q, nil)
	_, err := ing.Capture(context.Background(), ingest.CaptureRequest{CaptureID: "cap1", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: bytes.Repeat([]byte("x"), 1024)})
	if err != nil {
		t.Fatal(err)
	}
	derived := localaws.NewBlobStore(dir, "throne-derived-local")
	p := New(st, raw, derived, q, inference.New(0), nil)
	processed, err := p.Drain(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d", processed)
	}
	capture, _ := st.GetCapture("cap1")
	if capture.Status != "succeeded" {
		t.Fatalf("capture not succeeded: %+v", capture)
	}
	if _, err := derived.Get(context.Background(), "findings/cap1.json"); err != nil {
		t.Fatal(err)
	}
}

func TestDrainFailsCaptureOnPermanentRejection(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	raw := localaws.NewBlobStore(dir, ingest.DefaultRawBucket)
	q := localaws.NewQueue(dir, "jobs")
	ing := ingest.New(st, raw, q, nil)
	if _, err := ing.Capture(context.Background(), ingest.CaptureRequest{CaptureID: "cap-corrupt", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("this payload is CORRUPT")}); err != nil {
		t.Fatal(err)
	}
	derived := localaws.NewBlobStore(dir, "throne-derived-local")
	p := New(st, raw, derived, q, inference.New(0), nil)

	processed, err := p.Drain(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d", processed)
	}
	capture, _ := st.GetCapture("cap-corrupt")
	if capture.Status != "failed" {
		t.Fatalf("want failed capture, got %+v", capture)
	}
	if capture.Failure == "" {
		t.Fatal("expected failure reason to be recorded")
	}
	if _, err := derived.Get(context.Background(), "findings/cap-corrupt.json"); err == nil {
		t.Fatal("no derived artifact should be written for a failed capture")
	}
	// The message was acked, so a second drain finds nothing to redeliver.
	processed, err = p.Drain(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 0 {
		t.Fatalf("want 0 processed on second drain, got %d", processed)
	}
}
