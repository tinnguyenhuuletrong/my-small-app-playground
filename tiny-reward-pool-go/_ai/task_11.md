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

### Iter 3: Implement Snapshot and WAL Rotation Logging
- **Problem:** Snapshot creation and WAL rotation are not logged in the WAL, which is essential for the new recovery logic.
- **Plan:**
    1. **`internal/rewardpool/pool.go`:**
        - Inject the `WAL` instance into the `Pool` struct.
        - In `SaveSnapshot`, after successfully saving a snapshot, call `wal.LogSnapshot` to record the event and its path in the WAL.
    2. **`internal/recovery/recovery.go`:**
        - In `RecoverPool`, when rotating the WAL, use `wal.LogRotate` to add an entry to the *new* WAL file that points to the *old*, archived WAL file.
    3. **Verification:**
        - After implementation, run `make check` to check for compile errors.
        - Run `make test` to ensure all existing tests pass.
