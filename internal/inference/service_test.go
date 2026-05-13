package inference

import (
	"context"
	"testing"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

func TestInferDeterministic(t *testing.T) {
	svc := New(0)
	a, err := svc.Infer(context.Background(), domain.InferRequest{CaptureID: "c1"}, []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Infer(context.Background(), domain.InferRequest{CaptureID: "c1"}, []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	if a.Result != b.Result || a.Confidence != b.Confidence {
		t.Fatalf("not deterministic: %+v %+v", a, b)
	}
}

func TestInferRejectsCorrupt(t *testing.T) {
	_, err := New(0).Infer(context.Background(), domain.InferRequest{CaptureID: "c1"}, []byte("CORRUPT"))
	if err != ErrPermanent {
		t.Fatalf("expected permanent rejection, got %v", err)
	}
}
