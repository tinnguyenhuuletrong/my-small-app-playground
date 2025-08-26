# Tiny Reward Pool Go

A high-performance, in-memory Reward Pool Service written in Go. Designed for rapid, reliable reward distribution with robust logging and snapshotting.

<video controls src="doc/demo.mp4" title="Title"></video>

## Features
- In-memory reward pool with configurable item catalog via YAML.
- **gRPC Service**: Exposes `GetState` and `Draw` methods for programmatic access.
- **Unlimited Quantity**: Supports reward items with unlimited quantity.
- Interactive Terminal UI (TUI) for real-time monitoring and administration.
- Single-threaded processing model for low-latency, high-throughput.
- Write-Ahead Log (WAL) for deterministic recovery.
- Persistent request IDs that are unique and monotonically increasing across restarts.
- Asynchronous WAL streaming for replication.
- Snapshot support for fast state restoration.
- Modular design with testable interfaces.

## Getting Started

### Prerequisites
- Go 1.20+
- **Protocol Buffers**: `protoc` compiler.
- **gRPC Go**: `protoc-gen-go` and `protoc-gen-go-grpc`.
- **k6**: For load testing the gRPC service.

You can install the Go gRPC tools with:
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

### Configuration
1.  The application is configured via a YAML file. You can use the sample configuration at `samples/config.yaml`.
2.  To run the application, you need to pass the path to the configuration file using the `-config` flag.

### Build & Run
```sh
make build             # Build CLI binary
make run               # Run the interactive TUI
make test              # Run all unit tests
make proto-gen         # Generate Go code from .proto files
make distribution_test # Run distribution tests to verify reward probabilities
make bench-grpc        # Run k6 load test for the gRPC service
```

### Interactive TUI
The application features a rich interactive TUI with:
- A live-updating bar chart of reward item quantities.
- A command history and log viewer.
- REPL-like commands for interacting with the service (`h` for help).

### gRPC Service
The gRPC service can be enabled in the configuration file. It provides the following methods:
- `GetState`: Returns the current state of the reward pool.
- `Draw`: A bidirectional streaming RPC to draw items from the pool.

You can use `grpcurl` to interact with the service. See `_ai/ref/note_grpcurl.md` for examples.

## Project Structure
- `cmd/cli/main.go`: The main entry point for the interactive TUI.
- `internal/config`: Handles loading of `config.yaml`.
- `internal/actor`: Core actor model for processing and state management.
- `internal/wal`: Write-Ahead Log implementation.
- `internal/walstream`: WAL streaming for replication.
- `internal/rewardpool`: The reward pool implementation.
- `pkg/rewardpool-grpc-service`: The gRPC service implementation.
- `samples/config.yaml`: The main configuration file.

## Documentation
- See `_ai/doc/agent_note.md` for a quick project summary and onboarding.

---
MIT License
