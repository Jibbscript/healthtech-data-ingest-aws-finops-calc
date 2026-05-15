package main

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

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
	st := store.New(dataDir)
	_ = st.SeedDemo()
	svc := ingest.New(st, localaws.NewBlobStore(dataDir, ingest.DefaultRawBucket), localaws.NewQueue(dataDir, "throne-ingest-jobs"), slog.Default())
	go serveGRPC(svc)
	log.Println("throne-ingest HTTP compatibility endpoint listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", svc.Handler()))
}

func serveGRPC(svc *ingest.Service) {
	lis, err := net.Listen("tcp", ":8443")
	if err != nil {
		log.Printf("throne-ingest gRPC listener unavailable: %v", err)
		return
	}
	grpcServer := grpc.NewServer()
	ingest.RegisterGRPC(grpcServer, svc)
	log.Println("throne-ingest gRPC listening on :8443")
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("throne-ingest gRPC stopped: %v", err)
	}
}
