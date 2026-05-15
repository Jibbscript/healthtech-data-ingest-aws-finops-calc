package localaws

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBlobStorePutGetAndPresign(t *testing.T) {
	ctx := context.Background()
	blob := NewBlobStore(t.TempDir(), "bucket")

	hash, err := blob.Put(ctx, "captures/device-1/cap-1", []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("expected content hash")
	}

	got, err := blob.Get(ctx, "captures/device-1/cap-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "payload" {
		t.Fatalf("payload = %q", got)
	}

	url := blob.Presign("captures/device-1/cap-1", time.Minute)
	if !strings.HasPrefix(url, "local-s3://download?") || !strings.Contains(url, "bucket=bucket") {
		t.Fatalf("unexpected presign URL: %s", url)
	}
}

func TestBlobStoreRejectsDotPrefixedKeysOnWrite(t *testing.T) {
	_, err := NewBlobStore(t.TempDir(), "bucket").Put(context.Background(), "..shadow", []byte("payload"))
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestBlobStoreHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := NewBlobStore(t.TempDir(), "bucket").Put(ctx, "key", []byte("payload")); err == nil {
		t.Fatal("expected cancelled put error")
	}
	if _, err := NewBlobStore(t.TempDir(), "bucket").Get(ctx, "key"); err == nil {
		t.Fatal("expected cancelled get error")
	}
}

func TestBlobStoreDefaultBucketPathStaysUnderDataDir(t *testing.T) {
	dir := t.TempDir()
	blob := NewBlobStore(dir, "")
	if _, err := blob.Put(context.Background(), "a/b", []byte("payload")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir + "/s3/throne-raw-local/a/b"); err != nil {
		t.Fatal(err)
	}
}
