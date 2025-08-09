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
