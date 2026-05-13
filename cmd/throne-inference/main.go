package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
	"github.com/jibbscript/throne-backend-poc/internal/inference"
)

func main() {
	latency := time.Second
	if raw := os.Getenv("INFERENCE_LATENCY_MS"); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil {
			latency = time.Duration(ms) * time.Millisecond
		}
	}
	svc := inference.New(latency)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/v1/infer", func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		finding, err := svc.Infer(r.Context(), domain.InferRequest{CaptureID: r.URL.Query().Get("capture_id")}, payload)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		_ = json.NewEncoder(w).Encode(finding)
	})
	log.Println("throne-inference listening on :50052")
	log.Fatal(http.ListenAndServe(":50052", mux))
}
