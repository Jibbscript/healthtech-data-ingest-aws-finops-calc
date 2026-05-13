package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func TestJWTAndCaptureAuthorization(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	_ = st.SeedDemo()
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
	req = httptest.NewRequest(http.MethodGet, "/v1/captures/cap-api", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}
