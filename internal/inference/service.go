package inference

import (
	"context"
	"crypto/sha256"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

var ErrPermanent = errors.New("permanent inference rejection")

type Service struct {
	Latency      time.Duration
	ModelVersion string
}

func New(latency time.Duration) *Service {
	return &Service{Latency: latency, ModelVersion: "stub-sha256-v1"}
}

func (s *Service) Infer(ctx context.Context, req domain.InferRequest, payload []byte) (domain.Finding, error) {
	if s.Latency > 0 {
		timer := time.NewTimer(s.Latency)
		select {
		case <-ctx.Done():
			timer.Stop()
			return domain.Finding{}, ctx.Err()
		case <-timer.C:
		}
	}
	if len(payload) == 0 || strings.Contains(string(payload), "CORRUPT") {
		return domain.Finding{}, ErrPermanent
	}
	sum := sha256.Sum256(payload)
	confidence := 0.70 + (float64(sum[0]) / 255.0 * 0.29)
	label := "normal"
	if sum[1]%5 == 0 {
		label = "review"
	}
	return domain.Finding{ID: domain.NewID("finding"), CaptureID: req.CaptureID, Result: label, Confidence: math.Round(confidence*1000) / 1000, Reason: "deterministic synthetic finding from payload hash", CreatedAt: time.Now().UTC()}, nil
}
