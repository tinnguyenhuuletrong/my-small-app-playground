CLI=cmd/cli/main.go
BIN=bin/tiny-reward-pool-cli

.PHONY: run build test

run:
	go run $(CLI)

build:
	go build -o $(BIN) $(CLI)

check:
	go vet ./...

test:
	go test ./internal/...

clean:
	go clean -testcache

bench:
	go test -bench=. ./cmd/bench/

distribution_test:
	go test -v ./cmd/distribution_test/...