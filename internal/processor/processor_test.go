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
	p := New(st, raw, localaws.NewBlobStore(dir, "throne-derived-local"), q, inference.New(0), nil)
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
	if _, err := st.FindFindingByCapture("cap1"); err != nil {
		t.Fatal(err)
	}
}
