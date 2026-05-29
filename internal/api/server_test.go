package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func seedCapture(t *testing.T, dir string) (*store.Store, *localaws.BlobStore) {
	t.Helper()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	raw := localaws.NewBlobStore(dir, ingest.DefaultRawBucket)
	if _, err := ingest.New(st, raw, localaws.NewQueue(dir, "jobs"), nil).Capture(context.Background(), ingest.CaptureRequest{CaptureID: "cap-api", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("payload")}); err != nil {
		t.Fatal(err)
	}
	return st, raw
}

func TestCrossUserReadIsForbidden(t *testing.T) {
	st, raw := seedCapture(t, t.TempDir())
	token, _ := SignHS256("s", Claims{UserID: "someone_else"})
	h := (&Server{Store: st, Raw: raw, Secret: "s"}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/v1/captures/cap-api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 for cross-user read, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	st, raw := seedCapture(t, t.TempDir())
	token, _ := SignHS256("s", Claims{UserID: "user_demo", Exp: time.Now().Add(-time.Hour).Unix()})
	h := (&Server{Store: st, Raw: raw, Secret: "s"}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/v1/captures/cap-api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 for expired token, got %d", rr.Code)
	}
}

func TestTamperedSignatureIsRejected(t *testing.T) {
	token, _ := SignHS256("s", Claims{UserID: "user_demo"})
	last := token[len(token)-1]
	flip := byte('A')
	if last == 'A' {
		flip = 'B'
	}
	tampered := token[:len(token)-1] + string(flip)
	if _, err := VerifyHS256("s", tampered); err == nil {
		t.Fatal("expected tampered signature to be rejected")
	}
}

func TestJWTAndCaptureAuthorization(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	raw := localaws.NewBlobStore(dir, ingest.DefaultRawBucket)
	_, err := ingest.New(st, raw, localaws.NewQueue(dir, "jobs"), nil).Capture(context.Background(), ingest.CaptureRequest{CaptureID: "cap-api", UserID: "user_demo", DeviceID: "device_demo", Thumbprint: "DEV-THUMBPRINT", Data: []byte("payload")})
	if err != nil {
		t.Fatal(err)
	}
	token, _ := SignHS256("s", Claims{UserID: "user_demo"})
	h := (&Server{Store: st, Raw: raw, Secret: "s"}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/v1/captures/cap-api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("cap-api")) {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/v1/captures/cap-api/image-url", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("local-s3://download")) {
		t.Fatalf("image-url code=%d body=%s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/captures/cap-api", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestVerifyRejectsMissingExpiration(t *testing.T) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, _ := json.Marshal(Claims{UserID: "user_demo"})
	unsigned := b64(header) + "." + b64(payload)
	token := unsigned + "." + b64(mac("s", unsigned))

	if _, err := VerifyHS256("s", token); err == nil {
		t.Fatal("expected missing exp to be rejected")
	}
}

func TestMissingServerSecretDoesNotAcceptTokens(t *testing.T) {
	token, _ := SignHS256("s", Claims{UserID: "user_demo"})
	h := (&Server{Store: store.New(t.TempDir()), Raw: localaws.NewBlobStore(t.TempDir(), ingest.DefaultRawBucket)}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/v1/captures/cap-api", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 for missing server secret, got %d", rr.Code)
	}
}
