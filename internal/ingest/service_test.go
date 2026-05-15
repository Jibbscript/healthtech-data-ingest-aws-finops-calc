package ingest

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func TestCaptureWritesBlobAndQueue(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	svc := New(st, localaws.NewBlobStore(dir, DefaultRawBucket), localaws.NewQueue(dir, "jobs"), nil)
	payload := bytes.Repeat([]byte("a"), 5*1024*1024)
	ack, err := svc.Capture(context.Background(), CaptureRequest{CaptureID: "cap-test", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: payload, CapturedAt: time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if ack.Capture.ID != "cap-test" || ack.ContentHash == "" {
		t.Fatalf("bad ack: %+v", ack)
	}
	msgs, err := localaws.NewQueue(dir, "jobs").Receive(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Job.CaptureID != "cap-test" {
		t.Fatalf("unexpected queue messages: %+v", msgs)
	}
}

func TestCaptureRequiresMatchingDeviceCertificateAndOwner(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	svc := New(st, localaws.NewBlobStore(dir, DefaultRawBucket), localaws.NewQueue(dir, "jobs"), nil)

	for name, req := range map[string]CaptureRequest{
		"missing thumbprint": {CaptureID: "cap-no-thumb", UserID: "user_demo", DeviceID: "device_demo", Data: []byte("payload")},
		"wrong owner":        {CaptureID: "cap-wrong-owner", UserID: "other_user", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("payload")},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Capture(context.Background(), req); err == nil {
				t.Fatal("expected device authorization failure")
			}
		})
	}
}
