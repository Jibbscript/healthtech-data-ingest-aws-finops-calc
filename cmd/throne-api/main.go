package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/jibbscript/throne-backend-poc/internal/api"
	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
	"google.golang.org/grpc"
)

func main() {
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	secret := os.Getenv("THRONE_JWT_SECRET")
	if secret == "" {
		if !allowDevAuth() {
			log.Fatal("THRONE_JWT_SECRET is required; set THRONE_ALLOW_DEV_AUTH=true only for local demos")
		}
		log.Println("warning: using local demo JWT secret")
		secret = "dev-secret"
	}
	st := store.New(dataDir)
	_ = st.SeedDemo()
	srv := &api.Server{Store: st, Raw: localaws.NewBlobStore(dataDir, ingest.DefaultRawBucket), Secret: secret}
	go serveGRPC(srv)
	log.Println("throne-api REST gateway listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", srv.Handler()))
}

func allowDevAuth() bool {
	switch strings.ToLower(os.Getenv("THRONE_ALLOW_DEV_AUTH")) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func serveGRPC(srv *api.Server) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Printf("throne-api gRPC listener unavailable: %v", err)
		return
	}
	grpcServer := grpc.NewServer()
	srv.RegisterGRPC(grpcServer)
	log.Println("throne-api gRPC listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("throne-api gRPC stopped: %v", err)
	}
}
