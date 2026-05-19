package loadsim

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunSendsConfiguredPayloads(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/captures" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("x-device-thumbprint") != "DEV-THUMBPRINT" {
			t.Error("missing device thumbprint")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if len(body) != 32 {
			t.Errorf("payload bytes = %d", len(body))
		}
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	result := Run(Config{Devices: 1, RatePerDevice: 20, Duration: 120 * time.Millisecond, Target: server.URL, PayloadBytes: 32})
	if result.Sent == 0 {
		t.Fatal("expected at least one sent request")
	}
	if result.Failed != 0 {
		t.Fatalf("failed requests = %d", result.Failed)
	}
	if requests.Load() != int64(result.Sent) {
		t.Fatalf("requests=%d sent=%d", requests.Load(), result.Sent)
	}
}

func TestRandomPayloadReportsEntropyFailure(t *testing.T) {
	payload, err := randomPayload(32, failingReader{})
	if err == nil {
		t.Fatal("expected entropy failure")
	}
	if payload != nil {
		t.Fatalf("payload = %v, want nil on failure", payload)
	}
}

func TestRunCountsPayloadGenerationFailure(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	result := Run(Config{
		Devices:       1,
		RatePerDevice: 50,
		Duration:      80 * time.Millisecond,
		Target:        server.URL,
		PayloadBytes:  32,
		RandomReader:  failingReader{},
	})

	if result.Sent != 0 {
		t.Fatalf("sent = %d, want zero when payload generation fails", result.Sent)
	}
	if result.Failed == 0 {
		t.Fatal("expected payload generation failures to be counted")
	}
	if requests.Load() != 0 {
		t.Fatalf("requests = %d, want none when payload generation fails", requests.Load())
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}
