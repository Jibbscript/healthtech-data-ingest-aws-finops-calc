package main

import (
	"fmt"
	"github.com/jibbscript/throne-backend-poc/internal/store"
	"os"
)

func main() {
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	if err := store.New(dataDir).SeedDemo(); err != nil {
		panic(err)
	}
	fmt.Println("seeded user_demo/device_demo")
}
