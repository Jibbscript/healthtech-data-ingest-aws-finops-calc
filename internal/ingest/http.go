package ingest

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/captures", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(s.maxPayload()+1)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req := CaptureRequest{CaptureID: r.URL.Query().Get("capture_id"), UserID: r.URL.Query().Get("user_id"), DeviceID: r.URL.Query().Get("device_id"), Data: body, CapturedAt: time.Now().UTC(), Traceparent: r.Header.Get("traceparent"), Thumbprint: r.Header.Get("x-device-thumbprint")}
		ack, err := s.Capture(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(ack)
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("# HELP throne_ingest_up Ingest service up\n# TYPE throne_ingest_up gauge\nthrone_ingest_up 1\n"))
	})
	return mux
}
