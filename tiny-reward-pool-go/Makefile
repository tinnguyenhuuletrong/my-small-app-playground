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

# Ramp VUs from 0 to 100 over 10s, stay there for 60s, then 10s down to 0.
bench-grpc:
	k6 run -u 0 -s 10s:100 -s 60s:100 -s 10s:0 ./cmd/grpc_bench/k6_grpc_script.js
