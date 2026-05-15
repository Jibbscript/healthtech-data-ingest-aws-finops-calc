package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	required := []string{"buf", "protoc-gen-go", "protoc-gen-go-grpc", "protoc-gen-grpc-gateway", "protoc-gen-openapiv2"}
	for _, tool := range required {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s is required; run scripts/install-proto-tools.sh: %w", tool, err)
		}
	}
	if err := os.RemoveAll("internal/gen/throne/v1"); err != nil {
		return err
	}
	if err := os.RemoveAll("api/openapi"); err != nil {
		return err
	}
	cmd := exec.Command("buf", "generate", "--path", "api/proto/throne/v1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	if _, err := os.Stat("api/openapi/throne.swagger.json"); err == nil {
		if err := os.Rename("api/openapi/throne.swagger.json", "api/openapi/throne.v1.swagger.json"); err != nil {
			return err
		}
	}
	expected := []string{
		"internal/gen/throne/v1/api.pb.go",
		"internal/gen/throne/v1/api.pb.gw.go",
		"internal/gen/throne/v1/api_grpc.pb.go",
		"internal/gen/throne/v1/capture.pb.go",
		"internal/gen/throne/v1/capture_grpc.pb.go",
		"internal/gen/throne/v1/inference.pb.go",
		"internal/gen/throne/v1/inference_grpc.pb.go",
		"internal/gen/throne/v1/types.pb.go",
		"api/openapi/throne.v1.swagger.json",
	}
	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("expected generated artifact %s: %w", path, err)
		}
	}
	fmt.Println("generated protobuf, gRPC, grpc-gateway, and OpenAPI artifacts")
	return nil
}
