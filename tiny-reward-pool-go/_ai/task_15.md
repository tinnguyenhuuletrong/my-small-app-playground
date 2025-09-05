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
    *   Update tests in `wal/`, `wal/storage/`, `recovery/`, and `actor/` to reflect the new header-only logic.
    *   Add a specific test for chained recovery across multiple rotated WAL files.
    *   Add a test for crash recovery (reading a WAL with `Status: Open` and ensuring logs are read correctly).

---

## Iter 2

### Plan

1.  **Update `internal/types/types.go`**:
    *   Modify `WALHeader` struct: remove `NextWALPath` and add `SeqNo uint64`.
    *   Add new constant `WALBaseName = "wal"`.

2.  **Update `internal/utils/utils.go`**:
    *   Remove `GenRotatedWALPath` function.
    *   Add `GetWALFiles() ([]string, error)` to scan `walDir`, find files matching `wal.ddd`, and return them sorted numerically.
    *   Add `GenNextWALPath() (string, uint64, error)` to determine the next sequence number and return the new path and sequence number.

3.  **Update `internal/wal/storage/*.go`**:
    *   Update `New...Storage` functions to accept a `seqNo uint64` and write it to the `WALHeader` on creation.
    *   Remove `Rotate` method from the `Storage` interface.
    *   Rename `Finalize` to `FinalizeAndClose` and simplify it to only update the header status to `Closed` before closing the file. The `isRotated` and `nextPath` parameters will be removed.
    *   The `Close` method will now just call `FinalizeAndClose`.

4.  **Update `internal/wal/wal.go`**:
    *   Remove the `Rotate` method.
    *   Update `NewWAL` to accept the `seqNo` and pass it to the storage constructor.
    *   The `Close` method will call `storage.FinalizeAndClose()`.

5.  **Update `internal/actor/actor.go`**:
    *   Rewrite `handleWALFull` logic:
        1.  Call `a.ctx.WAL.Close()` to finalize the current full WAL.
        2.  Use `a.ctx.Utils.GenNextWALPath()` to get the path and sequence number for the new WAL.
        3.  Create a new `WAL` instance using the new path and sequence number.
        4.  Replace the old WAL instance in the actor's context: `a.ctx.WAL = newWAL`.
        5.  Proceed with creating a snapshot and replaying pending logs to the new WAL instance.

6.  **Update `internal/recovery/recovery.go`**:
    *   Remove the `unrollWALChain` function.
    *   Rewrite `RecoverPool` and `RecoverPoolFromConfig`:
        1.  Use the new `utils.GetWALFiles()` to get a sorted list of all WAL files.
        2.  Iterate through the files, parsing each one and appending its logs to a single list.
        3.  After collecting all logs, find the last snapshot and determine which logs to replay.
        4.  Load the snapshot and replay the necessary logs.
        5.  The logic for deleting old WAL files must be removed.
        6.  The recovery process should also return the path of the last WAL file, so the application can continue using it. If the last file is full, a new one should be created.

7.  **Update `cmd/cli/main.go`**:
    *   The main application logic needs to be updated to handle the new return values from `recovery.RecoverPool`. It will receive the last WAL path and must decide whether to continue with it or create a new one if it's full.

8.  **Build and Verify (`make check`):**
    *   After implementing the code changes from steps 1-6, run `make check` to fix all compilation errors and warnings before proceeding to the testing phase.

9.  **Update Tests**:
    *   Update unit and integration tests across `wal/`, `actor/`, and `recovery/` packages to align with the new sequential WAL file management.
    *   Add specific tests for the WAL file scanning and sorting logic in `utils`.
    *   Add tests to verify the recovery process correctly replays logs from a sequence of WAL files.
    *   Ensure tests for the WAL rotation in the actor correctly create the next sequential file.