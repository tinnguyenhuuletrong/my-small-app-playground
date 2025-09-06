# Project Quick Summary

## Goal

- High-performance, in-memory Reward Pool Service in Go
- Implements PRD requirements: reward pool, WAL, config, CLI, gRPC service, and robust processing model

## Recent Updates (Sep 2025)

- **Task_15:** Enhanced WAL file structure and recovery process.
  - **WAL Header:** Implemented a `<Header><Data>` layout for WAL files. The header contains metadata such as a magic number, version, status (`Open`/`Closed`), and a sequence number.
  - **Sequential WAL Files:** WAL rotation now creates sequential files (e.g., `wal.000`, `wal.001`, `wal.002`) instead of timestamped files. This simplifies WAL management.
  - **Simplified Recovery:** The recovery logic now scans the working directory for all `wal.xxx` files, sorts them numerically, and replays them in order to restore the state.
  - **Snapshot Integrity:** Snapshots now include a `SHA256` hash of the item catalog to ensure data integrity.
- **Task_14:** Added a gRPC service and support for unlimited quantity items.
  - **gRPC Service:** Implemented a gRPC service in `pkg/rewardpool-grpc-service` that exposes `GetState` and `Draw` methods. The service can be enabled and configured in `config.yaml`.
  - **Unlimited Quantity:** Added support for reward items with unlimited quantity by setting the `quantity` field to `-1`. The core logic in `rewardpool` and `selector` has been updated to handle this.
  - **New Tools:** Added `k6` for gRPC benchmarking and `protoc` for protobuf code generation.
- **Task_13:** Evolved the application into a configurable service with an interactive TUI.
  - **YAML Configuration:** Moved from hardcoded paths and JSON to a `config.yaml`-based setup, allowing configuration of the working directory, reward pool, and WAL parameters (`max_file_size_kb`, `max_request_buffer_size`, `formatter`, `flush_after_n_draw`).
  - **Interactive TUI:** Replaced the basic CLI with a full-featured, interactive terminal user interface (TUI) built with `bubbletea`.
  - **Live Dashboard:** The TUI provides a live-updating dashboard with a bar chart showing item quantities, a history of commands, and a debug log view.
  - **REPL-like Commands:** The TUI supports commands like `draw`, `update`, `status`, `help`, and `reload`, providing administrative control over the running service.
  - **Archived Old CLI:** The previous CLI demo was archived to `cmd/cli_v1`.
- **Task_12:** Implemented persistent request IDs and a WAL streaming mechanism.
  - **Persistent Request ID:** The `requestID` is now persistent across restarts. It is saved as part of the pool snapshot and correctly restored from both the snapshot and the WAL, ensuring it is always unique and increasing.
  - **WAL Streaming for Replication:** Introduced a new `internal/walstream` module and a dedicated `StreamingActor` to asynchronously stream WAL entries to a target, laying the groundwork for replica synchronization. This is designed to be non-blocking.
- **Task_11:** Made the WAL the source of truth by introducing multiple log types and re-architecting the recovery and actor systems.
  - **Enhanced WAL:** The WAL now supports multiple log types (`LogTypeUpdate`, `LogTypeSnapshot`, `LogTypeRotate`) beyond just `LogTypeDraw`, with a polymorphic `WalLogEntry` interface.
  - **Item Updates:** The system now supports transactional updates to item quantity and probability via the actor model.
  - **`replay` Module:** Created a new `internal/replay` module to centralize WAL replay logic, removing duplication from `recovery` and `actor` modules.
  - **WAL-Driven Recovery:** The recovery process is now strictly driven by the WAL. It requires the WAL to start with a snapshot, loads the state from it, and then replays subsequent logs.
  - **Robust WAL Rotation:** The actor's `flush` logic now gracefully handles `ErrWALFull` by preserving pending operations, reverting state, creating a snapshot, rotating the WAL, and then re-applying the pending operations to the new WAL file, preventing data loss.
- **Task_09:** Refactored WAL architecture for extensibility and reliability.
  - Moved WAL log format to JSON Lines (JSONL) with typed `WalLogItem`/`WalLogDrawItem` and `LogType`/`LogError` enums.
  - Introduced `LogFormatter` (JSON, StringLine) and `Storage` (File, FileMMap) interfaces; `WAL` now composes these.
- **Task_08:** Fixed a critical bug in the reward distribution logic. Refactored the `ItemSelector` implementations (`FenwickTreeSelector` and `PrefixSumSelector`) to correctly use `Probability` for weighted selection and `Quantity` for availability.

## Modules & Structure

- `internal/config`: Loads and parses the `config.yaml` file.
- `internal/actor`: Contains the core `RewardProcessorActor` for state management and the `StreamingActor` for WAL replication.
- `internal/walstream`: Defines the `WALStreamer` interface and provides `NoOpStreamer` and `LogStreamer` implementations.
- `internal/rewardpool`: Manages the reward items, now with snapshotting logic that includes the `last_request_id`.
- `internal/recovery`: Recovers state from snapshots and WAL files, now also restoring the `last_request_id`.
- `pkg/rewardpool-grpc-service`: The gRPC service implementation.
- `cmd/cli`: The new interactive TUI application.
  - `tui/model.go`: The main `bubbletea` model, handling UI state, commands, and rendering.
- `cmd/cli_v1`: The archived, non-interactive CLI.
- `samples/config.yaml`: The central configuration file for the application.

## Key Features

- **Sequential WAL:** WAL files are created sequentially (`wal.000`, `wal.001`, etc.) with headers for metadata, improving traceability and recovery.
- **Snapshot Integrity:** Snapshots include a `SHA256` hash for verifying data integrity.
- **gRPC Service:** Exposes `GetState` and `Draw` methods for programmatic access.
- **Unlimited Quantity:** Supports reward items with unlimited quantity.
- **YAML Configuration:** All major settings are now managed in a single `config.yaml` file.
- **Interactive TUI:** A rich, interactive terminal interface for managing and observing the reward pool in real-time.
- **Persistent Request IDs:** Guarantees that request IDs are unique and monotonically increasing even after restarts.
- **WAL Streaming:** Asynchronously streams WAL logs to an external target, enabling real-time replication.
- **WAL-First Principle:** Two-phase commit ensures WAL is written before in-memory state is mutated.
- **Pluggable WAL:** Formatters (JSONL, StringLine) and storages (File, MMap) are interchangeable via interfaces.
- **Auto Rotation & Snapshotting:** The processor automatically handles WAL file rotation and snapshot creation when the WAL is full.

## Testing and Verification

- **Check syntax:** Run `make check` to check source and report compile errors / warining.
- **Unit Tests:** Run `make test` to execute all unit tests.
- **Distribution Test:** Run `make distribution_test` to verify that the reward distribution is correct according to the configured probabilities.
- **Benchmarks:** Run `make bench` to run all benchmark tests.
- **Proto Generation:** Run `make proto-gen` to generate Go code from `.proto` files.
- **gRPC Benchmarking:** Run `make bench-grpc` to run the k6 load test for the gRPC service.

## Example Usage

- Configure the service in `samples/config.yaml`.
- Run `make run` to start the interactive TUI.
- Use commands like `h` (help), `d` (draw), and `u` (update) within the TUI.

## Next Steps

- Add more advanced TUI features, such as filtering and searching the command history.
- Explore more sophisticated WAL streaming backends like Kafka or gRPC streams.
- Enhance configuration validation and error handling.