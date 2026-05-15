package ingest

import (
	"context"
	"errors"
	"io"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
	thronev1 "github.com/jibbscript/throne-backend-poc/internal/gen/throne/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	thronev1.UnimplementedIngestServiceServer
	Service *Service
}

func RegisterGRPC(reg grpc.ServiceRegistrar, svc *Service) {
	thronev1.RegisterIngestServiceServer(reg, &GRPCServer{Service: svc})
}

func (s *GRPCServer) Capture(stream grpc.ClientStreamingServer[thronev1.CaptureChunk, thronev1.CaptureAck]) error {
	if s.Service == nil {
		return status.Error(codes.FailedPrecondition, "ingest service not configured")
	}
	var req CaptureRequest
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if req.Thumbprint == "" {
				req.Thumbprint = thumbprintFromContext(stream.Context())
			}
			ack, err := s.Service.Capture(stream.Context(), req)
			if err != nil {
				s.Service.RecordRequest("4xx")
				return status.Error(codes.InvalidArgument, err.Error())
			}
			s.Service.RecordRequest("2xx")
			return stream.SendAndClose(&thronev1.CaptureAck{
				CaptureId:   ack.Capture.ID,
				S3Key:       ack.S3Key,
				ContentHash: ack.ContentHash,
				Capture:     captureToProto(ack.Capture),
			})
		}
		if err != nil {
			s.Service.RecordRequest("5xx")
			return status.Error(codes.Internal, "failed to read capture stream")
		}
		switch payload := chunk.Payload.(type) {
		case *thronev1.CaptureChunk_Metadata:
			req.CaptureID = payload.Metadata.GetCaptureUuid()
			req.UserID = payload.Metadata.GetUserId()
			req.DeviceID = payload.Metadata.GetDeviceId()
			req.CapturedAt = payload.Metadata.GetCapturedAt().AsTime()
		case *thronev1.CaptureChunk_DataChunk:
			if len(req.Data)+len(payload.DataChunk) > s.Service.maxPayload() {
				s.Service.RecordRequest("4xx")
				return status.Errorf(codes.ResourceExhausted, "payload exceeds max %d bytes", s.Service.maxPayload())
			}
			req.Data = append(req.Data, payload.DataChunk...)
		default:
			s.Service.RecordRequest("4xx")
			return status.Error(codes.InvalidArgument, "capture chunk payload is required")
		}
	}
}

func thumbprintFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	for _, key := range []string{"x-device-thumbprint", "device-thumbprint"} {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func captureToProto(c domain.Capture) *thronev1.Capture {
	return &thronev1.Capture{
		Id:          c.ID,
		UserId:      c.UserID,
		DeviceId:    c.DeviceID,
		S3Key:       c.S3Key,
		ContentHash: c.ContentHash,
		SizeBytes:   c.SizeBytes,
		Status:      string(c.Status),
		CapturedAt:  timestamppb.New(c.CapturedAt),
	}
}
