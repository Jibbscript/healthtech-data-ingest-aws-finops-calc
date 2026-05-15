package ingest

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	thronev1 "github.com/jibbscript/throne-backend-poc/internal/gen/throne/v1"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGRPCCaptureWritesBlobAndQueue(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	if err := st.SeedDemo(); err != nil {
		t.Fatal(err)
	}
	q := localaws.NewQueue(dir, "jobs")
	svc := New(st, localaws.NewBlobStore(dir, DefaultRawBucket), q, nil)

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	RegisterGRPC(grpcServer, svc)
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := thronev1.NewIngestServiceClient(conn)
	streamCtx := metadata.AppendToOutgoingContext(ctx, "x-device-thumbprint", "DEV-THUMBPRINT")
	stream, err := client.Capture(streamCtx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&thronev1.CaptureChunk{Payload: &thronev1.CaptureChunk_Metadata{Metadata: &thronev1.CaptureMetadata{
		CaptureUuid: "cap-grpc",
		UserId:      "user_demo",
		DeviceId:    "device_demo",
		CapturedAt:  timestamppb.New(time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)),
	}}}); err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&thronev1.CaptureChunk{Payload: &thronev1.CaptureChunk_DataChunk{DataChunk: bytes.Repeat([]byte("g"), 1024)}}); err != nil {
		t.Fatal(err)
	}
	ack, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatal(err)
	}
	if ack.GetCaptureId() != "cap-grpc" || ack.GetContentHash() == "" {
		t.Fatalf("bad ack: %+v", ack)
	}
	msgs, err := q.Receive(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Job.CaptureID != "cap-grpc" {
		t.Fatalf("unexpected queue messages: %+v", msgs)
	}
}
