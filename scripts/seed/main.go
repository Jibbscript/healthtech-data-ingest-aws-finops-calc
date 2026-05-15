package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jibbscript/throne-backend-poc/internal/store"
)

func main() {
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	if err := store.New(dataDir).SeedDemo(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("seeded user_demo/device_demo")
}
