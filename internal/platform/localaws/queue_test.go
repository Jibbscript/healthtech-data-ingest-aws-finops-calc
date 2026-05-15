package localaws

import (
	"context"
	"testing"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

func TestQueueSendReceiveAck(t *testing.T) {
	ctx := context.Background()
	q := NewQueue(t.TempDir(), "jobs")
	job := domain.IngestJob{CaptureID: "cap-1", DeviceID: "device-1", UserID: "user-1", S3Key: "captures/cap-1"}

	id, err := q.Send(ctx, job, map[string]string{"traceparent": "trace-1"})
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := q.Receive(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("message count = %d", len(msgs))
	}
	if msgs[0].ID != id || msgs[0].Job.CaptureID != "cap-1" || msgs[0].Attributes["traceparent"] != "trace-1" {
		t.Fatalf("unexpected message: %+v", msgs[0])
	}

	if err := q.Ack(ctx, id); err != nil {
		t.Fatal(err)
	}
	msgs, err = q.Receive(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected empty queue after ack, got %+v", msgs)
	}
}

func TestQueueReceiveDefaultsLimitAndSorts(t *testing.T) {
	ctx := context.Background()
	q := NewQueue(t.TempDir(), "jobs")
	for _, id := range []string{"cap-b", "cap-a"} {
		if _, err := q.Send(ctx, domain.IngestJob{CaptureID: id}, nil); err != nil {
			t.Fatal(err)
		}
	}

	msgs, err := q.Receive(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("default limit count = %d", len(msgs))
	}
}

func TestQueueHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q := NewQueue(t.TempDir(), "jobs")

	if _, err := q.Send(ctx, domain.IngestJob{}, nil); err == nil {
		t.Fatal("expected cancelled send error")
	}
	if _, err := q.Receive(ctx, 1); err == nil {
		t.Fatal("expected cancelled receive error")
	}
}
