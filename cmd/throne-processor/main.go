package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/inference"
	"github.com/jibbscript/throne-backend-poc/internal/ingest"
	"github.com/jibbscript/throne-backend-poc/internal/platform/localaws"
	"github.com/jibbscript/throne-backend-poc/internal/processor"
	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func main() {
	once := flag.Bool("once", false, "run one drain and exit")
	flag.Parse()
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	p := processor.New(store.New(dataDir), localaws.NewBlobStore(dataDir, ingest.DefaultRawBucket), localaws.NewBlobStore(dataDir, "throne-derived-local"), localaws.NewQueue(dataDir, "throne-ingest-jobs"), inference.New(envLatency()), nil)
	for {
		n, err := p.Drain(context.Background(), 10)
		if err != nil {
			log.Printf("processor drain error: %v", err)
		}
		log.Printf("processor drained %d messages", n)
		if *once {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func envLatency() time.Duration { return time.Second }
