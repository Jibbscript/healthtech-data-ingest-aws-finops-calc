package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
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
	metrics := &inferenceMetrics{}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/infer", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() { metrics.observe(time.Since(start)) }()
		r.Body = http.MaxBytesReader(w, r.Body, int64(inferMaxPayload))
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		finding, err := svc.Infer(r.Context(), domain.InferRequest{CaptureID: r.URL.Query().Get("capture_id")}, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		_ = json.NewEncoder(w).Encode(finding)
	})
	mux.HandleFunc("/metrics", metrics.handler)
	log.Println("throne-inference listening on :50052")
	log.Fatal(http.ListenAndServe(":50052", mux))
}

const inferMaxPayload = 20 << 20

// inferenceBuckets are the histogram upper bounds, in seconds.
var inferenceBuckets = []float64{0.1, 0.5, 1, 5, 10, 30, 60}

// maxLatencySamples bounds the reservoir used to estimate p99 so the collector
// cannot grow without limit under sustained load.
const maxLatencySamples = 4096

type inferenceMetrics struct {
	mu           sync.Mutex
	count        int64
	sum          float64
	bucketCounts []int64
	samples      []float64
	sampleCursor int
}

func (m *inferenceMetrics) observe(d time.Duration) {
	seconds := d.Seconds()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	m.sum += seconds
	if m.bucketCounts == nil {
		m.bucketCounts = make([]int64, len(inferenceBuckets))
	}
	for i, bucket := range inferenceBuckets {
		if seconds <= bucket {
			m.bucketCounts[i]++
		}
	}
	if len(m.samples) < maxLatencySamples {
		m.samples = append(m.samples, seconds)
	} else {
		m.samples[m.sampleCursor] = seconds
		m.sampleCursor = (m.sampleCursor + 1) % maxLatencySamples
	}
}

func (m *inferenceMetrics) handler(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w.Header().Set("content-type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte("# HELP throne_inference_duration_seconds Inference request duration\n# TYPE throne_inference_duration_seconds histogram\n"))
	for i, bucket := range inferenceBuckets {
		var count int64
		if i < len(m.bucketCounts) {
			count = m.bucketCounts[i]
		}
		_, _ = fmt.Fprintf(w, "throne_inference_duration_seconds_bucket{le=\"%.1f\"} %d\n", bucket, count)
	}
	_, _ = fmt.Fprintf(w, "throne_inference_duration_seconds_bucket{le=\"+Inf\"} %d\n", m.count)
	_, _ = fmt.Fprintf(w, "throne_inference_duration_seconds_sum %.6f\n", m.sum)
	_, _ = fmt.Fprintf(w, "throne_inference_duration_seconds_count %d\n", m.count)
	_, _ = fmt.Fprintf(w, "throne_inference_duration_seconds_p99 %.6f\n", percentile(m.samples, 0.99))
}

func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	idx := int(float64(len(cp)-1) * q)
	return cp[idx]
}
