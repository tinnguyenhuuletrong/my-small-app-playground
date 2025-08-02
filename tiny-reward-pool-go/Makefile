CLI=cmd/cli/main.go
BIN=bin/tiny-reward-pool-cli

.PHONY: run build test

run:
	go run $(CLI)

build:
	go build -o $(BIN) $(CLI)

test:
	go test ./internal/...
