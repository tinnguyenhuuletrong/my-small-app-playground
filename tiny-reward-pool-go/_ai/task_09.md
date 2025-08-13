# Task 09: Optimize WAL Format and Efficiency

## Target

- Consider a more efficient format than raw text files.
- Review the interface for easier expansion later.
  - Stream WAL log over a network / Kafka stream.
  - Background job to sync archived WAL file, snapshot info to a Bucket.

## Iter 01

### Plan

1.  **Analysis Complete:** Reviewed `internal/wal/wal.go`, `internal/recovery/recovery.go`, and `internal/processing/processing.go`. The current implementation uses a fragile `fmt.Sprintf`/`fmt.Sscanf` approach and does not log error details.

2.  **Adopt JSONL Format:** Replace the current raw string format with JSONL (JSON Lines). Each log entry will be a self-contained JSON object on a new line.

3.  **Abstract the Log Entry Struct:** Make the `WalLogItem` more extensible by adding `Type` and `Error` fields. This allows for better-structured logging and easier future expansion.

    ```go
    // in internal/types/types.go
    type WalLogItem struct {
        Type      byte `json:"type"`      // e.g., "DRAW". Should have a Constant define map
        Error     byte   `json:"error,omitempty"` // To log failure reasons. Should have a Constant define map
    }

    type WalLogDrawItem struct {
        WalLogItem
        Type      byte `json:"type"`      // Type always = Draw
        RequestID uint64 `json:"request_id"`
        ItemID    string `json:"item_id,omitempty"`
        Success   bool   `json:"success"`
    }
    ```

4.  **Update `internal/types/types.go`:**
    *   Replace the existing `WalLogItem` struct with the new, more abstract version above.

5.  **Refactor `internal/processing/processing.go`:**
    *   Modify the `run()` loop to populate the new `WalLogItem` struct correctly.
    *   When a draw fails, the `Error` field of the log item will be populated with the error message from `p.pool.SelectItem(p.ctx)`.
    *   The `Type` field will be set to `"DRAW"`.

6.  **Refactor `internal/wal/wal.go`:**
    *   Modify `WAL.Flush()` to serialize the new `WalLogItem` struct to JSON using `json.Marshal`.
    *   Modify `ParseWAL()` to deserialize the JSONL data back into a `WalLogItem` struct using `json.Unmarshal`.

7.  **Verify `internal/recovery/recovery.go`:**
    *   No direct changes are expected, but its behavior must be verified with the new log format to ensure recovery logic still functions correctly.

8.  **Update Tests:**
    *   Update `internal/wal/wal_test.go` to test the new JSONL format and the more detailed log structure.
    *   Update `internal/processing/processing_test.go` to ensure it correctly handles the new logging logic.
    *   Ensure all other relevant tests, including `internal/recovery/recovery_test.go`, still pass.

9.  **Propose New Interface (for next iteration):**
    *   The plan to introduce a `WALWriter` and `WALReader` interface remains the focus for the next iteration to fully abstract the storage layer.

### Result

- Successfully migrated the WAL from a raw text format to a structured, extensible JSONL format.
- Implemented an abstract `WalLogItem` base struct and a specific `WalLogDrawItem` to allow for future log types, as per the plan.
- Added typed constants for log types (`LogType`) and errors (`LogError`) for efficiency and clarity.
- Refactored the `processing`, `wal`, `types`, and `recovery` packages to use the new abstract log structure.
- Updated and fixed all relevant unit tests, including mock objects, to align with the new interfaces and structs. All tests are passing.

### Problem

- The `WAL` interface in `types.go` currently specifies `LogDraw(item WalLogDrawItem)`. This is too specific. To support new log types in the future (e.g., `WalLogConfigChangeItem`), this method signature needs to be more generic.
- The `ParseWAL` function in `wal.go` is also specific to `WalLogDrawItem`. It needs to be refactored to handle different log types polymorphically.
- The core WAL implementation is still tied to a file-based system. The next iteration should focus on introducing `Reader` and `Writer` interfaces to abstract the underlying storage mechanism (file, network, etc.).

## Iter 02

### Plan

1.  **Goal:** Abstract the WAL's storage and formatting logic to allow for interchangeable backends (e.g., JSONL vs. String Line) and prepare for future storage mediums (e.g., network streams).

