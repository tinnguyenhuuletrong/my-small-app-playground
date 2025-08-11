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

2.  **Define Core Interfaces:** In a new file `internal/wal/storage.go`, define the core abstractions:
    *   **`LogFormatter` Interface:** To handle serialization and deserialization.
        ```go
        type LogFormatter interface {
            Encode(item types.WalLogDrawItem) ([]byte, error)
            Decode(data []byte) (types.WalLogDrawItem, error)
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
    *   **`JSONFormatter`:** Create a struct that implements `LogFormatter`. The `Encode` method will use `json.Marshal` and the `Decode` method will use `json.Unmarshal`. This will encapsulate the logic from Iteration 01.
    *   **`StringLineFormatter`:** Create a struct that implements `LogFormatter`. This will bring back the original `fmt.Sprintf` and `fmt.Sscanf` logic to represent the old format for benchmarking purposes.

4.  **Create Storage Implementation:**
    *   **`FileStorage`:** Create a struct that implements the `Storage` interface. It will manage the `os.File` handle, reading lines, and writing bytes. This will abstract all the direct file operations from the main `wal.go` file.

5.  **Refactor the `WAL` struct and its methods:**
    *   Modify the `WAL` struct in `internal/wal/wal.go` to be composed of the new interfaces:
        ```go
        type WAL struct {
            formatter LogFormatter
            storage   Storage
            buffer    [][]byte // Now stores pre-encoded data
        }
        ```
    *   Update `NewWAL` to accept `LogFormatter` and `Storage` interfaces as optional arguments (see `PoolOptional`. Let pick default LogFormatter=Json, Storage=File), allowing for dependency injection. 
    *   `LogDraw` will now use the `formatter` to encode the item and store the resulting `[]byte` in the buffer.
    *   `Flush` will iterate through the byte buffer and pass each entry to `storage.Write()`.
    *   The global `ParseWAL` function will be adapted to use the new components.

6.  **Update Application Wiring:**
    *   In `cmd/cli/main.go`, update the WAL initialization to create and inject the desired `JSONFormatter` and `FileStorage` into `NewWAL`.

7.  **Update Benchmarks:**
    *   Modify the benchmarks in `cmd/bench/` to construct `WAL` instances with both the `JSONFormatter` and the `StringLineFormatter` to allow for direct performance comparison between the two formats.

### Result

(To be filled in after implementation)

### Problem

(To be filled in after implementation)

