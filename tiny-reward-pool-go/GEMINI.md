# Gemini Code Assistant Context

This document provides context for the Gemini Code Assistant to understand and effectively assist with the development of this project.

## Project Overview

This project is a high-performance, in-memory Reward Pool Service written in Go. It's designed for rapid and reliable reward distribution, featuring a robust logging and snapshotting mechanism to ensure data integrity and fast recovery.

The core of the service is a single-threaded, transactional processing model that ensures low-latency and high-throughput for reward distribution. A Write-Ahead Log (WAL) is implemented for deterministic recovery, with support for in-memory buffering and batch flushing to optimize I/O performance. The system enforces a strict WAL-first recovery model, where every WAL file must begin with a snapshot, ensuring a consistent and reliable state restoration.

The project is structured into several internal modules, including `config`, `processing`, `recovery`, `replay`, `rewardpool`, `selector`, `types`, `utils`, and `wal`. A command-line interface (CLI) demo is provided in the `cmd/cli` directory, which showcases the usage of all modules, including graceful shutdown, periodic snapshotting, and WAL rotation.

The project uses Go modules for dependency management, with `github.com/edsrzf/mmap-go` being a key dependency for memory-mapped file I/O in the WAL implementation.

## Building and Running

The project uses a `Makefile` for common development tasks.

*   **Check compile errors / warnining:**
    ```sh
    make check
    ```

*   **Build the CLI binary:**
    ```sh
    make build
    ```

*   **Run the CLI demo:**
    ```sh
    make run
    ```

*   **Run all unit tests:**
    ```sh
    make test
    ```

## Development Conventions

*   **Configuration:** The application is configured via a central `samples/config.yaml` file.
*   **Interactive TUI:** The primary interface is an interactive terminal application built with `bubbletea`, located in `cmd/cli`.
*   **Persistent Request IDs:** The actor model ensures that `requestID` is persistent across restarts by saving it in snapshots and recovering it from the WAL.
*   **WAL Streaming:** A `walstream` module with a dedicated `StreamingActor` provides asynchronous, non-blocking streaming of WAL entries for replication.
*   **Modular Design:** The project follows a modular design, with clear separation of concerns between different packages.
*   **Interfaces:** Interfaces are used to define contracts between different modules, promoting testability and loose coupling. A key example is the `ItemSelector` interface, which abstracts the underlying data structure for weighted random item selection. The `PoolReward.Probability` field has been updated to `int64` to align with the `ItemSelector` module's requirements. Implementations include `FenwickTreeSelector` and `PrefixSumSelector`.
*   **Testing:** Unit tests are provided for all key modules, and the project includes benchmark tests for performance-critical components like the WAL.
*   **Dependency Injection:** The `Context` struct is used for dependency injection, and the `rewardpool.Pool` accepts an `ItemSelector` to allow for different selection strategies.
*   **Concurrency and Transactional Integrity:** A single-threaded processing model with a dedicated goroutine and buffered channels is used to handle state changes. The `Actor.Draw` method now returns a channel (`<-chan DrawResponse`) for a more idiomatic and developer-friendly API. To ensure data integrity and adhere to the WAL-first principle, the system uses a two-phase commit process, with the `ItemSelector` being the source of truth for all reward item states (quantity and probability):
    1.  **Stage:** An operation (like a draw or item update) is first staged in memory. For draws, the `ItemSelector` immediately decrements the item's quantity in its internal state to prevent over-draws during the transaction. The selection is based on the item's `Probability`, while availability is checked against its `Quantity`.
    2.  **Log:** The operation is written to the Write-Ahead Log's in-memory buffer. The WAL now supports multiple log types, including draws, item updates, and snapshots.
    3.  **Commit/Revert:** When the WAL's buffer is flushed to disk, if the write is successful, the staged operations are committed (e.g., `CommitDraw`). Since the selector's state was already updated, this step finalizes the transaction. If the write fails, the operation is reverted (`RevertDraw`), and the `ItemSelector` is updated to restore the item's original state, ensuring consistency.
