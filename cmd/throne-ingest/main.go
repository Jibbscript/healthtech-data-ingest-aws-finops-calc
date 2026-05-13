package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func main() {
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	st := store.New(dataDir)
	_ = st.SeedDemo()
	svc := ingest.New(st, localaws.NewBlobStore(dataDir, ingest.DefaultRawBucket), localaws.NewQueue(dataDir, "throne-ingest-jobs"), slog.Default())
	log.Println("throne-ingest listening on :8080 (REST gateway) and shaped for :8443 gRPC/mTLS in production")
	log.Fatal(http.ListenAndServe(":8080", svc.Handler()))
}
