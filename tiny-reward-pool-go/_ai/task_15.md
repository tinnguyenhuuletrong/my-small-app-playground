# Task 15: Enhance WAL File Structure

## Target

- Enhance WAL file structure to have a `<Header><Data>` layout.
- The header will contain all metadata, including version, status, and rotation information.
- The total size of the WAL file in the config should be respected.
- The existing recovery resilience model (fail-stop on parse error) is acceptable for this task.

---

## Iter 1

### Plan

1.  **Define Header Structure in `internal/types/types.go`:**
    *   Add the `WALHeader` struct to the `types.go` file.
    *   **`WALHeader` struct (fixed at 256 bytes):**
        *   `Magic uint32` (To identify it as our WAL file).
        *   `Version uint32`.
        *   `Status uint32` (`WALStatusOpen`, `WALStatusClosed`).
        *   `NextWALPath [200]byte` (Fixed-size array for the path, used on rotation).
        *   `Padding` for future use and to ensure the total size is 256 bytes.
    *   Define constants for the fixed `HeaderSize`, magic numbers, and statuses within the `types` package.

2.  **Update `internal/types/types.go`:**
    *   Remove the `WalLogRotateItem` struct.
    *   Remove the `LogRotate` method from the `WAL` interface.

3.  **Modify Storage Layer (`internal/wal/storage/*.go`):**
    *   **`New...Storage`:**
        *   On file creation, write the initial `WALHeader` with `Status: WALStatusOpen`.
        *   The write `offset` must start after the header (`HeaderSize`).
        *   `CanWrite` must account for the header reservation. For `mmap`, the data area is `total_size - HeaderSize`.
    *   **Add `Finalize(isRotated bool, nextPath string)` to `Storage` interface:**
        *   This method will handle the finalization sequence.
        *   Logic:
            1.  Flush any pending data to disk.
            2.  Seek back to the beginning of the file (offset 0).
            3.  Rewrite the `WALHeader`, updating the `Status` to `WALStatusClosed` and setting `NextWALPath` if `isRotated` is true.
            4.  Flush the header write to disk.
    *   **`Rotate` and `Close` methods:**
        *   The `Rotate` method will call `Finalize`, then close and rename the file.
        *   The `Close` method will call `Finalize` and then close the file.

4.  **Modify `internal/wal/wal.go`:**
    *   The `Rotate(path string)` method will call `storage.Rotate(path)`.
    *   The `Close()` method will call `storage.Close()`.
    *   The `LogRotate` method is removed.
    *   **Update `ParseWAL` function:**
        1.  It will now accept a file path.
        2.  It will open the file, read the `WALHeader` to get `HeaderSize`.
        3.  It will read the rest of the file content from `HeaderSize` to the end.
        4.  It will pass this data slice to `formatter.Decode`.

5.  **Modify `internal/actor/actor.go`:**
    *   No significant changes are expected here. The actor will continue to call `a.ctx.WAL.Rotate()` and `a.ctx.WAL.Close()`, and the WAL layer will abstract the new finalization logic.

6.  **Modify Recovery Logic (`internal/recovery/recovery.go`):**
    *   **Update `RecoverPool`'s main loop:**
        1.  Start with the primary WAL path and create a queue of paths to process.
        2.  Loop while the queue is not empty:
            a. Dequeue a path. Open the WAL file.
            b. Read its `WALHeader`.
            c. Parse the data section using the updated `wal.ParseWAL`. If parsing fails, the entire recovery fails.
            d. Replay the successfully parsed logs.
            e. Check `Header.Status`. If it's `Closed` and `NextWALPath` is set, enqueue the next path.

7.  **Build and Verify (`make check`):**
    *   After implementing the code changes from steps 1-6, run `make check` to fix all compilation errors and warnings before proceeding to the testing phase.

8.  **Update Tests:**
    *   Update tests in `wal/`, `wal/storage/`, `recovery/`, and `actor/` packages to reflect the new header-only logic.
    *   Add a specific test for chained recovery across multiple rotated WAL files.
    *   Add a test for crash recovery (reading a WAL with `Status: Open` and ensuring logs are read correctly).

