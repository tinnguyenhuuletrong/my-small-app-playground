CLI=cmd/cli/main.go
BIN=bin/tiny-reward-pool-cli

.PHONY: run build test proto-gen

run:
	go run $(CLI)

build:
	go build -o $(BIN) $(CLI)

check:
	go vet ./...

test:
	go test ./...

clean:
	go clean -testcache

bench:
	go test -bench=. ./cmd/bench/

distribution_test:
	go test -v ./cmd/distribution_test/...

proto-gen:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    pkg/rewardpool-grpc-service/rewardpool.proto
