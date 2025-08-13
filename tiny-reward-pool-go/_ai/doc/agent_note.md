# Project Quick Summary

## Goal

- High-performance, in-memory Reward Pool Service in Go
- Implements PRD requirements: reward pool, WAL, config, CLI, and robust processing model

## Recent Updates (Aug 2025)

- **Task_09:** Refactored WAL architecture for extensibility and reliability.
  - Moved WAL log format to JSON Lines (JSONL) with typed `WalLogItem`/`WalLogDrawItem` and `LogType`/`LogError` enums.
  - Introduced `LogFormatter` (JSON, StringLine) and `Storage` (File, FileMMap) interfaces; `WAL` now composes these.
  - Added `ErrWALFull` and `Storage.CanWrite(size)` to safely trigger rotation on full write targets.
  - Centralized WAL rotation and snapshot creation in `processing.Processor` using a `Utils` interface (`GenRotatedWALPath`, `GenSnapshotPath`, `GetLogger`).
  - Updated `recovery.RecoverPool` to accept `formatter` and `utils`, replay WAL via formatter, write snapshot, and archive/rotate WAL.
  - CLI wiring updated to construct `DefaultUtils`, select formatter/storage, and inject into `RecoverPool`/`NewWAL`.
  - New tests for formatter/storage and integration test for WAL rotation; all tests passing.
  - Benchmarks updated; see `_ai/doc/bench.md` for Task 09 analysis and results.
- **Task_08:** Fixed a critical bug in the reward distribution logic. Refactored the `ItemSelector` implementations (`FenwickTreeSelector` and `PrefixSumSelector`) to correctly use `Probability` for weighted selection and `Quantity` for availability. Delegated all state management from the `rewardpool.Pool` to the `ItemSelector`, making it the single source of truth. Added a `distribution_test` to verify the correctness of the reward distribution.
- **Task_06:** Refactored the `rewardpool.Pool` to use a more efficient in-memory data structure for weighted random item selection, improving the performance of the `SelectItem` operation. Introduced an `ItemSelector` interface and implemented `FenwickTreeSelector` using a Fenwick Tree for O(log N) selection. Updated `Pool` to use this abstraction, ensuring `pendingDraws` are correctly accounted for to prevent over-draws. All related tests were updated and passed.
- **Task_05:** Refactored the `Processor.Draw` method to return a channel (`<-chan DrawResponse`) for a more idiomatic and developer-friendly API. Optimized the channel-based implementation using `sync.Pool` to reduce allocation overhead. Refactored benchmarks to accurately measure performance, with `bench_draw_apis_test.go` providing a direct comparison of callback vs. channel `Draw` methods, and other benchmarks using the recommended channel-based approach.
- **Task_04:** Refactored the core processing logic to be strictly WAL-first and transactional.
  - **Two-Phase Commit:** Introduced a staging area (`pendingDraws`) in the `RewardPool`. Operations now follow a select/stage (`SelectItem`) and then commit/revert (`CommitDraw`/`RevertDraw`) pattern. This ensures the WAL is written before any in-memory state is finalized.
  - **WAL Buffering:** The WAL now buffers log entries in memory. The `LogDraw` operation appends to this buffer, and a `Flush` operation writes the buffer to disk, improving performance by reducing I/O calls.
  - **Batch Processing:** The `Processor` now supports batching draws with a `flushAfterNDraw` setting. It accumulates staged draws and commits them in a batch after a configurable number of operations, further optimizing throughput.
- **Benchmarking:** Added new benchmark for memory-mapped WAL (mmap WAL) in `cmd/bench/bench_wal_mmap_test.go`.
- **Metrics Collection:** Benchmarks now compare No WAL, Real WAL, and Mmap WAL. Mmap WAL achieves ~2M draws/sec, much faster than file WAL, but slower than mock WAL.
- **Documentation:** `_ai/doc/bench.md` updated with new results, metrics table, and analysis. Mmap WAL is a strong middle ground for performance and durability.
- **Makefile Update:** Added `distribution_test` target to `Makefile` for running distribution tests.
- **New Tests:** Added `bench_selector_test.go` for benchmarking selector performance and `cmd/distribution_test` for distribution tests.

## Modules & Structure

