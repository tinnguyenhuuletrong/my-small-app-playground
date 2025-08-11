# Task 08: Fix Reward Distribution Logic

## Target
- The reward distribution logic should correctly use item probabilities for selection.
- Items should be picked based on their assigned probability and their remaining quantity.
- Items with a quantity of 0 or less should not be selectable.
- All existing tests, including the distribution test, must pass after the fix.

## Iter 1

### Problem
The current implementation of the `PrefixSumSelector` and `FenwickTreeSelector` does not seem to respect the `Probability` field of the items in the reward pool. The distribution tests show a near-uniform distribution, whereas the configuration implies a weighted distribution.

The root cause is likely that the selectors are being built based on item `Quantity` instead of `Probability`. Additionally, when an item's quantity is exhausted, it is not correctly removed from the selection pool, leading to incorrect draws.

### Plan
1.  **Pool Method Analysis:** The methods in `internal/rewardpool/pool.go` (`GetItemRemaining`, `SelectItem`, `CommitDraw`, `RevertDraw`, `ApplyDrawLog`) have been reviewed. They correctly utilize the `ItemSelector` interface. The planned changes are confined to the selector implementations, and no changes are required in `pool.go`.

2.  **Refactor Selectors (`PrefixSumSelector` and `FenwickTreeSelector`):**
    -   Introduce a new field, `itemInfo map[string]types.PoolReward`, to each selector. This will store a copy of the full item details, allowing the selector to track both `Quantity` and `Probability` independently.
    -   Rename the existing `totalAvailable` field to `totalWeight` for clarity, as it will now represent the sum of probabilities of all *available* items.

3.  **Modify `Reset` Method in Both Selectors:**
    -   The `Reset` method will be updated to initialize the selector based on `Probability`.
    -   It will populate the new `itemInfo` map.
    -   It will iterate through the items, and for each item where `Quantity > 0`, it will add its `Probability` to the internal weight structure (`FenwickTree` or `prefixSums`) and the `totalWeight`.

4.  **Modify `Update` Method in Both Selectors:**
    -   This is the most critical change. The `Update` method will now correctly handle the distinction between `Quantity` and `Probability`.
    -   When called with a `delta` (e.g., -1 for a draw), it will update the `Quantity` in its internal `itemInfo` map.
    -   It will then check if the item's availability has changed (i.e., if `Quantity` has crossed the zero threshold).
        -   If `Quantity` drops to `0`, the item's `Probability` will be subtracted from the selector's weight structure, effectively removing it from future draws.
        -   If `Quantity` increases from `0` to a positive number (e.g., during a `RevertDraw`), the item's `Probability` will be added back, making it available for selection again.

5.  **Modify `GetItemRemaining` Method in Both Selectors:**
    -   This method will be updated to return the `Quantity` from the new `itemInfo` map.

6.  **Verification:**
    -   First, run `make test` to ensure all existing unit tests pass.
    -   Then, run `make distribution_test` to confirm that the reward distribution is now correctly weighted according to the item probabilities.

### Result
The `PrefixSumSelector` and `FenwickTreeSelector` were refactored to correctly use `Probability` for item selection and `Quantity` for availability tracking. A new `itemInfo` map was introduced to store full item details, and `totalAvailable` was renamed to `totalWeight`. The `Reset` method was updated to initialize selectors based on `Probability`, and the `Update` method was modified to handle `Quantity` changes and dynamically adjust item availability in the selection pool. The `GetItemRemaining` method was updated to reflect the actual remaining quantity. All unit tests passed, and the distribution tests confirmed correct weighted distribution.

## Iter 2

### Problem
The `rewardpool.Pool` struct still held a `catalog` field, which was redundant given that the `ItemSelector` now manages item quantities and probabilities. This created a duplication of state and unnecessary synchronization points. The `CommitDraw` and `ApplyDrawLog` methods in `Pool` also contained logic that directly manipulated item quantities, which should ideally be delegated entirely to the `ItemSelector`. Additionally, the `Pool` lacked a direct way to expose its current state (i.e., the catalog with updated quantities) without relying on internal fields.

### Plan
1.  **Remove Redundant `catalog` Field:** Eliminate the `catalog []types.PoolReward` field from the `rewardpool.Pool` struct.
2.  **Delegate Quantity Management to `ItemSelector`:**
    *   Modify `NewPool`, `Load`, and `LoadSnapshot` to initialize the `ItemSelector` directly with the provided catalog, removing any reliance on a `Pool`-held catalog.
    *   Remove the manual quantity decrementing loop from `CommitDraw` in `Pool`, as this is now handled by the `ItemSelector`'s internal state.
    *   Simplify `ApplyDrawLog` to directly call `p.selector.Update(itemID, -1)`, delegating the quantity update to the selector.
3.  **Introduce `State()` Method:** Add a new `State() []types.PoolReward` method to the `Pool` struct that returns the current catalog state by calling `p.selector.SnapshotCatalog()`.
4.  **Update Snapshotting:** In `SaveSnapshot`, retrieve the catalog for the snapshot directly from `p.selector.SnapshotCatalog()`.
5.  **Update CLI Usage:** Modify `cmd/cli/main.go` to use the new `pool.State()` method for printing the pool's current state.
6.  **Update Tests:** Adjust existing tests in `internal/rewardpool/pool_test.go` and `internal/rewardpool/pool_snapshot_test.go` to use `GetItemRemaining` and the new `State()` method, reflecting the removal of the direct `catalog` access.
7.  **Verification:** Run `make test` to ensure all unit tests pass after these changes.

### Result
The redundant `catalog` field was successfully removed from the `rewardpool.Pool` struct. Quantity management was fully delegated to the `ItemSelector`. `NewPool`, `Load`, and `LoadSnapshot` now initialize the `ItemSelector` directly. The manual quantity decrementing logic was removed from `CommitDraw` and `ApplyDrawLog` in `Pool`, with these operations now relying entirely on the `ItemSelector`. A new `State() []types.PoolReward` method was added to the `Pool` struct, providing a clean way to access the current state of the reward pool via the `ItemSelector`'s `SnapshotCatalog()` method. Snapshotting was updated to retrieve the catalog directly from `p.selector.SnapshotCatalog()`. The `cmd/cli/main.go` was updated to use `pool.State()` for displaying the pool's state. All relevant tests in `internal/rewardpool/pool_test.go` and `internal/rewardpool/pool_snapshot_test.go` were updated to use `GetItemRemaining` and the new `State()` method, confirming the successful refactoring and delegation of responsibilities. All unit tests passed after these changes.
