package localaws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BlobStore struct{ root, bucket string }

func NewBlobStore(dataDir, bucket string) *BlobStore {
	if dataDir == "" {
		dataDir = ".data"
	}
	if bucket == "" {
		bucket = "throne-raw-local"
	}
	return &BlobStore{root: filepath.Join(dataDir, "s3"), bucket: bucket}
}

func (b *BlobStore) Put(ctx context.Context, key string, payload []byte) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	clean := cleanKey(key)
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid key %q", key)
	}
	path := filepath.Join(b.root, b.bucket, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum[:]), nil
}

func (b *BlobStore) Get(ctx context.Context, key string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return os.ReadFile(filepath.Join(b.root, b.bucket, cleanKey(key)))
}

func (b *BlobStore) Presign(key string, ttl time.Duration) string {
	v := url.Values{}
	v.Set("bucket", b.bucket)
	v.Set("key", key)
	v.Set("expires", time.Now().Add(ttl).UTC().Format(time.RFC3339))
	return "local-s3://download?" + v.Encode()
}

func cleanKey(key string) string { return filepath.Clean("/" + key)[1:] }
