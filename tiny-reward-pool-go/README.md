# Tiny Reward Pool Go

A high-performance, in-memory Reward Pool Service written in Go. Designed for rapid, reliable reward distribution with robust logging and snapshotting.

## Features
- In-memory reward pool with configurable item catalog via YAML.
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

### Configuration
1.  Copy `samples/config.yaml` to the root of the project.
2.  Edit `config.yaml` to configure the reward pool, WAL settings, and other parameters.

### Build & Run
```sh
make build   # Build CLI binary
make run     # Run the interactive TUI
make test    # Run all unit tests
make distribution_test # Run distribution tests to verify reward probabilities
```

### Interactive TUI
The application features a rich interactive TUI with:
- A live-updating bar chart of reward item quantities.
- A command history and log viewer.
- REPL-like commands for interacting with the service (`h` for help).

## Project Structure
- `cmd/cli/main.go`: The main entry point for the interactive TUI.
- `internal/config`: Handles loading of `config.yaml`.
- `internal/actor`: Core actor model for processing and state management.
- `internal/wal`: Write-Ahead Log implementation.
- `internal/walstream`: WAL streaming for replication.
- `internal/rewardpool`: The reward pool implementation.
- `samples/config.yaml`: The main configuration file.

## Documentation
- See `_ai/doc/agent_note.md` for a quick project summary and onboarding.

---
MIT License