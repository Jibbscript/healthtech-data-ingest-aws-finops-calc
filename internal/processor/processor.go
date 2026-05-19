package processor

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
	"github.com/jibbscript/throne-backend-poc/internal/inference"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

type Processor struct {
	Store    *store.Store
	Raw      *localaws.BlobStore
	Derived  *localaws.BlobStore
	Queue    *localaws.Queue
	Infer    *inference.Service
	Logger   *slog.Logger
	Deadline time.Duration
}

func New(st *store.Store, raw, derived *localaws.BlobStore, q *localaws.Queue, infer *inference.Service, logger *slog.Logger) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{Store: st, Raw: raw, Derived: derived, Queue: q, Infer: infer, Logger: logger, Deadline: 30 * time.Second}
}

func (p *Processor) Drain(ctx context.Context, limit int) (int, error) {
	msgs, err := p.Queue.Receive(ctx, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, msg := range msgs {
		if err := p.Process(ctx, msg); err != nil {
			p.Logger.Warn("message processing failed", "message_id", msg.ID, "error", err)
			continue
		}
		processed++
	}
	return processed, nil
}

func (p *Processor) Process(ctx context.Context, msg localaws.QueuedMessage) error {
	capture, err := p.Store.GetCapture(msg.Job.CaptureID)
	if err != nil {
		return err
	}
	if capture.Status == domain.CaptureSucceeded {
		return p.Queue.Ack(ctx, msg.ID)
	}
	payload, err := p.Raw.Get(ctx, msg.Job.S3Key)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, p.Deadline)
	defer cancel()
	finding, err := p.Infer.Infer(callCtx, domain.InferRequest{CaptureID: msg.Job.CaptureID, S3Key: msg.Job.S3Key, ModelVersion: p.Infer.ModelVersion, ContentHash: msg.Job.ContentHash}, payload)
	if err != nil {
		if errors.Is(err, inference.ErrPermanent) {
			id, idErr := domain.NewID("finding")
			if idErr != nil {
				return idErr
			}
			failed := domain.Finding{ID: id, CaptureID: capture.ID, UserID: capture.UserID, Result: "failed", Confidence: 0, Reason: err.Error(), CreatedAt: time.Now().UTC()}
			if addErr := p.Store.AddFinding(failed); addErr != nil {
				return addErr
			}
			if updErr := p.Store.UpdateCaptureStatus(capture.ID, domain.CaptureFailed, err.Error()); updErr != nil {
				return updErr
			}
			return p.Queue.Ack(ctx, msg.ID)
		}
		return err
	}
	finding.UserID = capture.UserID
	if err := p.Store.AddFinding(finding); err != nil {
		return err
	}
	if err := p.Store.UpdateCaptureStatus(capture.ID, domain.CaptureSucceeded, ""); err != nil {
		return err
	}
	b, err := json.MarshalIndent(finding, "", "  ")
	if err != nil {
		return err
	}
	if _, err := p.Derived.Put(ctx, "findings/"+capture.ID+".json", b); err != nil {
		return err
	}
	return p.Queue.Ack(ctx, msg.ID)
}
