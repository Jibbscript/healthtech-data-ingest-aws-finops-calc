package localaws

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

type QueuedMessage struct {
	ID         string            `json:"id"`
	Job        domain.IngestJob  `json:"job"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Attempts   int               `json:"attempts"`
	CreatedAt  time.Time         `json:"created_at"`
}

type Queue struct{ dir string }

func NewQueue(dataDir, name string) *Queue {
	if dataDir == "" {
		dataDir = ".data"
	}
	if name == "" {
		name = "throne-ingest-jobs"
	}
	return &Queue{dir: filepath.Join(dataDir, "sqs", name)}
}

func (q *Queue) Send(ctx context.Context, job domain.IngestJob, attrs map[string]string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	if err := os.MkdirAll(filepath.Join(q.dir, "pending"), 0o755); err != nil {
		return "", err
	}
	id, err := domain.NewID("msg")
	if err != nil {
		return "", err
	}
	msg := QueuedMessage{ID: id, Job: job, Attributes: attrs, CreatedAt: time.Now().UTC()}
	b, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return "", err
	}
	return id, os.WriteFile(filepath.Join(q.dir, "pending", id+".json"), b, 0o600)
}

func (q *Queue) Receive(ctx context.Context, limit int) ([]QueuedMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if limit <= 0 {
		limit = 1
	}
	entries, err := os.ReadDir(filepath.Join(q.dir, "pending"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	out := make([]QueuedMessage, 0, limit)
	for _, e := range entries {
		if len(out) >= limit || e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(q.dir, "pending", e.Name()))
		if err != nil {
			return nil, err
		}
		var msg QueuedMessage
		if err := json.Unmarshal(b, &msg); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

func (q *Queue) Ack(_ context.Context, id string) error {
	return os.Remove(filepath.Join(q.dir, "pending", id+".json"))
}
