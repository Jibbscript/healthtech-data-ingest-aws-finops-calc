package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/loadsim"
)

func main() {
	var cfg loadsim.Config
	var dur string
	var metricsAddr string
	var capturesPerDevicePerDay float64
	var sent, failed atomic.Int64
	var totalLatencyMillis atomic.Int64

	flag.IntVar(&cfg.Devices, "devices", 10, "number of simulated devices")
	flag.Float64Var(&cfg.RatePerDevice, "rate-per-device", 1, "captures per second per device")
	flag.Float64Var(&capturesPerDevicePerDay, "captures-per-device-per-day", 0, "capture count per simulated day; compressed into --duration when set")
	flag.StringVar(&dur, "duration", "10s", "duration")
	flag.StringVar(&cfg.Target, "target", "http://localhost:8080", "ingest target")
	flag.IntVar(&cfg.PayloadBytes, "payload-bytes", 1024, "payload size in bytes")
	flag.StringVar(&metricsAddr, "metrics-addr", ":9091", "address for Prometheus load metrics")
	flag.Parse()

	parsedDuration, err := time.ParseDuration(dur)
	if err != nil {
		log.Fatalf("invalid --duration: %v", err)
	}
	cfg.Duration = parsedDuration
	if capturesPerDevicePerDay > 0 {
		cfg.RatePerDevice = capturesPerDevicePerDay / cfg.Duration.Seconds()
	}
	cfg.OnResult = func(success bool, latency time.Duration) {
		if success {
			sent.Add(1)
		} else {
			failed.Add(1)
		}
		totalLatencyMillis.Add(latency.Milliseconds())
	}

	metricsSrv := &http.Server{Addr: metricsAddr, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		s := sent.Load()
		f := failed.Load()
		avg := int64(0)
		if s+f > 0 {
			avg = totalLatencyMillis.Load() / (s + f)
		}
		_, _ = fmt.Fprintf(w, "# HELP throne_load_captures_sent_total Synthetic captures sent successfully\n# TYPE throne_load_captures_sent_total counter\nthrone_load_captures_sent_total %d\n# HELP throne_load_captures_failed_total Synthetic captures that failed\n# TYPE throne_load_captures_failed_total counter\nthrone_load_captures_failed_total %d\n# HELP throne_load_average_latency_ms Average synthetic upload latency\n# TYPE throne_load_average_latency_ms gauge\nthrone_load_average_latency_ms %d\n", s, f, avg)
	})}
	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("load metrics server stopped: %v", err)
		}
	}()

	result := loadsim.Run(cfg)
	_ = metricsSrv.Close()
	fmt.Printf("sent=%d failed=%d duration=%s\n", result.Sent, result.Failed, cfg.Duration)
}
