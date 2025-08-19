# Task 11: Enhance WAL with More Log Types

## Target
Currently, the WAL only logs `WalLogDrawItem`. This task will expand the WAL to include more log types to make the system more robust, covering item updates, snapshots, and WAL rotation.

To make it simple. We have a rule that begin of WAL file must be a `LogTypeSnapshot`

## Plan

### Iter 1: Refactor for Polymorphic Log Entries & Recovery Logic
- **Problem:** The current WAL implementation is tightly coupled to `WalLogDrawItem`, and the recovery process is not driven by the WAL content.
- **Plan:**
    1. **`internal/types/types.go`:**
        - Define new `LogType` constants: `LogTypeUpdate`, `LogTypeSnapshot`, and `LogTypeRotate`.
        - Same as `WalLogDrawItem` use embeded struct / interface to hold different log item types. Contain common attb `type` and `error` Runtime switch base on `type`, error handle by `error`
        - Define new structs for the new log types:
            - `WalLogUpdateItem{ItemID string, Quantity int, Probability int64}`
            - `WalLogSnapshotItem{Path string}`
            - `WalLogRotateItem{OldPath string, NewPath string}`
    2. **`internal/wal/formatter/` & `internal/wal/wal.go`:**
        - Update the `LogFormatter` interface and `JSONFormatter`, `StringLineFormatter` to handle `[]WalLogEntry`.
        - Update the `WAL` struct and methods to use `WalLogEntry` and add new logging functions (`LogUpdate`, `LogSnapshot`, `LogRotate`).
    3. **`internal/rewardpool/pool.go`:**
        - Add new methods to apply changes from the WAL: `ApplyUpdateLog(itemID string, quantity int, probability int64)`.
    4. **`internal/recovery/recovery.go`:**
        - Rework the `RecoverPool` function:
            - First line of WAL log must be `LogTypeSnapshot`
            - If a snapshot entry is found, it will load the pool state from the `Path` in that log entry.
            - Finally, it will replay the WAL entries that occurred *after* the loaded snapshot, using a type switch to call the appropriate `Apply...Log` method on the pool (`ApplyDrawLog`, `ApplyUpdateLog`, etc.).
    5. **`internal/actor/actor.go`:**:  
        - Implement an `Init()` method that checks if the WAL is empty. If it is, it creates an initial snapshot and flushes it to the WAL. This ensures the first entry in a new WAL file is always a `LogTypeSnapshot`.
    - **`internal/actor/system.go`:**
        - Call the `actor.Init()` method when creating a new system to ensure proper initialization of the WAL.
    6. **Verification:**
        - After implementation, run `make check` to check for compile errors.
        - Run `make test` to ensure all existing tests pass.

### Iter 2: Implement ConfigPool Item Update
- **Problem:** The reward pool does not currently support updating item properties, and the selector needs to handle these changes.
- **Plan:**
    1. **`internal/selector/`:**
        - Add a new method `UpdateItem(itemID string, quantity int, probability int64)` to the `ItemSelector` interface and its implementations. This will handle changes to both quantity and probability, which may require rebuilding internal structures.
    2. **`internal/rewardpool/pool.go`:**
        - Implement the `ApplyUpdateLog` method, which will call the selector's new `UpdateItem` method.
        - Add a user-facing `UpdateItem` method to the `Pool` that can be called during normal operation.
    3. **Integration:**
        - In a high-level component like `actor`, when an item update is requested, it should call the new `pool.UpdateItem` method and then log the change to the WAL using `wal.LogUpdate`.
    4. **Verification:**
        - After implementation, run `make check` to check for compile errors.
        - Run `make test` to ensure all existing tests pass.
- **Result:**
    - Successfully implemented the `UpdateItem` functionality across the selector, pool, and actor layers.
    - Added `UpdateItem` to the `ItemSelector` interface and implemented it in both `FenwickTreeSelector` and `PrefixSumSelector`.
    - Added `UpdateItem` to the `RewardPool` interface and `Pool` struct.
    - Integrated the update mechanism into the `ActorSystem` with a new `UpdateMessage`, allowing users to update item properties transactionally.
    - The actor now logs `WalLogUpdateItem` entries to the WAL after an update.
    - Added comprehensive unit tests for the new functionality at the selector, pool, and actor levels to ensure correctness.
    - All checks and tests passed successfully.

### Iter 3: Correct Snapshot and WAL Rotation Logging
-   **Problem:** The current `flush()` logic in `internal/actor/actor.go` discards pending WAL entries if a WAL rotation is triggered due to a full WAL file. This leads to data loss. The goal is to ensure all logs are persisted across WAL file rotations while maintaining the constraint that every WAL file must start with a snapshot.
-   **Plan:**
    1.  **Modify `internal/actor/actor.go`:**
        - Refactor the `flush()` method to correctly handle the `wal.ErrWALFull` error.
        - **Detailed Steps for `wal.ErrWALFull` scenario:**
            i.  **Preserve Pending State:** Before making any changes, create a temporary copy of the `a.pendingDraws` slices. This map holds wal log that have been applied in-memory but not yet successfully flushed to the WAL.
            ii. **Revert In-Memory Changes:** Call the `revertPending()` helper function. This will use the original `a.pendingDraws` map to revert the staged changes in the pool, bringing its state back to what is consistent with the last successful write in the (now full) WAL. This process will also clear `a.pendingDraws`.
            iii. **Create Snapshot and Rotate WAL:**
                - Create a new snapshot of the consistent, reverted pool state.
                - Call `a.wal.Rotate()` to archive the full WAL file and prepare a new, empty one.
            iv. **Initialize New WAL with Snapshot:**
                - Log the newly created snapshot as the first entry in the new WAL file using `a.wal.LogSnapshot()`.
            v.  **Re-apply and Re-log Operations:**
                - Iterate through the temporary copy of pending draws saved in step `i`.
                - For each draw operation, re-introduce it into the system:
                    - Re-stage the draw in the pool (`a.pool.StageDraw()`)
                    - Add the draw back to the `a.pendingDraws` map.
                    - Log the draw to the new WAL's in-memory buffer (`a.wal.LogDraw()`)
            vi. **Finalize the Flush:**
                - Call `a.wal.Flush()` again to write the re-logged pending operations to the new WAL file.
                - After a successful flush, call `commitPending()` to clear the `a.pendingDraws` map, finalizing the transactions.
    2.  **Verification:**
        - After implementation, run `make check` to check for compile errors.
        - Add a new unit test specifically designed to simulate the `wal.ErrWALFull` condition. This test should verify that the rotation occurs correctly and that no pending operations are lost in the process.
        - Run `make test` to ensure all existing and new tests pass.