*   **WAL Rotation and Snapshots:** The system ensures that no data is lost when the WAL file becomes full. The process is driven by the `actor` and follows these steps:
    1.  **Detect Full WAL:** When a WAL flush fails with `ErrWALFull`, the rotation process begins.
    2.  **Preserve and Revert:** All pending (un-flushed) operations are preserved in a temporary list. The actor then reverts the in-memory state of the `ItemSelector` to match the last successfully written state in the WAL.
    3.  **Snapshot:** A new snapshot of the consistent, reverted state is created and saved to disk.
    4.  **Rotate and Initialize:** The full WAL is archived, and a new, empty WAL file is created. The first entry written to this new WAL *must* be a `WalLogSnapshotItem` pointing to the newly created snapshot.
    5.  **Replay and Re-log:** The preserved pending operations are then re-staged in the `ItemSelector` and re-logged to the new WAL's in-memory buffer.
    6.  **Final Flush:** The buffer is flushed to the new WAL file, securing the re-logged operations. This robust process guarantees that every WAL file is a self-contained, recoverable unit starting with a complete snapshot.
*   **Error Handling:** Errors are handled explicitly, and the CLI demo includes error handling for recovery and WAL operations.
*   **Logging:** The CLI demo includes basic logging to the console to provide visibility into the system's state and operations.

## AI Agent Working Procedure

The AI agent's workflow is designed to be systematic, iterative, and well-documented, ensuring context is maintained and tasks are completed efficiently.

**1. Onboarding & Context Gathering:**

*   **Primary Directive:** Before starting any task, the agent must first read the entire `_ai/doc/` directory to understand the project's goals, architecture, and established workflow.
*   **Key Documents:**
    *   `_ai/doc/requirement.md`: The Product Requirements Document (PRD) that defines the project's features and constraints. All work must align with this document.
    *   `_ai/doc/working_flow.md`: The high-level guide on how to approach tasks.
    *   `_ai/doc/agent_note.md`: A quick summary and handover note from the previous agent to get up to speed on the latest project state.
    *   `_ai/doc/bench.md`: A summary of performance benchmarks, which informs decisions related to performance-critical code.

**2. Task Execution (Iterative Development):**

The development process is broken down into tasks, which are documented in files like `_ai/task_01.md`, `_ai/task_02.md`, etc.

*   **Task Structure:** Each task is composed of one or more iterations (`Iter`).
*   **The Iteration Cycle (Plan -> Do -> Document):**
    1.  **Review Previous Work:** Read the current task file to understand the overall `Target` and the `Problem` identified in the previous iteration.
    2.  **Formulate a `Plan`:** Based on the target and previous problems, create a clear and concise plan for the current iteration. This plan should be documented under the `Plan` section for the current `Iter`.
    3.  **Implement the Plan:** Execute the plan by writing or modifying the Go code and corresponding tests. The agent must adhere to the project's existing structure and conventions.
    4.  **Verify with Tests:** After implementation, run the project's tests to ensure all changes are correct and have not introduced regressions. The primary command for this is `go test ./internal/...`. For benchmarks, use `make bench`.
    5.  **Document the `Result`:** Once the implementation is complete and verified, document the outcome under the `Result` section of the current iteration.
    6.  **Identify `Problem`s:** If any limitations, bugs, or areas for improvement are found, document them in the `Problem` section. This sets the stage for the next iteration.

**3. Core Principles:**

*   **Iterative Refinement:** The agent works in small, incremental steps. Problems found in one iteration are addressed in the plan for the next.
*   **Compile-Driven:** Use `make check` after finish an implementation. Make sure fix all compile errors / warning before do a test `make test`
*   **Test-Driven:** Every functional component or module must be accompanied by unit tests.
*   **Documentation is a Priority:** The agent is responsible for keeping the `_ai` directory updated. This includes filling out the `Plan`, `Result`, and `Problem` sections for each iteration and updating benchmark documents when relevant.
*   **Performance-Aware:** When working on performance-sensitive areas, the agent should create and run benchmarks, analyze the results, and use the data to guide implementation choices.
