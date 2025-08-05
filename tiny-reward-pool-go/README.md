# Tiny Reward Pool Go

A high-performance, in-memory Reward Pool Service written in Go. Designed for rapid, reliable reward distribution with robust logging and snapshotting.

## Features
- In-memory reward pool with configurable item catalog
- Single-threaded processing model for low-latency, high-throughput
- Write-Ahead Log (WAL) for deterministic recovery
- Snapshot support for fast state restoration
- Atomic request ID generation
- CLI demo with graceful shutdown, periodic snapshot, and WAL rotation
- Modular design with testable interfaces

## Getting Started

### Prerequisites
- Go 1.20+

### Build & Run
```sh
make build   # Build CLI binary
make run     # Run CLI demo
make test    # Run all unit tests
```

### CLI Demo
- Recovery from WAL log
- Draws rewards in a loop
- Saves pool snapshot and rotates WAL every 5 seconds
- Graceful shutdown via SIGTERM/Ctrl+C

## Project Structure
- `cmd/cli/main.go` - CLI demo

## Documentation
- See `_ai/doc/agent_note.md` for quick project summary and onboarding

---
MIT License
