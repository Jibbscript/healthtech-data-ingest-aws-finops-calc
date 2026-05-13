package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/jibbscript/throne-backend-poc/internal/cost"
)

func main() {
	refresh := flag.Bool("refresh-pricing", false, "write bundled pricing snapshot and exit")
	flag.Parse()
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".cache"
	}
	svc := cost.New(dataDir)
	if *refresh {
		path, err := svc.RefreshSnapshot()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s", path)
		return
	}
	log.Println("throne-cost-api listening on :9000")
	log.Fatal(http.ListenAndServe(":9000", svc.Handler()))
}
