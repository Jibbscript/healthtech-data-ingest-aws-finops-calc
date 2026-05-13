package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jibbscript/throne-backend-poc/internal/api"
	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func main() {
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	secret := os.Getenv("THRONE_JWT_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}
	st := store.New(dataDir)
	_ = st.SeedDemo()
	srv := &api.Server{Store: st, Raw: localaws.NewBlobStore(dataDir, ingest.DefaultRawBucket), Secret: secret}
	log.Println("throne-api listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", srv.Handler()))
}