2.  **Define Core Interfaces:** In a new file `internal/types/types.go`, define the core abstractions:
    *   **`LogFormatter` Interface:** To handle serialization and deserialization.
        ```go
        type LogFormatter interface {
            // Batched encode. Should call in Flush
            Encode(items []types.WalLogDrawItem) ([]byte, error)

            // Batched decode. Should call in Parse
            Decode(data []byte) ([]types.WalLogDrawItem, error)
        }
        ```
    *   **`Storage` Interface:** To handle the physical writing, reading, and management of the log medium.
        ```go
        type Storage interface {
            WriteAll([][]byte) error
            ReadAll() ([][]byte, error)
            Flush() error
            Close() error
            Rotate(newPath string) error
        }
        ```

3.  **Create Formatter Implementations:**
    *   **`JSONFormatter`:** `internal/wal/formatter/json_formatter` Create a struct that implements `LogFormatter`. The `Encode` method will use `json.Marshal` and the `Decode` method will use `json.Unmarshal`. This will encapsulate the logic from Iteration 01.
    *   **`StringLineFormatter`:** `internal/wal/formatter/string_line_formatter` Create a struct that implements `LogFormatter`. This will bring back the original `fmt.Sprintf` and `fmt.Sscanf` logic to represent the old format for benchmarking purposes.

4.  **Create Storage Implementation:**
    *   **`FileStorage`:** `internal/wal/storage/file_storage.go` Create a struct that implements the `Storage` interface. It will manage the `os.File` handle, reading lines, and writing bytes. This will abstract all the direct file operations from the main `wal.go` file.
    *   **`FileMMapStorage`:** `internal/wal/storage/file_mmap_storage.go` Create a struct that implements the `Storage` interface. Copied logic as `cmd/bench/bench_wal_mmap_test.go`.

5.  **Refactor the `WAL` struct and its methods:**
    *   Modify the `WAL` struct in `internal/wal/wal.go` to be composed of the new interfaces:
        ```go
        type WAL struct {
            formatter LogFormatter
            storage   Storage
            buffer    [][]byte // Now stores pre-encoded data
        }
        ```
    *   Update `NewWAL` to accept `LogFormatter` and `Storage` interfaces as optional arguments (see `PoolOptional` as example. Let pick default LogFormatter=Json, Storage=File), allowing for configurable. 
    *   `LogDraw` will now use the `formatter` to encode the item and store the resulting `[]byte` in the buffer.
    *   `Flush` will Flush all of it as `WriteAll`.
    *   The global `ParseWAL` function will be adapted to use the new components.

6.  **Update Test:**
    * Make sure `make test` passed

7.  **Add new Test:**
    * Add test for storage and formater implement

8.  **Update Application Wiring:**
    *   In `cmd/cli/main.go`, update the WAL initialization to create and inject the desired `JSONFormatter` and `FileStorage` into `NewWAL`.

### Result

- Successfully abstracted the WAL's storage and formatting logic by introducing `LogFormatter` and `Storage` interfaces.
- Created `JSONFormatter` and `StringLineFormatter` implementations for `LogFormatter`.
- Created `FileStorage` and `FileMMapStorage` implementations for `Storage`.
- Refactored the `WAL` struct and its methods in `internal/wal/wal.go` to use the new interfaces.
- Updated all relevant tests (`internal/recovery/recovery_test.go` and `internal/wal/wal_test.go`) to correctly use and test the new interfaces and implementations. All tests are now passing.
- Updated the application wiring in `cmd/cli/main.go` to inject the `JSONFormatter` and `FileStorage` into `NewWAL` and `RecoverPool`.
- Updated the benchmark tests in `cmd/bench/bench_wal_test.go` to use the new `LogFormatter` and `Storage` interfaces.

### Problem

- The core WAL implementation is still tied to a file-based system. The next iteration should focus on introducing `Reader` and `Writer` interfaces to abstract the underlying storage mechanism (file, network, etc.).

## Iter 03

### Plan

1.  **Target**: Refine the `Storage` interface to be more practical and less abstract, avoiding over-engineering.

2.  **Problem Analysis**: The current `Storage` interface is too generic for its primary use case.
    *   `WriteAll([][]byte) error` is unnecessarily complex. The `LogFormatter` already encodes a batch of log items into a single `[]byte`. The storage layer should simply write this byte slice.
    *   `ReadAll() ([][]byte, error)` is only used by the `ParseWAL` function to read a WAL file from disk. This is a specific utility for WAL recovery, not a generic storage operation. It's better to move this into a dedicated utility function.

