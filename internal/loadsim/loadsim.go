package loadsim

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	Devices       int
	RatePerDevice float64
	Duration      time.Duration
	Target        string
	PayloadBytes  int
	OnResult      func(success bool, latency time.Duration)
	RandomReader  io.Reader
}

type Result struct {
	Sent   int
	Failed int
}

func randomPayload(n int, reader io.Reader) ([]byte, error) {
	if reader == nil {
		reader = rand.Reader
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(reader, b); err != nil {
		return nil, fmt.Errorf("generate random payload: %w", err)
	}
	return b, nil
}

func Run(cfg Config) Result {
	if cfg.Devices <= 0 {
		cfg.Devices = 1
	}
	if cfg.RatePerDevice <= 0 {
		cfg.RatePerDevice = 1
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}
	if cfg.Target == "" {
		cfg.Target = "http://localhost:8080"
	}
	if cfg.PayloadBytes <= 0 {
		cfg.PayloadBytes = 1024
	}
	client := &http.Client{Timeout: 30 * time.Second}
	deadline := time.Now().Add(cfg.Duration)
	randomReader := cfg.RandomReader
	if randomReader == nil {
		randomReader = rand.Reader
	}
	var mu sync.Mutex
	result := Result{}
	record := func(success bool, latency time.Duration) {
		mu.Lock()
		if !success {
			result.Failed++
		} else {
			result.Sent++
		}
		mu.Unlock()
		if cfg.OnResult != nil {
			cfg.OnResult(success, latency)
		}
	}
	var wg sync.WaitGroup
	for d := 0; d < cfg.Devices; d++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			interval := time.Duration(float64(time.Second) / cfg.RatePerDevice)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for time.Now().Before(deadline) {
				<-ticker.C
				start := time.Now()
				payload, err := randomPayload(cfg.PayloadBytes, randomReader)
				if err != nil {
					record(false, time.Since(start))
					continue
				}
				req, err := http.NewRequest(http.MethodPost, cfg.Target+"/v1/captures?user_id=user_demo&device_id=device_demo", bytes.NewReader(payload))
				if err != nil {
					record(false, time.Since(start))
					continue
				}
				req.Header.Set("x-device-thumbprint", "DEV-THUMBPRINT")
				resp, err := client.Do(req)
				elapsed := time.Since(start)
				success := err == nil && resp.StatusCode < 300
				record(success, elapsed)
				if resp != nil {
					_ = resp.Body.Close()
				}
			}
		}(d)
	}
	wg.Wait()
	return result
}
