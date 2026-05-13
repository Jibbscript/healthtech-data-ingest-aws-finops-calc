package loadsim

import (
	"bytes"
	"crypto/rand"
	"math"
	mrand "math/rand"
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
}

type Result struct {
	Sent      int
	Failed    int
	Latencies []time.Duration
}

func PayloadSize(seed int64, meanBytes int) int {
	if meanBytes <= 0 {
		meanBytes = 5 * 1024 * 1024
	}
	r := mrand.New(mrand.NewSource(seed))
	size := int(math.Exp(r.NormFloat64()*0.35 + math.Log(float64(meanBytes))))
	if size < 1024 {
		size = 1024
	}
	return size
}

func RandomPayload(n int) []byte { b := make([]byte, n); _, _ = rand.Read(b); return b }

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
	deadline := time.Now().Add(cfg.Duration)
	var mu sync.Mutex
	result := Result{}
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
				payload := RandomPayload(cfg.PayloadBytes)
				req, _ := http.NewRequest(http.MethodPost, cfg.Target+"/v1/captures?user_id=user_demo&device_id=device_demo", bytes.NewReader(payload))
				req.Header.Set("x-device-thumbprint", "DEV-THUMBPRINT")
				resp, err := http.DefaultClient.Do(req)
				elapsed := time.Since(start)
				mu.Lock()
				success := err == nil && resp.StatusCode < 300
				if !success {
					result.Failed++
				} else {
					result.Sent++
				}
				result.Latencies = append(result.Latencies, elapsed)
				mu.Unlock()
				if cfg.OnResult != nil {
					cfg.OnResult(success, elapsed)
				}
				if resp != nil {
					_ = resp.Body.Close()
				}
			}
		}(d)
	}
	wg.Wait()
	return result
}
