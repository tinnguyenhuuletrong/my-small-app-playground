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