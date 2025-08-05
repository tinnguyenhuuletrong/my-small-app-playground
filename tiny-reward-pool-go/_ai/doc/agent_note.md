# Project Quick Summary

## Goal
- High-performance, in-memory Reward Pool Service in Go
- Implements PRD requirements: reward pool, WAL, config, CLI, and robust processing model

## Recent Updates (Aug 2025)
- **Benchmarking:** Added new benchmark for memory-mapped WAL (mmap WAL) in `cmd/bench/bench_wal_mmap_test.go`.
- **Metrics Collection:** Benchmarks now compare No WAL, Real WAL, and Mmap WAL. Mmap WAL achieves ~2M draws/sec, much faster than file WAL, but slower than mock WAL.
- **Documentation:** `_ai/doc/bench.md` updated with new results, metrics table, and analysis. Mmap WAL is a strong middle ground for performance and durability.
- **Task_03 Iter 02:** Implemented mmap WAL, collected metrics, and documented findings. Plan for improvement now includes WAL format optimization, rotation, and more mmap strategies.

## Modules & Structure
- `internal/types/types.go`: Centralized type definitions and interfaces (`ConfigPool`, `PoolReward`, `WalLogItem`, `Context`, etc.)
- `internal/rewardpool/pool.go`: Reward pool logic, uses types and interfaces, supports save/load snapshot
- `internal/wal/wal.go`: WAL logging, implements WAL interface, supports flush and rotation
- `internal/config/config.go`: Config loading, implements Config interface
- `internal/utils/utils.go`: Random selection logic, implements Utils interface
- `internal/processing/processing.go`: Single-threaded processing model, uses atomic for request IDs, all operations via injected `Context`
- `internal/recovery/recovery.go`: WAL recovery logic, replays WAL log after snapshot, writes new snapshot, rotates WAL log
- `cmd/cli/main.go`: CLI demo, shows usage of all modules, supports graceful shutdown, snapshot, WAL rotation, and now uses recovery module for startup
- `cmd/bench`: Benchmark

## Key Features
- All state changes (draw, WAL, quantity) handled in a dedicated goroutine
- Requests sent via buffered channel, processed sequentially
- Request IDs generated safely with atomic operations
- Draw operation returns request ID immediately, result via callback
- WAL entry written before memory update and response
- Context struct used for dependency injection and testability
- Pool supports save/load snapshot (JSON)
- WAL supports flush and rotation
- WAL recovery module replays WAL log after snapshot, writes new snapshot, rotates WAL log on startup
- CLI demonstrates loading snapshot on start, WAL recovery, periodic snapshot save, WAL rotation, and graceful shutdown


## Benchmarks: 
Compare No WAL, Real WAL, and Mmap WAL for throughput, latency, and GC count. See `_ai/doc/bench.md`

## Testing
- Unit tests for RewardPool, WAL, Config, Utils, Processing, and Recovery modules
- All tests passing, including crash/restart recovery scenarios
- **Benchmark tests** for all WAL variants

## Example Usage
- See `cmd/cli/main.go` for demo: draws rewards in a loop, prints results, supports graceful shutdown, snapshot save/load, WAL rotation, and WAL recovery on startup
- See `cmd/bench/bench_wal_mmap_test.go` for mmap WAL benchmark

## Next Steps
- Continue iterating on features, add more tests, or extend modules as needed
- Consider more robust WAL log parsing, error handling, and support for transactional operations
- Explore further mmap WAL optimizations

---
This note is for quick onboarding and context transfer for future AI agents or developers.