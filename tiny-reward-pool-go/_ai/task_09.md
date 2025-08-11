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

```