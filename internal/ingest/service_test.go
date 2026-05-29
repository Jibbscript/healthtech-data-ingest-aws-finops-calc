package ingest

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	return New(st, localaws.NewBlobStore(dir, DefaultRawBucket), localaws.NewQueue(dir, "jobs"), nil)
}

func postCapture(t *testing.T, h http.Handler, query string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/captures"+query, bytes.NewReader(body))
	req.Header.Set("x-device-thumbprint", "DEV-THUMBPRINT")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

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

func TestHTTPCaptureClassifiesValidationAs4xx(t *testing.T) {
	svc := newTestService(t)
	h := svc.Handler()

	if rr := postCapture(t, h, "?user_id=user_demo&device_id=device_demo&capture_id=cap-ok", bytes.Repeat([]byte("a"), 1024)); rr.Code != http.StatusOK {
		t.Fatalf("valid capture: want 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := postCapture(t, h, "?user_id=user_demo&device_id=device_demo&capture_id=cap-empty", nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("empty payload: want 400, got %d", rr.Code)
	}
	if rr := postCapture(t, h, "?user_id=user_demo&capture_id=cap-nodev", []byte("x")); rr.Code != http.StatusBadRequest {
		t.Fatalf("missing device: want 400, got %d", rr.Code)
	}

	snap := svc.MetricsSnapshot()
	if snap["2xx"] != 1 || snap["4xx"] != 2 {
		t.Fatalf("unexpected metric buckets: %v", snap)
	}
}

func TestHTTPCaptureClassifiesInfraFailureAs5xx(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	// Point the blob root at a regular file so Blob.Put fails as an infra error.
	badRoot := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(badRoot, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := New(st, localaws.NewBlobStore(badRoot, DefaultRawBucket), localaws.NewQueue(dir, "jobs"), nil)

	rr := postCapture(t, svc.Handler(), "?user_id=user_demo&device_id=device_demo&capture_id=cap-infra", []byte("payload"))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("infra failure: want 500, got %d body=%s", rr.Code, rr.Body.String())
	}
	if svc.MetricsSnapshot()["5xx"] != 1 {
		t.Fatalf("want one 5xx metric, got %v", svc.MetricsSnapshot())
	}
}

func TestCaptureOversizePayloadIsValidationError(t *testing.T) {
	svc := newTestService(t)
	svc.MaxPayload = 8
	_, err := svc.Capture(context.Background(), CaptureRequest{CaptureID: "cap-big", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("far too many bytes")})
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError for oversize payload, got %v", err)
	}
}

func TestCaptureIsIdempotentOnCaptureID(t *testing.T) {
	svc := newTestService(t)
	t1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	ack1, err := svc.Capture(context.Background(), CaptureRequest{CaptureID: "cap-dup", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("payload"), CapturedAt: t1})
	if err != nil {
		t.Fatal(err)
	}
	ack2, err := svc.Capture(context.Background(), CaptureRequest{CaptureID: "cap-dup", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("different"), CapturedAt: t1.Add(48 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if !ack2.Capture.CapturedAt.Equal(ack1.Capture.CapturedAt) || ack2.Capture.ContentHash != ack1.Capture.ContentHash {
		t.Fatalf("expected idempotent capture to return the original record: %+v vs %+v", ack1.Capture, ack2.Capture)
	}
}
