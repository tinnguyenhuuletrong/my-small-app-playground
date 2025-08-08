# Task 06: Performance Improvement for Reward Selection

## Target

Refactor the `rewardpool.Pool` to use a more efficient in-memory data structure for weighted random item selection, improving the performance of the `SelectItem` operation.

## Iter 01

### Plan

1.  **Define `ItemSelector` Interface:**
    *   Create a new interface, e.g., `internal/rewardpool/item_selector.go`, that defines the contract for item selection. This interface will abstract the underlying data structure (Fenwick Tree).
    *   It should include methods like:
        *   `Select(ctx *types.Context) (string, error)`: To select an item.
        *   `Update(itemID string, quantity int64)`: To update an item's available quantity. A positive value increases availability, a negative value decreases it.
        *   `Reset(catalog []types.PoolReward)`: To clear the selector's state and re-initialize it with a new catalog.
        *   `TotalAvailable() int64`: To get the total count of all available items.
        *   `GetItemRemaining(itemID string) int`: To get the remaining quantity of a specific item.

2.  **Implement `FenwickTreeSelector`:**
    *   Create a new struct, `internal/rewardpool/fenwick_tree_selector.go`, that implements the `ItemSelector` interface.
    *   This struct will encapsulate the `FenwickTree` (from `internal/utils`), an `itemIndex` map (for O(1) lookup of an item's index in the tree), and a `totalAvailable` counter.
    *   The `Select` method will use the Fenwick Tree's `Find` method for efficient random selection.
    *   The `Update` method will modify the Fenwick Tree when item quantities change.
    *   The `Reset` method will clear the selector's state and rebuild the Fenwick Tree based on the provided catalog.
    *   The `GetItemRemaining` method will query the internal state.

3.  **Refactor `rewardpool.Pool`:**
    *   Modify the `Pool` struct in `internal/rewardpool/pool.go` to *contain* an `ItemSelector` interface instead of directly embedding the Fenwick Tree and related fields. This promotes dependency injection and testability.
    *   Update `NewPool`, `Load`, and `LoadSnapshot` to initialize and populate the `ItemSelector` (e.g., by creating a `FenwickTreeSelector` instance and passing it the initial catalog).
    *   Rewrite `SelectItem` to:
        *   Delegate to the `ItemSelector.Select` method to pick an item.
        *   *After* selection, add the item to `p.pendingDraws`.
        *   Crucially, call `p.selector.Update(selectedItemID, -1)` to immediately decrement the quantity of the selected item in the `ItemSelector`'s Fenwick Tree. This prevents over-draws by marking the item as staged and temporarily unavailable.
    *   Update `CommitDraw` to:
        *   Iterate through `p.pendingDraws` and permanently decrement quantities in `p.catalog`.
        *   The `ItemSelector`'s state is already correct because `SelectItem` already decremented the quantity in the Fenwick Tree. No further `ItemSelector.Update` is needed for committed items.
        *   Clear `p.pendingDraws`.
    *   Update `RevertDraw` to:
        *   Iterate through `p.pendingDraws`.
        *   For each item, call `p.selector.Update(itemID, int64(stagedCount))` to increment its quantity back in the `ItemSelector`'s Fenwick Tree, making it available again.
        *   Clear `p.pendingDraws`.
    *   Update `ApplyDrawLog` to:
        *   Decrement the quantity in `p.catalog`.
        *   Call `p.selector.Update(itemID, -1)` to decrement its quantity in the `ItemSelector`'s Fenwick Tree, reflecting the committed draw.
    *   Adjust `GetItemRemaining` to delegate to the `ItemSelector.GetItemRemaining` method.

4.  **Create Fenwick Tree Utility (as previously planned):**
    *   Implement the `FenwickTree` data structure in `internal/utils/fenwick_tree.go` with `Add`, `Query`, and `Find` operations.
    *   Create `internal/utils/fenwick_tree_test.go` with comprehensive unit tests. (This part is already done, and the tests have been written).

5.  **Update Tests:**
    *   Modify `internal/rewardpool/pool_test.go` to use the `ItemSelector` interface for testing. This might involve creating mock `ItemSelector` implementations for isolated testing of the `Pool`'s logic.
    *   Ensure all existing tests pass to verify that the external behavior of the pool remains unchanged.

### Result

All unit tests passed after implementing the `ItemSelector` interface with `FenwickTreeSelector` and refactoring the `rewardpool.Pool` to use it. Key fixes included:
-   Ensuring `Pool` instances are always initialized via `NewPool` to properly set up the `selector`.
-   Handling edge cases in `FenwickTree.Add` and `FenwickTree.Query` for zero-sized trees.
-   Correctly managing `initialCatalog` copies in `pool_test.go` to prevent unintended modifications across test sections.

### Problem

None. Iteration 01 is complete.