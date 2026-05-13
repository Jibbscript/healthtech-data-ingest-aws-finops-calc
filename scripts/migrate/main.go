package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dataDir := os.Getenv("THRONE_DATA_DIR")
	if dataDir == "" {
		dataDir = ".data"
	}
	path := filepath.Join(dataDir, "migrations-applied.txt")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte("000001_initial "+time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
	fmt.Println("applied local migration marker", path)
}