3.  **Refactoring Plan**:
    1.  **Simplify `Storage` Interface**: In `internal/types/types.go`, update the `Storage` interface:
        *   Change `WriteAll([][]byte) error` to `Write([]byte) error`.
        *   Remove `ReadAll() ([][]byte, error)`.
    2.  **Create WAL Reading Utility**: In `internal/utils/utils.go`, create a new utility function `ReadFileContent(path string) ([]byte, error)` to handle reading the entire content of a file. This will be used by the WAL parser.
    3.  **Update `Storage` Implementations**:
        *   In `internal/wal/storage/file_storage.go` and `internal/wal/storage/file_mmap_storage.go`, update the structs to implement the new `Storage` interface by replacing `WriteAll` with `Write` and removing `ReadAll`.
    4.  **Update `WAL` Logic**:
        *   In `internal/wal/wal.go`, modify the `Flush` method to pass the single `[]byte` from the formatter directly to `storage.Write()`.
        *   Update the `ParseWAL` function to use the new `utils.ReadFileContent()` to read the WAL file before decoding.
    5.  **Update Tests**:
        *   Adjust tests in `internal/wal/wal_test.go` and the storage implementation tests to reflect the interface changes.
    6.  **Verify and Fix**:
        *   Run `make test` to ensure all changes are correct and no regressions have been introduced.
        *   Run `make check` to find any compile errors or warnings and fix them.

### Result

- Successfully simplified the `Storage` interface by changing `WriteAll([][]byte) error` to `Write([]byte) error` and removing the `ReadAll` method.
- Moved the file reading logic to a new `ReadFileContent` function in the `internal/utils` package.
- Refactored `FileStorage` and `FileMMapStorage` to implement the new `Storage` interface.
- Updated the `WAL` implementation and its tests to work with the new interface.
- Fixed all compilation errors in `recovery` and `cmd/cli` that arose from the interface changes.
- All checks and tests are passing.

## Iter 04

### Plan

1.  **Target**: Centralize WAL rotation and snapshotting logic in the `Processor` to improve modularity and prevent write failures in `FileMMapStorage`.

2.  **Update `Storage` Interface**: In `internal/types/types.go`, add the `CanWrite(size int) bool` method to the `Storage` interface.
    ```go
    type Storage interface {
        Write([]byte) error
        CanWrite(size int) bool // New method
        Flush() error
        Close() error
        Rotate(newPath string) error
    }
    ```

3.  **Implement `CanWrite` Method**:
    *   **`FileStorage`**: The `CanWrite` method will return `true` if maximum file size not reached
    *   **`FileMMapStorage`**: The `CanWrite` method will return `true` only if the write size does not exceed the buffer's capacity.

4.  **Update `WAL` to Signal When Full**:
    *   In `internal/wal/wal.go`, modify the `Flush` method. Before writing, it will call `storage.CanWrite()`. If this returns `false`, `Flush` will return a new `ErrWALFull` error to signal that rotation is required.

5.  **Define `Utils` Interface for Lifecycle Management**:
    *   **Goal**: Abstract path generation and logging to make the `Processor` more testable and configurable.
    *   **Action**: Define a `Utils` interface in `internal/types/types.go`.
    *   **Definition**:
        ```go
        // Utils provides an interface for environment-specific operations like logging and path generation.
        type Utils interface {
            GetLogger() *slog.Logger
            GenRotatedWALPath() *string // Path for the archived WAL. nil means skip archiving.
            GenSnapshotPath() *string   // Path for the new snapshot. nil means skip snapshotting.
        }
        ```

6.  **Centralize Control in `Processor`**:
    *   **Action**:
        *   Update `processing.Processor` to accept the `Utils` interface on initialization.
        *   Refactor the `Processor.run()` method to handle the rotation and snapshot workflow.
    *   **Workflow**:
        1.  When `wal.Flush()` returns `ErrWALFull`:
        2.  The `Processor` calls `utils.GenRotatedWALPath()`.
        3. If the returned path is not `nil`, it calls `wal.Rotate()` with the new path. The Rotate method will implement the sequence you clarified: close -> move -> re-create.        
        4.  After rotation, the `Processor` calls `utils.GenSnapshotPath()`.
        5.  If the returned path is not `nil`, it calls `pool.Snapshot()` to create a new snapshot.

7.  **Simplify `cmd/cli`**:
    *   Remove all manual snapshotting and rotation logic (e.g., timers) from `cmd/cli/main.go`.
    *   The `main` function will now be responsible for creating a `Utils` implementation and injecting it into the `Processor`.

8.  **Update Tests**:
    *   Update unit tests for the `Processor` to use a mock `Utils` implementation.
    *   Add a new integration test in `internal/processing/processing_test.go` to verify the end-to-end rotation and snapshotting workflow.
    *   Run `make check` and `make test` to ensure all changes are correct.