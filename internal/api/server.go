package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jibbscript/throne-backend-poc/internal/domain"
	thronev1 "github.com/jibbscript/throne-backend-poc/internal/gen/throne/v1"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/protomap"
	"github.com/jibbscript/throne-backend-poc/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	Store  *store.Store
	Raw    *localaws.BlobStore
	Secret string
}

type GRPCServer struct {
	thronev1.UnimplementedThroneAPIServer
	Server *Server
}

func (s *Server) RegisterGRPC(reg grpc.ServiceRegistrar) {
	thronev1.RegisterThroneAPIServer(reg, &GRPCServer{Server: s})
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	gateway := runtime.NewServeMux()
	if err := thronev1.RegisterThroneAPIHandlerServer(context.Background(), gateway, &GRPCServer{Server: s}); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gateway registration failed", http.StatusInternalServerError)
		})
	}
	mux.Handle("/v1/", AuthMiddleware(s.Secret, gateway))
	return mux
}

func (g *GRPCServer) ListCaptures(ctx context.Context, req *thronev1.ListCapturesRequest) (*thronev1.ListCapturesResponse, error) {
	if err := g.requireUser(ctx, req.GetUserId()); err != nil {
		return nil, err
	}
	if err := g.audit(ctx, "api.list_captures", req.GetUserId()); err != nil {
		return nil, err
	}
	items, next, err := g.Server.Store.ListCapturesByUser(req.GetUserId(), int(req.GetPageSize()), req.GetCursor())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	out := make([]*thronev1.Capture, 0, len(items))
	for _, item := range items {
		out = append(out, protomap.CaptureToProto(item))
	}
	return &thronev1.ListCapturesResponse{Captures: out, NextCursor: next}, nil
}

func (g *GRPCServer) GetCapture(ctx context.Context, req *thronev1.GetCaptureRequest) (*thronev1.GetCaptureResponse, error) {
	capture, err := g.Server.Store.GetCapture(req.GetCaptureId())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "capture not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := g.requireUser(ctx, capture.UserID); err != nil {
		return nil, err
	}
	if err := g.audit(ctx, "api.get_capture", req.GetCaptureId()); err != nil {
		return nil, err
	}
	return &thronev1.GetCaptureResponse{Capture: protomap.CaptureToProto(capture)}, nil
}

func (g *GRPCServer) GetFinding(ctx context.Context, req *thronev1.GetFindingRequest) (*thronev1.GetFindingResponse, error) {
	finding, err := g.Server.Store.GetFinding(req.GetFindingId())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "finding not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := g.requireUser(ctx, finding.UserID); err != nil {
		return nil, err
	}
	if err := g.audit(ctx, "api.get_finding", req.GetFindingId()); err != nil {
		return nil, err
	}
	return &thronev1.GetFindingResponse{Finding: findingToProto(finding)}, nil
}

func (g *GRPCServer) CreateImageURL(ctx context.Context, req *thronev1.CreateImageURLRequest) (*thronev1.CreateImageURLResponse, error) {
	capture, err := g.Server.Store.GetCapture(req.GetCaptureId())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "capture not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := g.requireUser(ctx, capture.UserID); err != nil {
		return nil, err
	}
	if err := g.audit(ctx, "api.create_image_url", req.GetCaptureId()); err != nil {
		return nil, err
	}
	return &thronev1.CreateImageURLResponse{Url: g.Server.Raw.Presign(capture.S3Key, 5*time.Minute), ExpiresInSeconds: 300}, nil
}

func (g *GRPCServer) requireUser(ctx context.Context, expected string) error {
	if UserID(ctx) != expected {
		return status.Error(codes.PermissionDenied, "forbidden")
	}
	return nil
}

func (g *GRPCServer) audit(ctx context.Context, action string, resource string) error {
	if err := g.Server.Store.AddAuditEvent(domain.AuditEvent{Actor: UserID(ctx), Action: action, Resource: resource}); err != nil {
		return status.Error(codes.Internal, "internal error")
	}
	return nil
}

func findingToProto(f domain.Finding) *thronev1.Finding {
	return &thronev1.Finding{
		Id:         f.ID,
		CaptureId:  f.CaptureID,
		UserId:     f.UserID,
		Result:     f.Result,
		Confidence: f.Confidence,
		Reason:     f.Reason,
		CreatedAt:  timestamppb.New(f.CreatedAt),
	}
}
