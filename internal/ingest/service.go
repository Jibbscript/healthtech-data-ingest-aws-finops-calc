package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

const (
	MaxChunkSize      = 1 << 20
	DefaultMaxPayload = 20 << 20
	DefaultRawBucket  = "throne-raw-local"
)

type CaptureRequest struct {
	CaptureID   string
	UserID      string
	DeviceID    string
	Data        []byte
	CapturedAt  time.Time
	Traceparent string
	Thumbprint  string
}

type CaptureAck struct {
	Capture     domain.Capture `json:"capture"`
	S3Key       string         `json:"s3_key"`
	ContentHash string         `json:"content_hash"`
}

type Service struct {
	Store      *store.Store
	Blob       *localaws.BlobStore
	Queue      *localaws.Queue
	MaxPayload int
	Logger     *slog.Logger
}

func New(st *store.Store, blob *localaws.BlobStore, q *localaws.Queue, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{Store: st, Blob: blob, Queue: q, MaxPayload: DefaultMaxPayload, Logger: logger}
}

func (s *Service) Capture(ctx context.Context, req CaptureRequest) (CaptureAck, error) {
	if req.UserID == "" || req.DeviceID == "" {
		return CaptureAck{}, errors.New("user_id and device_id are required")
	}
	if len(req.Data) == 0 {
		return CaptureAck{}, errors.New("payload is empty")
	}
	if len(req.Data) > s.maxPayload() {
		return CaptureAck{}, fmt.Errorf("payload exceeds max %d bytes", s.maxPayload())
	}
	if !s.Store.DeviceAllowed(req.DeviceID, req.Thumbprint) {
		return CaptureAck{}, errors.New("device certificate is not allowlisted")
	}
	if req.CaptureID == "" {
		req.CaptureID = domain.NewID("capture")
	}
	if req.CapturedAt.IsZero() {
		req.CapturedAt = time.Now().UTC()
	}
	key := captureKey(req.DeviceID, req.CapturedAt, req.CaptureID)
	hash, err := s.Blob.Put(ctx, key, req.Data)
	if err != nil {
		return CaptureAck{}, err
	}
	capture := domain.Capture{ID: req.CaptureID, UserID: req.UserID, DeviceID: req.DeviceID, S3Key: key, ContentHash: hash, SizeBytes: int64(len(req.Data)), Status: domain.CapturePending, CapturedAt: req.CapturedAt}
	capture, existed, err := s.Store.AddCapture(capture)
	if err != nil {
		return CaptureAck{}, err
	}
	if !existed || capture.Status != domain.CaptureSucceeded {
		job := domain.IngestJob{CaptureID: capture.ID, DeviceID: capture.DeviceID, UserID: capture.UserID, S3Key: capture.S3Key, ContentHash: capture.ContentHash, CapturedAt: capture.CapturedAt, Traceparent: req.Traceparent}
		_, err = s.Queue.Send(ctx, job, map[string]string{"traceparent": req.Traceparent})
		if err != nil {
			return CaptureAck{}, err
		}
	}
	s.Logger.Info("capture accepted", "capture_id", capture.ID, "device_id", capture.DeviceID, "bytes", capture.SizeBytes)
	return CaptureAck{Capture: capture, S3Key: capture.S3Key, ContentHash: capture.ContentHash}, nil
}

func (s *Service) maxPayload() int {
	if s.MaxPayload <= 0 {
		return DefaultMaxPayload
	}
	return s.MaxPayload
}

func ValidateChunk(chunk []byte) error {
	if len(chunk) > MaxChunkSize {
		return fmt.Errorf("chunk exceeds max %d bytes", MaxChunkSize)
	}
	return nil
}

func captureKey(deviceID string, capturedAt time.Time, captureID string) string {
	safeDevice := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, deviceID)
	return path.Join("captures", safeDevice, capturedAt.Format("2006"), capturedAt.Format("01"), capturedAt.Format("02"), captureID)
}
