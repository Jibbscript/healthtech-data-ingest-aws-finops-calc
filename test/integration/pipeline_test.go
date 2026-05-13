package integration

import (
	"bytes"
	"context"
	"testing"

	"github.com/jibbscript/throne-backend-poc/internal/inference"
	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/processor"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func TestFullLocalPipeline(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	raw := localaws.NewBlobStore(dir, ingest.DefaultRawBucket)
	q := localaws.NewQueue(dir, "jobs")
	ing := ingest.New(st, raw, q, nil)
	for i := 0; i < 100; i++ {
		if _, err := ing.Capture(context.Background(), ingest.CaptureRequest{CaptureID: "cap-it-" + string(rune('a'+(i%26))) + string(rune('a'+(i/26))), UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: bytes.Repeat([]byte{byte(i)}, 256)}); err != nil {
			t.Fatal(err)
		}
	}
	p := processor.New(st, raw, localaws.NewBlobStore(dir, "throne-derived-local"), q, inference.New(0), nil)
	processed, err := p.Drain(context.Background(), 200)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 100 {
		t.Fatalf("processed=%d", processed)
	}
}
