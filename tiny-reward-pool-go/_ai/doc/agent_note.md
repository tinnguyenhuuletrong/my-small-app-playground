# Project Quick Summary

## Goal
- High-performance, in-memory Reward Pool Service in Go
- Implements PRD requirements: reward pool, WAL, config, CLI, and robust processing model

## Recent Updates (Aug 2025)
- **Task_04:** Refactored the core processing logic to be strictly WAL-first and transactional.
  - **Two-Phase Commit:** Introduced a staging area (`pendingDraws`) in the `RewardPool`. Operations now follow a select/stage (`SelectItem`) and then commit/revert (`CommitDraw`/`RevertDraw`) pattern. This ensures the WAL is written before any in-memory state is finalized.
  - **WAL Buffering:** The WAL now buffers log entries in memory. The `LogDraw` operation appends to this buffer, and a `Flush` operation writes the buffer to disk, improving performance by reducing I/O calls.
  - **Batch Processing:** The `Processor` now supports batching draws with a `flushAfterNDraw` setting. It accumulates staged draws and commits them in a batch after a configurable number of operations, further optimizing throughput.
- **Benchmarking:** Added new benchmark for memory-mapped WAL (mmap WAL) in `cmd/bench/bench_wal_mmap_test.go`.
- **Metrics Collection:** Benchmarks now compare No WAL, Real WAL, and Mmap WAL. Mmap WAL achieves ~2M draws/sec, much faster than file WAL, but slower than mock WAL.
- **Documentation:** `_ai/doc/bench.md` updated with new results, metrics table, and analysis. Mmap WAL is a strong middle ground for performance and durability.

## Modules & Structure
- `internal/types/types.go`: Centralized type definitions and interfaces (`ConfigPool`, `PoolReward`, `WalLogItem`, `Context`, etc.)
- `internal/rewardpool/pool.go`: Reward pool logic. Now includes a staging area (`PendingDraws`) and uses a two-phase commit model (`SelectItem`, `CommitDraw`, `RevertDraw`).
- `internal/wal/wal.go`: WAL logging. Now implements in-memory buffering with a `Flush` mechanism.
- `internal/config/config.go`: Config loading, implements Config interface
- `internal/utils/utils.go`: Random selection logic, implements Utils interface
- `internal/processing/processing.go`: Single-threaded processing model. Orchestrates the new transactional and batch-processing flow.
- `internal/recovery/recovery.go`: WAL recovery logic, replays WAL log after snapshot, writes new snapshot, rotates WAL log
- `cmd/cli/main.go`: CLI demo, shows usage of all modules, supports graceful shutdown, snapshot, WAL rotation, and now uses recovery module for startup
- `cmd/bench`: Benchmark

## Key Features
- All state changes are handled in a dedicated goroutine via a buffered channel.
- **WAL-First Principle:** A two-phase commit process ensures the WAL is written before the in-memory state is mutated.
- **Batch Processing:** The system can batch multiple draw operations before flushing the WAL and committing the changes, significantly improving throughput.
- Request IDs generated safely with atomic operations.
- Draw operation returns request ID immediately, result via callback.
- Context struct used for dependency injection and testability.
- Pool supports save/load snapshot (JSON).
- WAL supports flush and rotation.
- WAL recovery module replays WAL log after snapshot, writes new snapshot, rotates WAL log on startup.
- CLI demonstrates loading snapshot on start, WAL recovery, periodic snapshot save, WAL rotation, and graceful shutdown.


## Benchmarks: 
Compare No WAL, Real WAL, and Mmap WAL for throughput, latency, and GC count. See `_ai/doc/bench.md`

## Testing
- Unit tests for all modules, including the new transactional and batching logic.
- All tests passing, including crash/restart recovery scenarios.
- **Benchmark tests** for all WAL variants.

## Example Usage
- See `cmd/cli/main.go` for a demo of the complete system.
- See `cmd/bench/` for various WAL performance benchmarks.

## Next Steps
- Continue iterating on features, add more tests, or extend modules as needed.
- Consider more robust WAL log parsing and error handling.
- Explore further mmap WAL optimizations.

---
This note is for quick onboarding and context transfer for future AI agents or developers.
