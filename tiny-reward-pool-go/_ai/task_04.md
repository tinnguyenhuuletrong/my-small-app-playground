<!-- Read _ai/doc/*.md first -->

# Target
- Refactor the `RewardPool` and `Processor` to improve abstraction, prepare for future batch/transactional processing, and strictly adhere to the WAL-first principle.

## Iter 01
### Problem
The current implementation has several limitations that will hinder future development:
1.  **Violation of WAL-First Principle:** The `processing` loop modifies the pool's in-memory state (`pool.Draw()`) *before* writing to the Write-Ahead Log. The project's requirements state that the WAL entry must be written first.
2.  **No Preparation for Batching:** As correctly pointed out, the current `Draw` logic does not account for multiple requests within a single transaction or batch. This creates a race condition where the pool could promise more items than it has, leading to overdrawing.
3.  **Lack of Abstraction:** The `rewardpool.Draw()` function handles both item selection and state mutation, making the logic rigid and hard to test or extend.

### Plan
To address these issues, we will refactor the core logic to introduce a staging area (`pendingDraws`) and a two-phase commit process (select/stage, then commit/revert). This will make the system robust, transactional, and future-proof.

1.  **Introduce a Staging Area in `RewardPool`:**
    *   Add a `pendingDraws map[string]int` to the `rewardpool.Pool` struct. This map will track items that have been selected but not yet fully committed to the WAL.

2.  **Refactor `RewardPool` Interface and Implementation:**
    *   Replace the old `Draw` method with `SelectItem()`. This method will:
        *   Check if an item is available by comparing the main `Catalog` quantity against the `pendingDraws` count (`Catalog.Quantity - pendingDraws[itemID] > 0`).
        *   If available, increment the item's count in `pendingDraws` to stage it.
        *   Return a *copy* of the selected item.
    *   Create a new method: `CommitDraw(itemID string)`. This method will:
        *   Decrement the item's quantity in the main `Catalog`.
        *   Decrement the item's count in `pendingDraws`.
    *   Create a new method: `RevertDraw(itemID string)`. This method will be used if a WAL write fails. It will simply decrement the item's count in `pendingDraws` to release the staged item.

3.  **Update the `Processor` Loop:**
    *   Modify the `run()` loop in `internal/processing/processing.go` to orchestrate the new transactional flow:
        1.  Call `pool.SelectItem()` to stage a draw and get the outcome.
        2.  If an item was selected, attempt to `ctx.WAL.LogDraw()`.
        3.  If the WAL write is **successful**, call `pool.CommitDraw()` to finalize the state change.
        4.  If the WAL write **fails**, call `pool.RevertDraw()` to cancel the staged draw.
        5.  Invoke the callback with the final response.

4.  **Update Interfaces and Tests:**
    *   Update the `types.RewardPool` interface in `internal/types/types.go` to reflect the new `SelectItem`, `CommitDraw`, and `RevertDraw` methods.
    *   Update all unit tests for the `rewardpool` and `processing` packages to verify the new transactional logic, including success and failure paths.

### Result

- The RewardPool and Processor were refactored to strictly follow the WAL-first principle and support transactional, batch-ready logic.
- A staging area (`PendingDraws`) was added to the pool, and all draw operations now use a two-phase commit: select/stage, then commit/revert.
- The random selection now only considers available items, ensuring correct probability and avoiding overdrawing.
- All related interfaces and tests were updated. Unit tests for both rewardpool and processing pass, verifying success and WAL failure paths.
- The system is now robust, future-proof, and ready for batch/transactional extensions.


## Iter 02
### Problem
The current implementation has several limitations that will hinder future development:
1. the wall should have a buffer. write file on `Flush` instead of write to file everytime `LogDraw`

### Plan

To address the WAL buffering requirement and improve performance:

1. **Add a Buffer to WAL:**
   - Introduce an in-memory buffer (e.g., a slice of log lines or WalLogItem) in the WAL struct.
   - `LogDraw` should append to the buffer instead of writing directly to the file.

2. **Implement Flush Logic:**
   - `Flush` should write all buffered log entries to the file in one batch and then clear the buffer.
   - Ensure `Flush` is called at appropriate points (e.g., periodic flush, graceful shutdown, or after a batch/transaction).

3. **Update WAL Interface and Usage:**
   - Update the WAL interface in `types.go` to clarify the buffered behavior.
   - Update all usages of WAL in the codebase to expect buffered logging and explicit flushes.

4. **Update Tests:**
   - Add/modify unit tests in `wal_test.go` to verify that log entries are only written to disk after `Flush` is called.
   - Test edge cases: flush on shutdown, flush after batch, and flush with empty buffer.

5. **Document Buffering Behavior:**
   - Update documentation to describe the new WAL buffering and flush semantics for future maintainers.


## Iter 03
### Problem
The current implementation has several limitations that will hinder future development:
1. the rewardpool module `CommitDraw` and `RevertDraw` should update with no param. it will consume / discard all `pendingDraws` it has
2. the processing module should add params. `flushAfterNDraw`. So every n draw -> it Flush and `CommitDraw` or `RevertDraw`

### Plan

To address batch commit/revert and periodic WAL flush:

1. **Refactor RewardPool CommitDraw/RevertDraw:**
   - Change `CommitDraw()` and `RevertDraw()` to take no parameters.
   - `CommitDraw()` will consume all staged draws in `PendingDraws`, decrementing quantities in the catalog and clearing the staging map.
   - `RevertDraw()` will discard all staged draws in `PendingDraws` without modifying the catalog.

2. **Add flushAfterNDraw to Processor:**
   - Add a `flushAfterNDraw` parameter to `Processor`.
   - Track the number of staged draws since the last flush/commit.
   - After every N draws, call `WAL.Flush()` and then `pool.CommitDraw()` (or `pool.RevertDraw()` on error).
   - Reset the counter after each flush/commit.

3. **Update Processor Logic:**
   - In the draw loop, accumulate staged draws and only flush/commit after N draws or on shutdown.
   - Ensure correct error handling: if any WAL write fails, revert all staged draws.

4. **Update Interfaces and Tests:**
   - Update the `RewardPool` and `Processor` interfaces to reflect the new batch commit/revert logic.
   - Add/modify unit tests to verify batch commit, batch revert, and flush-after-N-draws behavior.
   - Test edge cases: partial batch, shutdown flush, error handling.

5. **Document Batch Semantics:**
   - Update documentation to describe batch commit/revert and periodic WAL flush for maintainers and future development.