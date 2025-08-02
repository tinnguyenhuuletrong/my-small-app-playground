# Project Quick Summary

## Goal
- High-performance, in-memory Reward Pool Service in Go
- Implements PRD requirements: reward pool, WAL, config, CLI, and robust processing model

## Modules & Structure
- `internal/types/types.go`: Centralized type definitions and interfaces (`ConfigPool`, `PoolReward`, `WalLogItem`, `Context`, etc.)
- `internal/rewardpool/pool.go`: Reward pool logic, uses types and interfaces, supports save/load snapshot
- `internal/wal/wal.go`: WAL logging, implements WAL interface, supports flush and rotation
- `internal/config/config.go`: Config loading, implements Config interface
- `internal/utils/utils.go`: Random selection logic, implements Utils interface
- `internal/processing/processing.go`: Single-threaded processing model, uses atomic for request IDs, all operations via injected `Context`
- `cmd/cli/main.go`: CLI demo, shows usage of all modules, supports graceful shutdown, snapshot, WAL rotation

## Key Features
- All state changes (draw, WAL, quantity) handled in a dedicated goroutine
- Requests sent via buffered channel, processed sequentially
- Request IDs generated safely with atomic operations
- Draw operation returns request ID immediately, result via callback
- WAL entry written before memory update and response
- Context struct used for dependency injection and testability
- Pool supports save/load snapshot (JSON)
- WAL supports flush and rotation
- CLI demonstrates loading snapshot on start, periodic snapshot save, WAL rotation, and graceful shutdown

## Testing
- Unit tests for RewardPool, WAL, Config, Utils, and Processing modules
- All tests passing

## Example Usage
- See `cmd/cli/main.go` for demo: draws rewards in a loop, prints results, supports graceful shutdown, snapshot save/load, WAL rotation

## Next Steps
- Continue iterating on features, add more tests, or extend modules as needed

---
This note is for quick onboarding and context transfer for future AI agents or developers.