- `internal/types/types.go`: Centralized type definitions and interfaces.
  - Data: `ConfigPool`, `PoolReward`, `WalLogItem`, `WalLogDrawItem`, `LogType`, `LogError`.
  - WAL abstractions: `LogFormatter` (Encode/Decode), `Storage` (Write/Flush/Close/Rotate/CanWrite), `WAL` interface.
  - Context/Utils: `Context` carries `WAL` and `Utils`. `Utils` provides `GetLogger`, `GenRotatedWALPath`, `GenSnapshotPath`.
  - Errors: `ErrWALFull`, `ErrWalBufferNotEmpty`, `ErrEmptyRewardPool`, `ErrPendingDrawsNotEmpty`, `ErrShutingDown`.
- `internal/wal/formatter/`: `JSONFormatter` (JSONL) and `StringLineFormatter` (compact string format) implementing `LogFormatter`.
- `internal/wal/storage/`: `FileStorage` (append/sync with capacity), `FileMMapStorage` (pre-sized mmap region) implementing `Storage`.
- `internal/wal/wal.go`: WAL composes `LogFormatter` and `Storage`, buffers `WalLogDrawItem`, `Flush` checks `CanWrite` and writes via storage, `ParseWAL` decodes via formatter.
- `internal/utils/utils.go`: `DefaultUtils` (logger, path generation for rotated WAL and snapshot). `ReadFileContent` utility (handles mmap zero padding). `test_utils.go` includes `MockUtils` and `MockWAL` for tests.
- `internal/processing/processing.go`: Single-threaded processor; transactional batching. On `ErrWALFull` from `Flush`, performs WAL rotation and snapshot via `Utils` and resumes.
- `internal/recovery/recovery.go`: Recovery now accepts `formatter` and `utils`, replays WAL via formatter, saves snapshot, and archives/removes old WAL using `utils.GenRotatedWALPath()`.
- `internal/rewardpool/pool.go`: Reward pool logic with staging (`pendingDraws`) and selector-driven state; provides snapshot load/save and apply log.
- `cmd/cli/main.go`: CLI demo wiring for `DefaultUtils`, formatter/storage selection, recovery, processing, and graceful shutdown.
- `cmd/bench`: Benchmarks for APIs, selectors, and WAL backends.

## Key Features

- All state changes are handled in a dedicated goroutine via a buffered channel.
- **WAL-First Principle:** Two-phase commit ensures WAL is written before in-memory state is mutated.
- **Batch Processing:** Multiple draws can be batched before flushing/committing to improve throughput.
- **Pluggable WAL:** Formatters (JSONL, StringLine) and storages (File, MMap) are interchangeable via interfaces.
- **Auto Rotation & Snapshotting:** Processor detects `ErrWALFull`, rotates WAL to an archive path, and creates a snapshot using `Utils`-provided paths; then continues processing.
- Request IDs generated safely with atomic operations.
- Channel-based `Draw` API for ergonomics; callback benchmark available for comparison.
- Pool supports save/load snapshot (JSON).
- WAL supports flush and rotation.
- WAL recovery replays WAL after snapshot, writes new snapshot, and archives rotated WAL on startup.

## Benchmarks:

Compare No WAL, Real WAL, and Mmap WAL for throughput, latency, and GC count. See `_ai/doc/bench.md` for detailed results and Task 09 analysis.

## Testing and Verification

- **Check syntax:** Run `make check` to check source and report compile errors / warining.
- **Unit Tests:** Run `make test` to execute all unit tests.
- **Distribution Test:** Run `make distribution_test` to verify that the reward distribution is correct according to the configured probabilities. This is a critical step to ensure the core logic is working as expected.
- **Benchmarks:** Run `make bench` to run all benchmark tests.

## Example Usage

- See `cmd/cli/main.go` for a demo of the complete system.
- See `cmd/bench/` for various WAL performance benchmarks.

## Next Steps

- Implement network/streaming WAL backends (e.g., Kafka/gRPC) using formatter/storage abstractions.
- Explore compact/binary serialization (e.g., Protobuf) to reduce WAL size and improve throughput.
- Add configuration-driven selection of formatter/storage and policy tuning (flush/rotation thresholds).
- Investigate parallel WAL replay and incremental snapshotting for faster recovery.
- Continue mmap/file storage tuning and selector/API ergonomics improvements.

---

This note is for quick onboarding and context transfer for future AI agents or developers.
